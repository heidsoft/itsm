package service

import (
	"context"
	"sort"
	"strings"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processbinding"
	"itsm-backend/service/scenario"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 部门默认流程绑定的 key 只允许有一个来源（service/scenario 清单）。
// 本文件锁两件事：清单里的 key 谁真的有 BPMN 载体，以及真实入口
// POST /api/v1/departments/:id/init-processes -> InitDepartmentDefaultBindings
// 在无载体/未部署时不再返回假成功。

// readyCatalogKeysBaseline 是 2026-10-02 实测的「部门清单中有 service/bpmn 载体的 key」。
// 双向棘轮：模板被删（ready 变少）或补了同名 .bpmn（ready 变多）都必须显式改本基线，
// 避免清单与部署能力悄悄漂移。
var readyCatalogKeysBaseline = []string{
	"change_normal_flow",
	"incident_emergency_flow",
	"release_approval_flow",
}

// unreadyCatalogKeysBaseline 是欠债清单：这些场景声明了流程 key，但产品还没有模板载体。
// 补齐模板后本条目必须删除，否则测试失败（防止基线变成永久挡箭牌）。
var unreadyCatalogKeysBaseline = []string{
	"budget_approval_flow",
	"change_emergency_flow",
	"change_requirement_flow",
	"expense_approval_flow",
	"leave_approval_flow",
	"procurement_flow",
	"recruitment_approval_flow",
	"release_test_flow",
}

func catalogProcessKeys(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, templates := range scenario.GetAllTemplates() {
		for _, tmpl := range templates {
			seen[tmpl.ProcessKey] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestDepartmentProcessCatalog_ReadinessBaseline(t *testing.T) {
	deployable, err := BuiltinProcessTemplateKeys()
	require.NoError(t, err)

	var ready, unready []string
	for _, key := range catalogProcessKeys(t) {
		if deployable[key] {
			ready = append(ready, key)
			continue
		}
		unready = append(unready, key)
	}

	assert.Equal(t, readyCatalogKeysBaseline, ready, "部门清单中有载体的 key 集合漂移")
	assert.Equal(t, unreadyCatalogKeysBaseline, unready, "部门清单中缺 BPMN 载体的 key 集合漂移")

	// 载体清单本身也必须能枚举，且与嵌入文件一致（不是手抄列表）。
	for _, key := range []string{"service_request_flow", "problem_management_flow", "ticket_general_flow"} {
		assert.True(t, deployable[key], "内置模板 %s 应可从 embed FS 枚举", key)
	}
}

func TestInitDepartmentDefaultBindings_BindsOnlyDeployableKeys(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenantID := 61
	deployment := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DEPT-BINDINGS").SetDeploymentName("dept-bindings").SetTenantID(tenantID).SaveX(ctx)
	for _, key := range readyCatalogKeysBaseline {
		client.ProcessDefinition.Create().
			SetKey(key).SetName(key).SetBpmnXML([]byte("<bpmn/>")).SetVersion("1").
			SetDeploymentID(deployment.ID).SetTenantID(tenantID).
			SetIsActive(true).SetIsLatest(true).SaveX(ctx)
	}

	svc := NewProcessBindingService(client)
	require.NoError(t, svc.InitDepartmentDefaultBindings(ctx, tenantID, 500, "operations"))

	bindings, err := client.ProcessBinding.Query().
		Where(processbinding.TenantID(tenantID), processbinding.DepartmentID(500)).
		All(ctx)
	require.NoError(t, err)

	keys := make([]string, 0, len(bindings))
	for _, b := range bindings {
		keys = append(keys, b.ProcessDefinitionKey)
	}
	sort.Strings(keys)
	// 4 条 operations 清单里 change_emergency_flow 无载体，只能落 3 条。
	assert.Equal(t, []string{"change_normal_flow", "incident_emergency_flow", "incident_emergency_flow"}, keys)

	// 幂等：重复初始化不产生重复绑定。
	require.NoError(t, svc.InitDepartmentDefaultBindings(ctx, tenantID, 500, "operations"))
	count, err := client.ProcessBinding.Query().
		Where(processbinding.TenantID(tenantID), processbinding.DepartmentID(500)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestInitDepartmentDefaultBindings_WithoutTemplateCarrierIsExplicitFailure(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenantID := 62

	svc := NewProcessBindingService(client)
	err := svc.InitDepartmentDefaultBindings(ctx, tenantID, 700, "finance")
	require.Error(t, err)

	message := err.Error()
	assert.Contains(t, message, "无 BPMN 模板载体")
	assert.Contains(t, message, "expense_approval_flow")
	assert.Contains(t, message, "budget_approval_flow")
	assert.Contains(t, message, "procurement_flow")

	count, qerr := client.ProcessBinding.Query().
		Where(processbinding.TenantID(tenantID)).
		Count(ctx)
	require.NoError(t, qerr)
	assert.Equal(t, 0, count, "无载体场景不得留下半截绑定")
}

func TestInitDepartmentDefaultBindings_CarrierButUndeployedIsExplicitFailure(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenantID := 63

	svc := NewProcessBindingService(client)
	err := svc.InitDepartmentDefaultBindings(ctx, tenantID, 800, "operations")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未在租户 63 部署")
	assert.True(t, strings.Contains(err.Error(), "incident_emergency_flow"), err.Error())
}

func TestInitDepartmentDefaultBindings_UnknownDepartmentType(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	svc := NewProcessBindingService(client)
	err := svc.InitDepartmentDefaultBindings(context.Background(), 64, 900, "legal")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知部门类型: legal")
}
