package dto

import "time"

// Known Error statuses
const (
	KnownErrorStatusDraft      = "draft"
	KnownErrorStatusActive     = "active"
	KnownErrorStatusResolved   = "resolved"
	KnownErrorStatusDeprecated = "deprecated"
)

// CreateKnownErrorRequest 创建已知错误请求
type CreateKnownErrorRequest struct {
	Title            string   `json:"title" binding:"required"`
	Description      string   `json:"description"`
	Symptoms         string   `json:"symptoms"`
	RootCause        string   `json:"rootCause"`
	Workaround       string   `json:"workaround"`
	Resolution       string   `json:"resolution"`
	Status           string   `json:"status" binding:"omitempty,oneof=draft active resolved deprecated"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity" binding:"omitempty,oneof=critical high medium low"`
	AffectedProducts []string `json:"affectedProducts"`
	AffectedCIs      []string `json:"affectedCis"`
	Keywords         []string `json:"keywords"`
	CreatedBy        int      `json:"createdBy" binding:"required"`
	TenantID         int      `json:"tenantId" binding:"required"`
}

// Known Error severities
const (
	KnownErrorSeverityCritical = "critical"
	KnownErrorSeverityHigh     = "high"
	KnownErrorSeverityMedium   = "medium"
	KnownErrorSeverityLow      = "low"
)

// KEDBCreateRequest 创建已知错误请求 (KEDB专用)
type KEDBCreateRequest struct {
	Title            string   `json:"title" binding:"required"`
	Description      string   `json:"description"`
	Symptoms         string   `json:"symptoms"`
	RootCause        string   `json:"rootCause"`
	Workaround       string   `json:"workaround"`
	Resolution       string   `json:"resolution"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity"`
	AffectedProducts []string `json:"affectedProducts"`
	AffectedCIs      []string `json:"affectedCis"`
	Keywords         []string `json:"keywords"`
	ProblemID        *int     `json:"problemId"`
}

// KEDBUpdateRequest 更新已知错误请求
type KEDBUpdateRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	Symptoms         *string  `json:"symptoms"`
	RootCause        *string  `json:"rootCause"`
	Workaround       *string  `json:"workaround"`
	Resolution       *string  `json:"resolution"`
	Category         *string  `json:"category"`
	Severity         *string  `json:"severity"`
	Status           *string  `json:"status"`
	AffectedProducts []string `json:"affectedProducts"`
	AffectedCIs      []string `json:"affectedCis"`
	Keywords         []string `json:"keywords"`
	IsKnownError     *bool    `json:"isKnownError"`
}

// KEDBResponse 已知错误响应 (KEDB专用，包含更多字段)
type KEDBResponse struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Symptoms         string     `json:"symptoms"`
	RootCause        string     `json:"rootCause"`
	Workaround       string     `json:"workaround"`
	Resolution       string     `json:"resolution"`
	Status           string     `json:"status"`
	Category         string     `json:"category"`
	Severity         string     `json:"severity"`
	AffectedProducts []string   `json:"affectedProducts"`
	AffectedCIs      []string   `json:"affectedCis"`
	Keywords         []string   `json:"keywords"`
	OccurrenceCount  int        `json:"occurrenceCount"`
	ProblemID        *int       `json:"problemId"`
	CreatedBy        int        `json:"createdBy"`
	TenantID         int        `json:"tenantId"`
	IsKnownError     bool       `json:"isKnownError"`
	FirstOccurrence  *time.Time `json:"firstOccurrence"`
	LastOccurrence   *time.Time `json:"lastOccurrence"`
	ResolvedAt       *time.Time `json:"resolvedAt"`
	ClosedAt         *time.Time `json:"closedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// KEDBListRequest 列出已知错误请求
type KEDBListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	Category string `form:"category"`
	Severity string `form:"severity"`
	Keyword  string `form:"keyword"`
}

// KEDBListResponse 已知错误列表响应
type KEDBListResponse struct {
	Items    []*KEDBResponse `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// KEDBStatsResponse KEDB统计响应
type KEDBStatsResponse struct {
	Total      int `json:"total"`
	Active     int `json:"active"`
	Resolved   int `json:"resolved"`
	Deprecated int `json:"deprecated"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
}
