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
// 共用一个寻址键：BPMN processInstanceId（PI-* 业务键）。数字 Ent ID 只出现在响应 DTO。
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

func TestBPMNProcessInstanceService_GetProcessInstance_NumericEntIDNotAAddressingKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_get_numeric?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}
	instance := createBPMNHistoryInstance(t, ctx, client, 1, "approval-flow", "PI-GET-003")

	got, err := svc.GetProcessInstance(ctx, strconv.Itoa(instance.ID))
	require.Error(t, err, "数字主键不再是寻址键，必须与 PI 键区分")
	assert.Nil(t, got)
}
