package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	accessTokenRevocationPrefix  = "jwt:revoked:"
	refreshTokenRevocationPrefix = "jwt:revoked:refresh:"
	// legacyRefreshBlacklistPrefix 只读兼容：2026-10-02 之前 refresh 吊销写在
	// handlers/common 的 Redis-only 黑名单里（键名含明文 JWT）。本版本起不再写入，
	// 条目 TTL 不超过 refresh 生命周期（7 天），过期后该分支必然不再命中，可整段删除。
	legacyRefreshBlacklistPrefix = "refresh:blacklist:"
	// userMinIATPrefix 存储每个用户的 token 最低可接受签发时间。
	// 早于该时间签发的 access 与 refresh token 都视为失效（用于角色变更/停用/改密后批量吊销存量会话）。
	userMinIATPrefix = "jwt:user_min_iat:"
	// userMinIATTTL 必须覆盖最长 token 生命周期：refresh token 有效期 7 天，
	// 因此约束要活过它，否则改密后遗留的旧 refresh token 会在 TTL 到期后重新可用。
	userMinIATTTL = 8 * 24 * time.Hour
)

type tokenRevocationStore interface {
	IsRevoked(ctx context.Context, token string) (bool, error)
	Revoke(ctx context.Context, token string, expiresAt time.Time) error
	// RevokeRefreshToken 吊销 refresh token，并返回本次调用是否首次吊销该 token。
	// 原子认领语义让「检查是否已用」与「标记已用」之间不存在竞态，防止 rotation 重放。
	RevokeRefreshToken(ctx context.Context, token string, expiresAt time.Time) (bool, error)
	IsRefreshRevoked(ctx context.Context, token string) (bool, error)
	// MinIssuedAt 返回该用户 token 的最低可接受签发时间；零值表示无约束。
	MinIssuedAt(ctx context.Context, userID int) (time.Time, error)
	// SetMinIssuedAt 将该用户 token 的最低签发时间提升到 t（只增不减）。
	SetMinIssuedAt(ctx context.Context, userID int, t time.Time) error
}

var (
	revocationStoreMu sync.RWMutex
	revocationStore   tokenRevocationStore = newMemoryTokenRevocationStore()
)

// ConfigureTokenRevocationRedis 让吊销状态跨实例共享。
// 未配置 Redis 时退化为进程内存储：单副本部署的登出/改密依然真实生效，
// 多副本部署必须由装配层显式配置，否则吊销只覆盖处理该请求的副本。
func ConfigureTokenRevocationRedis(client *redis.Client) {
	if client != nil {
		setTokenRevocationStore(&redisTokenRevocationStore{client: client})
	}
}

func setTokenRevocationStore(store tokenRevocationStore) {
	revocationStoreMu.Lock()
	defer revocationStoreMu.Unlock()
	revocationStore = store
}

func currentTokenRevocationStore() tokenRevocationStore {
	revocationStoreMu.RLock()
	defer revocationStoreMu.RUnlock()
	return revocationStore
}

// RevokeAccessToken invalidates one access token until its JWT expiry.
func RevokeAccessToken(ctx context.Context, token string, expiresAt time.Time) error {
	return currentTokenRevocationStore().Revoke(ctx, token, expiresAt)
}

func isAccessTokenRevoked(ctx context.Context, token string) (bool, error) {
	return currentTokenRevocationStore().IsRevoked(ctx, token)
}

// RevokeRefreshToken 吊销一个 refresh token（登出、rotation 后作废、重放检测）。
func RevokeRefreshToken(ctx context.Context, token string, expiresAt time.Time) (bool, error) {
	return currentTokenRevocationStore().RevokeRefreshToken(ctx, token, expiresAt)
}

// IsRefreshTokenRevoked 报告 refresh token 是否已被吊销；调用方必须按 fail-closed 处理错误。
func IsRefreshTokenRevoked(ctx context.Context, token string) (bool, error) {
	return currentTokenRevocationStore().IsRefreshRevoked(ctx, token)
}

// InvalidateUserTokens 将 userID 全部存量 token 置为失效：
// 之后任何签发时间早于 t 的 access 或 refresh token 都会被拒绝。
// 适用于：角色变更、账户停用、密码重置等权限收窄场景（P1-2 修复）。
func InvalidateUserTokens(ctx context.Context, userID int, t time.Time) error {
	return currentTokenRevocationStore().SetMinIssuedAt(ctx, userID, t)
}

// UserTokenMinIssuedAt 返回用户 token 的最低可接受签发时间；零值表示无约束。
func UserTokenMinIssuedAt(ctx context.Context, userID int) (time.Time, error) {
	return currentTokenRevocationStore().MinIssuedAt(ctx, userID)
}

// userMinIATKey 返回用户最低签发时间的存储键（分片按租户无关，仅按用户）。
func userMinIATKey(userID int) string {
	return fmt.Sprintf("%s%d", userMinIATPrefix, userID)
}

// accessTokenRevocationKey 用摘要做键名：吊销列表里的仍是未过期的有效凭证，
// 明文 JWT 进 keyspace 等于把凭据泄漏给任何能看 Redis 键的工具。
func accessTokenRevocationKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return accessTokenRevocationPrefix + hex.EncodeToString(sum[:])
}

func refreshTokenRevocationKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return refreshTokenRevocationPrefix + hex.EncodeToString(sum[:])
}

type redisTokenRevocationStore struct {
	client *redis.Client
}

func (s *redisTokenRevocationStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	count, err := s.client.Exists(ctx, accessTokenRevocationKey(token)).Result()
	return count > 0, err
}

func (s *redisTokenRevocationStore) Revoke(ctx context.Context, token string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return s.client.Set(ctx, accessTokenRevocationKey(token), "1", ttl).Err()
}

func (s *redisTokenRevocationStore) RevokeRefreshToken(ctx context.Context, token string, expiresAt time.Time) (bool, error) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// 自然过期的 token 无需占位，按既有语义放行。
		return true, nil
	}
	_, err := s.client.SetArgs(ctx, refreshTokenRevocationKey(token), "1", redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil // 已被其他并发请求认领，不是错误
		}
		return false, fmt.Errorf("revoke refresh token: %w", err)
	}
	return true, nil
}

func (s *redisTokenRevocationStore) IsRefreshRevoked(ctx context.Context, token string) (bool, error) {
	// 第二个键是只读的历史兼容位，见 legacyRefreshBlacklistPrefix 注释。
	count, err := s.client.Exists(ctx, refreshTokenRevocationKey(token), legacyRefreshBlacklistPrefix+token).Result()
	return count > 0, err
}

func (s *redisTokenRevocationStore) MinIssuedAt(ctx context.Context, userID int) (time.Time, error) {
	val, err := s.client.Get(ctx, userMinIATKey(userID)).Result()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	ts, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, nil // 脏数据按无约束处理
	}
	return time.Unix(ts, 0), nil
}

func (s *redisTokenRevocationStore) SetMinIssuedAt(ctx context.Context, userID int, t time.Time) error {
	// 只增不减：避免并发写回退约束。
	const luaScript = `local cur = redis.call('GET', KEYS[1])
local curTs = tonumber(cur) or 0
local newTs = tonumber(ARGV[1])
if newTs > curTs then
  redis.call('SET', KEYS[1], tostring(newTs), 'EX', tonumber(ARGV[2]))
end
return 1`
	return s.client.Eval(ctx, luaScript, []string{userMinIATKey(userID)},
		t.Unix(), int(userMinIATTTL.Seconds())).Err()
}

// memoryRevocationSweepThreshold 内存实现没有 TTL，写入时按阈值顺带清理过期条目，
// 避免登出/续签记录无上限累积。
const memoryRevocationSweepThreshold = 1024

type memoryTokenRevocationStore struct {
	mu      sync.Mutex
	expires map[string]time.Time
	// userMinIAT 记录每用户 token 最低可接受签发时间（内存实现）。
	userMinIAT map[int]time.Time
}

func newMemoryTokenRevocationStore() *memoryTokenRevocationStore {
	return &memoryTokenRevocationStore{
		expires:    make(map[string]time.Time),
		userMinIAT: make(map[int]time.Time),
	}
}

// purgeExpiredLocked 只在超过阈值时线性扫描；调用方必须持有 s.mu。
func (s *memoryTokenRevocationStore) purgeExpiredLocked(now time.Time) {
	if len(s.expires) < memoryRevocationSweepThreshold {
		return
	}
	for key, expiresAt := range s.expires {
		if !now.Before(expiresAt) {
			delete(s.expires, key)
		}
	}
}

func (s *memoryTokenRevocationStore) IsRevoked(_ context.Context, token string) (bool, error) {
	return s.isRevoked(accessTokenRevocationKey(token)), nil
}

func (s *memoryTokenRevocationStore) IsRefreshRevoked(_ context.Context, token string) (bool, error) {
	return s.isRevoked(refreshTokenRevocationKey(token)), nil
}

func (s *memoryTokenRevocationStore) isRevoked(key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.expires[key]
	if ok && !now.Before(expiresAt) {
		delete(s.expires, key)
		return false
	}
	return ok
}

func (s *memoryTokenRevocationStore) Revoke(_ context.Context, token string, expiresAt time.Time) error {
	if token == "" || !time.Now().Before(expiresAt) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(time.Now())
	s.expires[accessTokenRevocationKey(token)] = expiresAt
	return nil
}

func (s *memoryTokenRevocationStore) RevokeRefreshToken(_ context.Context, token string, expiresAt time.Time) (bool, error) {
	if token == "" {
		return true, nil
	}
	now := time.Now()
	if !now.Before(expiresAt) {
		return true, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	key := refreshTokenRevocationKey(token)
	if _, ok := s.expires[key]; ok {
		return false, nil
	}
	s.expires[key] = expiresAt
	return true, nil
}

func (s *memoryTokenRevocationStore) MinIssuedAt(_ context.Context, userID int) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userMinIAT[userID], nil
}

func (s *memoryTokenRevocationStore) SetMinIssuedAt(_ context.Context, userID int, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cur, ok := s.userMinIAT[userID]; !ok || t.After(cur) {
		s.userMinIAT[userID] = t
	}
	return nil
}
