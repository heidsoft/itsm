package change

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupChangeChainEntRepo 组装带审批链写路径的 ent 仓储（含 change_approvals /
// change_approval_chains 裸表），用于验证生产事务路径而非 mock 分支。
func setupChangeChainEntRepo(t *testing.T, suffix string) (*ent.Client, *sql.DB, int) {
	t.Helper()
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-")+"-"+suffix)
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))
	_, err = db.ExecContext(ctx, `
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

	tenant, err := client.Tenant.Create().
		SetName("EvalNode Tenant " + suffix).
		SetCode("EV" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain("evalnode.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return client, db, tenant.ID
}

// prod 实测的 P1-1：显式指定审批人时，BPMN 任务的 assignee 回落成创建人
// （assignmentSource=triggered_by），而业务审批链是调用方传入的 approverIds。
// 两套身份互不相交 → 审批人 403「已被他人认领」、创建人 403「不是待办审批人」，
// 审批永久卡 pending。
//
// 这里用真实 ent 数据复现该形态，并验证受控桥接路径能正常完成审批。
func TestApprovalBridge_TriggeredByFallbackDoesNotDeadlock(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_triggered_by")
	tenantID, creatorID := setupChangeBridgeActor(t, entClient, "trigby")
	ctx := context.Background()

	// 审批人：业务审批链里的成员，但 BPMN 任务认领给了创建人。
	approver, err := entClient.User.Create().
		SetUsername("trigby-approver").
		SetEmail("trigby-approver@example.com").
		SetName("TrigBy Approver").
		SetPasswordHash("hash").
		SetRole("manager").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)

	c := createTestChange(repo, tenantID, creatorID)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	_, err = repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID: c.ID, ApproverID: approver.ID, Status: "pending",
	})
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: approver.ID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))

	// 复现 prod 的数据形态：任务被创建人独占认领，来源标记为 triggered_by。
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "trigby1",
		fmt.Sprintf("change:%d", c.ID), creatorID, "Activity_CABApproval", "CAB审批")
	_, err = entClient.ProcessTask.UpdateOneID(taskID).
		SetTaskVariables(map[string]interface{}{
			"taskPurpose":     "approval",
			"assignmentSource": "triggered_by",
		}).
		Save(ctx)
	require.NoError(t, err)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, approver.ID, "approved", "同意实施", "manager")
	require.NoError(t, err, "业务审批链里的审批人必须能完成被创建人占住的 BPMN 待办")
	assert.Equal(t, "approved", updated.Status)

	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status, "审批待办应被完成，流程不再卡 pending")
}

// prod 实测的 P1-3：变更流程起流后停在「变更评估」，业务审批入口永远拿不到
// 审批待办（恒 409「请勿直接业务审批」）。提交动作应当把前置评估节点带过去。
//
// 这里用真实 EntRepository 走生产路径（事务内推进），不是 mock 分支。
func TestSubmitChange_AdvancesPreApprovalEvaluationNode(t *testing.T) {
	entClient, rawDB, tenantID := setupChangeChainEntRepo(t, "evalnode")
	ctx := context.Background()

	creator := mkChangeUser(t, entClient, tenantID, "end_user")
	approver := mkChangeUser(t, entClient, tenantID, "manager")

	repo := NewEntRepository(entClient, rawDB)
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)

	changeID := mkChangeDraft(t, entClient, tenantID, creator)

	// 流程实例停在「变更评估」——prod 上每个新建变更都是这个形态。
	assessmentTaskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "evalnode1",
		fmt.Sprintf("change:%d", changeID), creator, "Activity_Assessment", "变更评估")

	_, err := svc.SubmitChange(ctx, changeID, tenantID, creator, &dto.SubmitChangeRequest{
		ApproverIDs: []int{approver},
		Comment:     "submit",
	})
	require.NoError(t, err)

	assessment, err := entClient.ProcessTask.Get(ctx, assessmentTaskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", assessment.Status, "提交动作应把前置评估节点带过去")
}

// 反向验证：流程已经在审批节点时，提交/推进不得代签审批。
// 审批必须由审批人自己完成。
func TestAdvanceWorkTask_NeverSignsApprovalNode(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_no_autosign")
	tenantID, creatorID := setupChangeBridgeActor(t, entClient, "noautosign")
	ctx := context.Background()

	repo := newMockRepository()
	c := createTestChange(repo, tenantID, creatorID)
	c.Status = "draft"
	persistChangeInEntClient(t, entClient, c)

	// 直接在审批节点上：推进前置作业的逻辑不得动它。
	approvalTaskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "noautosign1",
		fmt.Sprintf("change:%d", c.ID), creatorID, "Activity_CABApproval", "CAB审批")

	bridge := service.NewBPMNApprovalBridge(entClient, zaptest.NewLogger(t).Sugar())
	advanced, err := bridge.AdvanceBusinessWorkTaskWithClient(
		ctx, entClient, tenantID, creatorID, "change", c.ID, nil)
	require.NoError(t, err)
	assert.Zero(t, advanced, "审批节点不是前置作业节点，不应被推进")

	task, err := entClient.ProcessTask.Get(ctx, approvalTaskID)
	require.NoError(t, err)
	assert.NotEqual(t, "completed", task.Status, "审批节点不得被自动签掉")
	assert.Equal(t, strconv.Itoa(creatorID), task.Assignee)
}
