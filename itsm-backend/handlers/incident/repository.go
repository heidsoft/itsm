package incident

import (
	"context"
	"errors"
	"time"

	"itsm-backend/ent"
	"itsm-backend/handlers/common/datascope"
)

// ErrStaleVersion 表示条件更新未命中任何行：读取快照后版本已被并发写入推进。
// service 层须将其映射为 409 冲突（提示刷新重试），不得退化成 500。
var ErrStaleVersion = errors.New("incident: stale version on conditional update")

// Repository defines the interface for incident data access
type Repository interface {
	// Incident operations
	Create(ctx context.Context, incident *Incident) (*Incident, error)
	Get(ctx context.Context, id int, tenantID int) (*Incident, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Incident, int, error)
	// Update 按 id + tenant_id + version 做条件更新并自增 version；
	// 传入的 incident.Version 必须是本次读取到的当前版本，未命中时返回 ErrStaleVersion。
	Update(ctx context.Context, incident *Incident) (*Incident, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error)
	CountByPeriod(ctx context.Context, tenantID int, start, end time.Time) (int, error)

	// Stats operations
	GetStats(ctx context.Context, tenantID int) (*IncidentStats, error)

	// GetReport 返回事件趋势报表读模型（窗口内分布与每日趋势）。
	// 与 GetStats 的租户全量标量是两套口径：本方法全部限定在 period 内，
	// 且只用 Ent 查询表达，以便在 SQLite 测试库上做契约验证。
	GetReport(ctx context.Context, tenantID int, period ReportPeriod) (*IncidentReport, error)

	// Event operations
	CreateEvent(ctx context.Context, event *IncidentEvent) (*IncidentEvent, error)
	ListEvents(ctx context.Context, incidentID int, tenantID int) ([]*IncidentEvent, error)

	// Rule operations
	ListActiveRules(ctx context.Context, tenantID int) ([]*IncidentRule, error)
	UpdateRuleStats(ctx context.Context, ruleID int, count int, lastExecutedAt time.Time) error

	// Read-model queries for handler-facing sub-resources
	GetIncidentWithEdges(ctx context.Context, id, tenantID int, edges ...string) (*ent.Incident, error)
	ListIncidentComments(ctx context.Context, incidentID, tenantID int) ([]*ent.IncidentEvent, error)
	CreateIncidentCommentEvent(ctx context.Context, event *ent.IncidentEvent) (*ent.IncidentEvent, error)
	CountTenantSLAViolations(ctx context.Context, tenantID int) (int, error)

	// GetUserNamesByIDs 批量查询用户姓名（id → name），用于列表响应回填
	// reporterName/assigneeName，避免前端展示裸用户 ID。一次 IN 查询，无 N+1。
	GetUserNamesByIDs(ctx context.Context, tenantID int, ids []int) (map[int]string, error)
}

// IncidentStats 聚合统计（按 tenant 隔离，全量口径，不受报表窗口影响）
// 字段名已统一为 camelCase JSON tag，对外暴露即前端期望字段。
type IncidentStats struct {
	TotalIncidents int `json:"totalIncidents"`
	// OpenIncidents 统计未终结事件，状态集合见 incidentOpenStatuses。
	// 2026-10 之前这里按 status='open' 过滤，而 'open' 不是事件域合法取值，
	// 导致新建/已确认/已分配等全部漏计，读数长期接近 0。
	OpenIncidents     int `json:"openIncidents"`
	CriticalIncidents int `json:"criticalIncidents"`
	MajorIncidents    int `json:"majorIncidents"`
	ResolvedIncidents int `json:"resolvedIncidents"`
	// AvgResolutionTime 平均解决时长（分钟），仅统计已解决/已关闭事件。
	AvgResolutionTime int `json:"avgResolutionTime"`
}
