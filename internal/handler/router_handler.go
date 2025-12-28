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
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_router_requests_total",
		Help: "Total number of requests handled by the router",
	}, []string{"provider", "status"})
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llm_router_request_duration_seconds",
		Help:    "Latency of requests per provider",
		Buckets: prometheus.DefBuckets,
	}, []string{"provider"})
)

type RouterHandler struct {
	Cfg    *config.RouterConfig
	Client *http.Client
}

// ResetMetrics unregisters and re-initializes metrics to zero them out
func (h *RouterHandler) ResetMetrics() {
	prometheus.Unregister(HttpRequestsTotal)
	prometheus.Unregister(RequestDuration)

	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llm_router_requests_total",
		Help: "Total number of requests handled by the router",
	}, []string{"provider", "status"})

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llm_router_request_duration_seconds",
		Help:    "Latency of requests per provider",
		Buckets: prometheus.DefBuckets,
	}, []string{"provider"})
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

	maxAttempts := 3
	var lastErr error

	for i := range maxAttempts {
		provider := h.Cfg.SelectProvider()
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No active providers"})
        	return
		}
		
		log.Printf("[INFO] Attempt %d/%d: Routing to %s", i+1, maxAttempts, provider.ID)

		start := time.Now()
		fullURL := provider.URL + "/v1/chat/completions"
		req, _ := http.NewRequest("POST", fullURL, bytes.NewBuffer(bodyBytes))
		req.Header = c.Request.Header.Clone()

		resp, err := h.Client.Do(req)
		duration := time.Since(start)

		if err == nil && resp.StatusCode < 500 {
			RequestDuration.WithLabelValues(provider.ID).Observe(duration.Seconds())
			HttpRequestsTotal.WithLabelValues(provider.ID, "success").Inc()

			log.Printf("[INFO] Success: provider=%s, status=%d, duration=%dms",
				provider.ID, resp.StatusCode, duration.Milliseconds())

			defer resp.Body.Close()
			for k, v := range resp.Header {
				c.Writer.Header()[k] = v
			}
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body)
			return
		}

		// Record Resilient Flow (Error/Failover)
		HttpRequestsTotal.WithLabelValues(provider.ID, "error").Inc()

		if err != nil {
			lastErr = err
			log.Printf("[ERROR] Attempt %d failed: %v", i+1, err)
		} else {
			lastErr = fmt.Errorf("provider %s returned %d", provider.ID, resp.StatusCode)
			log.Printf("[WARN] Attempt %d failed: status %d", i+1, resp.StatusCode)
			resp.Body.Close()
		}

		if i < maxAttempts-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   "All providers failed",
		"details": lastErr.Error(),
	})
}