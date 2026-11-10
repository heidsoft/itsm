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

// 批次 3 第 3 步：流程触发 / 流程绑定 / 层级域配置三组窄契约。
// 方法集等于 process_trigger.go 的真实调用面（5 + 8 + 3 = 16 个方法）。
// GetEffectiveConfig 的返回值与「各层级均未命中」的 sentinel 已下沉到 dto，
// 共享层保留同名别名，因此这里只需引用 dto、ent 与内置类型；
// 装配点仍传具体指针，组合根无需改动。

// ProcessTriggerRunner 流程实例触发生命周期
type ProcessTriggerRunner interface {
	TriggerProcess(ctx context.Context, req *dto.ProcessTriggerRequest) (*dto.ProcessTriggerResponse, error)
	GetProcessStatus(ctx context.Context, processInstanceID int, tenantID int) (*dto.ProcessTriggerResponse, error)
	CancelProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error
	SuspendProcess(ctx context.Context, processInstanceID int, reason string, tenantID int) error
	ResumeProcess(ctx context.Context, processInstanceID int, tenantID int) error
}

// ProcessBindingCatalog 流程绑定与部门默认流程模板
type ProcessBindingCatalog interface {
	CreateBinding(ctx context.Context, binding *dto.ProcessBinding) (*dto.ProcessBinding, error)
	GetBinding(ctx context.Context, id int, tenantID int) (*dto.ProcessBinding, error)
	QueryBindings(ctx context.Context, req *dto.ProcessBindingQueryRequest) ([]*dto.ProcessBinding, error)
	UpdateBinding(ctx context.Context, id int, binding *dto.ProcessBinding) (*dto.ProcessBinding, error)
	DeleteBinding(ctx context.Context, id int, tenantID int) error
	GetBindingsByBusinessType(ctx context.Context, businessType dto.BusinessType, tenantID int) ([]*dto.ProcessBinding, error)
	GetDepartmentBindings(ctx context.Context, tenantID, departmentID int) ([]*dto.ProcessBinding, error)
	InitDepartmentDefaultBindings(ctx context.Context, tenantID, departmentID int, departmentType string) error
}

// 批次 3 第 4 步：引擎 / 流程定义 / 实例 / 任务 / 版本组窄契约。
//
// ProcessEngine 的三个访问器返回的是共享层包内的命名接口，Go 要求方法签名
// 的类型完全一致（不能换成另一个结构等价的域内接口），因此这三个子服务接口
// 的唯一定义收敛到 dto，共享层保留同名别名；本契约引用同一批 dto 类型，
// `service.ProcessEngine` 的方法集因此覆盖 WorkflowEngine，组合根无需改动。
// 版本服务只需方法集一致，装配点仍传具体指针。

// WorkflowEngine 是 workflow.go 消费的引擎门面。
type WorkflowEngine interface {
	ProcessDefinitionService() dto.ProcessDefinitionCatalog
	ProcessInstanceService() dto.ProcessInstanceCatalog
	TaskService() dto.TaskCatalog
	StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error)
	SuspendProcess(ctx context.Context, processInstanceID string, reason string) error
	ResumeProcess(ctx context.Context, processInstanceID string) error
	TerminateProcess(ctx context.Context, processInstanceID string, reason string) error
}

// VersionCatalog 是 workflow.go 版本管理路由消费的窄契约（8 个方法）。
type VersionCatalog interface {
	CreateVersion(ctx context.Context, req *dto.CreateVersionRequest) (*dto.ProcessVersion, error)
	GetVersion(ctx context.Context, processKey string, version int, tenantID int) (*dto.ProcessVersion, error)
	ListVersions(ctx context.Context, processKey string, tenantID int) ([]*dto.ProcessVersion, error)
	ActivateVersion(ctx context.Context, processKey string, version int, tenantID int) error
	RollbackToVersion(ctx context.Context, processKey string, targetVersion int, tenantID int, reason string) error
	CompareVersions(ctx context.Context, processKey string, baseVersion, targetVersion int, tenantID int) (*dto.VersionComparison, error)
	GetChangeLogsByProcessKey(ctx context.Context, processKey string, tenantID int) ([]*ent.ProcessVersionChangelog, error)
	GetChangeLogsByProcessDefinitionID(ctx context.Context, processDefID, tenantID int) ([]*ent.ProcessVersionChangelog, error)
}

// AIGeneratorCatalog 是 ai_generator.go 消费的窄契约（生成 / 预览 / 模板推荐
// 三条路由的调用并集，缺一条就会让对应端点在编译期丢失方法）。
type AIGeneratorCatalog interface {
	GenerateBPMN(ctx context.Context, req *dto.GenerateBPMNRequest, autoDeploy bool) (*dto.GenerateBPMNResponse, error)
	PreviewBPMN(ctx context.Context, req *dto.PreviewBPMNRequest) (*dto.PreviewBPMNResponse, error)
	SuggestTemplates(ctx context.Context, tenantID int, keyword, processType string) ([]*dto.BPMNTemplateSuggestion, error)
}

// DomainConfigResolver 全局 → 租户 → 部门 → 团队的层级配置解析
type DomainConfigResolver interface {
	ListConfigs(ctx context.Context, tenantID int, configType string) ([]*ent.DomainConfig, error)
	SetConfig(
		ctx context.Context,
		tenantID, departmentID, teamID int,
		configType, configKey string,
		configValue map[string]interface{},
		inheritMode string,
		description string,
	) error
	GetEffectiveConfig(
		ctx context.Context,
		tenantID, departmentID, teamID int,
		configType, configKey string,
	) (*dto.ResolvedConfig, error)
}
