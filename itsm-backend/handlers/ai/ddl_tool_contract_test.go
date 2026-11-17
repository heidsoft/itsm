package ai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/handlers/ai"
	"itsm-backend/internal/services"
)

// Phase B4（2026-11-17）DDL/SOP AI 节点工具的三重交集 fail-closed 契约。
//
// 交集的三重由既有机制天然构成，本测试把它们锁死，防止未来有人改工具声明
// 或放宽授权时静默失效：
//   1. 用户角色维度：Gate 2 的 HasResourcePermission(role, tool.Resource, tool.Action)
//   2. 工具声明维度：四个 DDL/SOP 工具必须声明 resource=ai / action=triage，
//      否则第 1 重校验的就不是 ai:triage 码，end_user 可能越过分诊面。
//   3. 运行期策略维度：Gate 2.5 的 principal 零信任（本文件不重复测，见
//      middleware/auth_service_principal_test.go 与 Gate 2.5 既有契约）。
//
// 空交集（无 ai:triage 的角色）必须被拒绝并落 denied 审计，绝不静默放行。

func TestDDLToolDeclarationsUseAITriage(t *testing.T) {
	reg := service.NewToolRegistry(nil, nil, nil, nil)
	for _, name := range []string{"analyze_ddl", "classify_ddl", "generate_sop", "refine_sop"} {
		td := reg.GetTool(name)
		require.NotNil(t, td, "工具 %s 必须已注册", name)
		assert.True(t, td.ReadOnly, "工具 %s 必须是只读（D4：只读免审批）", name)
		assert.Equal(t, "ai", td.Resource, "工具 %s 的 resource 必须是 ai", name)
		assert.Equal(t, "triage", td.Action, "工具 %s 的 action 必须是 triage（否则 Gate 2 校验不到 ai:triage 码）", name)
	}
}

// end_user 不持 ai:triage：调用 DDL 工具必须在 Gate 2 被拒，且审计记为 denied。
// 这是「权限交集为空 → fail-closed」的核心反向断言。
func TestDDLTool_NoAITriageRoleDenied(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	_, invID, err := env.svc.ExecuteTool(context.Background(), 7, 10, "end_user", "analyze_ddl",
		map[string]interface{}{"sqlText": "DROP TABLE users;"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrToolPermissionDenied)
	assert.Zero(t, invID, "拒绝时不应返回 invocation ID")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv, "拒绝路径也必须落审计")
	assert.Equal(t, "denied", inv.PermissionCheck)
	assert.Contains(t, inv.PermissionReason, "ai")
	assert.Contains(t, inv.PermissionReason, "triage")
	assert.Equal(t, "end_user", inv.RoleSnapshot)
}

// technician 持 ai:triage：只读分析工具应放行并获得 executed 审计。
// 这是正向断言，证明交集非空时不会误拒。
func TestDDLTool_WithAITriageRoleExecuted(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	// analyze_ddl 是纯规则工具，不依赖注入的 rag/incident/cmdb，可在 nil 依赖下执行。
	res, invID, err := env.svc.ExecuteTool(context.Background(), 8, 10, "technician", "analyze_ddl",
		map[string]interface{}{"sqlText": "ALTER TABLE t ADD COLUMN c INT;", "ticketId": 42})

	require.NoError(t, err)
	// 只读工具的审计走 recordToolAudit（同步落库），不创建待审批任务，
	// 因此 ExecuteTool 第二返回值（invocation ID）为 0 是既有契约，不是失败。
	assert.Zero(t, invID)
	require.NotNil(t, res)
	analysis, ok := res.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, analysis, "riskLevel")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "passed", inv.PermissionCheck)
	assert.Equal(t, "executed", inv.Status)
	assert.Equal(t, 42, inv.TicketID, "B6：ticketId 应写入 ToolInvocation 强关联列")
}
