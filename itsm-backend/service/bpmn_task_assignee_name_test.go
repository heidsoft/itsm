package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListUserTaskViews_ResolvesAssigneeName 验证「我的待办」视图会把数字 assignee
// 解析成负责人显示名，而不是让审批中心直接渲染裸用户 ID（此前列里全是「1」）。
// 修复前：BPMNTaskResponse 无 AssigneeName，断言失败；修复后：填充为 User.Name。
func TestListUserTaskViews_ResolvesAssigneeName(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Assignee Tenant").SetCode("assignee-t").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("zhangsan").SetEmail("zhangsan@example.com").SetName("张三").
		SetPasswordHash("x").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-AN").SetDeploymentName("an").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("anDemo").SetName("an").SetBpmnXML([]byte("<x/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-AN-1").SetProcessDefinitionKey("anDemo").
		SetProcessDefinitionID(def.ID).SetTenantID(tenant.ID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetStartTime(time.Now()).Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-AN-1").SetProcessInstanceID(inst.ID).
		SetProcessDefinitionKey("anDemo").SetTaskDefinitionKey("UserTask_1").
		SetTaskName("审批").SetTaskType("user_task").SetStatus("created").
		SetAssignee(strconv.Itoa(u.ID)).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	svc := &bpmnTaskService{client: client, logger: logger}
	resp, total, err := svc.ListUserTaskViews(ctx, &ListUserTasksRequest{
		TenantID: tenant.ID,
		AllTasks: true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, resp, 1)
	assert.Equal(t, strconv.Itoa(u.ID), resp[0].Assignee)
	assert.Equal(t, "张三", resp[0].AssigneeName, "负责人应解析为显示名而非裸 ID")
}
