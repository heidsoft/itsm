package change

import (
	"context"
	"time"

	"itsm-backend/internal/datascope"
)

// Repository interface for Change domain
type Repository interface {
	// Change CRUD
	Create(ctx context.Context, c *Change) (*Change, error)
	Get(ctx context.Context, id int, tenantID int) (*Change, error)
	List(ctx context.Context, tenantID int, page, size int, status, search, riskLevel string, dataScope datascope.DataScope, currentUserID int) ([]*Change, int, error)
	Update(ctx context.Context, c *Change) (*Change, error)
	// UpdateStatusCAS 条件更新变更状态（乐观锁）：仅当记录当前状态等于
	// expectedStatus 时才推进到 targetStatus。返回 false 表示状态已被并发
	// 修改（或记录不存在/不属于该租户），调用方据此返回 409 冲突。
	// 这是状态转换消除 TOCTOU 竞态的关键原语，禁止绕过。
	UpdateStatusCAS(ctx context.Context, id, tenantID int, expectedStatus, targetStatus string) (bool, error)
	// SetOutcomeNote 记录实施结果说明（失败原因 / 回滚原因 / 完成说明）。
	// 有事务句柄时 service 直接在同一条状态 UPDATE 里写该列；只有无事务句柄的
	// 退化装配（mock/legacy）才走本方法，避免"状态变了、原因没留"。
	SetOutcomeNote(ctx context.Context, id, tenantID int, note string) error
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*Stats, error)
	SubmitForApproval(ctx context.Context, changeID, tenantID int, plan []ApprovalLevelPlan, comment string) error

	// Approvals
	// 审批决定的写入只有两条路径：SubmitApprovalRecordTx（提交审批记录）与
	// BPMN 桥接写 ProcessApprovalDecision（审批决定）。原先的
	// UpdateApprovalRecord（按 ID 盲改 change_approvals.status）已随无路由的
	// ProcessApproval 一并删除——它绕过"待办归属"校验，且与状态推进不同事务。
	CreateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error)
	GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error)

	// SubmitApprovalRecordTx 在同一 *sql.Tx 中原子写入审批记录并按需
	// 把变更从 draft 推进到 pending（CAS：仅当当前 status='draft'）。
	// 仓储内部开闭事务，无需 service 传入 *sql.DB。
	// 返回：
	//   - rec:     本事务写入的审批记录（已回填 ID/CreatedAt），调用方直接返回给前端；
	//   - promoted: 本次提交是否把变更从 draft 推进到 pending。
	// 审批记录与状态推进的原子性依赖本原语，禁止绕过。
	SubmitApprovalRecordTx(ctx context.Context, changeID, tenantID, approverID int, comment string, now time.Time) (rec *ApprovalRecord, promoted bool, err error)

	// Approval Workflow/Chain
	GetApprovalChain(ctx context.Context, changeID int, tenantID int) ([]*ApprovalChain, error)
	DeleteApprovalChain(ctx context.Context, changeID int, tenantID int) error
	ReplaceApprovalChain(ctx context.Context, changeID, tenantID int, chain []*ApprovalChain) error

	// Rollback checklist（回滚清单：一行一步，逐项确认与留痕）
	//
	// 清单是「失败可回退」的可验证形态：回滚前必须逐项确认，未确认的步骤会
	// 拦住 rolled_back 状态推进（见 Service.validateRollbackChecklist），
	// 因此「计划」与「执行事实」不会各写一份。
	ListRollbackSteps(ctx context.Context, changeID, tenantID int) ([]*RollbackStep, error)
	ReplaceRollbackSteps(ctx context.Context, changeID, tenantID int, steps []*RollbackStep) error
	// ResolveRollbackStep 条件更新（仅 pending 可被确认）。返回 false 表示
	// 该步骤不存在于本租户本变更，或已被别人确认过——调用方据此区分 404/409。
	ResolveRollbackStep(ctx context.Context, stepID, changeID, tenantID, actorID int, status, note string, now time.Time) (bool, error)

	// Risk Assessment
	CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error)
	GetRiskAssessment(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error)
	UpdateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error)

	// Tenant validation
	ValidateApproverBelongsToTenant(ctx context.Context, approverID, tenantID int) (bool, error)

	// Calendar view
	ListByDateRange(ctx context.Context, tenantID int, startDate, endDate, status string) ([]*Change, error)

	// FindOverlappingScheduled 查询与给定时间窗重叠且未进入终态的变更
	// （排除 changeID 自身），用于提交/排期前的窗口冲突检查。
	// 重叠判定：existing.start < window.end && existing.end > window.start。
	// 状态过滤：仅 pending/approved/scheduled/in_progress 参与
	// （draft 无排期约束，终态/失败已释放窗口）。
	FindOverlappingScheduled(ctx context.Context, tenantID int, excludeChangeID int, windowStart, windowEnd time.Time) ([]*Change, error)
}

// statusNoteUpdater 终态退化路径的组合写原语：状态 CAS 与结果说明合成同一条
// 条件 UPDATE。仅无事务句柄的装配（mock/legacy）会走到它——有事务句柄时
// service 在事务内的状态 UPDATE 里直接写 outcome_note。仓储实现本接口即可
// 消除「状态已 failed、原因为空」的崩溃窗口；未实现的旧 mock 自动退回
// UpdateStatusCAS + SetOutcomeNote 两段写，行为与历史一致。
type statusNoteUpdater interface {
	UpdateStatusCASWithNote(ctx context.Context, id, tenantID int, expectedStatus, targetStatus, note string) (bool, error)
}
