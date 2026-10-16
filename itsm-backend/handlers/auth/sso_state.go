package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SSO state 的一次性记录。
//
// state 是登录 CSRF 防线：发起时生成、回调时消费（单次）。
// PKCE verifier 与 nonce 同 state 绑定存放，避免把任意一个单独作为可信输入。
type SSOStateRecord struct {
	ProviderID int       `json:"providerId"`
	TenantID   int       `json:"tenantId"`
	Mode       string    `json:"mode"` // login | bind
	BindUserID int       `json:"bindUserId,omitempty"`
	Nonce      string    `json:"nonce"`
	Verifier   string    `json:"verifier"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SSOStateStore 是 state 的存取契约。Consume 必须原子完成「读取 + 删除」，
// 否则并发回调可以把同一个 state 消费两次（重放）。
type SSOStateStore interface {
	Save(ctx context.Context, state string, record *SSOStateRecord, ttl time.Duration) error
	Consume(ctx context.Context, state string) (*SSOStateRecord, error)
}

// ErrSSOStateNotFound 表示 state 不存在、已消费或已过期。
// 调用方必须把它当认证失败处理，不得回退到「无 state 也继续」。
var ErrSSOStateNotFound = errors.New("sso state not found")

const ssoStatePrefix = "sso:state:"

// SSOStateTTL 覆盖「用户完成 IdP 登录」的合理上限：10 分钟。
// TTL 必须短于 PKCE/nonce 的自然生命周期，过期的 state 没有保留价值。
const SSOStateTTL = 10 * time.Minute

type redisSSOStateStore struct {
	client *redis.Client
}

func newRedisSSOStateStore(client *redis.Client) *redisSSOStateStore {
	return &redisSSOStateStore{client: client}
}

// consumeScript 原子完成 GET+DEL：单次消费语义不依赖调用方自觉。
var ssoStateConsumeScript = redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if not v then return nil end
redis.call('DEL', KEYS[1])
return v`)

func (s *redisSSOStateStore) Save(ctx context.Context, state string, record *SSOStateRecord, ttl time.Duration) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal sso state: %w", err)
	}
	// NX：state 碰撞（随机数意外复用）时拒绝覆盖已有记录。
	if err := s.client.SetArgs(ctx, ssoStatePrefix+state, payload, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return errors.New("sso state collision")
		}
		return fmt.Errorf("save sso state: %w", err)
	}
	return nil
}

func (s *redisSSOStateStore) Consume(ctx context.Context, state string) (*SSOStateRecord, error) {
	raw, err := ssoStateConsumeScript.Run(ctx, s.client, []string{ssoStatePrefix + state}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSSOStateNotFound
		}
		return nil, fmt.Errorf("consume sso state: %w", err)
	}
	text, ok := raw.(string)
	if !ok || text == "" {
		return nil, ErrSSOStateNotFound
	}
	var record SSOStateRecord
	if err := json.Unmarshal([]byte(text), &record); err != nil {
		return nil, fmt.Errorf("decode sso state: %w", err)
	}
	return &record, nil
}

// memorySSOStateStore 是 Redis 缺失时的进程内降级：单副本部署语义完整，
// 多副本部署回调可能落到另一副本而失败——装配层必须为此留下告警。
type memorySSOStateStore struct {
	mu      sync.Mutex
	records map[string]memorySSOStateEntry
}

type memorySSOStateEntry struct {
	record   SSOStateRecord
	expireAt time.Time
}

func newMemorySSOStateStore() *memorySSOStateStore {
	return &memorySSOStateStore{records: make(map[string]memorySSOStateEntry)}
}

func (s *memorySSOStateStore) Save(_ context.Context, state string, record *SSOStateRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[state]; exists {
		return errors.New("sso state collision")
	}
	s.records[state] = memorySSOStateEntry{record: *record, expireAt: time.Now().Add(ttl)}
	return nil
}

func (s *memorySSOStateStore) Consume(_ context.Context, state string) (*SSOStateRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.records[state]
	if !exists {
		return nil, ErrSSOStateNotFound
	}
	delete(s.records, state)
	if !time.Now().Before(entry.expireAt) {
		return nil, ErrSSOStateNotFound
	}
	record := entry.record
	return &record, nil
}

// NewSSOStateStore 按可用性选择实现：Redis 可用则跨副本共享，否则进程内降级。
func NewSSOStateStore(client *redis.Client) SSOStateStore {
	if client != nil {
		return newRedisSSOStateStore(client)
	}
	return newMemorySSOStateStore()
}
