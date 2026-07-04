package dto

import "time"

// StandardChangeResponse 标准变更模板响应
type StandardChangeResponse struct {
	ID                 int       `json:"id"`                 // 模板ID
	Title              string    `json:"title"`              // 模板标题
	Description        string    `json:"description"`        // 模板描述
	ImplementationPlan string    `json:"implementationPlan"` // 实施计划步骤
	RollbackPlan       string    `json:"rollbackPlan"`       // 回滚计划
	Justification      string    `json:"justification"`      // 变更理由
	Category           string    `json:"category"`           // 分类
	RiskLevel          string    `json:"riskLevel"`          // 风险等级
	ImpactScope        string    `json:"impactScope"`        // 影响范围
	ExpectedDuration   int       `json:"expectedDuration"`   // 预计工期（分钟）
	ApprovalRequired   bool      `json:"approvalRequired"`   // 是否需要审批
	AffectedCis        []string  `json:"affectedCis"`        // 典型受影响的配置项
	Prerequisites      []string  `json:"prerequisites"`      // 前置条件
	Remarks            string    `json:"remarks"`            // 备注
	CreatedBy          int       `json:"createdBy"`          // 创建人ID
	TenantID           int       `json:"tenantId"`           // 租户ID
	IsActive           bool      `json:"isActive"`           // 是否启用
	CreatedAt          time.Time `json:"createdAt"`          // 创建时间
	UpdatedAt          time.Time `json:"updatedAt"`          // 更新时间
}

// CreateStandardChangeRequest 创建标准变更模板请求
type CreateStandardChangeRequest struct {
	Title              string   `json:"title" binding:"required"`              // 模板标题
	Description        string   `json:"description"`                           // 模板描述
	ImplementationPlan string   `json:"implementationPlan" binding:"required"` // 实施计划
	RollbackPlan       string   `json:"rollbackPlan" binding:"required"`       // 回滚计划
	Justification      string   `json:"justification"`                         // 变更理由
	Category           string   `json:"category"`                              // 分类
	RiskLevel          string   `json:"riskLevel"`                             // 风险等级
	ImpactScope        string   `json:"impactScope"`                           // 影响范围
	ExpectedDuration   int      `json:"expectedDuration"`                      // 预计工期
	ApprovalRequired   bool     `json:"approvalRequired"`                      // 是否需要审批
	AffectedCis        []string `json:"affectedCis"`                           // 受影响配置项
	Prerequisites      []string `json:"prerequisites"`                         // 前置条件
	Remarks            string   `json:"remarks"`                               // 备注
}

// UpdateStandardChangeRequest 更新标准变更模板请求
type UpdateStandardChangeRequest struct {
	Title              *string  `json:"title"`              // 模板标题
	Description        *string  `json:"description"`        // 模板描述
	ImplementationPlan *string  `json:"implementationPlan"` // 实施计划
	RollbackPlan       *string  `json:"rollbackPlan"`       // 回滚计划
	Justification      *string  `json:"justification"`      // 变更理由
	Category           *string  `json:"category"`           // 分类
	RiskLevel          *string  `json:"riskLevel"`          // 风险等级
	ImpactScope        *string  `json:"impactScope"`        // 影响范围
	ExpectedDuration   *int     `json:"expectedDuration"`   // 预计工期
	ApprovalRequired   *bool    `json:"approvalRequired"`   // 是否需要审批
	AffectedCis        []string `json:"affectedCis"`        // 受影响配置项
	Prerequisites      []string `json:"prerequisites"`      // 前置条件
	Remarks            *string  `json:"remarks"`            // 备注
	IsActive           *bool    `json:"isActive"`           // 是否启用
}

// StandardChangeListResponse 标准变更模板列表响应
type StandardChangeListResponse struct {
	Total     int                      `json:"total"`     // 总数
	Templates []StandardChangeResponse `json:"templates"` // 模板列表
}

// InstantiateStandardChangeRequest 从模板实例化变更请求
type InstantiateStandardChangeRequest struct {
	Title            string     `json:"title"`            // 可选：自定义标题
	PlannedStartDate *time.Time `json:"plannedStartDate"` // 计划开始时间
	PlannedEndDate   *time.Time `json:"plannedEndDate"`   // 计划结束时间
	AffectedCis      []string   `json:"affectedCis"`      // 受影响配置项（可覆盖）
}

// StandardChangeCategory 标准变更分类统计
type StandardChangeCategory struct {
	Category string `json:"category"` // 分类名称
	Count    int    `json:"count"`    // 该分类下的模板数量
}
