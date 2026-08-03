package ai_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/handlers/ai"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

// rbacMockRepo 记录 CreateToolInvocation 调用，其余方法返回零值。
// P2-6 测试只关心 Gate 2 的权限校验结果与审计写入，不需要真实持久化。
type rbacMockRepo struct {
	mu              sync.Mutex
	toolInvocations []*ai.ToolInvocation
}

func (m *rbacMockRepo) CreateToolInvocation(_ context.Context, i *ai.ToolInvocation) (*ai.ToolInvocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 分配递增 ID，模拟数据库自增主键，使 Service 能返回 > 0 的 invocation ID
	i.ID = len(m.toolInvocations) + 1
	m.toolInvocations = append(m.toolInvocations, i)
	return i, nil
}

func (m *rbacMockRepo) lastInvocation() *ai.ToolInvocation {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.toolInvocations) == 0 {
		return nil
	}
	return m.toolInvocations[len(m.toolInvocations)-1]
}

func (m *rbacMockRepo) invocations() []ai.ToolInvocation {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ai.ToolInvocation, 0, len(m.toolInvocations))
	for _, inv := range m.toolInvocations {
		out = append(out, *inv)
	}
	return out
}

// 其余 Repository 方法：返回零值，测试不依赖
func (m *rbacMockRepo) CreateConversation(_ context.Context, c *ai.Conversation) (*ai.Conversation, error) {
	return c, nil
}

func (m *rbacMockRepo) GetConversation(_ context.Context, _ int, _ int) (*ai.Conversation, error) {
	return nil, nil
}

func (m *rbacMockRepo) ListConversations(_ context.Context, _ int, _ int) ([]*ai.Conversation, error) {
	return nil, nil
}

func (m *rbacMockRepo) CreateMessage(_ context.Context, msg *ai.Message) (*ai.Message, error) {
	return msg, nil
}

func (m *rbacMockRepo) GetMessages(_ context.Context, _ int) ([]*ai.Message, error) {
	return nil, nil
}

func (m *rbacMockRepo) GetToolInvocation(_ context.Context, _ int, _ int) (*ai.ToolInvocation, error) {
	return nil, nil
}

func (m *rbacMockRepo) UpdateToolInvocation(_ context.Context, i *ai.ToolInvocation) (*ai.ToolInvocation, error) {
	return i, nil
}

func (m *rbacMockRepo) CreateRCA(_ context.Context, r *ai.RootCauseAnalysis) (*ai.RootCauseAnalysis, error) {
	return r, nil
}

func (m *rbacMockRepo) GetRCAByTicket(_ context.Context, _ int, _ int) (*ai.RootCauseAnalysis, error) {
	return nil, nil
}

func (m *rbacMockRepo) UpdateRCA(_ context.Context, r *ai.RootCauseAnalysis) (*ai.RootCauseAnalysis, error) {
	return r, nil
}

// rbacTestEnv 统一构造 Service + mock repo，开启硬编码权限模式以便无 DB 测试。
//
// 设计要点：
//   - 使用 create_ticket（写工具，需审批）作为测试目标，避免触发真实 ToolRegistry.Execute
//     （后者依赖 incident/rag/cmdb 等运行时服务）。写工具走审批流，仅调用 mock repo。
//   - PermissionConfig.Mode 切换为 HardcodeOnly，使 hasResourcePermission 不触碰数据库，
//     直接读取 middleware.RolePermissions 硬编码表，便于断言放行/拒绝。
//   - s.entClient 设为非 nil 空壳（HardcodeOnly 不使用它），满足 Gate 2 进入条件。
type rbacTestEnv struct {
	svc      *ai.Service
	repo     *rbacMockRepo
	prevMode middleware.PermissionConfigMode
}

