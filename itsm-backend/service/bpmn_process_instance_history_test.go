package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBPMNProcessInstanceService_GetProcessInstanceHistory_ReturnsOrderedHistory(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_history?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}

	instance := createBPMNHistoryInstance(t, ctx, client, 1, "approval-flow", "PI-HISTORY-001")
	t1 := time.Date(2026, 7, 8, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(10 * time.Minute)

	createBPMNExecutionHistory(t, ctx, client, instance.ID, 1, "hist-2", "task_1", "审批任务", "user_task", "complete", t2)
	createBPMNExecutionHistory(t, ctx, client, instance.ID, 1, "hist-1", "start", "开始", "start_event", "start", t1)

	history, err := svc.GetProcessInstanceHistory(ctx, "1",)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, "hist-1", history[0].HistoryID)
	assert.Equal(t, "hist-2", history[1].HistoryID)
}

func TestBPMNProcessInstanceService_GetProcessInstanceHistory_RespectsTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_history_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := &bpmnProcessInstanceService{client: client, logger: logger}

	ctxTenant1 := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 1)
	ctxTenant2 := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 2)

	instanceTenant1 := createBPMNHistoryInstance(t, ctxTenant1, client, 1, "approval-flow", "PI-HISTORY-002")
	_ = createBPMNHistoryInstance(t, ctxTenant2, client, 2, "approval-flow", "PI-HISTORY-003")

	createBPMNExecutionHistory(t, ctxTenant1, client, instanceTenant1.ID, 1, "tenant1-hist", "start", "开始", "start_event", "start", time.Now().UTC())

	history, err := svc.GetProcessInstanceHistory(ctxTenant2, "1")
	require.NoError(t, err)
	assert.Empty(t, history)
}

func createBPMNHistoryInstance(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, definitionKey, instanceKey string) *ent.ProcessInstance {
	t.Helper()

	definition := createBPMNHistoryDefinition(t, ctx, client, tenantID, definitionKey)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID(instanceKey).
		SetBusinessKey("ticket:1").
		SetProcessDefinitionKey(definitionKey).
		SetProcessDefinitionID(definition.ID).
		SetStatus("running").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance
}

func createBPMNHistoryDefinition(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, key string) *ent.ProcessDefinition {
	t.Helper()

	definition, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key).
		SetVersion("v1").
		SetIsActive(true).
		SetTenantID(tenantID).
		SetBpmnXML([]byte("<bpmn></bpmn>")).
		Save(ctx)
	require.NoError(t, err)
	return definition
}

func createBPMNExecutionHistory(t *testing.T, ctx context.Context, client *ent.Client, processInstanceID, tenantID int, historyID, activityID, activityName, activityType, eventType string, ts time.Time) {
	t.Helper()

	_, err := client.ProcessExecutionHistory.Create().
		SetHistoryID(historyID).
		SetProcessInstanceID(processInstanceID).
		SetProcessDefinitionKey("approval-flow").
		SetActivityID(activityID).
		SetActivityName(activityName).
		SetActivityType(activityType).
		SetEventType(eventType).
		SetTenantID(tenantID).
		SetTimestamp(ts).
		Save(ctx)
	require.NoError(t, err)
}
