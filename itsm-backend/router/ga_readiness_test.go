package router

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGAReadinessReportsStaleWorkflowBacklog(t *testing.T) {
	_, client := setupTestEngine(t)
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Workflow Health Tenant").
		SetCode("workflow-health").
		SetDomain("workflow-health.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	assignee, err := client.User.Create().
		SetUsername("workflow-health-admin").
		SetEmail("workflow-health@example.com").
		SetName("Workflow Health Admin").
		SetPasswordHash("not-used-in-test").
		SetRole("admin").
		SetTenantID(tenant.ID).
		SetActive(true).
		Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("deployment-health").
		SetDeploymentName("Workflow Health").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	definition, err := client.ProcessDefinition.Create().
		SetKey("workflow_health").
		SetName("Workflow Health").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("instance-health").
		SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessTask.Create().
		SetTaskID("task-health").
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(definition.Key).
		SetTaskDefinitionKey("approval").
		SetTaskName("审批").
		SetAssignee(fmt.Sprintf("%d", assignee.ID)).
		SetStatus("created").
		SetCreatedTime(time.Now().Add(-25 * time.Hour)).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	readiness := buildGAReadiness(ctx, client)
	assert.Equal(t, "ready", readiness.Status)
	assert.Equal(t, 1, readiness.Summary["stale_workflow_tasks"])
	assert.Equal(t, 1, readiness.Summary["overdue_workflow_tasks"])
	assert.Equal(t, 0, readiness.Summary["orphaned_workflow_tasks"])
	require.NotEmpty(t, readiness.Checks)
	assert.Equal(t, "warning", readiness.Checks[len(readiness.Checks)-1].Status)
}
