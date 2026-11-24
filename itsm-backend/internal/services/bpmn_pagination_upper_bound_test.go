package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// P2-06 守卫：三个 service 列表入口必须在进 SQL 前归一化分页。
//
// 旧实现 `if req.Page > 0 && req.PageSize > 0` 有两个缺口：
//   - 0/0 跳过整个 LIMIT 分支 = 整租户全量拉取；
//   - pageSize=5000 原样下传，无上界。
//
// 每入口 seed 105 行是刻意的：common.MaxPageSize=100，只有超过 100 行才能
// 区分「夹紧到 100」与「原样下传 5000」——25 行时两者返回值相同，测了等于没测。
// ---------------------------------------------------------------------------

const paginationSeedRows = 105

// seedPaginationBase 建租户 + deployment，返回 ctx（含租户键）与基础外键。
func seedPaginationBase(t *testing.T, dsn string) (*ent.Client, context.Context, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx, tenantID := bizTenantCtx(t, client, "pag")

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-PAG").SetDeploymentName("Pag").SetIsActive(true).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return client, ctx, tenantID, deployment.ID
}

func TestListUserTasksPaginationUpperBound(t *testing.T) {
	client, ctx, tenantID, deploymentID := seedPaginationBase(t, "file:bpmn_pag_tasks?mode=memory&cache=shared&_fk=1")

	def, err := client.ProcessDefinition.Create().
		SetKey("pag_tasks").SetName("Pag").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deploymentID).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-PAG-TASKS").SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < paginationSeedRows; i++ {
		_, err := client.ProcessTask.Create().
			SetTaskID(fmt.Sprintf("T-PAG-%03d", i)).
			SetTaskDefinitionKey("T").SetTaskName(fmt.Sprintf("task %d", i)).
			SetProcessDefinitionKey(def.Key).SetProcessInstanceID(inst.ID).
			SetStatus("created").SetTenantID(tenantID).
			SetTaskVariables(map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}

	svc := &bpmnTaskService{client: client, logger: zaptest.NewLogger(t).Sugar()}

	t.Run("0/0 归一化为 1/20 而不是全量", func(t *testing.T) {
		req := &dto.ListUserTasksRequest{TenantID: tenantID, AllTasks: true}
		tasks, total, err := svc.ListUserTasks(ctx, req)
		require.NoError(t, err)
		assert.Len(t, tasks, 20, "0/0 必须走默认 LIMIT，不得整租户拉取")
		assert.Equal(t, paginationSeedRows, total)
	})

	t.Run("页长 5000 夹紧到 MaxPageSize", func(t *testing.T) {
		req := &dto.ListUserTasksRequest{TenantID: tenantID, AllTasks: true, Page: 1, PageSize: 5000}
		tasks, _, err := svc.ListUserTasks(ctx, req)
		require.NoError(t, err)
		assert.Len(t, tasks, common.MaxPageSize, "超上限页长必须夹紧，不得原样下传")
	})

	t.Run("第二页在夹紧后语义一致", func(t *testing.T) {
		req := &dto.ListUserTasksRequest{TenantID: tenantID, AllTasks: true, Page: 2, PageSize: 100}
		tasks, _, err := svc.ListUserTasks(ctx, req)
		require.NoError(t, err)
		assert.Len(t, tasks, paginationSeedRows-common.MaxPageSize)
	})
}

func TestListProcessInstancesPaginationUpperBound(t *testing.T) {
	client, ctx, tenantID, deploymentID := seedPaginationBase(t, "file:bpmn_pag_instances?mode=memory&cache=shared&_fk=1")

	def, err := client.ProcessDefinition.Create().
		SetKey("pag_inst").SetName("Pag").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deploymentID).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < paginationSeedRows; i++ {
		_, err := client.ProcessInstance.Create().
			SetProcessInstanceID(fmt.Sprintf("PI-PAG-%03d", i)).
			SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
			SetStatus("running").SetVariables(map[string]interface{}{}).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	svc := &bpmnProcessInstanceService{client: client, logger: zaptest.NewLogger(t).Sugar()}

	t.Run("0/0 归一化为 1/20 而不是全量", func(t *testing.T) {
		req := &dto.ListProcessInstancesRequest{TenantID: tenantID}
		instances, total, err := svc.ListProcessInstances(ctx, req)
		require.NoError(t, err)
		assert.Len(t, instances, 20)
		assert.Equal(t, paginationSeedRows, total)
	})

	t.Run("页长 5000 夹紧到 MaxPageSize", func(t *testing.T) {
		req := &dto.ListProcessInstancesRequest{TenantID: tenantID, Page: 1, PageSize: 5000}
		instances, _, err := svc.ListProcessInstances(ctx, req)
		require.NoError(t, err)
		assert.Len(t, instances, common.MaxPageSize)
	})
}

func TestListProcessDefinitionsPaginationUpperBound(t *testing.T) {
	client, ctx, tenantID, deploymentID := seedPaginationBase(t, "file:bpmn_pag_defs?mode=memory&cache=shared&_fk=1")

	for i := 0; i < paginationSeedRows; i++ {
		_, err := client.ProcessDefinition.Create().
			SetKey(fmt.Sprintf("pag_def_%03d", i)).SetName(fmt.Sprintf("def %d", i)).
			SetVersion("1").SetIsLatest(true).SetIsActive(true).
			SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deploymentID).SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	svc := &bpmnProcessDefinitionService{client: client, logger: zaptest.NewLogger(t).Sugar()}

	t.Run("0/0 归一化为 1/20 而不是全量", func(t *testing.T) {
		req := &dto.ListProcessDefinitionsRequest{TenantID: tenantID}
		defs, total, err := svc.ListProcessDefinitions(ctx, req)
		require.NoError(t, err)
		assert.Len(t, defs, 20)
		assert.Equal(t, paginationSeedRows, total)
	})

	t.Run("页长 5000 夹紧到 MaxPageSize", func(t *testing.T) {
		req := &dto.ListProcessDefinitionsRequest{TenantID: tenantID, Page: 1, PageSize: 5000}
		defs, _, err := svc.ListProcessDefinitions(ctx, req)
		require.NoError(t, err)
		assert.Len(t, defs, common.MaxPageSize)
	})
}