func newRBACTestEnv(t *testing.T) *rbacTestEnv {
	t.Helper()
	repo := &rbacMockRepo{}
	// NewToolRegistry 全部依赖 nil：测试只用写工具（create_ticket），不会调用 Execute
	tools := service.NewToolRegistry(nil, nil, nil, nil)
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, tools, nil, nil, nil, nil, nil, nil, nil)
	// 非空 ent client（HardcodeOnly 不使用，仅满足 Gate 2 的 != nil 守卫）
	svc.SetEntClient(&ent.Client{})

	env := &rbacTestEnv{svc: svc, repo: repo, prevMode: middleware.PermissionConfig.Mode}
	// 切换到硬编码模式：无 DB 也可判定角色权限
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	// 清空权限缓存，避免其他测试残留影响
	middleware.InvalidateAllPermissionCaches()
	// 重置 RBAC Feature Flag 的 sync.Once，允许本测试重新读取环境变量
	ai.ResetRBACFlagForTest()

	t.Cleanup(func() {
		middleware.PermissionConfig.Mode = env.prevMode
		middleware.InvalidateAllPermissionCaches()
		ai.ResetRBACFlagForTest()
	})
	return env
}

// setFlag 设置 AI_TOOL_RBAC 开关并重新加载
func setFlag(t *testing.T, enabled, enforce bool) {
	t.Helper()
	ai.ResetRBACFlagForTest()
	if err := os.Setenv("AI_TOOL_RBAC_ENABLED", boolStr(enabled)); err != nil {
		t.Fatalf("set AI_TOOL_RBAC_ENABLED: %v", err)
	}
	if err := os.Setenv("AI_TOOL_RBAC_ENFORCE", boolStr(enforce)); err != nil {
		t.Fatalf("set AI_TOOL_RBAC_ENFORCE: %v", err)
	}
	// 触发重新加载
	_ = ai.IsToolRBACEnabled()
	// 还原环境变量，避免污染后续测试
	t.Cleanup(func() {
		_ = os.Unsetenv("AI_TOOL_RBAC_ENABLED")
		_ = os.Unsetenv("AI_TOOL_RBAC_ENFORCE")
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ===== T1: super_admin 始终放行（即使 RBAC enforce 开启） =====
// 期望：super_admin 跳过 Gate 2，permCheck=skipped，进入 create_ticket 审批流。
func TestExecuteTool_T1_SuperAdminBypass(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true) // enforce 模式，应最严格

	args := map[string]interface{}{"title": "x"}
	_, invID, err := env.svc.ExecuteTool(context.Background(), 1, 10, "super_admin", "create_ticket", args)

	require.NoError(t, err, "super_admin 不应被 RBAC 拦截")
	assert.Greater(t, invID, 0, "写工具应返回待审批 invocation ID")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "skipped", inv.PermissionCheck, "super_admin 跳过工具级 RBAC")
	assert.Equal(t, "super_admin", inv.RoleSnapshot)
	assert.Equal(t, "pending", inv.ApprovalState)
	assert.Equal(t, 10, inv.TenantID, "租户上下文应透传到审计记录")
}

// ===== T2: end_user 对 create_ticket（ticket:write）有权限 -> 放行 + passed =====
// 硬编码表 end_user 拥有 ticket:write，应通过 Gate 2。
func TestExecuteTool_T2_PermissionPassed(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	args := map[string]interface{}{"title": "x"}
	_, invID, err := env.svc.ExecuteTool(context.Background(), 5, 10, "end_user", "create_ticket", args)

	require.NoError(t, err, "end_user 拥有 ticket:write，应放行")
	assert.Greater(t, invID, 0)

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "passed", inv.PermissionCheck)
	assert.Equal(t, "end_user", inv.RoleSnapshot)
	assert.Equal(t, "pending", inv.ApprovalState)
}

// ===== T3: security 对 create_ticket（ticket:write）无权限 -> enforce 拒绝 =====
// 硬编码表 security 只有 ticket:read，没有 ticket:write。
func TestExecuteTool_T3_PermissionDeniedEnforce(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true) // enforce：拒绝并返回错误

	args := map[string]interface{}{"title": "x"}
	_, _, err := env.svc.ExecuteTool(context.Background(), 7, 10, "security", "create_ticket", args)

	require.Error(t, err, "security 缺少 ticket:write，应被拒绝")
	assert.True(t, errors.Is(err, ai.ErrToolPermissionDenied), "应返回 ErrToolPermissionDenied")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv, "拒绝路径也应写入审计记录")
	assert.Equal(t, "denied", inv.PermissionCheck)
	assert.Contains(t, inv.PermissionReason, "ticket")
	assert.Contains(t, inv.PermissionReason, "write")
	assert.Equal(t, "security", inv.RoleSnapshot)
	assert.NotEqual(t, "pending", inv.ApprovalState, "enforce 拒绝时不应创建待审批任务")
}

