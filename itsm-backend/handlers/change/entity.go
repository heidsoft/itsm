package change

import (
	"time"
)

// Change domain entity
type Change struct {
	ID                 int
	ChangeNumber       string
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
	TenantID     int
	Level        int
	ApproverID   int
	ApproverName string
	Role         string
	Status       string
	IsRequired   bool
	// Quorum 元数据（014 迁移新增）：本层法定人数语义。
	// ApprovalType: serial|parallel|all|or；Threshold: 本层需要的批准人数。
	ApprovalType string
	Threshold    int
	CreatedAt    time.Time
}

// ApprovalLevelPlan 是提交审批时的一层计划（来自引擎求值或旧逻辑兼容）。
// 多个 approver 同属一层时按 ApprovalType/Threshold 计算会签/或签。
type ApprovalLevelPlan struct {
	Level        int
	ApprovalType string // serial | parallel | all | or
	Threshold    int    // 本层需要的批准人数（0 表示由下游按类型推导）
	Required     bool
	ApproverIDs  []int
}

// ApprovalRecord represents an individual approval action
type ApprovalRecord struct {
	ID           int        `json:"id"`
	ChangeID     int        `json:"changeId"`
	TenantID     int        `json:"tenantId"`
	ApproverID   int        `json:"approverId"`
	ApproverName string     `json:"approverName"`
	Status       string     `json:"status"`
	Comment      *string    `json:"comment,omitempty"`
	ApprovedAt   *time.Time `json:"approvedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	// Levels 该审批人在本变更审批链中所属的层级（可能跨多层）。
	// 由 change_approval_chains 派生，用于按 (approverID, level) 双重匹配，
	// 避免跨层互相串（P1 修复）。
	Levels []int `json:"levels,omitempty"`
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
