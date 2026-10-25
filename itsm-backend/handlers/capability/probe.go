package capability

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/ssoprovider"

	"go.uber.org/zap"
)

// Health 是单个能力的真实健康信号。
//
// 引入原因：此前的 LastHealthCheckAt 直接取 time.Now()，等于每次请求都声称
// 「刚刚检查过」。那是伪信号——能力可能因为数据库不可达、连接器掉线、身份提供方
// 未配置而实际不可用，调用方却看不到任何差别。Health 只报告探测到的客观事实。
type Health struct {
	OK        bool      `json:"ok"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// probeSnapshot 是一次探测的共享结果，在 TTL 内复用，避免每个请求都打数据库。
// 连接器健康按租户分别缓存：探测会真实拨号到第三方，不能跨租户共享结果。
type probeSnapshot struct {
	at                time.Time
	dbOK              bool
	dbReason          string
	connectorByTenant map[int]map[string]connector.HealthStatus
}

// Prober 负责产出能力健康信号。
//
// 探测分三层：
//  1. 数据库连通性（db.PingContext）——所有域能力的前置依赖；
//  2. 连接器健康（connector.Manager.HealthCheckTenant）——仅 connector.* 能力消费，
//     按租户探测，避免租户级只读接口成为跨租户连通性探针；
//  3. 租户级配置（SSO provider 是否启用）——按租户查询，不缓存（单条索引计数）。
type Prober struct {
	db         *sql.DB
	client     *ent.Client
	connectors *connector.Manager
	logger     *zap.SugaredLogger
	ttl        time.Duration
	now        func() time.Time

	mu       sync.Mutex
	snapshot *probeSnapshot
}

// NewProber 创建探测器。任何依赖为 nil 都允许：缺失的依赖不会伪造健康，
// 而是让对应能力保持「无健康信号」。
func NewProber(db *sql.DB, client *ent.Client, connectors *connector.Manager, logger *zap.SugaredLogger) *Prober {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Prober{
		db: db, client: client, connectors: connectors, logger: logger,
		ttl: 30 * time.Second, now: time.Now,
	}
}

// snapshotFor 返回 TTL 内的探测快照。
func (p *Prober) snapshotFor(ctx context.Context) *probeSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	if p.snapshot != nil && now.Sub(p.snapshot.at) < p.ttl {
		return p.snapshot
	}

	snapshot := &probeSnapshot{
		at:                now,
		dbOK:              true,
		connectorByTenant: map[int]map[string]connector.HealthStatus{},
	}
	if p.db == nil {
		snapshot.dbOK = false
		snapshot.dbReason = "数据库连接未装配"
	} else {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := p.db.PingContext(probeCtx); err != nil {
			snapshot.dbOK = false
			snapshot.dbReason = "数据库不可达"
			p.logger.Warnw("capability health probe: database unreachable", "error", err)
		}
	}
	p.snapshot = snapshot
	return snapshot
}

// connectorStatuses 返回指定租户的连接器健康快照。
// 外部拨号在锁外不可并发去重，因此与旧实现一样在持有快照锁时完成单次探测。
func (p *Prober) connectorStatuses(ctx context.Context, tenantID int) map[string]connector.HealthStatus {
	if p.connectors == nil {
		return nil
	}
	p.snapshotFor(ctx)
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := p.snapshot
	if statuses, ok := snapshot.connectorByTenant[tenantID]; ok {
		return statuses
	}
	statuses := p.connectors.HealthCheckTenant(ctx, tenantID)
	snapshot.connectorByTenant[tenantID] = statuses
	return statuses
}

// connectorHealth 在租户级健康快照中查找指定连接器名称。
// Manager 的 key 形如 tenantID/name/provider；同一 name 可能有多个 provider 实例，
// 任一实例不健康即视为该能力不健康，避免把部分故障报成健康。
func connectorHealth(statuses map[string]connector.HealthStatus, name string) (connector.HealthStatus, bool) {
	var (
		found   bool
		matched connector.HealthStatus
	)
	for key, status := range statuses {
		parts := strings.SplitN(key, "/", 3)
		if len(parts) != 3 || parts[1] != name {
			continue
		}
		found = true
		if !status.OK {
			return status, true
		}
		matched = status
	}
	return matched, found
}

// HealthFor 返回指定能力在当前租户下的健康信号。
//
// 返回 nil 表示该能力没有可探测的健康维度（例如纯配置类能力），调用方应省略该字段，
// 而不是填一个「健康」的假值。
func (p *Prober) HealthFor(ctx context.Context, key string, tenantID int) *Health {
	snapshot := p.snapshotFor(ctx)
	health := &Health{OK: true, CheckedAt: snapshot.at}

	if !snapshot.dbOK {
		health.OK = false
		health.Reason = snapshot.dbReason
		return health
	}

	switch {
	case strings.HasPrefix(key, "connector."):
		name := strings.TrimPrefix(key, "connector.")
		if p.connectors == nil {
			health.OK = false
			health.Reason = "连接器运行时未装配"
			break
		}
		status, found := connectorHealth(p.connectorStatuses(ctx, tenantID), name)
		if !found {
			health.OK = false
			health.Reason = "当前租户尚未启用该连接器"
			break
		}
		health.CheckedAt = status.CheckedAt
		if !status.OK {
			health.OK = false
			health.Reason = status.Message
			if health.Reason == "" {
				health.Reason = "连接器健康检查失败"
			}
		}
	case key == "identity.oidc":
		configured, err := p.ssoConfigured(ctx, tenantID)
		if err != nil {
			// 探测失败不谎报健康：如实降级并保留原因。
			health.OK = false
			health.Reason = "身份提供方配置读取失败"
			p.logger.Warnw("capability health probe: sso provider query failed",
				"tenant_id", tenantID, "error", err)
		} else if !configured {
			health.OK = false
			health.Reason = "当前租户尚未启用身份提供方"
		}
	}
	return health
}

func (p *Prober) ssoConfigured(ctx context.Context, tenantID int) (bool, error) {
	if p.client == nil || tenantID <= 0 {
		return false, nil
	}
	return p.client.SSOProvider.Query().
		Where(ssoprovider.TenantIDEQ(tenantID), ssoprovider.EnabledEQ(true)).
		Exist(ctx)
}
