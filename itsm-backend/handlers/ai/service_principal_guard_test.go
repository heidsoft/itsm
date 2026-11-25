package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/handlers/ai"
	"itsm-backend/internal/authz"
)

// 这组测试锁住 ExecuteTool 的**身份守卫**对 service 主体的处理。
//
// 背景（2026-11-18，真实 MCP 客户端联调时暴露）：
// ExecuteTool 入口原有守卫 `if userID <= 0 || tenantID <= 0 { 拒绝 }`。
// 该守卫来自一次 P0 风险修复（54894938f），意图是「无身份不得执行工具」。
//
// 但它与 Phase B1 的 service 主体语义**直接冲突**：service 主体的 JWT 里
// user_id **恒为 0**（进程身份，不是登录用户、不进用户表）。于是：
//   - service 主体被无条件挡在门外，永远无法执行任何工具；
//   - MCP 通道与 sidecar 回连全部不可用；
//   - 错误信息只说「tool permission denied」，看不出是身份判定把人挡了。
//
// 修复口径：守卫按 principal 区分 ——
//   - human/agent：userID 必须 > 0（用户身份须有真实 user_id）
//   - service：userID=0 属正常，准入由 Gate 2（载体角色）与 Gate 2.5
//     （allowedPrincipals）决定
//
// 反向验证：把守卫改回 `userID <= 0` 一律拒绝 → 用例必须变红。

// TestExecuteTool_ServicePrincipalPassesIdentityGuard 验证 service 主体
// 能通过身份守卫（不再被 userID=0 误伤）。
func TestExecuteTool_ServicePrincipalPassesIdentityGuard(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	// service 主体的身份：user_id=0、载体角色 audit_admin（只读）。
	// 上下文里带 principal=service（AuthMiddleware 注入 request context 的那个点）。
	ctx := authz.WithPrincipal(context.Background(), authz.PrincipalService)

	// audit_admin 是只读角色：选一个只读工具，避免写工具进审批流干扰断言。
	_, _, err := env.svc.ExecuteTool(ctx, 0, 10, "audit_admin", "get_incident_stats", map[string]interface{}{})
	// 要点：不得因 userID=0 被**身份守卫**拒绝。
	//
	// 身份守卫的拒绝特征是「无任何附加信息」的裸 ErrToolPermissionDenied；
	// 而 Gate 2/2.5 的拒绝会带上具体原因（如「AI 主体 service 未获授权
	// incident:read」）。所以断言的是「错误里出现了判定原因」——
	// 说明请求已越过身份守卫、进入了权限判定链。
	//
	// 本测试环境未接运行期策略目录（ent client 为空），Gate 2.5 会报
	// "permission catalog unavailable" —— 这正是「已越过身份守卫」的证据。
	if err != nil {
		assert.Contains(t, err.Error(), "未获授权",
			"service 主体不应被身份守卫按 userID=0 拒绝（应进入权限判定链并给出原因），实际错误: %v", err)
	}
}

// TestExecuteTool_HumanPrincipalWithZeroUserIDStillRejected 是反向锁：
// human 主体没有真实 user_id 时仍必须被拒（不许因放开 service 而放宽 human）。
func TestExecuteTool_HumanPrincipalWithZeroUserIDStillRejected(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	ctx := authz.WithPrincipal(context.Background(), authz.PrincipalHuman)

	_, _, err := env.svc.ExecuteTool(ctx, 0, 10, "technician", "get_incident_stats", map[string]interface{}{})

	require.Error(t, err, "human 主体 userID=0 必须被身份守卫拒绝")
	assert.True(t, errors.Is(err, ai.ErrToolPermissionDenied),
		"应返回 ErrToolPermissionDenied，实际: %v", err)
}

// TestExecuteTool_ServicePrincipalWithZeroTenantStillRejected 验证租户维度
// 仍然 fail-closed：service 主体也必须有有效租户。
func TestExecuteTool_ServicePrincipalWithZeroTenantStillRejected(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	ctx := authz.WithPrincipal(context.Background(), authz.PrincipalService)

	_, _, err := env.svc.ExecuteTool(ctx, 0, 0, "audit_admin", "get_incident_stats", map[string]interface{}{})

	require.Error(t, err, "租户无效必须被拒绝（service 主体也不例外）")
	assert.True(t, errors.Is(err, ai.ErrToolPermissionDenied),
		"应返回 ErrToolPermissionDenied，实际: %v", err)
}