// ===== T4: 影子模式（enabled 但非 enforce）拒绝时只记录不拦截 =====
// security 无权限，但未开 enforce：应继续进入审批流，仅 permCheck=denied。
func TestExecuteTool_T4_ShadowModeRecordsDeniedButProceeds(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, false) // 影子模式

	args := map[string]interface{}{"title": "x"}
	_, invID, err := env.svc.ExecuteTool(context.Background(), 7, 10, "security", "create_ticket", args)

	require.NoError(t, err, "影子模式不拦截请求")
	assert.Greater(t, invID, 0, "影子模式仍应进入审批流")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "denied", inv.PermissionCheck, "影子模式仍记录权限校验结果")
	assert.Contains(t, inv.PermissionReason, "ticket:write")
	assert.Equal(t, "pending", inv.ApprovalState, "影子模式应继续走审批流")
}

// ===== T5: 未知工具 -> 返回 ErrUnknownTool + denied 审计 =====
func TestExecuteTool_T5_UnknownTool(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	args := map[string]interface{}{}
	_, _, err := env.svc.ExecuteTool(context.Background(), 5, 10, "end_user", "no_such_tool", args)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ai.ErrUnknownTool), "应返回 ErrUnknownTool")

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv, "未知工具应记录 denied 审计")
	assert.Equal(t, "denied", inv.PermissionCheck)
	assert.Equal(t, "unknown tool", inv.PermissionReason)
	assert.Equal(t, "no_such_tool", inv.ToolName)
}

// ===== T6: 租户上下文透传到审计记录（继承 P0 fail-closed 的 tenant 隔离） =====
// 不同 tenantID 调用应各自记录对应租户，避免跨租户审计混淆。
func TestExecuteTool_T6_TenantContextPropagatedToAudit(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)

	// 同一 super_admin 在两个不同租户下调用
	_, _, err := env.svc.ExecuteTool(context.Background(), 1, 100, "super_admin", "create_ticket", map[string]interface{}{})
	require.NoError(t, err)
	_, _, err = env.svc.ExecuteTool(context.Background(), 1, 200, "super_admin", "create_ticket", map[string]interface{}{})
	require.NoError(t, err)

	invos := env.repo.invocations()
	require.Len(t, invos, 2)
	assert.Equal(t, 100, invos[0].TenantID)
	assert.Equal(t, 200, invos[1].TenantID, "审计记录必须按调用时的 tenantID 分离")
}

// ===== T7: Feature Flag 关闭 -> 跳过 Gate 2，permCheck=skipped =====
// 默认 off：兼容历史行为，不校验、不拦截。
func TestExecuteTool_T7_FlagDisabledSkipsCheck(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, false, false)

	// security 对 ticket:write 无权限，但 flag 关闭应完全跳过
	_, invID, err := env.svc.ExecuteTool(context.Background(), 7, 10, "security", "create_ticket", map[string]interface{}{})

	require.NoError(t, err, "flag 关闭时不做 RBAC 校验")
	assert.Greater(t, invID, 0)

	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "skipped", inv.PermissionCheck, "flag 关闭时权限校验应被跳过")
	assert.Empty(t, inv.PermissionReason)
	assert.Equal(t, "security", inv.RoleSnapshot, "即便跳过校验仍记录角色快照")
}

// ===== 附加：entClient 为 nil 时跳过 Gate 2（防御性降级） =====
// 若 entClient 未注入（旧部署/单元测试），不应 panic，应降级为 skipped。
func TestExecuteTool_NilEntClientSkipsGate(t *testing.T) {
	env := newRBACTestEnv(t)
	setFlag(t, true, true)
	// 显式清空 ent client，模拟未注入场景
	env.svc.SetEntClient(nil)

	_, invID, err := env.svc.ExecuteTool(context.Background(), 7, 10, "security", "create_ticket", map[string]interface{}{})

	require.NoError(t, err, "entClient 为 nil 时应降级放行，不 panic")
	assert.Greater(t, invID, 0)
	inv := env.repo.lastInvocation()
	require.NotNil(t, inv)
	assert.Equal(t, "skipped", inv.PermissionCheck)
}
