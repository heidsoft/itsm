package dto

import "time"

// ChangeStatus 变更状态
type ChangeStatus string

const (
	ChangeStatusDraft         ChangeStatus = "draft"         // 草稿
	ChangeStatusPending       ChangeStatus = "pending"       // 待审批
	ChangeStatusApproved      ChangeStatus = "approved"      // 已批准
	ChangeStatusRejected      ChangeStatus = "rejected"      // 已拒绝
	ChangeStatusInProgress    ChangeStatus = "in_progress"   // 实施中
	ChangeStatusCompleted     ChangeStatus = "completed"     // 已完成
	ChangeStatusRolledBack    ChangeStatus = "rolled_back"   // 已回滚
	ChangeStatusCancelled     ChangeStatus = "cancelled"     // 已取消
)

// ChangeType 变更类型
type ChangeType string

const (
	ChangeTypeNormal   ChangeType = "normal"   // 普通变更
	ChangeTypeStandard ChangeType = "standard" // 标准变更
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
	Title              string      `json:"title" binding:"required"`              // 变更标题
	Description        string      `json:"description" binding:"required"`        // 变更描述
	Justification     string      `json:"justification" binding:"required"`      // 变更理由
	Type              ChangeType  `json:"type" binding:"required"`               // 变更类型
	Priority          ChangePriority `json:"priority" binding:"required"`        // 优先级
	ImpactScope       ChangeImpact `json:"impact_scope" binding:"required"`      // 影响范围
	RiskLevel         ChangeRisk   `json:"risk_level" binding:"required"`        // 风险等级
	PlannedStartDate  *time.Time  `json:"planned_start_date"`                    // 计划开始时间
	PlannedEndDate    *time.Time  `json:"planned_end_date"`                      // 计划结束时间
	ImplementationPlan string     `json:"implementation_plan" binding:"required"` // 实施计划
	RollbackPlan      string     `json:"rollback_plan" binding:"required"`       // 回滚计划
	AffectedCIs       []string   `json:"affected_cis"`                           // 受影响的配置项
	RelatedTickets    []string   `json:"related_tickets"`                        // 相关工单
}

// UpdateChangeRequest 更新变更请求
type UpdateChangeRequest struct {
	Title              *string      `json:"title"`               // 变更标题
	Description        *string      `json:"description"`         // 变更描述
	Justification     *string      `json:"justification"`       // 变更理由
	Type              *ChangeType  `json:"type"`                // 变更类型
	Priority          *ChangePriority `json:"priority"`          // 优先级
	ImpactScope       *ChangeImpact `json:"impact_scope"`        // 影响范围
	RiskLevel         *ChangeRisk   `json:"risk_level"`          // 风险等级
	PlannedStartDate  *time.Time   `json:"planned_start_date"`  // 计划开始时间
	PlannedEndDate    *time.Time   `json:"planned_end_date"`    // 计划结束时间
	ImplementationPlan *string     `json:"implementation_plan"`  // 实施计划
	RollbackPlan      *string     `json:"rollback_plan"`         // 回滚计划
	AffectedCIs       []string    `json:"affected_cis"`          // 受影响的配置项
	RelatedTickets    []string    `json:"related_tickets"`       // 相关工单
}

// ChangeResponse 变更响应
type ChangeResponse struct {
	ID                 int         `json:"id"`                   // 变更ID
	Title              string      `json:"title"`                // 变更标题
	Description        string      `json:"description"`          // 变更描述
	Justification     string      `json:"justification"`        // 变更理由
	Type              ChangeType  `json:"type"`                 // 变更类型
	Status            ChangeStatus `json:"status"`               // 状态
	Priority          ChangePriority `json:"priority"`           // 优先级
	ImpactScope       ChangeImpact `json:"impact_scope"`         // 影响范围
	RiskLevel         ChangeRisk   `json:"risk_level"`           // 风险等级
	AssigneeID        *int        `json:"assignee_id"`           // 处理人ID
	AssigneeName      *string     `json:"assignee_name"`         // 处理人姓名
	CreatedBy         int         `json:"created_by"`            // 创建人ID
	CreatedByName     string      `json:"created_by_name"`       // 创建人姓名
	TenantID          int         `json:"tenant_id"`             // 租户ID
	PlannedStartDate  *time.Time  `json:"planned_start_date"`   // 计划开始时间
	PlannedEndDate    *time.Time  `json:"planned_end_date"`     // 计划结束时间
	ActualStartDate   *time.Time  `json:"actual_start_date"`    // 实际开始时间
	ActualEndDate     *time.Time  `json:"actual_end_date"`      // 实际结束时间
	ImplementationPlan string     `json:"implementation_plan"`   // 实施计划
	RollbackPlan      string     `json:"rollback_plan"`          // 回滚计划
	AffectedCIs       []string   `json:"affected_cis"`           // 受影响的配置项
	RelatedTickets    []string   `json:"related_tickets"`        // 相关工单
	CreatedAt         time.Time  `json:"created_at"`             // 创建时间
	UpdatedAt         time.Time  `json:"updated_at"`             // 更新时间
}

// ChangeListResponse 变更列表响应
type ChangeListResponse struct {
	Total   int              `json:"total"`   // 总数
	Changes []ChangeResponse `json:"changes"` // 变更列表
}

// ChangeStatsResponse 变更统计响应
type ChangeStatsResponse struct {
	Total           int `json:"total"`            // 总变更数
	Pending         int `json:"pending"`          // 待审批
	Approved        int `json:"approved"`         // 已批准
	InProgress      int `json:"in_progress"`      // 实施中
	Completed       int `json:"completed"`        // 已完成
	RolledBack      int `json:"rolled_back"`      // 已回滚
	Rejected        int `json:"rejected"`         // 已拒绝
	Cancelled       int `json:"cancelled"`        // 已取消
}

// ChangeApprovalRequest 变更审批请求
type ChangeApprovalRequest struct {
	Status  ChangeStatus `json:"status" binding:"required"`  // 审批状态
	Comment *string      `json:"comment"`                    // 审批意见
}

// ChangeStatusUpdateRequest 变更状态更新请求
type ChangeStatusUpdateRequest struct {
	Status ChangeStatus `json:"status" binding:"required"` // 新状态
	Comment *string     `json:"comment"`                   // 状态变更说明
}
