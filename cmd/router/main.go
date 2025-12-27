package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-router/internal/config"
	"llm-router/internal/handler"
	"llm-router/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] Starting LLM Router service...")

	cfg := &config.RouterConfig{
		Providers: []config.Provider{
			{ID: "mock-a", URL: "http://mock-a:8080", Weight: 80, Active: true},
			{ID: "mock-b", URL: "http://mock-b:8080", Weight: 20, Active: true},
		},
	}

	log.Printf("[INFO] Router initialized with %d providers", len(cfg.Providers))
	for _, p := range cfg.Providers {
		log.Printf("[INFO] Provider configured: id=%s, url=%s, weight=%.1f, active=%v", p.ID, p.URL, p.Weight, p.Active)
	}

	r := gin.Default()
	r.StaticFile("/admin-ui", "./admin/index.html")
	limiter := middleware.NewRateLimiter()
	routerHandler := &handler.RouterHandler{
		Cfg:    cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// 1. Observability: Metrics endpoint for Prometheus scraping
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 2. Management: Admin API for Hot Reload of configurations
	r.POST("/admin/config", func(c *gin.Context) {
		var newProviders []config.Provider
		if err := c.ShouldBindJSON(&newProviders); err != nil {
			log.Printf("[ERROR] Failed to parse config update request from %s: %v", c.ClientIP(), err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[INFO] Received config update request from %s with %d providers", c.ClientIP(), len(newProviders))
		cfg.UpdateProviders(newProviders)
		log.Printf("[INFO] Configuration successfully updated with %d providers", len(newProviders))
		c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
	})

	// 3. Core LLM Routing
	r.POST("/v1/chat/completions", limiter.GinLimit(), routerHandler.HandleChat)

	srv := &http.Server{
		Addr:    ":3000",
		Handler: r,
	}

	r.GET("/admin/stats", func(c *gin.Context) {
		// Collect the current provider list
		providers := cfg.GetProviders()

		// Create a simplified view for the Admin UI
		stats := gin.H{
			"status":           "healthy",
			"active_providers": len(providers),
			"server_time":      time.Now().Format(time.RFC3339),
			"provider_details": providers,
		}

		c.JSON(http.StatusOK, stats)
	})

	// Endpoint to clear/reset metrics without restarting
	r.POST("/admin/metrics/reset", func(c *gin.Context) {
		prometheus.Unregister(handler.HttpRequestsTotal)
		prometheus.Unregister(handler.RequestDuration)

		// The promauto calls in router.go will re-initialize them on next use
		c.JSON(http.StatusOK, gin.H{"message": "Metrics reset successfully"})
	})

	go func() {
		log.Printf("[INFO] HTTP server listening on %s", srv.Addr)
		log.Println("[INFO] Endpoints available: GET /metrics, POST /admin/config, POST /v1/chat/completions")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Failed to start HTTP server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[INFO] Received signal %v, initiating graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("[INFO] Closing active connections...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Forced shutdown due to error: %v", err)
		os.Exit(1)
	}

	log.Println("[INFO] Server shutdown completed successfully")
}
