package cloudsignal

import (
	"context"
	"database/sql"
	"time"

	"itsm-backend/connector/alert"
)

// AlertSourceCloudMonitor 是 CloudMonitor 轮询写入 alerts 时使用的 source 值。
// 它与建单桥的 JSONB 状态字段一起构成幂等边界，不要改。
const AlertSourceCloudMonitor = "cloudmonitor"

// SyncStateRepository 读写每个账号每种信号的同步游标与健康状态。
type SyncStateRepository interface {
	List(ctx context.Context, tenantID int, kind SignalKind) ([]*SyncState, error)
	// Get 返回 nil, nil 表示该账号还没有同步记录（首次运行）。
	Get(ctx context.Context, tenantID int, accountID int64, kind SignalKind) (*SyncState, error)
	MarkSuccess(ctx context.Context, tenantID int, accountID int64, kind SignalKind, watermark, now time.Time, items int64) error
	// MarkFailure 记录一次同步失败；state 只能是 unavailable。
	MarkFailure(ctx context.Context, tenantID int, accountID int64, kind SignalKind, state, code, message string, now time.Time) error
	// MarkUnconfigured 表示账号未配置凭据。它保留历史游标，便于补齐凭据后续传。
	MarkUnconfigured(ctx context.Context, tenantID int, accountID int64, kind SignalKind, now time.Time) error
}

// FindingRepository 读写云安全中心漏洞快照。
type FindingRepository interface {
	Upsert(ctx context.Context, tenantID int, finding *SecurityFinding) (id int64, created bool, err error)
	Get(ctx context.Context, tenantID int, id int64) (*SecurityFinding, error)
	List(ctx context.Context, tenantID int, filter FindingFilter) ([]*SecurityFinding, int, error)
	Summary(ctx context.Context, tenantID int) (*FindingSummary, error)
	// SetStatus 写入终态（fixed/ignored）；状态机与权限由 service 校验。
	SetStatus(ctx context.Context, tenantID int, id int64, status string, now time.Time) (bool, error)
	// MarkUnseenResolved 把本轮完整拉取中未出现的未修复漏洞标记为已修复。
	MarkUnseenResolved(ctx context.Context, tenantID int, accountID int64, vulnType string, syncedBefore, now time.Time) (int64, error)
	// LinkCIs 按 (provider, cloud_resource_id) 把漏洞关联到已发现的配置项。
	LinkCIs(ctx context.Context, tenantID int, accountID int64) (int64, error)
}

// AlertRepository 是云监控告警的持久化面。upsert 与建单桥的状态迁移都用
// 条件更新，避免并发 worker 重复建单。
type AlertRepository interface {
	Upsert(ctx context.Context, tenantID int, a *alert.StandardAlert, now time.Time) (id int64, created bool, err error)
	List(ctx context.Context, tenantID int, filter AlertFilter) ([]*AlertRecord, int, error)
	// Get 返回 nil, nil 表示告警不存在或不属于该租户。
	Get(ctx context.Context, tenantID int, id int64) (*AlertRecord, error)
	Acknowledge(ctx context.Context, tenantID int, id int64, now time.Time) (bool, error)
	Resolve(ctx context.Context, tenantID int, id int64, now time.Time) (bool, error)
	// ClaimBridgeCandidates 认领一批可建单的 P1/P2 告警。过期认领可被回收，
	// 失败达到 maxAttempts 的告警不再返回。
	ClaimBridgeCandidates(ctx context.Context, tenantID int, severities []string, limit, maxAttempts int, staleBefore, now time.Time) ([]*BridgeCandidate, error)
	MarkBridged(ctx context.Context, tenantID int, id, incidentID int64, now time.Time) error
	MarkBridgeFailed(ctx context.Context, tenantID int, id int64, reason string, now time.Time) error
}

func NewSyncStateRepository(db *sql.DB) SyncStateRepository {
	if db == nil {
		return nil
	}
	return &syncStateRepository{db: db}
}

func NewFindingRepository(db *sql.DB) FindingRepository {
	if db == nil {
		return nil
	}
	return &findingRepository{db: db}
}

func NewAlertRepository(db *sql.DB) AlertRepository {
	if db == nil {
		return nil
	}
	return &alertRepository{db: db}
}
