package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RedisLimiter struct {
	client *redis.Client
	limit  int64
	burst  int
}

// NewGlobalLimiter initializes a connection to Redis for distributed rate limiting
func NewGlobalLimiter(redisAddr string, limit float64, burst int) *RedisLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[ERROR] Could not connect to Redis at %s: %v. Global limiting may fail.", redisAddr, err)
	} else {
		log.Printf("[INFO] Global Redis Limiter initialized at %s (Limit: %.1f/s)", redisAddr, limit)
	}

	return &RedisLimiter{
		client: rdb,
		limit:  int64(limit),
		burst:  burst,
	}
}

// GinLimit implements a sliding window counter logic using Redis
func (rl *RedisLimiter) GinLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// We use a per-second key for the sliding window
		// Format: rate_limit:2025-12-28T12:01:05
		now := time.Now()
		key := "rate_limit:" + now.Format("2006-01-02T15:04:05")

		// Increment the counter for the current second
		count, err := rl.client.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("[ERROR] Redis connection error: %v", err)
			// Fail-open strategy: allow request if Redis is down, or use c.Abort() to fail-closed
			c.Next()
			return
		}

		// Set expiration on new keys so they clean up automatically
		if count == 1 {
			rl.client.Expire(ctx, key, 2*time.Second)
		}

		// Check if global limit is exceeded
		if count > rl.limit {
			log.Printf("[WARN] Global rate limit exceeded (Redis): key=%s, count=%d", key, count)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Global distributed rate limit exceeded",
				"retry_after": "1s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
