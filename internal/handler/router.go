package handler

import (
	"bytes"
	"fmt"
	"io"
	"llm-router/internal/config"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metric for tracking total requests per provider and result status
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_router_requests_total",
		Help: "Total number of requests handled by the router",
	}, []string{"provider", "status"})

	// Metric for tracking request latency
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "llm_router_request_duration_seconds",
		Help: "Latency of requests per provider",
	}, []string{"provider"})
)

type RouterHandler struct {
	Cfg    *config.RouterConfig
	Client *http.Client
}

func (h *RouterHandler) HandleChat(c *gin.Context) {
	clientIP := c.ClientIP()
	log.Printf("[INFO] Received chat request from %s, path=%s", clientIP, c.Request.URL.Path)

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read request body from %s: %v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	log.Printf("[DEBUG] Request body size: %d bytes from %s", len(bodyBytes), clientIP)

	maxAttempts := 3
	var lastErr error

	for i := 0; i < maxAttempts; i++ {
		provider := h.Cfg.GetProvider()
		if provider == nil {
			log.Printf("[WARN] Attempt %d/%d: No provider available from %s", i+1, maxAttempts, clientIP)
			continue
		}
		log.Printf("[INFO] Attempt %d/%d: Routing request to provider=%s, url=%s from client=%s", i+1, maxAttempts, provider.ID, provider.URL, clientIP)

		start := time.Now()
		fullURL := provider.URL + "/v1/chat/completions"
		req, _ := http.NewRequest("POST", fullURL, bytes.NewBuffer(bodyBytes))
		req.Header = c.Request.Header.Clone()

		resp, err := h.Client.Do(req)
		duration := time.Since(start)

		if err == nil && resp.StatusCode < 500 {
			// Record Success Metrics
			requestDuration.WithLabelValues(provider.ID).Observe(duration.Seconds())
			httpRequestsTotal.WithLabelValues(provider.ID, "success").Inc()

			log.Printf("[INFO] Request successful: provider=%s, status=%d, duration=%.2fms, client=%s",
				provider.ID, resp.StatusCode, float64(duration.Milliseconds()), clientIP)

			defer resp.Body.Close()
			for k, v := range resp.Header {
				c.Writer.Header()[k] = v
			}
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body)
			return
		}

		// Record Error Metric
		httpRequestsTotal.WithLabelValues(provider.ID, "error").Inc()
		if err != nil {
			lastErr = err
			log.Printf("[ERROR] Provider request failed: provider=%s, attempt=%d/%d, error=%v, duration=%.2fms, client=%s",
				provider.ID, i+1, maxAttempts, err, float64(duration.Milliseconds()), clientIP)
		} else {
			lastErr = fmt.Errorf("provider %s returned %d", provider.ID, resp.StatusCode)
			log.Printf("[WARN] Provider returned error status: provider=%s, status=%d, attempt=%d/%d, duration=%.2fms, client=%s",
				provider.ID, resp.StatusCode, i+1, maxAttempts, float64(duration.Milliseconds()), clientIP)
			resp.Body.Close()
		}

		if i < maxAttempts-1 {
			log.Printf("[INFO] Retrying request after 50ms: attempt=%d/%d, client=%s", i+1, maxAttempts, clientIP)
			time.Sleep(50 * time.Millisecond)
		}
	}

	log.Printf("[ERROR] All providers exhausted after %d attempts, client=%s, last_error=%v", maxAttempts, clientIP, lastErr)
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   "All providers failed",
		"details": lastErr.Error(),
	})
}
