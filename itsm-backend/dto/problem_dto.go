package dto

import "time"

// CreateProblemRequest 创建问题请求
type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required,min=2,max=200"`
	Description string `json:"description" binding:"required,min=10,max=5000"`
	Priority    string `json:"priority" binding:"required"`
	Category    string `json:"category"`
	RootCause   string `json:"rootCause"`
	Impact      string `json:"impact"`
	ImpactScope string `json:"impactScope"` // 影响范围
}

// UpdateProblemRequest 更新问题请求
type UpdateProblemRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=2,max=200"`
	Description *string `json:"description" binding:"omitempty,min=10,max=5000"`
	Priority    *string `json:"priority" binding:"omitempty"`
	Status      *string `json:"status" binding:"omitempty"`
	Category    *string `json:"category" binding:"omitempty"`
	RootCause   *string `json:"rootCause" binding:"omitempty"`
	Impact      *string `json:"impact" binding:"omitempty"`
}

// ListProblemsRequest 获取问题列表请求
type ListProblemsRequest struct {
	Page      int        `json:"page" form:"page"`
	PageSize  int        `json:"pageSize" form:"pageSize"`
	Status    string     `json:"status" form:"status"`
	Priority  string     `json:"priority" form:"priority"`
	Category  string     `json:"category" form:"category"`
	Keyword   string     `json:"keyword" form:"keyword"`
	DateFrom  *time.Time `json:"dateFrom" form:"dateFrom"`
	DateTo    *time.Time `json:"dateTo" form:"dateTo"`
	SortBy    string     `json:"sortBy" form:"sortBy"`
	SortOrder string     `json:"sortOrder" form:"sortOrder"`
}

// ProblemResponse 问题响应
type ProblemResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Category    string    `json:"category"`
	RootCause   string    `json:"rootCause"`
	Impact      string    `json:"impact"`
	AssigneeID  *int      `json:"assigneeId,omitempty"`
	CreatedBy   int       `json:"createdBy"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	// 关联数据
	AssociatedTickets   []*AssociatedItemResponse `json:"associatedTickets,omitempty"`
	AssociatedIncidents []*AssociatedItemResponse `json:"associatedIncidents,omitempty"`
	AssociatedChanges   []*AssociatedItemResponse `json:"associatedChanges,omitempty"`
}

// AssociatedItemResponse 关联项响应
type AssociatedItemResponse struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Number string `json:"number,omitempty"`
	Type   string `json:"type,omitempty"`
}

// ProblemAssociationRequest 关联管理请求
type ProblemAssociationRequest struct {
	RelatedType string `json:"relatedType" binding:"required,oneof=ticket incident change"`
	RelatedIDs  []int  `json:"relatedIds" binding:"required,min=1"`
}

// ProblemRemoveAssociationRequest 移除关联请求
type ProblemRemoveAssociationRequest struct {
	RelatedType string `json:"relatedType" binding:"required,oneof=ticket incident change"`
	RelatedID   int    `json:"relatedId" binding:"required"`
}

// ProblemAssociationResponse 关联管理响应
type ProblemAssociationResponse struct {
	Tickets   []*AssociatedItemResponse `json:"tickets"`
	Incidents []*AssociatedItemResponse `json:"incidents"`
	Changes   []*AssociatedItemResponse `json:"changes"`
}

// ListProblemsResponse 问题列表响应
type ListProblemsResponse struct {
	Problems   []*ProblemResponse `json:"problems"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	TotalPages int                `json:"totalPages"`
}

// ProblemStatsResponse 问题统计响应
type ProblemStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	HighPriority int `json:"highPriority"`
}

// ProblemDetailResponse 问题详情响应
type ProblemDetailResponse struct {
	Problem ProblemResponse `json:"problem"`
}

// CreateKnownErrorFromProblemRequest 从问题创建已知错误请求
type CreateKnownErrorFromProblemRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	Symptoms         *string  `json:"symptoms"`
	RootCause        *string  `json:"rootCause"`
	Workaround       *string  `json:"workaround"`
	Resolution       *string  `json:"resolution"`
	Category         *string  `json:"category"`
	Severity         *string  `json:"severity" binding:"omitempty,oneof=critical high medium low"`
	AffectedProducts []string `json:"affectedProducts"`
	AffectedCIs      []string `json:"affectedCis"`
	Keywords         []string `json:"keywords"`
}
