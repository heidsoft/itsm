package problem

import (
	"time"
)

// Problem domain entity
type Problem struct {
	ID          int
	Title       string
	Description string
	Status      string
	Priority    string
	Category    string
	RootCause   string
	Workaround  string
	Resolution  string
	Impact      string
	AssigneeID  *int
	CreatedBy   int
	TenantID    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ResolvedAt  *time.Time
	ClosedAt    *time.Time
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
