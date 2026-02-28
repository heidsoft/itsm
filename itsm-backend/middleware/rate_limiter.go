package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitMiddleware returns rate limiting middleware
// Default: 5 requests per minute
func RateLimitMiddleware() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  5,
		Period: 1 * time.Minute,
	}
	store := memory.NewStore()
	
	return func(c *gin.Context) {
		key := c.ClientIP()
		context := limiter.DefaultContext{
			Key: key,
		}
		
		allowed, remaining, _, err := rate.Allow(store, context)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "限流服务异常",
			})
			c.Abort()
			return
		}
		
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":       1001,
				"message":    "请求过于频繁，请稍后再试",
				"remaining":  remaining,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// StrictRateLimitMiddleware returns stricter rate limiting middleware
// For endpoints that need more protection: 3 requests per minute
func StrictRateLimitMiddleware() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  3,
		Period: 1 * time.Minute,
	}
	store := memory.NewStore()
	
	return func(c *gin.Context) {
		key := c.ClientIP()
		context := limiter.DefaultContext{
			Key: key,
		}
		
		allowed, remaining, _, err := rate.Allow(store, context)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "限流服务异常",
			})
			c.Abort()
			return
		}
		
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":       1001,
				"message":    "请求过于频繁，请稍后再试",
				"remaining":  remaining,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// LoginRateLimiter middleware for login endpoint
// 5 attempts per minute per IP address
func LoginRateLimiter() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  5, // 5 attempts
		Period: 1 * time.Minute, // per minute
	}
	store := memory.NewStore()
	
	return func(c *gin.Context) {
		key := c.ClientIP() // Use IP address as the key
		context := limiter.DefaultContext{
			Key: key,
		}
		
		allowed, remaining, _, err := rate.Allow(store, context)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "限流服务异常",
			})
			c.Abort()
			return
		}
		
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":       1001,
				"message":    "登录请求过于频繁，请稍后再试",
				"remaining":  remaining,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
