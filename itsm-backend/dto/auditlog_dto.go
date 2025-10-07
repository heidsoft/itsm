package dto

import "time"

// AuditLog DTO 用于对外返回审计日志数据
type AuditLog struct {
    ID          int       `json:"id"`
    CreatedAt   time.Time `json:"created_at"`
    TenantID    int       `json:"tenant_id"`
    UserID      int       `json:"user_id"`
    RequestID   string    `json:"request_id"`
    IP          string    `json:"ip"`
    Resource    string    `json:"resource"`
    Action      string    `json:"action"`
    Path        string    `json:"path"`
    Method      string    `json:"method"`
    StatusCode  int       `json:"status_code"`
    RequestBody string    `json:"request_body"`
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