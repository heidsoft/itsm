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
	// dbQueryMaxSeqFn 从DB查询当前最大序列号的函数 (key格式: "sequence:ticket:tenant:YYYYMM")
	// 如果为nil，则跳过DB同步
	dbQueryMaxSeqFn func(key string) (int64, error)
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
		client:          rdb,
		logger:          logger,
		dbQueryMaxSeqFn: nil,
	}
}

// SetDBQueryFunc 设置DB序列查询函数（用于Redis初始化时从DB同步）
func (s *SequenceService) SetDBQueryFunc(fn func(key string) (int64, error)) {
	s.dbQueryMaxSeqFn = fn
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
// 方案B：Redis初始化时从DB同步起点，避免Redis重置后编号碰撞
// expiredAt: 序列过期时间（用于按月重置等场景）
//
// 并发安全修复：原实现 GET->判断->SET 存在 check-then-act 竞态--并发时某协程
// 可能在 current==0 分支用 SET 直接覆盖已被其他协程 INCR 推进的计数器（回拨），
// 导致多个请求拿到同一序列号。现改为：① 仅当 key 不存在/为0时用 SETNX 播种
// 起点（绝不覆盖既有值）；② 统一走 INCR 原子递增，保证单调不回拨。
func (s *SequenceService) GetNextSequenceWithExpiry(ctx context.Context, key string, expiredAt time.Time) (int64, error) {
	// 检查 key 是否已初始化
	current, err := s.client.Get(ctx, key).Int64()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to get sequence: %w", err)
	}

	// 仅当 key 不存在或为0时播种起点；SETNX 保证只有第一个请求写入成功，
	// 后来者写入失败也无妨--直接走 INCR 即可，绝不会回拨计数器
	if err == redis.Nil || current == 0 {
		start := int64(1)
		if s.dbQueryMaxSeqFn != nil {
			if dbMax, dbErr := s.dbQueryMaxSeqFn(key); dbErr == nil && dbMax > 0 {
				start = dbMax
				s.logger.Infow("Redis sequence seeded from DB", "key", key, "dbMax", dbMax)
			}
		}
		if setErr := s.client.SetArgs(ctx, key, start, redis.SetArgs{
			Mode: "NX",
			TTL:  time.Until(expiredAt),
		}).Err(); setErr != nil && setErr != redis.Nil {
			s.logger.Warnw("SETNX seed failed, will rely on INCR", "key", key, "error", setErr)
		}
	}

	// 原子递增：无论播种成败都安全（未播种成功时从 0 -> 1）
	seq, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment sequence: %w", err)
	}

	// 补充过期时间（SETNX 写入时已带 TTL，此处兜底无 TTL 的旧 key）
	if ttl, terr := s.client.TTL(ctx, key).Result(); terr == nil && ttl <= 0 {
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
