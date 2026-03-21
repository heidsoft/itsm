package dto

import "time"

// ChangeStatus 变更状态
type ChangeStatus string

const (
	ChangeStatusDraft      ChangeStatus = "draft"       // 草稿
	ChangeStatusPending    ChangeStatus = "pending"     // 待审批
	ChangeStatusApproved   ChangeStatus = "approved"    // 已批准
	ChangeStatusRejected   ChangeStatus = "rejected"    // 已拒绝
	ChangeStatusInProgress ChangeStatus = "in_progress" // 实施中
	ChangeStatusCompleted  ChangeStatus = "completed"   // 已完成
	ChangeStatusRolledBack ChangeStatus = "rolled_back" // 已回滚
	ChangeStatusCancelled  ChangeStatus = "cancelled"   // 已取消
)

// ChangeType 变更类型
type ChangeType string

const (
	ChangeTypeNormal    ChangeType = "normal"    // 普通变更
	ChangeTypeStandard  ChangeType = "standard"  // 标准变更
	ChangeTypeEmergency ChangeType = "emergency" // 紧急变更
)

// ChangePriority 变更优先级
type ChangePriority string

const (
	ChangePriorityLow      ChangePriority = "low"      // 低
	ChangePriorityMedium   ChangePriority = "medium"   // 中
	ChangePriorityHigh     ChangePriority = "high"     // 高
	ChangePriorityCritical ChangePriority = "critical" // 紧急
)

// ChangeImpact 变更影响范围
type ChangeImpact string

const (
	ChangeImpactLow    ChangeImpact = "low"    // 低
	ChangeImpactMedium ChangeImpact = "medium" // 中
	ChangeImpactHigh   ChangeImpact = "high"   // 高
)

// ChangeRisk 变更风险等级
type ChangeRisk string

const (
	ChangeRiskLow    ChangeRisk = "low"    // 低
	ChangeRiskMedium ChangeRisk = "medium" // 中
	ChangeRiskHigh   ChangeRisk = "high"   // 高
)

// CreateChangeRequest 创建变更请求
type CreateChangeRequest struct {
	Title              string     `json:"title" binding:"required"` // 变更标题
	Description        string     `json:"description"`              // 变更描述
	Justification      string     `json:"justification"`            // 变更理由
	Type               string     `json:"type"`                     // 变更类型: normal, standard, emergency
	Priority           string     `json:"priority"`                 // 优先级: low, medium, high, critical
	ImpactScope        string     `json:"impact_scope"`             // 影响范围: low, medium, high
	RiskLevel          string     `json:"risk_level"`               // 风险等级: low, medium, high
	PlannedStartDate   *time.Time `json:"planned_start_date"`       // 计划开始时间
	PlannedEndDate     *time.Time `json:"planned_end_date"`         // 计划结束时间
	ImplementationPlan string     `json:"implementation_plan"`      // 实施计划
	RollbackPlan       string     `json:"rollback_plan"`            // 回滚计划
	AffectedCIs        []string   `json:"affected_cis"`             // 受影响的配置项
}

// UpdateChangeRequest 更新变更请求
type UpdateChangeRequest struct {
	Title              *string         `json:"title"`               // 变更标题
	Description        *string         `json:"description"`         // 变更描述
	Justification      *string         `json:"justification"`       // 变更理由
	Type               *ChangeType     `json:"type"`                // 变更类型
	Priority           *ChangePriority `json:"priority"`            // 优先级
	ImpactScope        *ChangeImpact   `json:"impact_scope"`        // 影响范围
	RiskLevel          *ChangeRisk     `json:"risk_level"`          // 风险等级
	PlannedStartDate   *time.Time      `json:"planned_start_date"`  // 计划开始时间
	PlannedEndDate     *time.Time      `json:"planned_end_date"`    // 计划结束时间
	ImplementationPlan *string         `json:"implementation_plan"` // 实施计划
	RollbackPlan       *string         `json:"rollback_plan"`       // 回滚计划
	AffectedCIs        []string        `json:"affected_cis"`        // 受影响的配置项
	RelatedTickets     []string        `json:"related_tickets"`     // 相关工单
}

