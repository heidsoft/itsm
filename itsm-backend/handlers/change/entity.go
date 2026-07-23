package change

import (
	"time"
)

// Change domain entity
type Change struct {
	ID                 int
	Title              string
	Description        string
	Justification      string
	Type               string
	Status             string
	Priority           string
	ImpactScope        string
	RiskLevel          string
	AssigneeID         *int
	Assignee           *User
	CreatedBy          int
	CreatedByUser      *User
	TenantID           int
	PlannedStartDate   *time.Time
	PlannedEndDate     *time.Time
	ActualStartDate    *time.Time
	ActualEndDate      *time.Time
	ImplementationPlan string
	RollbackPlan       string
	AffectedCIs        []string
	RelatedTickets     []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// User is the minimal user projection needed by the Change domain response.
// Keeping this projection local avoids exposing the persistence model while
// still allowing repositories to hydrate user display information.
type User struct {
	ID   int
	Name string
}

// ApprovalChain represents an item in the approval workflow
type ApprovalChain struct {
	ID           int
	ChangeID     int
	Level        int
	ApproverID   int
	ApproverName string
	Role         string
	Status       string
	IsRequired   bool
	CreatedAt    time.Time
}

// ApprovalRecord represents an individual approval action
type ApprovalRecord struct {
	ID           int
	ChangeID     int
	ApproverID   int
	ApproverName string
	Status       string
	Comment      *string
	ApprovedAt   *time.Time
	CreatedAt    time.Time
}

// RiskAssessment represents the risk evaluation of a change
type RiskAssessment struct {
	ID                 int
	ChangeID           int
	TenantID           int
	RiskLevel          string
	RiskDescription    string
	ImpactAnalysis     string
	MitigationMeasures string
	ContingencyPlan    string
	RiskOwner          string
	RiskReviewDate     *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Stats represents change statistics.
// Field names and JSON tags mirror dto.ChangeStatsResponse so callers can either
// consume the domain struct directly or map it through the DTO. Status values
// follow the canonical set defined in dto.ChangeStatus (draft, pending, approved,
// scheduled, in_progress, completed, failed, rolled_back, rejected, cancelled).
type Stats struct {
	Total      int `json:"total"`
	Draft      int `json:"draft"`
	Pending    int `json:"pending"`
	Approved   int `json:"approved"`
	Scheduled  int `json:"scheduled"`
	InProgress int `json:"inProgress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	RolledBack int `json:"rolledBack"`
	Rejected   int `json:"rejected"`
	Cancelled  int `json:"cancelled"`
}
