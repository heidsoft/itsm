package service

import (
	"context"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GET /api/v1/bpmn/process-instances/:id 与同族 suspend/resume/terminate/approval-history
// 共用一个寻址键：BPMN processInstanceId（PI-* 业务键）。列表响应的 id 是数字
// Ent ID，但调用方（CLI/MCP/agent）天然会拿列表 id 再调详情 —— 因此数字形态
// 在租户内按主键兜底解析，避免「列表给的 id 必 404」的陷阱。
// 本组测试锁死该契约：E2E ticket-type-full-chain 曾因按 PI 键读取被 strconv 拒绝而 404。

func TestBPMNProcessInstanceService_GetProcessInstance_ByProcessInstanceKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_get_by_key?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}
	instance := createBPMNHistoryInstance(t, ctx, client, 1, "approval-flow", "PI-GET-001")

	got, err := svc.GetProcessInstance(ctx, "PI-GET-001")
	require.NoError(t, err)
	assert.Equal(t, instance.ID, got.ID)
	assert.Equal(t, "PI-GET-001", got.ProcessInstanceID)
}

func TestBPMNProcessInstanceService_GetProcessInstance_CrossTenantRejected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_get_cross_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctxA := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	ctxB := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 2)
	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}
	createBPMNHistoryInstance(t, ctxA, client, 1, "approval-flow", "PI-GET-002")

	got, err := svc.GetProcessInstance(ctxB, "PI-GET-002")
	require.Error(t, err, "tenant B must not resolve tenant A's instance by PI key")
	assert.Nil(t, got)
}

func TestBPMNProcessInstanceService_GetProcessInstance_ResolvesListIDWithinTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_get_numeric?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}
	instance := createBPMNHistoryInstance(t, ctx, client, 1, "approval-flow", "PI-GET-003")

	got, err := svc.GetProcessInstance(ctx, strconv.Itoa(instance.ID))
	require.NoError(t, err, "列表响应的数字 id 必须能直接用于详情读取")
	require.NotNil(t, got)
	assert.Equal(t, "PI-GET-003", got.ProcessInstanceID)

	// 数字兜底仍必须遵守租户隔离。
	ctxOther := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 2)
	got, err = svc.GetProcessInstance(ctxOther, strconv.Itoa(instance.ID))
	require.Error(t, err, "tenant B must not resolve tenant A's instance by numeric id")
	assert.Nil(t, got)

	// 不存在的数字 id 仍是 not found，而不是空成功。
	got, err = svc.GetProcessInstance(ctx, "999999")
	require.Error(t, err)
	assert.Nil(t, got)
}