// ChangeResponse 变更响应
type ChangeResponse struct {
	ID                 int            `json:"id"`                  // 变更ID
	Title              string         `json:"title"`               // 变更标题
	Description        string         `json:"description"`         // 变更描述
	Justification      string         `json:"justification"`       // 变更理由
	Type               ChangeType     `json:"type"`                // 变更类型
	Status             ChangeStatus   `json:"status"`              // 状态
	Priority           ChangePriority `json:"priority"`            // 优先级
	ImpactScope        ChangeImpact   `json:"impactScope"`        // 影响范围
	RiskLevel          ChangeRisk     `json:"riskLevel"`          // 风险等级
	AssigneeID         *int           `json:"assigneeId"`         // 处理人ID
	AssigneeName       *string        `json:"assigneeName"`       // 处理人姓名
	CreatedBy          int            `json:"createdBy"`          // 创建人ID
	CreatedByName      string         `json:"createdByName"`     // 创建人姓名
	TenantID           int            `json:"tenantId"`           // 租户ID
	PlannedStartDate   *time.Time     `json:"plannedStartDate"`  // 计划开始时间
	PlannedEndDate     *time.Time     `json:"plannedEndDate"`    // 计划结束时间
	ActualStartDate    *time.Time     `json:"actualStartDate"`   // 实际开始时间
	ActualEndDate      *time.Time     `json:"actualEndDate"`     // 实际结束时间
	ImplementationPlan string         `json:"implementationPlan"` // 实施计划
	RollbackPlan       string         `json:"rollbackPlan"`       // 回滚计划
	AffectedCIs        []string       `json:"affectedCIs"`        // 受影响的配置项
	RelatedTickets     []string       `json:"relatedTickets"`     // 相关工单
	CreatedAt          time.Time      `json:"createdAt"`          // 创建时间
	UpdatedAt          time.Time      `json:"updatedAt"`          // 更新时间
}

// ChangeListResponse 变更列表响应
type ChangeListResponse struct {
	Total   int              `json:"total"`   // 总数
	Changes []ChangeResponse `json:"changes"` // 变更列表
}

// ChangeStatsResponse 变更统计响应
type ChangeStatsResponse struct {
	Total      int `json:"total"`       // 总变更数
	Pending    int `json:"pending"`     // 待审批
	Approved   int `json:"approved"`    // 已批准
	InProgress int `json:"in_progress"` // 实施中
	Completed  int `json:"completed"`   // 已完成
	RolledBack int `json:"rolled_back"` // 已回滚
	Rejected   int `json:"rejected"`    // 已拒绝
	Cancelled  int `json:"cancelled"`   // 已取消
}

// ChangeApprovalRequest 变更审批请求
type ChangeApprovalRequest struct {
	Status  ChangeStatus `json:"status" binding:"required"` // 审批状态
	Comment *string      `json:"comment"`                   // 审批意见
}

// ChangeStatusUpdateRequest 变更状态更新请求
type ChangeStatusUpdateRequest struct {
	Status  ChangeStatus `json:"status" binding:"required"` // 新状态
	Comment *string      `json:"comment"`                   // 状态变更说明
}

// ChangeApproval 变更审批记录
type ChangeApproval struct {
	ID           int          `json:"id"`            // 审批ID
	ChangeID     int          `json:"changeId"`     // 变更ID
	ApproverID   int          `json:"approverId"`   // 审批人ID
	ApproverName string       `json:"approverName"` // 审批人姓名
	Status       ChangeStatus `json:"status"`        // 审批状态
	Comment      *string      `json:"comment"`       // 审批意见
	ApprovedAt   *time.Time   `json:"approvedAt"`   // 审批时间
	CreatedAt    time.Time    `json:"createdAt"`    // 创建时间
}

// ChangeApprovalChain 变更审批链
type ChangeApprovalChain struct {
	ID           int       `json:"id"`            // 审批链ID
	ChangeID     int       `json:"changeId"`     // 变更ID
	Level        int       `json:"level"`         // 审批级别
	ApproverID   int       `json:"approverId"`   // 审批人ID
	ApproverName string    `json:"approverName"` // 审批人姓名
	Role         string    `json:"role"`          // 审批角色
	Status       string    `json:"status"`        // 审批状态：pending, approved, rejected
	IsRequired   bool      `json:"isRequired"`   // 是否必需审批
	CreatedAt    time.Time `json:"createdAt"`    // 创建时间
}

// ChangeRiskAssessment 变更风险评估
type ChangeRiskAssessment struct {
	ID                 int        `json:"id"`                  // 风险评估ID
	ChangeID           int        `json:"changeId"`           // 变更ID
	RiskLevel          ChangeRisk `json:"riskLevel"`          // 风险等级
	RiskDescription    string     `json:"riskDescription"`    // 风险描述
	ImpactAnalysis     string     `json:"impactAnalysis"`     // 影响分析
	MitigationMeasures string     `json:"mitigationMeasures"` // 缓解措施
	ContingencyPlan    string     `json:"contingencyPlan"`    // 应急计划
	RiskOwner          string     `json:"riskOwner"`          // 风险责任人
	RiskReviewDate     *time.Time `json:"riskReviewDate"`    // 风险评审日期
	CreatedAt          time.Time  `json:"createdAt"`          // 创建时间
	UpdatedAt          time.Time  `json:"updatedAt"`          // 更新时间
}

