package dto

import "time"

// ProvisioningTaskResponse 交付任务响应
type ProvisioningTaskResponse struct {
	ID              int            `json:"id"`
	ServiceRequestID int           `json:"service_request_id"`
	Provider        string         `json:"provider"`
	ResourceType    string         `json:"resource_type"`
	Status          string         `json:"status"`
	Payload         map[string]any `json:"payload,omitempty"`
	Result          map[string]any `json:"result,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// StartProvisioningResponse 启动交付响应
type StartProvisioningResponse struct {
	Task *ProvisioningTaskResponse `json:"task"`
}


