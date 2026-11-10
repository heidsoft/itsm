package bpmn

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// WorkflowTemplateCatalog 是模板目录 handler 消费的窄契约（批次 3 第 1 步）。
//
// 方法集只由 handlers/bpmn/workflow_template.go 的真实调用面决定；值类型与
// 错误 sentinel 全部取自 dto，因此本文件不引用共享层包（Gate C.8 / L.3 全文
// 扫描会把注释里的包路径也算命中，这里刻意不写）。
// 后续各组按同样方式在本文件追加接口，直到整个 bpmn 域归零。
type WorkflowTemplateCatalog interface {
	List(ctx context.Context, tenantID int, keyword, domain, status string, page, pageSize int) (*dto.WorkflowTemplateListResponse, error)
	Get(ctx context.Context, tenantID int, key, version string) (*dto.WorkflowTemplate, error)
	CreateDraft(ctx context.Context, tenantID, userID int, req *dto.CreateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error)
	UpdateDraft(ctx context.Context, tenantID int, key string, req *dto.UpdateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error)
	Publish(ctx context.Context, tenantID int, key string) (*dto.WorkflowTemplate, error)
	Archive(ctx context.Context, tenantID int, key string) error
	Reload(ctx context.Context, tenantID int, key string) (*dto.ReloadWorkflowTemplateResponse, error)
	Versions(ctx context.Context, tenantID int, key string) ([]*dto.WorkflowTemplate, error)
}

// 批次 3 第 2 步：监控 / 审计 / 指标 / SLA / 租户统计五组窄契约。
// 方法集等于 dashboard.go + monitoring.go 的真实调用面（14 个方法），
// 请求与响应值类型全部下沉到 dto 并由共享层保留别名，因此这里只需引用 dto、
// ent 与内置类型；注入点仍传具体指针，组合根无需改动。

// DashboardMetricsProvider 仪表盘 / 流程指标 / 瓶颈分析
type DashboardMetricsProvider interface {
	GetDashboardMetrics(ctx context.Context, tenantID int, startTime, endTime time.Time) (*dto.DashboardMetrics, error)
	GetProcessMetrics(ctx context.Context, processDefinitionKey string, tenantID int, startTime, endTime time.Time) (*dto.BPMNProcessMetrics, error)
	GetBottleneckAnalysis(ctx context.Context, processDefinitionKey string, tenantID int) ([]dto.BottleneckInfo, error)
}

// BPMNAuditLogReader 审计日志查询与用户活动
type BPMNAuditLogReader interface {
	QueryAuditLogs(ctx context.Context, req *dto.QueryAuditLogsRequest) ([]*ent.ProcessAuditLog, int, error)
	GetUserActivity(ctx context.Context, userID int, tenantID int, startTime, endTime time.Time) ([]*ent.ProcessAuditLog, error)
}

// BPMNTenantStatsProvider 租户维度 BPMN 统计
type BPMNTenantStatsProvider interface {
	GetTenantStatistics(ctx context.Context, tenantID int) (*dto.TenantBPMNStats, error)
}

// BPMNSLAProvider SLA 违规与合规率
type BPMNSLAProvider interface {
	CheckSLAViolations(ctx context.Context, tenantID int) ([]*dto.SLAViolation, error)
	GetSLAComplianceRate(ctx context.Context, processDefinitionKey string, startTime, endTime time.Time, tenantID int) (float64, int, int, error)
}

// ProcessMonitoringProvider 流程监控（指标 / 实例状态 / 时间线 / 告警 / 健康度）
type ProcessMonitoringProvider interface {
	GetProcessMetrics(ctx context.Context, req *dto.ProcessMetricsRequest) (*dto.ProcessMetrics, error)
	GetProcessInstanceStatus(ctx context.Context, processInstanceID int, tenantID int) (*dto.ProcessInstanceStatus, error)
	ListProcessInstancesStatus(ctx context.Context, query *dto.ListProcessInstanceStatusQuery) ([]*dto.ProcessInstanceStatus, int, error)
	GetProcessTimeline(ctx context.Context, processInstanceKey string, tenantID int) ([]*dto.ProcessTimelineEntry, error)
	GetPerformanceAlerts(ctx context.Context, tenantID int) ([]map[string]interface{}, error)
	GetSystemHealth(ctx context.Context, tenantID int) (map[string]interface{}, error)
}
