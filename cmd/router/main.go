package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"llm-router/internal/config"
	"llm-router/internal/handler"
	"llm-router/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// getEnv is a helper to read environment variables with a default fallback
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] Starting LLM Router service...")

	// 1. Load Configuration from Environment Variables
	weightA, _ := strconv.ParseFloat(getEnv("PROVIDER_A_WEIGHT", "80.0"), 64)
	weightB, _ := strconv.ParseFloat(getEnv("PROVIDER_B_WEIGHT", "20.0"), 64)

	cfg := &config.RouterConfig{
		Providers: []config.Provider{
			{
				ID:     getEnv("PROVIDER_A_ID", "mock-a"),
				URL:    getEnv("PROVIDER_A_URL", "http://mock-a:8080"),
				Weight: weightA,
				Active: true,
			},
			{
				ID:     getEnv("PROVIDER_B_ID", "mock-b"),
				URL:    getEnv("PROVIDER_B_URL", "http://mock-b:8080"),
				Weight: weightB,
				Active: true,
			},
		},
	}

	// 2. Initialize Distributed Rate Limiter (Redis-backed for K8s)
	redisURL := getEnv("REDIS_URL", "redis:6379")
	limitRPS, _ := strconv.ParseFloat(getEnv("GLOBAL_LIMIT", "5.0"), 64)
	burst, _ := strconv.Atoi(getEnv("GLOBAL_BURST", "5"))
	
	limiter := middleware.NewGlobalLimiter(redisURL, limitRPS, burst)

	// 3. Setup Router and Handlers
	r := gin.Default()

	routerHandler := &handler.RouterHandler{
		Cfg:    cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// --- ROUTES ---

	// A. Observability: Metrics endpoint for Prometheus
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// B. Management API: Specific routes MUST come before Static wildcards
	// These provide data to the Admin UI
	r.GET("/admin/stats", func(c *gin.Context) {
		providers := cfg.GetProviders()
		c.JSON(http.StatusOK, gin.H{
			"status":           "healthy",
			"active_providers": len(providers),
			"server_time":      time.Now().Format(time.RFC3339),
			"provider_details": providers,
		})
	})

	r.POST("/admin/config", func(c *gin.Context) {
		var newProviders []config.Provider
		if err := c.ShouldBindJSON(&newProviders); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg.UpdateProviders(newProviders)
		c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
	})

	r.POST("/admin/metrics/reset", func(c *gin.Context) {
		prometheus.Unregister(handler.HttpRequestsTotal)
		prometheus.Unregister(handler.RequestDuration)
		c.JSON(http.StatusOK, gin.H{"message": "Metrics reset successfully"})
	})

	// C. Static UI: Served at /dashboard to avoid prefix conflicts with /admin API
	r.Static("/dashboard", "./admin")

	// D. Core LLM Routing with Global Rate Limiting
	r.POST("/v1/chat/completions", limiter.GinLimit(), routerHandler.HandleChat)

	// --- SERVER STARTUP & SHUTDOWN ---

	port := getEnv("PORT", "3000")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("[INFO] HTTP server listening on :%s", port)
		log.Printf("[INFO] Admin UI available at http://localhost:%s/dashboard", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Failed to start server: %v", err)
		}
	}()

	// Signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exiting")
}