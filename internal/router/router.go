package router

import (
	"net/http"
	"os"
	"strings"
	"time"

	"llm-router/internal/config"
	"llm-router/internal/handler"
	"llm-router/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(cfg *config.RouterConfig, limiter *middleware.RedisLimiter) *gin.Engine {
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	customTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	}

	httpClient := &http.Client{
		Transport: customTransport,
		Timeout:   30 * time.Second,
	}

	// 3. Initialize Handlers
	routerHandler := &handler.RouterHandler{
		Cfg:    cfg,
		Client: httpClient,
	}
	adminHandler := &handler.AdminHandler{
		Cfg: cfg,
	}

	// 4. Feature Toggle: Global Rate Limiting
	rateLimitEnabled := strings.ToLower(os.Getenv("RATE_LIMIT_ENABLED")) == "true"

	// --- ROUTES ---

	// A. Observability
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// B. Management API (Grouped for easier future middleware/auth)
	adminGroup := r.Group("/admin")
	{
		adminGroup.GET("/stats", adminHandler.GetStats)
		adminGroup.POST("/config", adminHandler.UpdateConfig)
		adminGroup.POST("/metrics/reset", adminHandler.ResetMetrics)
	}

	// C. Static UI
	r.Static("/dashboard", "./admin")

	// D. Core LLM Routing Logic
	// Only apply the Redis middleware if enabled via ENV
	if rateLimitEnabled && limiter != nil {
		r.POST("/v1/chat/completions", limiter.GinLimit(), routerHandler.HandleChat)
	} else {
		r.POST("/v1/chat/completions", routerHandler.HandleChat)
	}

	return r
}