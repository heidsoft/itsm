package authz

// 运行期权限注册表：权限码的**唯一运行时真源**是 DB 的 permissions 表。
//
// 与 Go 侧 Definitions() 的分工：
//   - Definitions() 只在「全新库首次引导」时由 seeder 调用，之后不再参与任何判定；
//   - Registry 每次鉴权读 DB（带 TTL 缓存），读不到 / 解析失败一律 fail-closed；
//   - 管理员在运行期改策略 → 调 Invalidate → 下一次判定立即生效。
//
// 这样「加一个权限码」不再需要改 Go 代码、重新编译、重新发版；
// 而「路由声明引用了一个 DB 里不存在的码」仍由 CI 守卫在构建期拦下。

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/permission"

	"go.uber.org/zap"
)

// DefaultRegistryTTL 与 middleware 的权限缓存 TTL 对齐，避免两层缓存口径不一。
const DefaultRegistryTTL = 5 * time.Minute

// Definition 是一条完整的运行期权限定义。
type Definition struct {
	ID          int
	Code        string
	Name        string
	Description string
	Resource    string
	Action      string
	TenantID    int
	Policy      PermissionPolicy
}

// Catalog 是鉴权调用方需要的最小读面（便于单测注入 fake）。
type Catalog interface {
	Lookup(ctx context.Context, tenantID int, code string) (*Definition, error)
	AuthorizePrincipal(ctx context.Context, tenantID int, code, principal string) (bool, int, error)
}

// ErrDefinitionNotFound / ErrCatalogUnavailable 让调用方区分「明确没这个码」
// 与「读不出来」。两者在鉴权路径上都应 deny，但日志语义不同。
var (
	ErrDefinitionNotFound   = fmt.Errorf("permission definition not found")
	ErrCatalogUnavailable   = fmt.Errorf("permission catalog unavailable")
	ErrDefinitionInactive   = fmt.Errorf("permission definition inactive")
	ErrDefinitionCorrupt    = fmt.Errorf("permission definition corrupt")
	ErrDefinitionDuplicated = fmt.Errorf("permission definition duplicated")
)

type registryEntry struct {
	defs     []Definition
	byCode   map[string]*Definition
	loadedAt time.Time
}

// Registry 按租户缓存 DB 权限定义，写操作后通过 Invalidate 精确失效。
type Registry struct {
	client *ent.Client
	ttl    time.Duration
	mu     sync.RWMutex
	cache  map[int]*registryEntry
}

// NewRegistry 创建注册表。client 为 nil 时任何查询都返回 ErrCatalogUnavailable
// （fail-closed，不会退化成 Go 侧的引导清单）。
func NewRegistry(client *ent.Client) *Registry {
	return &Registry{
		client: client,
		ttl:    DefaultRegistryTTL,
		cache:  make(map[int]*registryEntry),
	}
}

// sharedRegistry 是进程级共享实例。所有写入方（权限管理 API）与读取方
// （AI 执行链）必须用同一个实例，否则写后失效只清掉自己那份缓存。
// 默认指向一个未装配 client 的注册表：未装配时查询 fail-closed。
var sharedRegistry atomic.Pointer[Registry]

func init() {
	sharedRegistry.Store(NewRegistry(nil))
}

// ConfigureShared 在 bootstrap 装配 ent client 时调用，返回共享实例。
func ConfigureShared(client *ent.Client) *Registry {
	r := NewRegistry(client)
	sharedRegistry.Store(r)
	return r
}

// Shared 返回进程级共享注册表。
func Shared() *Registry { return sharedRegistry.Load() }

// SetTTL 调整缓存窗口（测试可设为 0 强制每次回源）。
func (r *Registry) SetTTL(ttl time.Duration) { r.ttl = ttl }

// Invalidate 使指定租户的缓存失效。权限/策略写入后必须调用。
func (r *Registry) Invalidate(tenantID int) {
	r.mu.Lock()
	delete(r.cache, tenantID)
	r.mu.Unlock()
}

// InvalidateAll 清空全部租户缓存。
func (r *Registry) InvalidateAll() {
	r.mu.Lock()
	clear(r.cache)
	r.mu.Unlock()
}

