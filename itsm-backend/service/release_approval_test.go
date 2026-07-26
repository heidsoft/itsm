package service

import (
	"context"
	"fmt"
	"testing"

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
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "approve", "同意发布")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "scheduled", resp.Status)
}

func TestApplyReleaseApproval_RejectMovesDraftToCancelled(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_reject_ok")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "rok")
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	resp, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID, "reject", "风险过高")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "cancelled", resp.Status)
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

func TestApplyReleaseApproval_UnknownApproverRejected(t *testing.T) {
	client := newReleaseApprovalTestClient(t, "rel_unknown_actor")
	tenantID, _, approverID, releaseID := setupReleaseApprovalFixture(t, client, "ua")
	svc := NewReleaseService(client, zaptest.NewLogger(t).Sugar())

	_, err := svc.ApplyReleaseApproval(context.Background(), releaseID, tenantID, approverID+9999, "approve", "")
	require.Error(t, err, "审批人必须是本租户有效用户")
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
