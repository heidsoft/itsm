package dto

import "time"

// ProvisioningTaskResponse 交付任务响应
type ProvisioningTaskResponse struct {
	ID               int            `json:"id"`
	ServiceRequestID int            `json:"serviceRequestId"`
	Provider         string         `json:"provider"`
	ResourceType     string         `json:"resourceType"`
	Status           string         `json:"status"`
	Payload          map[string]any `json:"payload,omitempty"`
	Result           map[string]any `json:"result,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	// AttemptCount / MaxAttempts：调用方据此判断「还能不能重试」，
	// 而不是靠试一次拿 409 才知道。
	AttemptCount  int        `json:"attemptCount"`
	MaxAttempts   int        `json:"maxAttempts"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
	// ResolvedByUserID / ResolutionNote：人工接管的留痕（谁判定、依据是什么）。
	ResolvedByUserID int       `json:"resolvedByUserId,omitempty"`
	ResolutionNote   string    `json:"resolutionNote,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// StartProvisioningResponse 启动交付响应
type StartProvisioningResponse struct {
	Task *ProvisioningTaskResponse `json:"task"`
}

// RetryProvisioningTaskRequest 重试交付任务请求。
//
// reason 非必填但会被记录：运维点「重试」时写一句为什么，事后复盘能区分
// 「网络抖动自动重试」与「换了凭据人工重试」。
type RetryProvisioningTaskRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ResolveProvisioningTaskRequest 人工接管判定请求。
//
// outcome 只允许 delivered|failed：交付的终态只有两种，不接受「待定」。
// note 必填——没有依据的人工判定无法审计，等于把系统状态交给不可追溯的操作。
type ResolveProvisioningTaskRequest struct {
	Outcome string `json:"outcome" binding:"required,oneof=delivered failed"`
	Note    string `json:"note" binding:"required"`
}