// All 返回租户内全部权限定义（含已停用），按 code 稳定排序，供管理页展示。
func (r *Registry) All(ctx context.Context, tenantID int) ([]Definition, error) {
	entry, err := r.entry(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := append([]Definition(nil), entry.defs...)
	return out, nil
}

// Lookup 按权限码取定义。
//
// fail-closed 语义：码不存在、已停用、信封损坏、DB 不可用一律返回 error，
// 调用方不得把 error 当成「放行」或「回退到硬编码」。
func (r *Registry) Lookup(ctx context.Context, tenantID int, code string) (*Definition, error) {
	entry, err := r.entry(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	def, ok := entry.byCode[strings.TrimSpace(code)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionNotFound, code)
	}
	if !def.Policy.IsActive {
		return nil, fmt.Errorf("%w: %s", ErrDefinitionInactive, code)
	}
	copy := *def
	return &copy, nil
}

// AuthorizePrincipal 是 AI 执行链的单一判定入口：
//  1. 码必须在 DB 中登记且启用；
//  2. 主体类型必须被该码显式允许（默认只有 human → AI 零权限）；
//  3. 返回该主体执行时需要的人类确认强度。
//
// 返回 (needsApproval, minApprovals, error)。error != nil 一律 deny。
func (r *Registry) AuthorizePrincipal(ctx context.Context, tenantID int, code, principal string) (bool, int, error) {
	def, err := r.Lookup(ctx, tenantID, code)
	if err != nil {
		return false, 0, err
	}
	if !def.Policy.Allows(principal) {
		return false, 0, fmt.Errorf("principal %q is not granted permission %q", principal, code)
	}
	needs, min := def.Policy.ApprovalFor(principal)
	return needs, min, nil
}

func (r *Registry) entry(ctx context.Context, tenantID int) (*registryEntry, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("%w: invalid tenant_id %d", ErrCatalogUnavailable, tenantID)
	}
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("%w: ent client not wired", ErrCatalogUnavailable)
	}

	r.mu.RLock()
	cached := r.cache[tenantID]
	r.mu.RUnlock()
	if cached != nil && r.ttl > 0 && time.Since(cached.loadedAt) < r.ttl {
		return cached, nil
	}

	rows, err := r.client.Permission.Query().
		Where(permission.TenantIDEQ(tenantID)).
		Order(ent.Asc(permission.FieldCode)).
		All(ctx)
	if err != nil {
		zap.S().Warnw("permission catalog load failed; failing closed",
			"tenant_id", tenantID, "error_class", "db_error")
		return nil, fmt.Errorf("%w: %v", ErrCatalogUnavailable, err)
	}

	defs := make([]Definition, 0, len(rows))
	byCode := make(map[string]*Definition, len(rows))
	for _, row := range rows {
		code := strings.TrimSpace(row.Code)
		resource := strings.TrimSpace(row.Resource)
		action := strings.TrimSpace(row.Action)
		if code == "" || resource == "" || action == "" || code != resource+":"+action {
			zap.S().Errorw("permission catalog: malformed code triple; failing closed",
				"tenant_id", tenantID, "id", row.ID, "code", row.Code,
				"resource", row.Resource, "action", row.Action)
			return nil, fmt.Errorf("%w: id=%d code=%q", ErrDefinitionCorrupt, row.ID, row.Code)
		}
		if _, exists := byCode[code]; exists {
			zap.S().Errorw("permission catalog: duplicate code; failing closed",
				"tenant_id", tenantID, "code", code)
			return nil, fmt.Errorf("%w: %s", ErrDefinitionDuplicated, code)
		}
		desc, policy, err := DecodeDefinition(row.Description, resource, action)
		if err != nil {
			zap.S().Errorw("permission catalog: invalid policy envelope; failing closed",
				"tenant_id", tenantID, "code", code, "error", err)
			return nil, fmt.Errorf("%w: %s: %v", ErrDefinitionCorrupt, code, err)
		}
		def := Definition{
			ID:          row.ID,
			Code:        code,
			Name:        row.Name,
			Description: desc,
			Resource:    resource,
			Action:      action,
			TenantID:    row.TenantID,
			Policy:      policy,
		}
		defs = append(defs, def)
		// 边追加边登记：重复码检测必须在同一批租户行里生效，
		// 不能等到循环结束后才回填 byCode（否则同批重复永远查不出来）。
		byCode[code] = &defs[len(defs)-1]
	}

	entry := &registryEntry{defs: defs, byCode: byCode, loadedAt: time.Now()}
	r.mu.Lock()
	r.cache[tenantID] = entry
	r.mu.Unlock()
	return entry, nil
}

// ValidateBootstrap 校验「引导清单 ⊆ DB 定义」与「码三元组一致」，
// 供生产启动自检使用：引导清单只是初始化输入，不能替代 DB 成为第二真源。
func (r *Registry) ValidateBootstrap(ctx context.Context, tenantID int) error {
	defs, err := r.All(ctx, tenantID)
	if err != nil {
		return err
	}
	actual := make(map[string]Definition, len(defs))
	for _, d := range defs {
		actual[d.Code] = d
	}
	missing := make([]string, 0)
	for _, def := range Definitions() {
		row, ok := actual[def.Code]
		if !ok {
			missing = append(missing, def.Code)
			continue
		}
		if row.Resource != def.Resource || row.Action != def.Action {
			return fmt.Errorf("permission %q: DB triple %s:%s != bootstrap %s:%s",
				def.Code, row.Resource, row.Action, def.Resource, def.Action)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("DB permission catalog is missing %d bootstrap codes: %s",
			len(missing), strings.Join(missing, ", "))
	}
	return nil
}
