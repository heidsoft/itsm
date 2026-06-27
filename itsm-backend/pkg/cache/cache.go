package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CacheService 缓存服务
type CacheService struct {
	client *redis.Client
	logger *zap.Logger
}

// CacheConfig 缓存配置
type CacheConfig struct {
	DefaultTTL time.Duration
	Prefix     string
}

// NewCacheService 创建缓存服务
func NewCacheService(client *redis.Client, logger *zap.Logger) *CacheService {
	return &CacheService{
		client: client,
		logger: logger,
	}
}

// Set 设置缓存
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("Failed to marshal cache value", zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to set cache", zap.Error(err), zap.String("key", key))
		return err
	}

	c.logger.Debug("Cache set successfully", zap.String("key", key))
	return nil
}

// Get 获取缓存
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache key not found")
		}
		c.logger.Error("Failed to get cache", zap.Error(err), zap.String("key", key))
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.logger.Error("Failed to unmarshal cache value", zap.Error(err))
		return err
	}

	return nil
}

// Delete 删除缓存
func (c *CacheService) Delete(ctx context.Context, keys ...string) error {
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.logger.Error("Failed to delete cache", zap.Error(err), zap.Strings("keys", keys))
		return err
	}

	c.logger.Debug("Cache deleted successfully", zap.Strings("keys", keys))
	return nil
}

// DeleteByPattern 根据模式删除缓存
func (c *CacheService) DeleteByPattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		c.logger.Error("Failed to scan keys", zap.Error(err))
		return err
	}

	if len(keys) > 0 {
		return c.Delete(ctx, keys...)
	}

	return nil
}

// Exists 检查缓存是否存在
func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	val, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// SetNX 设置缓存（如果不存在）
func (c *CacheService) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	res, err := c.client.SetArgs(ctx, key, data, redis.SetArgs{Mode: "NX", TTL: ttl}).Result()
	if err != nil {
		return false, err
	}
	return res == "OK", nil
}

// GetOrSet 获取缓存，如果不存在则设置
func (c *CacheService) GetOrSet(ctx context.Context, key string, dest interface{}, ttl time.Duration, fn func() (interface{}, error)) error {
	// 尝试从缓存获取
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	// 缓存不存在，调用函数获取数据
	value, err := fn()
	if err != nil {
		return err
	}

	// 设置缓存
	if err := c.Set(ctx, key, value, ttl); err != nil {
		c.logger.Warn("Failed to set cache, continuing without caching", zap.Error(err))
	}

	// 将值赋给dest
	data, _ := json.Marshal(value)
	return json.Unmarshal(data, dest)
}

// Increment 计数器增加
func (c *CacheService) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Decrement 计数器减少
func (c *CacheService) Decrement(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// Expire 设置过期时间
func (c *CacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// TTL 获取剩余过期时间
func (c *CacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}
