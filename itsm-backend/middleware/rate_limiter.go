package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// LoginRateLimiter middleware for login endpoint
// 5 attempts per minute per IP address
func LoginRateLimiter() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  5,
		Period: 1 * time.Minute,
	}
	store := memory.NewStore()
	limiterInstance := limiter.New(store, rate)

	return func(c *gin.Context) {
		key := c.ClientIP()

		// 检查是否允许
		context, err := limiterInstance.Get(c.Request.Context(), key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "限流服务异常",
			})
			c.Abort()
			return
		}

		if !context.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":      1001,
				"message":   "登录请求过于频繁，请稍后再试",
				"remaining": context.Remaining,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
