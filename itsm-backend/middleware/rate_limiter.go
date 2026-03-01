package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	memoryStore "github.com/ulule/limiter/v3/drivers/store/memory"
)

// LoginRateLimiter middleware for login endpoint
// 5 attempts per minute per IP address using ulule/limiter v3
func LoginRateLimiter() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  5,    // 5 attempts
		Period: 1 * time.Minute, // per minute
	}
	store := memoryStore.NewStore()
	limiterInstance := limiter.New(store, rate)

	return func(c *gin.Context) {
		key := c.ClientIP() // Use IP address as the key
		
		// Peek checks without incrementing (to check if already limited)
		ctx, err := limiterInstance.Peek(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "限流服务异常",
			})
			c.Abort()
			return
		}

		if ctx.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":     1001,
				"message":  "登录请求过于频繁，请稍后再试",
				"remaining": ctx.Remaining,
			})
			c.Abort()
			return
		}

		// Actually increment the counter for this valid request
		_, _ = limiterInstance.Get(c.Request.Context(), key)

		c.Next()
	}
}
