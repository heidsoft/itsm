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
	// OutcomeNote 实施结果说明：failed 记失败原因、rolled_back 记回滚原因、
	// completed 记完成说明。与 Justification（事前变更理由）分开存放，
	// 避免事后结论覆盖事前理由。
	OutcomeNote    string
	AffectedCIs    []string
	RelatedTickets []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

// TypeCount 是某个变更类型的数量，来自真实的 GROUP BY type 聚合。
type TypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// Stats represents change statistics.
// Field names and JSON tags mirror dto.ChangeStatsResponse so callers can either
// consume the domain struct directly or map it through the DTO. Status values
// follow the canonical set defined in dto.ChangeStatus (draft, pending, approved,
// scheduled, in_progress, completed, failed, rolled_back, rejected, cancelled,
// closed).
type Stats struct {
	Total      int         `json:"total"`
	Draft      int         `json:"draft"`
	Pending    int         `json:"pending"`
	Approved   int         `json:"approved"`
	Scheduled  int         `json:"scheduled"`
	InProgress int         `json:"inProgress"`
	Completed  int         `json:"completed"`
	Failed     int         `json:"failed"`
	RolledBack int         `json:"rolledBack"`
	Rejected   int         `json:"rejected"`
	Cancelled  int         `json:"cancelled"`
	Closed     int         `json:"closed"`
	ByType     []TypeCount `json:"byType"`
}

// 回滚清单步骤状态。
//
// 与 changes.status 不同，这些状态描述的是「一步回滚动作的实际结果」：
//   - pending：尚未确认。清单里还有 pending 的步骤时，变更不得被标记为已回滚；
//   - done：已执行（成功）；
//   - skipped：本步不适用（必须写明依据，否则事后无法判断是「跳过」还是「漏了」）；
//   - failed：执行失败（必须写明现象，这正是 PIR 判断影响是否复原的输入）。
//
// failed 也算「已确认」：回滚失败是真实结果，把它卡在 pending 只会让人
// 无法收尾，反而逼迫使用者把失败谎报成完成。
const (
	RollbackStepPending = "pending"
	RollbackStepDone    = "done"
	RollbackStepSkipped = "skipped"
	RollbackStepFailed  = "failed"
)

// IsRollbackStepStatusValid 校验步骤状态取值；未知值一律拒绝（失败关闭），
// 避免前端拼错状态名后写入无法解释的行。
func IsRollbackStepStatusValid(status string) bool {
	switch status {
	case RollbackStepPending, RollbackStepDone, RollbackStepSkipped, RollbackStepFailed:
		return true
	default:
		return false
	}
}

// RollbackStep 是回滚清单的一项：回滚时逐项确认，并留下结果。
//
// 清单本身即计划（description 有序），执行结果写在同一行（status/note/
// resolved_by/resolved_at），因此「计划」与「执行事实」不会分叉成两份记录。
type RollbackStep struct {
	ID             int
	ChangeID       int
	TenantID       int
	StepNo         int
	Description    string
	Status         string
	Note           string
	ResolvedBy     *int
	ResolvedByName string
	ResolvedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Resolved 表示该步已确认（可继续下一步/可收尾回滚）。
func (s *RollbackStep) Resolved() bool { return s.Status != RollbackStepPending }
