package ai_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/handlers/ai"
	"itsm-backend/metrics"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

// persistFailMockRepo 继承 rbacMockRepo 的字段，扩展 CreateToolInvocation 错误注入能力。
//
// 设计目的：L8 修复回归测试。
// 背景：handlers/ai/service.go 中 5 处 _, _ = 吞错模式导致 DB 抖动时 AI 对话/审计静默失败。
// 修复后：失败路径必须同时具备 (a) 不阻塞主业务、(b) 上报结构化错误日志、
// (c) 上报 itsm_ai_persist_errors_total 计数器。
// 本 mock 通过让 CreateToolInvocation 返回错误，触发 recordToolAudit 的失败分支，
// 验证 (a) 不抛错（best-effort 保留）和 (c) 指标累加。
type persistFailMockRepo struct {
	*rbacMockRepo
	injectCreateToolInvocationErr error
	injectCreateMessageErr        error
}

func (m *persistFailMockRepo) CreateToolInvocation(_ context.Context, i *ai.ToolInvocation) (*ai.ToolInvocation, error) {
	if m.injectCreateToolInvocationErr != nil {
		return nil, m.injectCreateToolInvocationErr
	}
	// 复用父类分配 ID 的逻辑
	i.ID = 100
	return i, nil
}

func (m *persistFailMockRepo) CreateMessage(_ context.Context, msg *ai.Message) (*ai.Message, error) {
	if m.injectCreateMessageErr != nil {
		return nil, m.injectCreateMessageErr
	}
	return msg, nil
}

// newPersistFailEnv 构造 Service + 可注入错误的 mock repo。
//
// 与 newRBACTestEnv 的差异：
//   - 返回 *persistFailMockRepo 而不是 *rbacMockRepo；
//   - 沿用同样的 HardcodeOnly 模式，避免引入 DB 依赖；
//   - 调用方可设置 inject*Err 字段触发失败分支。
func newPersistFailEnv(t *testing.T) (*ai.Service, *persistFailMockRepo) {
	t.Helper()
	repo := &persistFailMockRepo{rbacMockRepo: &rbacMockRepo{}}
	tools := service.NewToolRegistry(nil, nil, nil, nil)
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, tools, nil, nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(&ent.Client{})

	prevMode := middleware.PermissionConfig.Mode
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	middleware.InvalidateAllPermissionCaches()
	ai.ResetRBACFlagForTest()
	t.Cleanup(func() {
		middleware.PermissionConfig.Mode = prevMode
		middleware.InvalidateAllPermissionCaches()
		ai.ResetRBACFlagForTest()
	})
	return svc, repo
}

// ===== T1: recordToolAudit 在 DB 失败时不应阻塞主业务 =====
// 审计是 observability 副作用，写入失败不能让 ExecuteTool 返回 DB 错误，
// 必须让原业务错误（unknown tool / denied）继续走。
func TestRecordToolAudit_DBFailureDoesNotPropagate(t *testing.T) {
	svc, repo := newPersistFailEnv(t)
	setFlag(t, true, true)

	repo.injectCreateToolInvocationErr = errors.New("simulated DB outage")

	// 通过"未知工具"路径触发 recordToolAudit
	_, _, err := svc.ExecuteTool(context.Background(), 1, 10, "super_admin", "no_such_tool_xyz", map[string]interface{}{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ai.ErrUnknownTool),
		"DB 失败时仍应返回业务错误 ErrUnknownTool，而不是泄露 DB 错误给上层")
	assert.NotContains(t, err.Error(), "simulated DB outage",
		"DB 错误细节不得出现在响应中（防止日志注入/敏感信息泄露）")
}

// ===== T2: DB 失败必须上报 itsm_ai_persist_errors_total 指标 =====
// 不阻塞业务是对的，但失败本身必须可观测，否则就是 2026-06-13 审计里那个静默失败坑。
func TestRecordToolAudit_EmitsMetric(t *testing.T) {
	svc, repo := newPersistFailEnv(t)
	setFlag(t, true, true)

	tenantLabel := "10"
	// 读基线（避免并发测试污染采用相对差值）
	before := testutil.ToFloat64(metrics.AIPersistErrors.WithLabelValues(
		"create_tool_invocation", "", tenantLabel,
	))

	repo.injectCreateToolInvocationErr = errors.New("simulated DB outage")
	_, _, _ = svc.ExecuteTool(context.Background(), 1, 10, "super_admin", "no_such_tool_xyz", map[string]interface{}{})

	after := testutil.ToFloat64(metrics.AIPersistErrors.WithLabelValues(
		"create_tool_invocation", "", tenantLabel,
	))
	assert.Equal(t, before+1, after,
		"DB 失败必须累加 itsm_ai_persist_errors_total{operation=create_tool_invocation,role=,tenant_id=10}")
}

// ===== T3: Happy path 不应增加指标（确保修复未误报） =====
// 防止"修复反而让所有成功路径也报错"——回归保护。
func TestRecordToolAudit_HappyPathNoMetricIncrement(t *testing.T) {
	svc, _ := newPersistFailEnv(t)
	setFlag(t, true, true)

	tenantLabel := "10"
	before := testutil.ToFloat64(metrics.AIPersistErrors.WithLabelValues(
		"create_tool_invocation", "", tenantLabel,
	))

	// 不注入错误，正常 super_admin 调用 create_ticket（写工具走 pending 路径）
	_, _, err := svc.ExecuteTool(context.Background(), 1, 10, "super_admin", "create_ticket", map[string]interface{}{})
	require.NoError(t, err)

	after := testutil.ToFloat64(metrics.AIPersistErrors.WithLabelValues(
		"create_tool_invocation", "", tenantLabel,
	))
	assert.Equal(t, before, after,
		"happy path 不应触发持久化失败计数器")
}

// ===== T4: 并发安全验证（修复涉及多 goroutine 同时上报指标） =====
// recordToolAudit / CreateMessage 在并发请求下被调用，prometheus.CounterVec 本身线程安全，
// 但 Service.logger.Errorw 需要确认 zap.SugaredLogger 也是线程安全的（zap 官方保证）。
// 本测试作为烟雾测试，验证无数据竞争。
func TestRecordToolAudit_ConcurrentSafe(t *testing.T) {
	svc, repo := newPersistFailEnv(t)
	setFlag(t, true, true)
	repo.injectCreateToolInvocationErr = errors.New("simulated DB outage")

	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _, _ = svc.ExecuteTool(context.Background(), 1, 10, "super_admin", "no_such_tool_xyz", map[string]interface{}{})
		}()
	}
	wg.Wait()

	// 不崩溃就算通过；具体指标值不在此断言（被其他测试干扰）
	// 只需确认 no panic / no race（go test -race 会兜底）
}