// ChangeImplementationPlan 变更实施计划
type ChangeImplementationPlan struct {
	ID              int        `json:"id"`               // 实施计划ID
	ChangeID        int        `json:"changeId"`        // 变更ID
	Phase           string     `json:"phase"`            // 实施阶段
	Description     string     `json:"description"`      // 阶段描述
	Tasks           []string   `json:"tasks"`            // 具体任务
	Responsible     string     `json:"responsible"`      // 负责人
	StartDate       *time.Time `json:"startDate"`       // 开始时间
	EndDate         *time.Time `json:"endDate"`         // 结束时间
	Prerequisites   []string   `json:"prerequisites"`    // 前置条件
	Dependencies    []string   `json:"dependencies"`     // 依赖关系
	SuccessCriteria string     `json:"successCriteria"` // 成功标准
	Status          string     `json:"status"`           // 状态：pending, in_progress, completed, failed
	CreatedAt       time.Time  `json:"createdAt"`       // 创建时间
	UpdatedAt       time.Time  `json:"updatedAt"`       // 更新时间
}

// ChangeRollbackPlan 变更回滚计划
type ChangeRollbackPlan struct {
	ID                int       `json:"id"`                 // 回滚计划ID
	ChangeID          int       `json:"changeId"`          // 变更ID
	TriggerConditions []string  `json:"triggerConditions"` // 触发条件
	RollbackSteps     []string  `json:"rollbackSteps"`     // 回滚步骤
	Responsible       string    `json:"responsible"`        // 负责人
	EstimatedTime     string    `json:"estimatedTime"`     // 预估时间
	CommunicationPlan string    `json:"communicationPlan"` // 沟通计划
	TestPlan          string    `json:"testPlan"`          // 测试计划
	ApprovalRequired  bool      `json:"approvalRequired"`  // 是否需要审批
	CreatedAt         time.Time `json:"createdAt"`         // 创建时间
	UpdatedAt         time.Time `json:"updatedAt"`         // 更新时间
}

// ChangeRollbackExecution 变更回滚执行记录
type ChangeRollbackExecution struct {
	ID              int        `json:"id"`                // 执行记录ID
	ChangeID        int        `json:"changeId"`         // 变更ID
	RollbackPlanID  int        `json:"rollbackPlanId"`  // 回滚计划ID
	TriggerReason   string     `json:"triggerReason"`    // 触发原因
	InitiatedBy     int        `json:"initiatedBy"`      // 发起人ID
	InitiatedByName string     `json:"initiatedByName"` // 发起人姓名
	Status          string     `json:"status"`            // 状态：initiated, in_progress, completed, failed
	StartTime       *time.Time `json:"startTime"`        // 开始时间
	EndTime         *time.Time `json:"endTime"`          // 结束时间
	Result          string     `json:"result"`            // 执行结果
	Comments        string     `json:"comments"`          // 备注
	CreatedAt       time.Time  `json:"createdAt"`        // 创建时间
	UpdatedAt       time.Time  `json:"updatedAt"`        // 更新时间
}

// CreateChangeApprovalRequest 创建变更审批请求
type CreateChangeApprovalRequest struct {
	ChangeID   int     `json:"change_id" binding:"required"`   // 变更ID
	ApproverID int     `json:"approver_id" binding:"required"` // 审批人ID
	Comment    *string `json:"comment"`                        // 审批意见
}

// UpdateChangeApprovalRequest 更新变更审批请求
type UpdateChangeApprovalRequest struct {
	Status  ChangeStatus `json:"status" binding:"required"` // 审批状态
	Comment *string      `json:"comment"`                   // 审批意见
}

// CreateChangeRiskAssessmentRequest 创建变更风险评估请求
type CreateChangeRiskAssessmentRequest struct {
	ChangeID           int        `json:"change_id" binding:"required"`           // 变更ID
	RiskLevel          ChangeRisk `json:"risk_level" binding:"required"`          // 风险等级
	RiskDescription    string     `json:"risk_description" binding:"required"`    // 风险描述
	ImpactAnalysis     string     `json:"impact_analysis" binding:"required"`     // 影响分析
	MitigationMeasures string     `json:"mitigation_measures" binding:"required"` // 缓解措施
	ContingencyPlan    string     `json:"contingency_plan"`                       // 应急计划
	RiskOwner          string     `json:"risk_owner" binding:"required"`          // 风险责任人
	RiskReviewDate     *time.Time `json:"risk_review_date"`                       // 风险评审日期
}

