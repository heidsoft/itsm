package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTimeoutScannerClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(dbName, "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

func setupTimeoutScannerTenant(t *testing.T, client *ent.Client, suffix string) (tenantID, userID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Timeout Tenant " + suffix).
		SetCode("timeout-" + suffix).
		SetDomain("timeout-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("timeout-user-" + suffix).
		SetEmail("timeout-" + suffix + "@example.com").
		SetName("Timeout User " + suffix).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, user.ID
}

func createOverdueTaskFixture(t *testing.T, client *ent.Client, tenantID int, suffix string, dueDate time.Time, taskVars map[string]interface{}, assignee string, status string) int {
	t.Helper()
	ctx := context.Background()

	defKey := "timeout_test_" + suffix
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TIMEOUT-" + suffix).
		SetDeploymentName("Timeout Deployment " + suffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("Timeout Test " + suffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(`<bpmn:definitions/>`)).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-TIMEOUT-" + suffix).
		SetProcessDefinitionKey(defKey).
		SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	builder := client.ProcessTask.Create().
		SetTaskID("TASK-TIMEOUT-" + suffix).
		SetTaskDefinitionKey("TimeoutTask_1").
		SetTaskName("超时测试任务").
		SetTaskType("user_task").
		SetProcessDefinitionKey(defKey).
		SetProcessInstanceID(instance.ID).
		SetStatus(status).
		SetDueDate(dueDate).
		SetTenantID(tenantID)
	if assignee != "" {
		builder = builder.SetAssignee(assignee)
	}
	if taskVars != nil {
		builder = builder.SetTaskVariables(taskVars)
	}
	task, err := builder.Save(ctx)
	require.NoError(t, err)
	return task.ID
}

func TestExtractTimeoutAction(t *testing.T) {
	tests := []struct {
		name     string
		vars     map[string]interface{}
		expected string
	}{
		{"nil vars", nil, ""},
		{"empty vars", map[string]interface{}{}, ""},
		{"no timeoutAction key", map[string]interface{}{"other": "value"}, ""},
		{"non-string value", map[string]interface{}{"timeoutAction": 42}, ""},
		{"notify", map[string]interface{}{"timeoutAction": "notify"}, "notify"},
		{"escalate", map[string]interface{}{"timeoutAction": "escalate"}, "escalate"},
		{"auto_reject", map[string]interface{}{"timeoutAction": "auto_reject"}, "auto_reject"},
		{"auto_approve", map[string]interface{}{"timeoutAction": "auto_approve"}, "auto_approve"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractTimeoutAction(tc.vars))
		})
	}
}

func TestScanOverdueTasks_InvalidTenantID(t *testing.T) {
	client := newTimeoutScannerClient(t, "invalid_tenant")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())

	_, err := scanner.ScanOverdueTasks(context.Background(), 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant ID")

	_, err = scanner.ScanOverdueTasks(context.Background(), -1)
	require.Error(t, err)
}

func TestScanOverdueTasks_NoOverdueTasks(t *testing.T) {
	client := newTimeoutScannerClient(t, "no_overdue")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, _ := setupTimeoutScannerTenant(t, client, "no_overdue")

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Zero(t, processed)
}

func TestScanOverdueTasks_FutureDueDateSkipped(t *testing.T) {
	client := newTimeoutScannerClient(t, "future_due")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "future_due")
	createOverdueTaskFixture(t, client, tenantID, "future",
		time.Now().Add(24*time.Hour),
		map[string]interface{}{"timeoutAction": "notify"},
		fmt.Sprintf("%d", userID),
		common.ProcessTaskStatusAssigned,
	)

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Zero(t, processed)

	task, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusAssigned, task.Status)
}

func TestScanOverdueTasks_CompletedTasksSkipped(t *testing.T) {
	client := newTimeoutScannerClient(t, "completed_skip")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "completed")
	createOverdueTaskFixture(t, client, tenantID, "completed",
		time.Now().Add(-2*time.Hour),
		nil,
		fmt.Sprintf("%d", userID),
		common.ProcessTaskStatusCompleted,
	)

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Zero(t, processed)
}

