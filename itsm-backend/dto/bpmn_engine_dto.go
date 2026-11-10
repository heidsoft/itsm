package dto

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/schema"
)

// 批次 3 第 4 步：BPMN 引擎 / 流程定义 / 实例 / 任务 / 版本组的共享值类型与
// 子服务契约。原先这些名字定义在共享层，handler 为了引用它们必须 import
// 共享层包；现在定义收敛到 dto，共享层只保留同名别名（见 service/ 下四个
// 文件），因此接口签名在编译期仍是同一类型身份，errors/JSON 形状零变化。
//
// 方法集逐条复制自原共享层 ProcessDefinitionService / ProcessInstanceService /
// TaskService 三个接口，不做裁剪：共享层内部也经由 ProcessEngine 的三个访问器
// 调用这些方法，裁剪会让 service 包内的调用点丢失方法。

// ---------------------------------------------------------------------------
// 流程定义

type CreateProcessDefinitionRequest struct {
	Key              string                 `json:"key" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml" binding:"required"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	// ApprovalConfig 流程级审批配置（auto_approve_roles 等），与设计器「流程配置」页签对应，
	// 必须随创建/更新写入对应的 DB 列，否则前端保存会静默丢失。
	ApprovalConfig *schema.ApprovalConfig `json:"approvalConfig"`
	SLAConfig      map[string]interface{} `json:"slaConfig"`
	TenantID       int                    `json:"tenantId" binding:"required"`
	Publish        bool                   `json:"publish"`
}

type UpdateProcessDefinitionRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	IsActive         *bool                  `json:"isActive"`
	// ApprovalConfig 非 nil 时整体替换流程级审批配置（读改写由前端负责）。
	ApprovalConfig *schema.ApprovalConfig `json:"approvalConfig"`
	SLAConfig      map[string]interface{} `json:"slaConfig"`
	// CandidateDefinition is the AI-generated business contract. It is kept
	// alongside the BPMN draft so publishing can enforce the same safety gate
	// after a human edits the diagram.
	CandidateDefinition map[string]interface{} `json:"candidateDefinition"`
}

type ListProcessDefinitionsRequest struct {
	Key      string `form:"key" json:"key"`
	Category string `form:"category" json:"category"`
	IsActive *bool  `form:"isActive" json:"isActive"`
	TenantID int    `form:"-" json:"tenantId"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

// ProcessDefinitionCatalog 流程定义服务接口（共享层同名接口的唯一定义）。
type ProcessDefinitionCatalog interface {
	CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error)
	GetProcessDefinitionByID(ctx context.Context, id int) (*ent.ProcessDefinition, error)
	GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error)
	UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	PublishProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	DeleteProcessDefinition(ctx context.Context, key string, version string) error
	ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error)
	SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error
}

// ---------------------------------------------------------------------------
// 流程实例

// ListProcessInstancesRequest 流程实例列表请求。
//
// handler 用 ShouldBindQuery 绑定：没有 form 标签时 Gin 只按 Go 字段名匹配，
// camelCase 查询参数会静默丢空，service 侧的过滤谓词因此永远收不到条件。
// TenantID 只认认证上下文注入，禁止从请求绑定。
type ListProcessInstancesRequest struct {
	ProcessDefinitionKey string `form:"processDefinitionKey" json:"processDefinitionKey"`
	Status               string `form:"status" json:"status"`
	BusinessKey          string `form:"businessKey" json:"businessKey"`
	TenantID             int    `form:"-" json:"tenantId"`
	Page                 int    `form:"page" json:"page"`
	PageSize             int    `form:"pageSize" json:"pageSize"`
}

// InstanceStatisticsRequest 实例统计请求
//
// StartDate/EndDate 的 wire 格式尚未收敛，保持不绑定：按时间过滤当前不生效，
// 但不会因为猜错格式把统计请求变成 400。
type InstanceStatisticsRequest struct {
	ProcessDefinitionKey string     `form:"processDefinitionKey" json:"processDefinitionKey"`
	Status               string     `form:"status" json:"status"`
	TenantID             int        `form:"-" json:"tenantId"`
	StartDate            *time.Time `form:"-" json:"startDate"`
	EndDate              *time.Time `form:"-" json:"endDate"`
}

// InstanceStatistics 实例统计
type InstanceStatistics struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Completed  int `json:"completed"`
	Suspended  int `json:"suspended"`
	Terminated int `json:"terminated"`
}

