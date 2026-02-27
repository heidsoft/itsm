package dto

import "time"

// ApprovalChainStepDTO 审批链步骤
type ApprovalChainStepDTO struct {
	Level      int    `json:"level"`
	ApproverID int    `json:"approver_id,omitempty"`
	Role       string `json:"role"`
	Name       string `json:"name"`
	IsRequired bool   `json:"is_required"`
}

// ApprovalChainRequest 创建审批链请求
type ApprovalChainRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	EntityType  string                 `json:"entity_type" binding:"required"`
	Chain       []ApprovalChainStepDTO `json:"chain" binding:"required"`
	Status      string                 `json:"status"`
	TenantID    int                    `json:"-"`
}

// ApprovalChainResponse 审批链响应
type ApprovalChainResponse struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EntityType  string                 `json:"entity_type"`
	Chain       []ApprovalChainStepDTO `json:"chain"`
	Status      string                 `json:"status"`
	CreatedBy   int                    `json:"created_by"`
	TenantID    int                    `json:"tenant_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ApprovalChainListResponse 审批链列表响应
type ApprovalChainListResponse struct {
	Data  []ApprovalChainResponse `json:"data"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
}

// ApprovalChainStats 审批链统计
type ApprovalChainStats struct {
	Total            int     `json:"total"`
	Active           int     `json:"active"`
	Inactive         int     `json:"inactive"`
	TotalSteps       int     `json:"total_steps"`
	AvgStepsPerChain float64 `json:"avg_steps_per_chain"`
}
