package dto

import "time"

// Known Error statuses
const (
	KnownErrorStatusDraft      = "draft"
	KnownErrorStatusActive     = "active"
	KnownErrorStatusResolved   = "resolved"
	KnownErrorStatusDeprecated = "deprecated"
)

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
	RootCause        string   `json:"root_cause"`
	Workaround       string   `json:"workaround"`
	Resolution       string   `json:"resolution"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity"`
	AffectedProducts []string `json:"affected_products"`
	AffectedCIs      []string `json:"affected_cis"`
	Keywords         []string `json:"keywords"`
	ProblemID        *int     `json:"problem_id"`
}

// KEDBUpdateRequest 更新已知错误请求
type KEDBUpdateRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	Symptoms         *string  `json:"symptoms"`
	RootCause        *string  `json:"root_cause"`
	Workaround       *string  `json:"workaround"`
	Resolution       *string  `json:"resolution"`
	Category         *string  `json:"category"`
	Severity         *string  `json:"severity"`
	Status           *string  `json:"status"`
	AffectedProducts []string `json:"affected_products"`
	AffectedCIs      []string `json:"affected_cis"`
	Keywords         []string `json:"keywords"`
	IsKnownError     *bool    `json:"is_known_error"`
}

// KEDBResponse 已知错误响应 (KEDB专用，包含更多字段)
type KEDBResponse struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Symptoms         string     `json:"symptoms"`
	RootCause        string     `json:"root_cause"`
	Workaround       string     `json:"workaround"`
	Resolution       string     `json:"resolution"`
	Status           string     `json:"status"`
	Category         string     `json:"category"`
	Severity         string     `json:"severity"`
	AffectedProducts []string   `json:"affected_products"`
	AffectedCIs      []string   `json:"affected_cis"`
	Keywords         []string   `json:"keywords"`
	OccurrenceCount  int        `json:"occurrence_count"`
	ProblemID        *int       `json:"problem_id"`
	CreatedBy        int        `json:"created_by"`
	TenantID         int        `json:"tenant_id"`
	IsKnownError     bool       `json:"is_known_error"`
	FirstOccurrence  *time.Time `json:"first_occurrence"`
	LastOccurrence   *time.Time `json:"last_occurrence"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	ClosedAt         *time.Time `json:"closed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
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
	PageSize int             `json:"page_size"`
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
