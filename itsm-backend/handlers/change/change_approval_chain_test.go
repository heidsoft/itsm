package change

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupChangeChainTest 组装带审批链求值引擎的变更 Service（测试库隔离）。
func setupChangeChainTest(t *testing.T) (*Service, *ent.Client, *sql.DB, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE change_approvals (
			id INTEGER PRIMARY KEY AUTOINCREMENT, change_id INTEGER NOT NULL, tenant_id INTEGER NOT NULL,
			approver_id INTEGER NOT NULL, status TEXT NOT NULL, comment TEXT, approved_at DATETIME, created_at DATETIME, updated_at DATETIME
		);
		CREATE TABLE change_approval_chains (
			id INTEGER PRIMARY KEY AUTOINCREMENT, change_id INTEGER NOT NULL, tenant_id INTEGER NOT NULL,
			level INTEGER NOT NULL, approver_id INTEGER NOT NULL, role TEXT NOT NULL, status TEXT NOT NULL,
			is_required BOOLEAN NOT NULL, approval_type TEXT NOT NULL DEFAULT 'serial', threshold INTEGER NOT NULL DEFAULT 1, created_at DATETIME
		);
	`)
	require.NoError(t, err)

	tenant, err := client.Tenant.Create().SetName("ChangeChainTenant").SetCode("CC" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain("cc.test").SetStatus("active").Save(context.Background())
	require.NoError(t, err)

	repo := NewEntRepository(client, db)
	logger := zaptest.NewLogger(t).Sugar()
	svc := &Service{
		repo:          repo,
		logger:        logger,
		entClient:     client,
		approvalChain: service.NewApprovalChainService(client, logger),
	}
	return svc, client, db, tenant.ID
}

func mkChangeUser(t *testing.T, client *ent.Client, tenantID int, role string) int {
	t.Helper()
	u, err := client.User.Create().
		SetUsername("cc-" + role + "-" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetEmail(role + "-" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com").
		SetName("CC " + role).SetPasswordHash("h").SetRole(entuser.Role(role)).SetActive(true).SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

func mkChangeDraft(t *testing.T, client *ent.Client, tenantID, creatorID int) int {
	t.Helper()
	c, err := client.Change.Create().SetTitle("Chain Change").SetDescription("test").
		SetStatus("draft").SetCreatedBy(creatorID).SetTenantID(tenantID).Save(context.Background())
	require.NoError(t, err)
	return c.ID
}

func queryChangeChainApprovers(t *testing.T, db *sql.DB, changeID, tenantID int) []int {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT approver_id FROM change_approval_chains WHERE change_id=$1 AND tenant_id=$2 ORDER BY level, approver_id`,
		changeID, tenantID)
	require.NoError(t, err)
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		require.NoError(t, rows.Scan(&id))
		out = append(out, id)
	}
	return out
}

