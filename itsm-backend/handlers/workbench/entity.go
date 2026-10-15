package workbench

import (
	"time"

	"itsm-backend/internal/datascope"
)

// WorkbenchItem represents a unified work item from any ITIL domain.
type WorkbenchItem struct {
	ID          int       `json:"id"`
	RecordType  string    `json:"recordType"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Phase       string    `json:"phase"`
	AssigneeID  *int      `json:"assigneeId,omitempty"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WorkbenchQuery represents query parameters for the workbench API.
type WorkbenchQuery struct {
	TenantID   int      `json:"tenantId"`
	AssigneeID *int     `json:"assigneeId,omitempty"`
	Phase      []string `json:"phase,omitempty"`
	Priority   []string `json:"priority,omitempty"`
	RecordType []string `json:"recordType,omitempty"`
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`

	// 行级数据权限。零值 DataScopeAll 表示不过滤；
	// 非管理角色必须显式传入 datascope.DataScopeOwnedOrAssigned。
	//
	// 此前 workbench 只靠路由层 dashboard:read 兜底，导致普通角色的跨域工作台
	// 能看到全租户的工单/事件/问题/变更。行级过滤的单一源是 internal/datascope。
	DataScope datascope.DataScope `json:"-"`
	// ViewerID 是当前访问者 ID，用于 owner/assignee 维度过滤。
	ViewerID int `json:"-"`
}

// WorkbenchResponse represents the paginated workbench response.
type WorkbenchResponse struct {
	Items      []WorkbenchItem `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}
