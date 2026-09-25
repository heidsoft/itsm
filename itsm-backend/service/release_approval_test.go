package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

func newReleaseApprovalTestClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { client.Close() })
	return client
}

// setupReleaseApprovalFixture 创建租户、创建人、审批人和一条草稿态发布
func setupReleaseApprovalFixture(t *testing.T, client *ent.Client, code string) (tenantID, creatorID, approverID, releaseID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Release Tenant " + code).
		SetCode("rel-" + code).
		SetDomain("rel-" + code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("rel-creator-" + code).
		SetEmail("rel-creator-" + code + "@example.com").
		SetName("Release Creator " + code).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("rel-approver-" + code).
		SetEmail("rel-approver-" + code + "@example.com").
		SetName("Release Approver " + code).
		SetPasswordHash("hash").
		SetRole("manager").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	releaseEntity, err := client.Release.Create().
		SetReleaseNumber("REL-" + code).
		SetTitle("Release " + code).
		SetStatus("draft").
		SetCreatedBy(creator.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, creator.ID, approver.ID, releaseEntity.ID
}

func TestApplyReleaseApproval_ApproveMovesDraftToScheduled(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_approve_ok")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "aok")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "aok", fmt.Sprintf("release:%d", releaseID), approverID)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "approve", "同意发布")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "scheduled", resp.Status)
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.Equal(t, "approved", task.TaskVariables["approvalResult"])
	decision, err := client.ProcessApprovalDecision.Query().Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, approverID, decision.ActorID)
}

func TestApplyReleaseApproval_RejectMovesDraftToCancelled(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_reject_ok")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "rok")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "rok", fmt.Sprintf("release:%d", releaseID), approverID)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "reject", "风险过高")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "cancelled", resp.Status)
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)
	require.Equal(t, "rejected", task.TaskVariables["approvalResult"])
	decision, err := client.ProcessApprovalDecision.Query().Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, approverID, decision.ActorID)
}

// Serialize persisted fields, excluding Ent's non-comparable driver/log config.
func releaseApprovalSnapshot(t *testing.T, rows ...any) []byte {
	t.Helper()
	data, err := json.Marshal(rows)
	require.NoError(t, err)
	return data
}

func TestApplyReleaseApproval_NoBPMNDoesNotWrite(t *testing.T) {
	for _, state := range []string{"noInstance", "noPendingTask"} {
		for _, action := range []string{"approve", "reject"} {
			t.Run(state+"/"+action, func(t *testing.T) {
				client := newReleaseApprovalTestClient(t, "release-"+state+"-"+action)
				tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "absent")
				ctx := context.Background()
				if state == "noPendingTask" {
					_, taskID := createBridgeProcessFixture(t, client, tenantID, "absent", fmt.Sprintf("release:%d", releaseID), approverID)
					_, err := client.ProcessTask.UpdateOneID(taskID).SetStatus("completed").Save(ctx)
					require.NoError(t, err)
				}
				before, err := client.Release.Get(ctx, releaseID)
				require.NoError(t, err)
				tasksBefore, err := client.ProcessTask.Query().All(ctx)
				require.NoError(t, err)
				instancesBefore, err := client.ProcessInstance.Query().All(ctx)
				require.NoError(t, err)
				svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())
				for attempt := 0; attempt < 2; attempt++ {
					resp, err := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, action, "decision")
					var businessErr *common.BusinessError
					require.ErrorAs(t, err, &businessErr)
					require.Equal(t, common.ConflictCode, businessErr.Code)
					require.Nil(t, resp)
					after, err := client.Release.Get(ctx, releaseID)
					require.NoError(t, err)
					tasksAfter, err := client.ProcessTask.Query().All(ctx)
					require.NoError(t, err)
					instancesAfter, err := client.ProcessInstance.Query().All(ctx)
					require.NoError(t, err)
					require.Equal(t, releaseApprovalSnapshot(t, before, tasksBefore, instancesBefore),
						releaseApprovalSnapshot(t, after, tasksAfter, instancesAfter))
					decisions, err := client.ProcessApprovalDecision.Query().Count(ctx)
					require.NoError(t, err)
					require.Zero(t, decisions)
					audits, err := client.ProcessAuditLog.Query().Count(ctx)
					require.NoError(t, err)
					require.Zero(t, audits)
				}
			})
		}
	}
}

func TestApplyReleaseApproval_CreatorCannotSelfApprove(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_self")
	tenantID, creatorID, _, releaseID := setupReleaseApprovalFixture(t, client, "self")
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	_, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, creatorID, "approve", "")
	require.Error(t, err, "发布创建人不能审批自己的发布")

	// 状态未被改动
	releaseEntity, gerr := client.Release.Get(context.Background(), releaseID)
	require.NoError(t, gerr)
	assert.Equal(t, "draft", releaseEntity.Status)
}

func TestApplyReleaseApproval_NonDraftStatusRejected(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_nondraft")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "nd")
	ctx := context.Background()
	_, err := client.Release.UpdateOneID(releaseID).SetStatus("scheduled").Save(ctx)
	require.NoError(t, err)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	_, err = svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, "approve", "")
	require.Error(t, err, "非草稿态发布不允许重复审批")
}