// TestChange_SubmitChange_UsesResolvedChainApprovers 验证：提交变更时若存在激活的
// "change" 审批链，则以链解析出的审批人（租户隔离）生成审批链，而非调用方传入/创建人自审。
func TestChange_SubmitChange_UsesResolvedChainApprovers(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	chainReq := &dto.ApprovalChainRequest{
		Name:       "Change Chain",
		EntityType: "change",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "L1", IsRequired: true, ApprovalType: "serial"},
			{Level: 2, Role: "security", Name: "L2", IsRequired: true, ApprovalType: "serial"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	mgr := mkChangeUser(t, client, tenantID, "manager")
	sec := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// 调用方未传审批人 → 应被链解析出的 [mgr, sec] 覆盖，而非创建人自审
	req := &dto.SubmitChangeRequest{ApproverIDs: nil, Comment: "submit"}
	_, err = svc.SubmitChange(ctx, changeID, tenantID, creator, req)
	require.NoError(t, err)

	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.Equal(t, []int{mgr, sec}, approvers, "审批链应解析出 manager+security，而非创建人自审")
	require.NotContains(t, approvers, creator, "创建人不应出现在审批人中")
}

// TestChange_SubmitChange_ChainBlockedFailsClosed 验证：激活链存在但必需层无审批人且
// 策略为阻断时，提交失败关闭（不创建审批记录、变更保持 draft）。
func TestChange_SubmitChange_ChainBlockedFailsClosed(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	// 必需层绑定一个租户内不存在的角色 → 解析不到审批人 → 默认 fallback=block → 阻塞
	chainReq := &dto.ApprovalChainRequest{
		Name:       "Blocking Chain",
		EntityType: "change",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "no_such_role_xyz", Name: "L1", IsRequired: true, ApprovalType: "serial"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	req := &dto.SubmitChangeRequest{ApproverIDs: nil, Comment: "submit"}
	_, err = svc.SubmitChange(ctx, changeID, tenantID, creator, req)
	require.Error(t, err, "链阻塞应失败关闭")

	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.Empty(t, approvers, "阻塞时不应写入任何审批记录")
}

// TestChange_SubmitChange_NoChainFallsBackToCreator 验证：无激活链时回退旧逻辑
// （调用方未传审批人 → 默认创建人），保持向后兼容。
func TestChange_SubmitChange_NoChainFallsBackToCreator(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	req := &dto.SubmitChangeRequest{ApproverIDs: nil, Comment: "submit"}
	_, err := svc.SubmitChange(ctx, changeID, tenantID, creator, req)
	require.NoError(t, err)

	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.Equal(t, []int{creator}, approvers, "无激活链时应回退创建人自审（旧逻辑）")
}

// seedApprovalDecisions 按 approverIDs 注入 ProcessApprovalDecision 审计行（businessType=change）。
// dc5fbf97 起审批决定由 BPMN bridge 写入 ProcessApprovalDecision（不可变审计表），
// change_approvals legacy 表已停止写入；测试直接注入审计行驱动 quorum 求值。
func seedApprovalDecisions(t *testing.T, client *ent.Client, tenantID, changeID, actorID int, approverIDs []int) {
	t.Helper()
	ctx := context.Background()
	for _, aid := range approverIDs {
		_, err := client.ProcessApprovalDecision.Create().
			SetProcessInstanceID(changeID).SetProcessTaskID(changeID*100 + aid).
			SetProcessInstanceKey(fmt.Sprintf("change-evt-%d", changeID)).
			SetTaskID(fmt.Sprintf("TASK-%d-%d", changeID, aid)).
			SetProcessDefinitionKey("change").SetNodeKey("Approval_1").
			SetBusinessType("change").SetBusinessID(strconv.Itoa(changeID)).
			SetActorID(aid).SetActorName(fmt.Sprintf("approver-%d", aid)).
			SetAction("approve").SetDecision("approved").
			SetTenantID(tenantID).Save(ctx)
		require.NoError(t, err)
	}
	_ = actorID
}

// queryChangeStatus 读取变更当前状态。
func queryChangeStatus(t *testing.T, client *ent.Client, changeID int) string {
	t.Helper()
	c, err := client.Change.Get(context.Background(), changeID)
	require.NoError(t, err)
	return c.Status
}

// TestChange_Advancement_ParallelAllMustApprove 验证：同一级 parallel（会签）要求
// 全部候选人通过；少一人仍 pending，集齐后整体 approved。
// dc5fbf97 契约：审批决定以 ProcessApprovalDecision 审计行为权威，
// quorum 求值由 checkAndTransitionChange 按 (approver, level) 双重匹配驱动。
func TestChange_Advancement_ParallelAllMustApprove(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	a := mkChangeUser(t, client, tenantID, "manager")
	b := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	plan := []ApprovalLevelPlan{{Level: 1, ApprovalType: "parallel", Threshold: 2, Required: true, ApproverIDs: []int{a, b}}}
	require.NoError(t, svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "submit"))
	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.Equal(t, []int{a, b}, approvers)

	// 仅一人批准 → 仍 pending
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{a})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusPending), queryChangeStatus(t, client, changeID), "会签未集齐应仍 pending")

	// 第二人批准 → approved（同一任务多行审计，验证唯一索引已放宽）
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{b})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusApproved), queryChangeStatus(t, client, changeID))
}

