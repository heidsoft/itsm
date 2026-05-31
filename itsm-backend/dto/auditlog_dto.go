package dto

import "time"

// AuditLog DTO 用于对外返回审计日志数据
type AuditLog struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	TenantID    int       `json:"tenantId"`
	UserID      int       `json:"userId"`
	RequestID   string    `json:"requestId"`
	IP          string    `json:"ip"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	StatusCode  int       `json:"statusCode"`
	RequestBody string    `json:"requestBody"`
}

// ListAuditLogsRequest 审计日志查询请求参数
type ListAuditLogsRequest struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	UserID     *int   `form:"user_id"`
	Resource   string `form:"resource"`
	Action     string `form:"action"`
	Method     string `form:"method"`
	StatusCode *int   `form:"status_code"`
	Path       string `form:"path"`
	RequestID  string `form:"request_id"`
	From       string `form:"from"`
	To         string `form:"to"`
}

// ListAuditLogsResponse 审计日志查询响应结构
type ListAuditLogsResponse struct {
	Logs     []*AuditLog `json:"logs"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
