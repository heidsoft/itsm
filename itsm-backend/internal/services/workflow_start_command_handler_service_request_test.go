package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// workflow.start 消费侧的服务请求契约：审批人来自落库的审批记录，
// 流程变量必须携带 need_approval / total_levels / approval_plan，
// 否则模板会在无审批人可指派的节点上卡死或直接跳过审批。

func newServiceRequestWorkflowFixture(t *testing.T) (*ent.Client, context.Context, int, int) {
	t.Helper()
	client, ctx := setupChangeWorkflowTest(t)
	tenant, err := createRouteTestTenant(ctx, client, fmt.Sprintf("sr-%d", nextRouteTestSeq()))
	require.NoError(t, err)
	requester := createRouteUser(t, client, ctx, tenant.ID)
	return client, ctx, tenant.ID, requester
}

func createServiceRequestForWorkflowTest(t *testing.T, client *ent.Client, ctx context.Context, tenantID, requesterID int) *ent.ServiceRequest {
	t.Helper()
	seq := nextRouteTestSeq()
	sr, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(1).
		SetRequesterID(requesterID).
		SetTitle(fmt.Sprintf("交付契约测试服务请求 %d", seq)).
		SetStatus("submitted").
		SetCurrentLevel(1).
		SetTotalLevels(2).
		Save(ctx)
	require.NoError(t, err)
	return sr
}

func createServiceRequestApprovalForWorkflowTest(t *testing.T, client *ent.Client, ctx context.Context, tenantID, serviceRequestID, level, approverID int, threshold int) {
	t.Helper()
	_, err := client.ServiceRequestApproval.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(serviceRequestID).
		SetLevel(level).
		SetStep("manager").
		SetStatus("pending").
		SetNode(map[string]interface{}{
			"approver_ids":     []interface{}{float64(approverID)},
			"quorum_threshold": float64(threshold),
		}).
		Save(ctx)
	require.NoError(t, err)
}

func createServiceRequestWorkflowCommand(t *testing.T, client *ent.Client, ctx context.Context, tenantID, serviceRequestID int) *ent.OperationalCommand {
	t.Helper()
	cmd, err := client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("service_request").
		SetAggregateID(serviceRequestID).
		SetIdempotencyKey(fmt.Sprintf("service_request:%d:workflow:start:test", serviceRequestID)).
		SetPayload(map[string]interface{}{"businessType": "service_request", "businessId": serviceRequestID}).
		SetStatus(commandbus.StatusPending).
		Save(ctx)
	require.NoError(t, err)
	return cmd
}

func TestWorkflowStart_ServiceRequestBuildsApprovalPlan(t *testing.T) {
	client, ctx, tenantID, requesterID := newServiceRequestWorkflowFixture(t)
	sr := createServiceRequestForWorkflowTest(t, client, ctx, tenantID, requesterID)

	approverA := createRouteUser(t, client, ctx, tenantID)
	approverB := createRouteUser(t, client, ctx, tenantID)
	createServiceRequestApprovalForWorkflowTest(t, client, ctx, tenantID, sr.ID, 1, approverA, 2)
	createServiceRequestApprovalForWorkflowTest(t, client, ctx, tenantID, sr.ID, 2, approverB, 1)

	// 同 ID 的跨租户审批行不得进入计划（tenant scope 是最容易漏的一处）。
	foreignTenant, err := createRouteTestTenant(ctx, client, fmt.Sprintf("sr-foreign-%d", nextRouteTestSeq()))
	require.NoError(t, err)
	foreignApprover := createRouteUser(t, client, ctx, foreignTenant.ID)
	createServiceRequestApprovalForWorkflowTest(t, client, ctx, foreignTenant.ID, sr.ID, 3, foreignApprover, 1)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())
	require.NoError(t, handler.Handle(ctx, createServiceRequestWorkflowCommand(t, client, ctx, tenantID, sr.ID)))

	require.Len(t, trigger.reqs, 1)
	req := trigger.reqs[0]
	assert.Equal(t, "service_request_flow", req.ProcessDefinitionKey)
	assert.Equal(t, true, req.Variables["need_approval"])
	assert.Equal(t, 2, req.Variables["total_levels"])

	plan, ok := req.Variables["approval_plan"].([]interface{})
	require.True(t, ok, "approval_plan 必须是可被引擎按层解析的切片")
	require.Len(t, plan, 2)

	first, ok := plan[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, first["level"])
	assert.Equal(t, 2, first["threshold"])
	assert.Equal(t, []interface{}{float64(approverA)}, first["approver_ids"])

	second, ok := plan[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, second["level"])
	assert.Equal(t, []interface{}{float64(approverB)}, second["approver_ids"])
}

func TestWorkflowStart_ServiceRequestWithoutApprovalsSkipsApproval(t *testing.T) {
	client, ctx, tenantID, requesterID := newServiceRequestWorkflowFixture(t)
	sr := createServiceRequestForWorkflowTest(t, client, ctx, tenantID, requesterID)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())
	require.NoError(t, handler.Handle(ctx, createServiceRequestWorkflowCommand(t, client, ctx, tenantID, sr.ID)))

	require.Len(t, trigger.reqs, 1)
	req := trigger.reqs[0]
	assert.Equal(t, false, req.Variables["need_approval"], "没有审批记录时流程必须直接进入履约")
	assert.Equal(t, 0, req.Variables["total_levels"])
	plan, ok := req.Variables["approval_plan"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, plan)
}
