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
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// StartProvisioningResponse 启动交付响应
type StartProvisioningResponse struct {
	Task *ProvisioningTaskResponse `json:"task"`
}