func TestApplyReleaseApproval_InvalidApproverForbidden(t *testing.T) {
	for _, identity := range []string{"foreign", "inactive", "unknown"} {
		for _, action := range []string{"approve", "reject"} {
			t.Run(identity+"/"+action, func(t *testing.T) {
				client := newReleaseApprovalTestClient(t, "rel_invalid_actor_"+identity+"_"+action)
				tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "local")
				ctx := context.Background()
				switch identity {
				case "foreign":
					_, _, approverID, _ = setupReleaseApprovalFixture(t, client, "foreign")
				case "inactive":
					_, err := client.User.UpdateOneID(approverID).SetActive(false).Save(ctx)
					require.NoError(t, err)
				case "unknown":
					approverID += 9999
				}
				mutations := 0
				client.Use(func(next ent.Mutator) ent.Mutator {
					return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
						mutations++
						return next.Mutate(ctx, mutation)
					})
				})
				svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())
				for attempt := 0; attempt < 2; attempt++ {
					resp, err := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, action, "decision")
					var appErr *common.AppError
					require.ErrorAs(t, err, &appErr)
					require.Equal(t, common.ErrCodeForbidden, appErr.Code)
					require.Equal(t, "审批人不存在或已停用", appErr.Message)
					require.Nil(t, resp)
				}
				require.Zero(t, mutations)
			})
		}
	}
}

func TestApplyReleaseApproval_TenantIsolation(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_tenant_iso")
	_, _, _, releaseID := setupReleaseApprovalFixture(t, client, "tiA")
	tenantB, _, approverB, _ := setupReleaseApprovalFixture(t, client, "tiB")
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	// 租户 B 审批租户 A 的发布：按未找到处理（返回 nil, nil）
	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantB, approverB, "approve", "")
	require.NoError(t, err)
	assert.Nil(t, resp, "跨租户不得命中其他租户的发布")
}

func TestApplyReleaseApproval_BridgesBPMNTask(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_bridge")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "br")
	// 关联运行中的 BPMN 审批流程，待办任务指派给审批人
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "relbr1",
		fmt.Sprintf("release:%d", releaseID), approverID)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "approve", "同意")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "scheduled", resp.Status)

	// BPMN 待办任务应被桥接完成
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)
}

func TestApplyReleaseApproval_BridgeFailClosedForUnauthorizedActor(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_bridge_fc")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "fc")
	// BPMN 任务指派给其他人：审批人不是流程任务审批人，必须 fail-closed
	createBridgeProcessFixture(t, client, tenantID, "relfc1",
		fmt.Sprintf("release:%d", releaseID), approverID+1000)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	_, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "approve", "")
	require.Error(t, err, "BPMN 任务审批人不匹配时必须中止业务侧审批")

	// 发布状态未被改动
	releaseEntity, gerr := client.Release.Get(context.Background(), releaseID)
	require.NoError(t, gerr)
	assert.Equal(t, "draft", releaseEntity.Status)
}

// 审批成功后再次审批必须被拒绝，状态与流程均不得变化。
func TestApplyReleaseApproval_DuplicateApprovalRejected(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_dup")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "dup")
	ctx := context.Background()
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "dup1",
		fmt.Sprintf("release:%d", releaseID), approverID)
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, "approve", "同意")
	require.NoError(t, err)
	require.Equal(t, "scheduled", resp.Status)

	snapAfterFirst := releaseApprovalSnapshot(t,
		mustRelease(t, client, releaseID),
		mustTask(t, client, taskID))

	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			resp2, err2 := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, action, "重复")
			require.Error(t, err2, "已审批的发布不得再次审批")
			require.Nil(t, resp2)
			require.Equal(t, snapAfterFirst, releaseApprovalSnapshot(t,
				mustRelease(t, client, releaseID),
				mustTask(t, client, taskID)))
		})
	}
}

// 业务更新失败时 BPMN 任务完成必须回滚，防止"流程已批、业务未批"的不一致。
func TestApplyReleaseApproval_AtomicCommit_RollbackOnBusinessFailure(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_rollback")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "rb")
	ctx := context.Background()
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "rb1",
		fmt.Sprintf("release:%d", releaseID), approverID)

	failReleaseUpdates := true
	client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if _, ok := mutation.(*ent.ReleaseMutation); ok && failReleaseUpdates {
				return nil, fmt.Errorf("simulated release update failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})

	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())
	resp, err := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, "approve", "同意")
	require.Error(t, err, "业务更新失败时审批必须失败")
	require.Nil(t, resp)

	rel, gerr := client.Release.Get(ctx, releaseID)
	require.NoError(t, gerr)
	assert.Equal(t, "draft", rel.Status, "发布状态不得变更")

	task, terr := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, terr)
	assert.Equal(t, "assigned", task.Status, "BPMN 任务必须回滚到原始状态")

	failReleaseUpdates = false
	resp2, err2 := svc.ApplyReleaseApproval(ctx, releaseID, tenantID, approverID, "approve", "重试")
	require.NoError(t, err2)
	require.NotNil(t, resp2)
	assert.Equal(t, "scheduled", resp2.Status)

	task2, terr2 := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, terr2)
	assert.Equal(t, "completed", task2.Status)
}

func mustRelease(t *testing.T, client *ent.Client, id int) *ent.Release {
	t.Helper()
	r, err := client.Release.Get(context.Background(), id)
	require.NoError(t, err)
	return r
}

func mustTask(t *testing.T, client *ent.Client, id int) *ent.ProcessTask {
	t.Helper()
	task, err := client.ProcessTask.Get(context.Background(), id)
	require.NoError(t, err)
	return task
}