func TestScanOverdueTasks_NotifyActionSetsTimeoutStatus(t *testing.T) {
	client := newTimeoutScannerClient(t, "notify_action")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "notify")
	createOverdueTaskFixture(t, client, tenantID, "notify",
		time.Now().Add(-1*time.Hour),
		map[string]interface{}{"timeoutAction": "notify"},
		fmt.Sprintf("%d", userID),
		common.ProcessTaskStatusAssigned,
	)

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	task, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, task.Status)
}

func TestScanOverdueTasks_DefaultActionIsNotify(t *testing.T) {
	client := newTimeoutScannerClient(t, "default_notify")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "default")
	createOverdueTaskFixture(t, client, tenantID, "default",
		time.Now().Add(-1*time.Hour),
		nil,
		fmt.Sprintf("%d", userID),
		common.ProcessTaskStatusStarted,
	)

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	task, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, task.Status)
}

func TestScanOverdueTasks_EscalateActionSetsEscalatedStatus(t *testing.T) {
	client := newTimeoutScannerClient(t, "escalate_action")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "escalate")
	createOverdueTaskFixture(t, client, tenantID, "escalate",
		time.Now().Add(-1*time.Hour),
		map[string]interface{}{"timeoutAction": "escalate"},
		fmt.Sprintf("%d", userID),
		common.ProcessTaskStatusAssigned,
	)

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	task, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusEscalated, task.Status)
	assert.Equal(t, "任务超时自动升级", task.TaskVariables["escalation_reason"])
	assert.NotEmpty(t, task.TaskVariables["escalated_time"])
}

func TestScanOverdueTasks_TenantIsolation(t *testing.T) {
	client := newTimeoutScannerClient(t, "tenant_isolation")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())

	tenantA, userA := setupTimeoutScannerTenant(t, client, "A")
	tenantB, userB := setupTimeoutScannerTenant(t, client, "B")

	createOverdueTaskFixture(t, client, tenantA, "tenantA",
		time.Now().Add(-1*time.Hour),
		map[string]interface{}{"timeoutAction": "notify"},
		fmt.Sprintf("%d", userA),
		common.ProcessTaskStatusCreated,
	)
	createOverdueTaskFixture(t, client, tenantB, "tenantB",
		time.Now().Add(-1*time.Hour),
		map[string]interface{}{"timeoutAction": "escalate"},
		fmt.Sprintf("%d", userB),
		common.ProcessTaskStatusAssigned,
	)

	processedA, err := scanner.ScanOverdueTasks(context.Background(), tenantA)
	require.NoError(t, err)
	assert.Equal(t, 1, processedA)

	taskA, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantA)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, taskA.Status)

	taskB, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantB)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusAssigned, taskB.Status, "tenant B task should not be affected by tenant A scan")

	processedB, err := scanner.ScanOverdueTasks(context.Background(), tenantB)
	require.NoError(t, err)
	assert.Equal(t, 1, processedB)

	taskBAfter, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantB)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusEscalated, taskBAfter.Status)
}

func TestScanOverdueTasks_MultipleOverdueTasksProcessed(t *testing.T) {
	client := newTimeoutScannerClient(t, "multiple_tasks")
	scanner := NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
	tenantID, userID := setupTimeoutScannerTenant(t, client, "multi")

	for i, action := range []string{"notify", "escalate"} {
		createOverdueTaskFixture(t, client, tenantID, fmt.Sprintf("multi_%d", i),
			time.Now().Add(-1*time.Hour),
			map[string]interface{}{"timeoutAction": action},
			fmt.Sprintf("%d", userID),
			common.ProcessTaskStatusAssigned,
		)
	}

	processed, err := scanner.ScanOverdueTasks(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, processed)

	tasks, err := client.ProcessTask.Query().
		Where(processtask.TenantID(tenantID)).
		All(context.Background())
	require.NoError(t, err)
	statuses := make(map[string]int)
	for _, task := range tasks {
		statuses[task.Status]++
	}
	assert.Equal(t, 1, statuses[common.ProcessTaskStatusTimeout])
	assert.Equal(t, 1, statuses[common.ProcessTaskStatusEscalated])
}