// CreateChangeImplementationPlanRequest 创建变更实施计划请求
type CreateChangeImplementationPlanRequest struct {
	ChangeID        int        `json:"change_id" binding:"required"`        // 变更ID
	Phase           string     `json:"phase" binding:"required"`            // 实施阶段
	Description     string     `json:"description" binding:"required"`      // 阶段描述
	Tasks           []string   `json:"tasks" binding:"required"`            // 具体任务
	Responsible     string     `json:"responsible" binding:"required"`      // 负责人
	StartDate       *time.Time `json:"start_date"`                          // 开始时间
	EndDate         *time.Time `json:"end_date"`                            // 结束时间
	Prerequisites   []string   `json:"prerequisites"`                       // 前置条件
	Dependencies    []string   `json:"dependencies"`                        // 依赖关系
	SuccessCriteria string     `json:"success_criteria" binding:"required"` // 成功标准
}

// CreateChangeRollbackPlanRequest 创建变更回滚计划请求
type CreateChangeRollbackPlanRequest struct {
	ChangeID          int      `json:"change_id" binding:"required"`          // 变更ID
	TriggerConditions []string `json:"trigger_conditions" binding:"required"` // 触发条件
	RollbackSteps     []string `json:"rollback_steps" binding:"required"`     // 回滚步骤
	Responsible       string   `json:"responsible" binding:"required"`        // 负责人
	EstimatedTime     string   `json:"estimated_time" binding:"required"`     // 预估时间
	CommunicationPlan string   `json:"communication_plan"`                    // 沟通计划
	TestPlan          string   `json:"test_plan"`                             // 测试计划
	ApprovalRequired  bool     `json:"approval_required"`                     // 是否需要审批
}

// ChangeApprovalResponse 变更审批响应
type ChangeApprovalResponse struct {
	ID           int          `json:"id"`            // 审批ID
	ChangeID     int          `json:"changeId"`     // 变更ID
	ApproverID   int          `json:"approverId"`   // 审批人ID
	ApproverName string       `json:"approverName"` // 审批人姓名
	Status       ChangeStatus `json:"status"`        // 审批状态
	Comment      *string      `json:"comment"`       // 审批意见
	ApprovedAt   *time.Time   `json:"approvedAt"`   // 审批时间
	CreatedAt    time.Time    `json:"createdAt"`    // 创建时间
}

// ChangeApprovalChainResponse 变更审批链响应
type ChangeApprovalChainResponse struct {
	ID           int       `json:"id"`            // 审批链ID
	ChangeID     int       `json:"changeId"`     // 变更ID
	Level        int       `json:"level"`         // 审批级别
	ApproverID   int       `json:"approverId"`   // 审批人ID
	ApproverName string    `json:"approverName"` // 审批人姓名
	Role         string    `json:"role"`          // 审批角色
	Status       string    `json:"status"`        // 审批状态
	IsRequired   bool      `json:"isRequired"`   // 是否必需审批
	CreatedAt    time.Time `json:"createdAt"`    // 创建时间
}

// ChangeRiskAssessmentResponse 变更风险评估响应
type ChangeRiskAssessmentResponse struct {
	ID                 int        `json:"id"`                  // 风险评估ID
	ChangeID           int        `json:"changeId"`           // 变更ID
	RiskLevel          ChangeRisk `json:"riskLevel"`          // 风险等级
	RiskDescription    string     `json:"riskDescription"`    // 风险描述
	ImpactAnalysis     string     `json:"impactAnalysis"`     // 影响分析
	MitigationMeasures string     `json:"mitigationMeasures"` // 缓解措施
	ContingencyPlan    string     `json:"contingencyPlan"`    // 应急计划
	RiskOwner          string     `json:"riskOwner"`          // 风险责任人
	RiskReviewDate     *time.Time `json:"riskReviewDate"`    // 风险评审日期
	CreatedAt          time.Time  `json:"createdAt"`          // 创建时间
	UpdatedAt          time.Time  `json:"updatedAt"`          // 更新时间
}

