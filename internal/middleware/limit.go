package middleware

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	global *rate.Limiter
	keys   map[string]*rate.Limiter
	mu     sync.Mutex
}

// NewRateLimiter initializes a global limit and a map for per-API-key limits
func NewRateLimiter() *RateLimiter {
	log.Println("[INFO] Rate limiter initialized: global_limit=5req/s, burst=5")
	return &RateLimiter{
		// Set global limit to 5 requests per second with a burst of 5
		global: rate.NewLimiter(rate.Limit(5), 5),
		keys:   make(map[string]*rate.Limiter),
	}
}

// GinLimit implements the rate limiting middleware for Gin [cite: 13, 19]
func (rl *RateLimiter) GinLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Global Rate Limit check [cite: 19]
		if !rl.global.Allow() {
			log.Printf("[WARN] Global rate limit exceeded: client_ip=%s, path=%s", c.ClientIP(), c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Global rate limit exceeded",
			}) // Returns 429
			return
		}

		// 2. Per-API-Key Rate Limit check [cite: 20]
		apiKey := c.GetHeader("Authorization")
		if apiKey != "" {
			rl.mu.Lock()
			limiter, exists := rl.keys[apiKey]
			if !exists {
				// Default: 5 requests per second per unique API key
				limiter = rate.NewLimiter(rate.Limit(5), 5)
				rl.keys[apiKey] = limiter
				masked := maskAPIKey(apiKey)
				log.Printf("[INFO] New API key registered for rate limiting: api_key=%s, limit=5req/s", masked)
			}
			rl.mu.Unlock()

			if !limiter.Allow() {
				masked := maskAPIKey(apiKey)
				log.Printf("[WARN] API key rate limit exceeded: api_key=%s, client_ip=%s, path=%s", masked, c.ClientIP(), c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "API key rate limit exceeded",
				}) // Returns 429
				return
			}
		}

		c.Next()
	}
}

// maskAPIKey masks sensitive API key data for logging (shows first 4 and last 4 chars)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
