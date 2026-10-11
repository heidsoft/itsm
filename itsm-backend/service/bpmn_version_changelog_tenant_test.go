package service

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
)

// GET /bpmn/process-definitions/changelogs/:id 按全局自增 ID 查询，
// 缺 tenant 谓词时跨租户可枚举他租户版本变更内容。
func TestGetChangeLogsByProcessDefinitionID_RejectsCrossTenant(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant2, err := client.Tenant.Create().
		SetName("CL T2").SetCode("cl-t2").SetDomain("cl-t2.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	// 另建一个 tenant1 的流程定义，模拟攻击者持有自己租户内合法 ID 的场景。
	tenant1, err := client.Tenant.Create().
		SetName("CL T1").SetCode("cl-t1").SetDomain("cl-t1.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	seed := func(tenantID int, key string) int {
		deployment, err := client.ProcessDeployment.Create().
			SetDeploymentID(fmt.Sprintf("DEP-%s", key)).
			SetDeploymentName(key).
			SetIsActive(true).SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		def, err := client.ProcessDefinition.Create().
			SetKey(key).SetName(key).SetVersion("1").
			SetIsLatest(true).SetIsActive(true).
			SetBpmnXML([]byte("<definitions><process id=\"p1\" name=\"proc\"></process></definitions>")).
			SetDeploymentID(deployment.ID).SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return def.ID
	}

	defID1 := seed(tenant1.ID, "cl_flow_t1")
	defID2 := seed(tenant2.ID, "cl_flow_t2")

	_, err = client.ProcessVersionChangelog.Create().
		SetProcessDefinitionID(defID1).
		SetVersion("v1").
		SetChangeLog("本租户的流程变更内容").
		SetChangeType("deploy").
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessVersionChangelog.Create().
		SetProcessDefinitionID(defID2).
		SetVersion("v1").
		SetChangeLog("他租户的流程变更内容").
		SetChangeType("deploy").
		SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := &BPMNVersionService{client: client, logger: logger}

	result, err := svc.GetChangeLogsByProcessDefinitionID(ctx, defID1, tenant1.ID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "本租户的流程变更内容", result[0].ChangeLog)

	// 携带他租户定义 ID 直接 fail-closed 为空，而不是泄漏内容。
	result, err = svc.GetChangeLogsByProcessDefinitionID(ctx, defID2, tenant1.ID)
	require.NoError(t, err)
	require.Empty(t, result)

	result, err = svc.GetChangeLogsByProcessDefinitionID(ctx, defID2, tenant2.ID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "他租户的流程变更内容", result[0].ChangeLog)

	// 缺租户上下文必须 fail closed，而不是退化成全平台查询。
	result, err = svc.GetChangeLogsByProcessDefinitionID(ctx, defID2, 0)
	require.Error(t, err)
	require.Empty(t, result)
}
