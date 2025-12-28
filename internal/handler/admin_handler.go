package handler

import (
	"net/http"
	"time"

	"llm-router/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type AdminHandler struct {
	Cfg *config.RouterConfig
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	providers := h.Cfg.GetProviders()
	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"active_providers": len(providers),
		"server_time":      time.Now().Format(time.RFC3339),
		"provider_details": providers,
	})
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var newProviders []config.Provider
	if err := c.ShouldBindJSON(&newProviders); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.Cfg.UpdateProviders(newProviders)
	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

func (h *AdminHandler) ResetMetrics(c *gin.Context) {
	prometheus.Unregister(HttpRequestsTotal)
	prometheus.Unregister(RequestDuration)
	c.JSON(http.StatusOK, gin.H{"message": "Metrics reset successfully"})
}