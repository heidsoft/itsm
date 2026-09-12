package dto

import "time"

// AuditLog DTO 用于对外返回审计日志数据
type AuditLog struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	TenantID  int       `json:"tenantId"`
	UserID    int       `json:"userId"`
	// UserName 后端 service 层 join user 表填充（name 优先，缺失回退 username）。
	// 该字段变更属 DTO 增量，不影响审计中间件写入路径。
	UserName    *string `json:"userName,omitempty"`
	RequestID   string  `json:"requestId"`
	IP          string  `json:"ip"`
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	Path        string  `json:"path"`
	Method      string  `json:"method"`
	StatusCode  int     `json:"statusCode"`
	RequestBody string  `json:"requestBody"`
}

// ListAuditLogsRequest 审计日志查询请求参数
type ListAuditLogsRequest struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	UserID     *int   `form:"userId"`
	Resource   string `form:"resource"`
	Action     string `form:"action"`
	Method     string `form:"method"`
	StatusCode *int   `form:"statusCode"`
	Path       string `form:"path"`
	RequestID  string `form:"requestId"`
	From       string `form:"from"`
	To         string `form:"to"`
}

// ListAuditLogsResponse 审计日志查询响应结构
type ListAuditLogsResponse struct {
	Logs     []*AuditLog `json:"logs"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
