package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SequenceService Redis 序列服务
type SequenceService struct {
	client *redis.Client
	logger *zap.SugaredLogger
}

// NewSequenceService 创建序列服务
func NewSequenceService(host string, port int, password string, db int, logger *zap.SugaredLogger) *SequenceService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warnw("Redis connection failed, sequence service will use fallback", "error", err)
		rdb.Close()
		return nil // 返回nil，让上层使用数据库回退
	}

	return &SequenceService{
		client: rdb,
		logger: logger,
	}
}

// GetNextSequence 获取下一个序列号
// key: 序列键名，如 "sequence:ticket:202603" 表示 2026年3月的工单序列
func (s *SequenceService) GetNextSequence(ctx context.Context, key string) (int64, error) {
	// 使用 Redis INCR 原子递增
	seq, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}
	return seq, nil
}

// GetNextSequenceWithExpiry 获取下一个序列号并设置过期时间
// expiredAt: 序列过期时间（用于按月重置等场景）
func (s *SequenceService) GetNextSequenceWithExpiry(ctx context.Context, key string, expiredAt time.Time) (int64, error) {
	// 尝试获取当前值
	current, err := s.client.Get(ctx, key).Int64()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to get sequence: %w", err)
	}

	// 如果序列不存在或为0，设置为1
	if current == 0 {
		current = 1
		err := s.client.Set(ctx, key, current, time.Until(expiredAt)).Err()
		if err != nil {
			return 0, fmt.Errorf("failed to init sequence: %w", err)
		}
		return current, nil
	}

	// 递增
	seq, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}

	// 检查是否需要更新过期时间
	ttl, err := s.client.TTL(ctx, key).Result()
	if err == nil && ttl <= 0 {
		s.client.Expire(ctx, key, time.Until(expiredAt))
	}

	return seq, nil
}

// GetCurrentSequence 获取当前序列号（不递增）
func (s *SequenceService) GetCurrentSequence(ctx context.Context, key string) (int64, error) {
	seq, err := s.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get sequence: %w", err)
	}
	return seq, nil
}

// ResetSequence 重置序列
func (s *SequenceService) ResetSequence(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Close 关闭连接
func (s *SequenceService) Close() error {
	return s.client.Close()
}