// ProcessInstanceCatalog 流程实例服务接口（共享层同名接口的唯一定义）。
type ProcessInstanceCatalog interface {
	GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error)
	ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error)
	GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error)
	SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error
	GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error)
	GetInstanceStatistics(ctx context.Context, req *InstanceStatisticsRequest) (*InstanceStatistics, error)
}

// ---------------------------------------------------------------------------
// 任务

type ListUserTasksRequest struct {
	Assignee        string `form:"-" json:"assignee"`
	CandidateUsers  string `form:"-" json:"candidateUsers"`
	CandidateGroups string `form:"-" json:"candidateGroups"`
	// UserID 为「我的待办」语义：查询"分配给我 OR 我在候选人 OR 我所在组作为候选组"的任务。
	// 传入后：Assignee/CandidateUsers/CandidateGroups 会被忽略（可选透传）。
	// 身份三字段一律 form:"-"：只能由 handler 从认证上下文写入。
	UserID int `form:"-" json:"userId"`
	// AllTasks is an internal authorization decision made by the controller after
	// the task:admin middleware succeeds. It must never be populated from HTTP input.
	AllTasks bool `json:"-" form:"-"`
	// ProcessInstanceID 是 process_tasks 的数值外键；列表按 PI-* 业务键过滤尚未收敛，
	// 故 form:"-"，避免前端传 "PI-xxx" 时 Gin 把它变成 400。
	ProcessInstanceID    int    `form:"-" json:"processInstanceId"`
	Status               string `form:"status" json:"status"`
	ProcessDefinitionKey string `form:"processDefinitionKey" json:"processDefinitionKey"`
	TenantID             int    `form:"-" json:"tenantId"`
	Page                 int    `form:"page" json:"page"`
	PageSize             int    `form:"pageSize" json:"pageSize"`
}

type TaskStatisticsRequest struct {
	ProcessDefinitionKey string     `form:"processDefinitionKey" json:"processDefinitionKey"`
	Assignee             string     `form:"assignee" json:"assignee"`
	Status               string     `form:"status" json:"status"`
	TenantID             int        `form:"-" json:"tenantId"`
	StartDate            *time.Time `form:"-" json:"startDate"`
	EndDate              *time.Time `form:"-" json:"endDate"`
}

type TaskStatistics struct {
	TotalTasks        int                    `json:"totalTasks"`
	CompletedTasks    int                    `json:"completedTasks"`
	PendingTasks      int                    `json:"pendingTasks"`
	OverdueTasks      int                    `json:"overdueTasks"`
	AverageCompletion float64                `json:"averageCompletion"`
	StatusBreakdown   map[string]int         `json:"statusBreakdown"`
	AssigneeBreakdown map[string]int         `json:"assigneeBreakdown"`
	TimeDistribution  map[string]interface{} `json:"timeDistribution"`
}

// CounterSignStatus 会签状态
type CounterSignStatus struct {
	ParentTaskID string `json:"parentTaskId"`
	Total        int    `json:"total"`
	Completed    int    `json:"completed"`
	Approved     int    `json:"approved"`
	Rejected     int    `json:"rejected"`
	Pending      int    `json:"pending"`
	Status       string `json:"status"` // pending, approved, rejected
}

// CounterSignRequest 会签请求
type CounterSignRequest struct {
	ApprovalType string   `json:"approvalType"` // serial, parallel
	Approvers    []string `json:"approvers"`
	Threshold    int      `json:"threshold"`
}

// VoteRequest 投票请求
type VoteRequest struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment"`
}

