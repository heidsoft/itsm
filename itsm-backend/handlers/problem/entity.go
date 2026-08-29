package problem

import (
	"time"
)

// Problem domain entity
type Problem struct {
	ID            int
	ProblemNumber string
	Title         string
	Description   string
	Status        string
	Priority      string
	Category      string
	RootCause     string
	Workaround    string
	Resolution    string
	Impact        string
	AssigneeID    *int
	CreatedBy     int
	TenantID      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ResolvedAt    *time.Time
	ClosedAt      *time.Time
	// 关联数据 (eager-loaded)
	Tickets   []*AssociatedItem
	Incidents []*AssociatedItem
	Changes   []*AssociatedItem
}

// AssociatedItem 关联项
type AssociatedItem struct {
	ID     int
	Title  string
	Status string
	Number string
	Type   string
}

// ProblemStats domain entity
type ProblemStats struct {
	Total        int
	Open         int
	InProgress   int
	Resolved     int
	Closed       int
	HighPriority int
}

// CategoryCount represents a category and its count.
type CategoryCount struct {
	Category string
	Count    int
}

// MonthlyCount represents a monthly count for trend analysis.
type MonthlyCount struct {
	Month    string
	Count    int
	Resolved int
	Open     int
}

// ProblemTrendData holds trend analytics data.
type ProblemTrendData struct {
	Period                 string
	TotalProblems          int
	ResolvedProblems       int
	OpenProblems           int
	ResolutionRate         float64
	AvgResolutionTimeHours float64
	CategoryBreakdown      map[string]int
	PriorityBreakdown      map[string]int
	TrendDirection         string
	TopCategories          []CategoryCount
	MonthlyTrend           []MonthlyCount
}

// ProblemHotspotData holds hotspot analytics data.
type ProblemHotspotData struct {
	PeriodStart       string
	PeriodEnd         string
	CategoryBreakdown map[string]int
	PriorityBreakdown map[string]int
	Hotspots          []string
	AvgPerCategory    float64
}
