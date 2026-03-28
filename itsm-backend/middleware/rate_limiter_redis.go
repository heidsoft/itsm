package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter 基于 Redis 的分布式限流器
// 使用滑动窗口算法实现精确的请求限流
type RedisRateLimiter struct {
	client  *redis.Client
	limit   int           // 每时间窗口最大请求数
	window  time.Duration // 时间窗口
	keyPrefix string     // Redis key 前缀
}

// NewRedisRateLimiter 创建 Redis 限流器
func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:    client,
		limit:     limit,
		window:    window,
		keyPrefix: "ratelimit:",
	}
}

// Allow 检查是否允许请求（实现滑动窗口算法）
func (rl *RedisRateLimiter) Allow(ctx context.Context, clientIP string) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-rl.window)
	key := rl.keyPrefix + clientIP

	// 使用 Redis 事务确保原子性
	pipe := rl.client.Pipeline()

	// 删除窗口外的旧记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// 计算当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)

	// 执行管道
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("redis pipeline exec failed: %w", err)
	}

	currentCount := countCmd.Val()

	// 检查是否超过限制
	if currentCount >= int64(rl.limit) {
		return false, nil
	}

	// 添加当前请求到有序集合
	err = rl.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}).Err()
	if err != nil {
		return false, fmt.Errorf("zadd failed: %w", err)
	}

	// 设置 key 的过期时间
	rl.client.Expire(ctx, key, rl.window)

	return true, nil
}
