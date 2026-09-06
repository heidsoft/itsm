package middleware

import (
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	memoryStore "github.com/ulule/limiter/v3/drivers/store/memory"
	"itsm-backend/common"
)

// LoginRateLimiter middleware for login endpoint
// 10 attempts per minute per IP address using ulule/limiter v3
//
// P0-2（2026-09-06 UAT 修复）：从 5/分钟提到 10/分钟（真实用户首次输错常试 2-3 次）。
// 命中时返回 retry_after_seconds，前端可展示倒计时；同时设 Retry-After HTTP 头。
func LoginRateLimiter() gin.HandlerFunc {
	rate := limiter.Rate{
		Limit:  10,              // 10 attempts（UAT 后从 5 提到 10）
		Period: 1 * time.Minute, // per minute
	}
	store := memoryStore.NewStore()
	limiterInstance := limiter.New(store, rate)

	return func(c *gin.Context) {
		key := c.ClientIP() // Use IP address as the key

		// Peek checks without incrementing (to check if already limited)
		ctx, err := limiterInstance.Peek(c.Request.Context(), key)
		if err != nil {
			common.Fail(c, common.InternalErrorCode, "限流服务异常")
			c.Abort()
			return
		}

		if ctx.Reached {
			resetAt := time.Unix(ctx.Reset, 0)
			retryAfter := int(math.Ceil(time.Until(resetAt).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", itoa(retryAfter))
			common.FailWithData(c, common.ForbiddenCode, "登录请求过于频繁，请稍后再试", gin.H{
				"retryAfterSeconds": retryAfter,
			})
			c.Abort()
			return
		}

		// Actually increment the counter for this valid request
		if _, err := limiterInstance.Get(c.Request.Context(), key); err != nil {
			common.Fail(c, common.InternalErrorCode, "限流服务异常")
			c.Abort()
			return
		}

		c.Next()
	}
}

// itoa 避免 strconv 依赖（ulule/limiter 已经传递 time.Time Reset，简化返回整数秒）
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
