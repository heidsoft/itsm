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

// TestListUserTaskViews_ResolvesProcessDefinitionName 验证「我的待办」视图把
// processDefinitionKey 解析成定义显示名（D10）：审批中心「流程」列不得直接暴露
// ticket_general_flow 这类工程标识；定义缺失时名称为空，由前端回退展示 Key。
func TestListUserTaskViews_ResolvesProcessDefinitionName(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("DefName Tenant").SetCode("defname-t").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("lisi").SetEmail("lisi@example.com").SetName("李四").
		SetPasswordHash("x").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DN").SetDeploymentName("dn").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 同名 key 两个版本：旧版本名称不得覆盖 is_latest 版本
	_, err = client.ProcessDefinition.Create().
		SetKey("ticket_general_flow").SetName("工单通用流程 v1").SetVersion("1").SetBpmnXML([]byte("<x/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenant.ID).SetIsActive(false).SetIsLatest(false).Save(ctx)
	require.NoError(t, err)
	latestDef, err := client.ProcessDefinition.Create().
		SetKey("ticket_general_flow").SetName("工单通用流程").SetVersion("2").SetBpmnXML([]byte("<x/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	mkTask := func(taskID, defKey string, defID int) {
		inst, err := client.ProcessInstance.Create().
			SetProcessInstanceID("PI-DN-" + taskID).SetProcessDefinitionKey(defKey).
			SetProcessDefinitionID(defID).SetTenantID(tenant.ID).SetStatus("running").
			SetVariables(map[string]interface{}{}).SetStartTime(time.Now()).Save(ctx)
		require.NoError(t, err)
		_, err = client.ProcessTask.Create().
			SetTaskID(taskID).SetProcessInstanceID(inst.ID).
			SetProcessDefinitionKey(defKey).SetTaskDefinitionKey("UserTask_1").
			SetTaskName("审批").SetTaskType("user_task").SetStatus("created").
			SetAssignee(strconv.Itoa(u.ID)).SetTenantID(tenant.ID).Save(ctx)
		require.NoError(t, err)
	}
	mkTask("TASK-DN-1", "ticket_general_flow", latestDef.ID)
	// 无对应定义的任务：名称应为空，前端回退展示 Key
	mkTask("TASK-DN-2", "ghost_flow", latestDef.ID)

	svc := &bpmnTaskService{client: client, logger: logger}
	resp, total, err := svc.ListUserTaskViews(ctx, &ListUserTasksRequest{
		TenantID: tenant.ID,
		AllTasks: true,
	})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, resp, 2)

	nameByTaskID := make(map[string]string, len(resp))
	for _, r := range resp {
		nameByTaskID[r.TaskID] = r.ProcessDefinitionName
	}
	assert.Equal(t, "工单通用流程", nameByTaskID["TASK-DN-1"], "应取 is_latest 版本的定义名")
	assert.Empty(t, nameByTaskID["TASK-DN-2"], "定义缺失时名称应为空，前端回退展示 Key")
}

// TestListUserTaskViews_DefinitionNameTenantScoped 跨租户同名 key 不得串名：
// 定义查询必须带租户谓词。
func TestListUserTaskViews_DefinitionNameTenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("def-tenant-a").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").SetCode("def-tenant-b").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("wangwu").SetEmail("wangwu@example.com").SetName("王五").
		SetPasswordHash("x").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)

	depB, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DNB").SetDeploymentName("dnb").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)
	defB, err := client.ProcessDefinition.Create().
		SetKey("shared_flow").SetName("B 租户专属流程名").SetBpmnXML([]byte("<x/>")).
		SetDeploymentID(depB.ID).SetTenantID(tenantB.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	// 租户 A 的任务引用同名 key，但 A 下没有该定义
	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DN-A1").SetProcessDefinitionKey("shared_flow").
		SetProcessDefinitionID(defB.ID).SetTenantID(tenantA.ID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetStartTime(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-DN-A1").SetProcessInstanceID(inst.ID).
		SetProcessDefinitionKey("shared_flow").SetTaskDefinitionKey("UserTask_1").
		SetTaskName("审批").SetTaskType("user_task").SetStatus("created").
		SetAssignee(strconv.Itoa(u.ID)).SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)

	svc := &bpmnTaskService{client: client, logger: logger}
	resp, total, err := svc.ListUserTaskViews(ctx, &ListUserTasksRequest{
		TenantID: tenantA.ID,
		AllTasks: true,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, resp, 1)
	assert.Empty(t, resp[0].ProcessDefinitionName, "不得读取其他租户的同名流程定义名")
}