// TaskCatalog 任务管理服务接口（共享层同名接口的唯一定义）。
type TaskCatalog interface {
	GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error)
	GetTaskByID(ctx context.Context, id int) (*ent.ProcessTask, error)
	AuthorizeTaskAccess(ctx context.Context, task *ent.ProcessTask, requireActor bool) error
	CompleteTaskByID(ctx context.Context, id int, variables map[string]interface{}) error
	ClaimTask(ctx context.Context, taskID string, userID string) error
	ClaimTaskByID(ctx context.Context, id int, userID int) error
	ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error)
	ListUserTaskViews(ctx context.Context, req *ListUserTasksRequest) ([]*BPMNTaskResponse, int, error)
	AssignTask(ctx context.Context, taskID string, assignee string) error
	ReassignTask(ctx context.Context, taskID string, newAssigneeID int, reason string) error
	TerminateTask(ctx context.Context, taskID string, reason string) error
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	CancelTask(ctx context.Context, taskID string, reason string) error
	GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error)
	SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error
	HandleTaskTimeout(ctx context.Context, taskID string) error
	RetryTask(ctx context.Context, taskID string, maxRetries int) error
	DelegateTask(ctx context.Context, taskID string, newAssignee string) error
	DelegateTaskByID(ctx context.Context, id int, newAssignee string) error
	AddApproverTask(ctx context.Context, taskID string, newApprover string) error
	AddApproverTaskByID(ctx context.Context, id int, newApprover string) error
	EscalateTask(ctx context.Context, taskID string, reason string) error
	BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string, tenantID int) error
	GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error)
	ListApprovalDecisions(ctx context.Context, processInstanceKey string) ([]*ent.ProcessApprovalDecision, error)
	// 会签相关
	CreateCounterSignTasks(ctx context.Context, parentTaskID string, req *CounterSignRequest) ([]*ent.ProcessTask, error)
	GetCounterSignStatus(ctx context.Context, parentTaskID string) (*CounterSignStatus, error)
	Vote(ctx context.Context, taskID string, req *VoteRequest) error
}

// ---------------------------------------------------------------------------
// 版本

// ProcessVersion 流程版本信息
type ProcessVersion struct {
	ID                   string    `json:"id"`
	ProcessDefinitionKey string    `json:"processDefinitionKey"`
	Version              int       `json:"version"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	BPMNXML              string    `json:"bpmnXml"`
	DeploymentID         string    `json:"deploymentId"`
	IsActive             bool      `json:"isActive"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	CreatedBy            string    `json:"createdBy"`
	TenantID             int       `json:"tenantId"`
	ChangeLog            string    `json:"changeLog"`
	CompatibilityNotes   string    `json:"compatibilityNotes"`
}

// CreateVersionRequest 创建版本请求
type CreateVersionRequest struct {
	ProcessDefinitionKey string `json:"processDefinitionKey" binding:"required"`
	Name                 string `json:"name" binding:"required"`
	Description          string `json:"description"`
	BPMNXML              string `json:"bpmnXml" binding:"required"`
	ChangeLog            string `json:"changeLog"`
	CompatibilityNotes   string `json:"compatibilityNotes"`
	TenantID             int    `json:"tenantId"`
	CreatedBy            string `json:"createdBy"`
}

// UpdateVersionRequest 更新版本请求
type UpdateVersionRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	BPMNXML            string `json:"bpmnXml"`
	ChangeLog          string `json:"changeLog"`
	CompatibilityNotes string `json:"compatibilityNotes"`
}

// VersionComparison 版本比较结果
type VersionComparison struct {
	BaseVersion     *ProcessVersion `json:"baseVersion"`
	TargetVersion   *ProcessVersion `json:"targetVersion"`
	Changes         []ChangeDetail  `json:"changes"`
	BreakingChanges []string        `json:"breakingChanges"`
	Compatibility   string          `json:"compatibility"`
}

// ChangeDetail 变更详情
type ChangeDetail struct {
	Type        string `json:"type"`        // "added", "removed", "modified"
	ChangeType  string `json:"changeType"`  // 变更类型
	ElementType string `json:"elementType"` // "task", "gateway", "event", "flow"
	ElementID   string `json:"elementId"`
	ElementName string `json:"elementName"`
	Description string `json:"description"`
	Impact      string `json:"impact"`             // "low", "medium", "high", "critical"
	OldValue    string `json:"oldValue,omitempty"` // 旧值
	NewValue    string `json:"newValue,omitempty"` // 新值
}

// ParseVersionNumber 将版本字符串解析为整数版本号。
// 支持纯整数（"1","2"）和 semver（"1.0.0","1.1.0"）两种格式，semver 取主版本号。
//
// 原实现位于共享层，handler 因 5 处调用必须 import 共享层包。现收敛到 dto，
// 共享层保留同名函数委托到这里（单一实现），handler 经注入的函数字段调用。
func ParseVersionNumber(versionStr string) (int, error) {
	if v, err := strconv.Atoi(versionStr); err == nil {
		return v, nil
	}
	parts := strings.Split(versionStr, ".")
	if len(parts) > 0 {
		if v, err := strconv.Atoi(parts[0]); err == nil {
			return v, nil
		}
	}
	return 0, fmt.Errorf("无效的版本格式: %s", versionStr)
}
