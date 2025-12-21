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
	Impact      string
	AssigneeID  *int
	CreatedBy   int
	TenantID    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
