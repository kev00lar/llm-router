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
	"llm-router/internal/middleware"
	"llm-router/internal/router"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] Starting LLM Router service...")

	// 1. Load Configuration
	weightA, _ := strconv.ParseFloat(getEnv("PROVIDER_A_WEIGHT", "80.0"), 64)
	weightB, _ := strconv.ParseFloat(getEnv("PROVIDER_B_WEIGHT", "20.0"), 64)

	cfg := &config.RouterConfig{
		Providers: []config.Provider{
			{ID: getEnv("PROVIDER_A_ID", "mock-a"), URL: getEnv("PROVIDER_A_URL", "http://mock-a:8080"), Weight: weightA, Active: true},
			{ID: getEnv("PROVIDER_B_ID", "mock-b"), URL: getEnv("PROVIDER_B_URL", "http://mock-b:8080"), Weight: weightB, Active: true},
		},
	}

	// 2. Initialize Infrastructure
	limiter := middleware.NewGlobalLimiter(
		getEnv("REDIS_URL", "redis:6379"),
		getEnvAsFloat("GLOBAL_LIMIT", 5.0),
		getEnvAsInt("GLOBAL_BURST", 5),
	)

	// 3. Setup Router
	r := router.NewRouter(cfg, limiter)

	// 4. Start Server
	port := getEnv("PORT", "3000")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("[INFO] Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Failed to start: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}

// Helpers for main.go
func getEnvAsFloat(key string, fallback float64) float64 {
	val, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil { return fallback }
	return val
}

func getEnvAsInt(key string, fallback int) int {
	val, err := strconv.Atoi(os.Getenv(key))
	if err != nil { return fallback }
	return val
}