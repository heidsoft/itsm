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
type probeSnapshot struct {
	at          time.Time
	dbOK        bool
	dbReason    string
	connectors  map[string]connector.HealthStatus
	connectorOK bool
}

// Prober 负责产出能力健康信号。
//
// 探测分三层：
//  1. 数据库连通性（db.PingContext）——所有域能力的前置依赖；
//  2. 连接器健康（connector.Manager.HealthCheckAll）——仅 connector.* 能力消费；
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

	snapshot := &probeSnapshot{at: now, dbOK: true}
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
	if p.connectors != nil {
		snapshot.connectors = p.connectors.HealthCheckAll(ctx)
		snapshot.connectorOK = true
	}
	p.snapshot = snapshot
	return snapshot
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
		if snapshot.connectorOK {
			status, exists := snapshot.connectors[name]
			if exists && !status.OK {
				health.OK = false
				health.Reason = status.Message
				if health.Reason == "" {
					health.Reason = "连接器健康检查失败"
				}
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