// TestChange_Advancement_OrTwoChooseOne 验证：or（或签，阈值=1）任意一人批准即整体 approved。
func TestChange_Advancement_OrTwoChooseOne(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	a := mkChangeUser(t, client, tenantID, "manager")
	b := mkChangeUser(t, client, tenantID, "security")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	plan := []ApprovalLevelPlan{{Level: 1, ApprovalType: "or", Threshold: 1, Required: true, ApproverIDs: []int{a, b}}}
	require.NoError(t, svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "submit"))
	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.Equal(t, []int{a, b}, approvers)

	// 第一人批准即满足阈值 → approved
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{a})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusApproved), queryChangeStatus(t, client, changeID))
}

// TestChange_Advancement_ThresholdNofM 验证：parallel 但阈值<N（3 候选需 2 票）的多数决。
func TestChange_Advancement_ThresholdNofM(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	a := mkChangeUser(t, client, tenantID, "manager")
	b := mkChangeUser(t, client, tenantID, "security")
	c := mkChangeUser(t, client, tenantID, "technician")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	plan := []ApprovalLevelPlan{{Level: 1, ApprovalType: "parallel", Threshold: 2, Required: true, ApproverIDs: []int{a, b, c}}}
	require.NoError(t, svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "submit"))
	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.ElementsMatch(t, []int{a, b, c}, approvers)

	// 1 票 → 仍 pending
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{a})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusPending), queryChangeStatus(t, client, changeID), "2 票阈值未达时应仍 pending")

	// 2 票 → approved
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{b})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusApproved), queryChangeStatus(t, client, changeID), "达到 2 票阈值应 approved")
}

// TestChange_Advancement_ReviewResolver 验证：审批链步骤 role=review:REVIEW 时，引擎解析出
// 评审组活跃成员作为审批人；parallel 需全员通过才整体 approved（消除原独立评审双路径）。
func TestChange_Advancement_ReviewResolver(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	ca := mkChangeUser(t, client, tenantID, "manager")
	cb := mkChangeUser(t, client, tenantID, "security")
	_, err := client.ChangeReviewMember.Create().SetUserID(ca).SetType("REVIEW").SetRole("member").SetTenantID(tenantID).SetIsActive(true).Save(ctx)
	require.NoError(t, err)
	_, err = client.ChangeReviewMember.Create().SetUserID(cb).SetType("REVIEW").SetRole("member").SetTenantID(tenantID).SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	// 激活链：评审步骤（review:REVIEW），parallel 全员通过。
	chainReq := &dto.ApprovalChainRequest{
		Name:       "Review Chain",
		EntityType: "change",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "review:REVIEW", Name: "Review", IsRequired: true, ApprovalType: "parallel"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err = acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	req := &dto.SubmitChangeRequest{ApproverIDs: nil, Comment: "submit"}
	_, err = svc.SubmitChange(ctx, changeID, tenantID, creator, req)
	require.NoError(t, err)

	approvers := queryChangeChainApprovers(t, db, changeID, tenantID)
	require.ElementsMatch(t, []int{ca, cb}, approvers, "review:REVIEW 步骤应解析出评审组活跃成员")

	// 评审 parallel 全员通过：先一人 → pending，再一人 → approved
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{ca})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusPending), queryChangeStatus(t, client, changeID))
	seedApprovalDecisions(t, client, tenantID, changeID, creator, []int{cb})
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))
	require.Equal(t, string(dto.ChangeStatusApproved), queryChangeStatus(t, client, changeID))
}
