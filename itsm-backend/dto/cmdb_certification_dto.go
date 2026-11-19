package dto

import "time"

// CI 数据认证任务（ServiceNow CMDB Health / Compliance 维度）。
//
// 一次人工确认落成一条 status=confirmed 的记录，形成可审计历史；
// pending/overdue 由 due_at 相对当前时间派生，不单独存枚举。
// CI 上的 last_confirmed_at/confirmed_by 是本表最近一条 confirmed 的冗余。

// CICertificationTaskResponse 认证任务读模型。
type CICertificationTaskResponse struct {
	ID          int        `json:"id"`
	CIID        int        `json:"ciId"`
	CIName      string     `json:"ciName,omitempty"`
	Status      string     `json:"status"`
	AssigneeID  int        `json:"assigneeId,omitempty"`
	ConfirmedBy int        `json:"confirmedBy,omitempty"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	DueAt       *time.Time `json:"dueAt,omitempty"`
	Evidence    string     `json:"evidence,omitempty"`
	Source      string     `json:"source,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// CICertificationListResponse 认证任务分页列表。
type CICertificationListResponse struct {
	Items      []*CICertificationTaskResponse `json:"items"`
	Total      int                            `json:"total"`
	Pending    int                            `json:"pending"`
	Overdue    int                            `json:"overdue"`
	Confirmed  int                            `json:"confirmed"`
	Page       int                            `json:"page"`
	PageSize   int                            `json:"pageSize"`
	TotalPages int                            `json:"totalPages"`
}

// CreateCICertificationRequest 发起认证任务请求。
type CreateCICertificationRequest struct {
	CIID       int        `json:"ciId" binding:"required,min=1"`
	AssigneeID int        `json:"assigneeId,omitempty"`
	DueAt      *time.Time `json:"dueAt,omitempty"`
	Evidence   string     `json:"evidence,omitempty" binding:"omitempty,max=2000"`
}

// ConfirmCICertificationRequest 完成认证请求。
type ConfirmCICertificationRequest struct {
	Evidence string `json:"evidence,omitempty" binding:"omitempty,max=2000"`
}