// ChangeImplementationPlanResponse 变更实施计划响应
type ChangeImplementationPlanResponse struct {
	ID              int        `json:"id"`               // 实施计划ID
	ChangeID        int        `json:"changeId"`        // 变更ID
	Phase           string     `json:"phase"`            // 实施阶段
	Description     string     `json:"description"`      // 阶段描述
	Tasks           []string   `json:"tasks"`            // 具体任务
	Responsible     string     `json:"responsible"`      // 负责人
	StartDate       *time.Time `json:"startDate"`       // 开始时间
	EndDate         *time.Time `json:"endDate"`         // 结束时间
	Prerequisites   []string   `json:"prerequisites"`    // 前置条件
	Dependencies    []string   `json:"dependencies"`     // 依赖关系
	SuccessCriteria string     `json:"successCriteria"` // 成功标准
	Status          string     `json:"status"`           // 状态
	CreatedAt       time.Time  `json:"createdAt"`       // 创建时间
	UpdatedAt       time.Time  `json:"updatedAt"`       // 更新时间
}

// ChangeRollbackPlanResponse 变更回滚计划响应
type ChangeRollbackPlanResponse struct {
	ID                int       `json:"id"`                 // 回滚计划ID
	ChangeID          int       `json:"changeId"`          // 变更ID
	TriggerConditions []string  `json:"triggerConditions"` // 触发条件
	RollbackSteps     []string  `json:"rollbackSteps"`     // 回滚步骤
	Responsible       string    `json:"responsible"`        // 负责人
	EstimatedTime     string    `json:"estimatedTime"`     // 预估时间
	CommunicationPlan string    `json:"communicationPlan"` // 沟通计划
	TestPlan          string    `json:"testPlan"`          // 测试计划
	ApprovalRequired  bool      `json:"approvalRequired"`  // 是否需要审批
	CreatedAt         time.Time `json:"createdAt"`         // 创建时间
	UpdatedAt         time.Time `json:"updatedAt"`         // 更新时间
}

// ChangeRollbackExecutionResponse 变更回滚执行记录响应
type ChangeRollbackExecutionResponse struct {
	ID              int        `json:"id"`                // 执行记录ID
	ChangeID        int        `json:"changeId"`         // 变更ID
	RollbackPlanID  int        `json:"rollbackPlanId"`  // 回滚计划ID
	TriggerReason   string     `json:"triggerReason"`    // 触发原因
	InitiatedBy     int        `json:"initiatedBy"`      // 发起人ID
	InitiatedByName string     `json:"initiatedByName"` // 发起人姓名
	Status          string     `json:"status"`            // 状态
	StartTime       *time.Time `json:"startTime"`        // 开始时间
	EndTime         *time.Time `json:"endTime"`          // 结束时间
	Result          string     `json:"result"`            // 执行结果
	Comments        string     `json:"comments"`          // 备注
	CreatedAt       time.Time  `json:"createdAt"`        // 创建时间
	UpdatedAt       time.Time  `json:"updatedAt"`        // 更新时间
}

// ChangeApprovalWorkflowRequest 变更审批工作流请求
type ChangeApprovalWorkflowRequest struct {
	ChangeID      int                       `json:"change_id" binding:"required"`      // 变更ID
	ApprovalChain []ChangeApprovalChainItem `json:"approval_chain" binding:"required"` // 审批链
}

// ChangeApprovalChainItem 变更审批链项目
type ChangeApprovalChainItem struct {
	Level      int    `json:"level" binding:"required"`       // 审批级别
	ApproverID int    `json:"approver_id" binding:"required"` // 审批人ID
	Role       string `json:"role" binding:"required"`        // 审批角色
	IsRequired bool   `json:"is_required"`                    // 是否必需审批
}

// ChangeApprovalSummary 变更审批摘要
type ChangeApprovalSummary struct {
	ChangeID         int                           `json:"change_id"`         // 变更ID
	CurrentLevel     int                           `json:"current_level"`     // 当前审批级别
	TotalLevels      int                           `json:"total_levels"`      // 总审批级别
	ApprovalStatus   string                        `json:"approval_status"`   // 审批状态
	NextApprover     *string                       `json:"next_approver"`     // 下一个审批人
	ApprovalHistory  []ChangeApprovalResponse      `json:"approval_history"`  // 审批历史
	PendingApprovals []ChangeApprovalChainResponse `json:"pending_approvals"` // 待审批项目
}

// SubmitChangeRequest 提交变更审批请求
type SubmitChangeRequest struct {
	ApproverIDs []int  `json:"approver_ids"` // 审批人ID列表
	Comment     string `json:"comment"`      // 提交说明（可选）
}
