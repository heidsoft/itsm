package workbench

import "time"

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
}

// WorkbenchResponse represents the paginated workbench response.
type WorkbenchResponse struct {
	Items      []WorkbenchItem `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}
