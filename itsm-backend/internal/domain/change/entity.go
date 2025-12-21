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
	CreatedBy          int
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

// Stats represents change statistics
type Stats struct {
	Total      int
	Pending    int
	Approved   int
	InProgress int
	Completed  int
	RolledBack int
	Rejected   int
	Cancelled  int
}
