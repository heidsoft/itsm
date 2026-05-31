package dto

import "time"

// ApprovalChainStepDTO 审批链步骤
type ApprovalChainStepDTO struct {
	Level      int    `json:"level"`
	ApproverID int    `json:"approver_id,omitempty"`
	Role       string `json:"role"`
	Name       string `json:"name"`
	IsRequired bool   `json:"isRequired"`
}

// ApprovalChainRequest 创建审批链请求
type ApprovalChainRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	EntityType  string                 `json:"entityType" binding:"required"`
	Chain       []ApprovalChainStepDTO `json:"chain" binding:"required"`
	Status      string                 `json:"status"`
	TenantID    int                    `json:"-"`
}

// ApprovalChainResponse 审批链响应
type ApprovalChainResponse struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EntityType  string                 `json:"entityType"`
	Chain       []ApprovalChainStepDTO `json:"chain"`
	Status      string                 `json:"status"`
	CreatedBy   int                    `json:"createdBy"`
	TenantID    int                    `json:"tenantId"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
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
	TotalSteps       int     `json:"totalSteps"`
	AvgStepsPerChain float64 `json:"avgStepsPerChain"`
}
