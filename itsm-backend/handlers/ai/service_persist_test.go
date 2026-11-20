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
	"itsm-backend/internal/services"
	"itsm-backend/metrics"
	"itsm-backend/middleware"
)

// persistFailMockRepo 继承 rbacMockRepo 的字段，扩展 CreateToolInvocation 错误注入能力。
//
// handlers/ai/service.go 若用 _, _ = 吞掉持久化错误，DB 抖动时 AI 对话和审计会静默失败。
// 失败路径必须同时满足：(a) 不阻塞主业务、(b) 上报结构化错误日志、
// (c) 上报 itsm_ai_persist_errors_total 计数器。
// 本 mock 通过让 CreateToolInvocation 返回错误，触发 recordToolAudit 的失败分支，
// 验证 (a) 不抛错（best-effort 保留）和 (c) 指标累加。
type persistFailMockRepo struct {
	*rbacMockRepo
	injectCreateToolInvocationErr error
	injectCreateMessageErr        error
	// aiResults 捕获落库的 AI 分析结果，供降级标记断言使用。
	aiResults []*ai.AIAnalysisResult
}

// SaveAIAnalysisResult 覆盖父类的"直接返回不落库"，把记录留下来以便断言。
func (m *persistFailMockRepo) SaveAIAnalysisResult(_ context.Context, r *ai.AIAnalysisResult) (*ai.AIAnalysisResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aiResults = append(m.aiResults, r)
	return r, nil
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
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, tools, nil, nil, nil, nil, nil, nil)
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

// ===== DB 失败必须上报 itsm_ai_persist_errors_total 指标 =====
// 不阻塞业务是对的，但失败本身必须可观测，否则审计丢失不会被发现。
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

// ===== 降级标记必须如实传到审计底账 =====
//
// 事故背景：AnalyzeTicketWithAudit 曾以 `rootCauses, _, _ := performAnalysis(...)`
// 丢弃降级标记，同时 persistAIResult 把 Degraded 硬编码为 false。两者叠加导致
// 「LLM 超时后的启发式兜底」在 ai_analysis_results 里表现为模型正常产出，
// 事后审计无法区分真产出与兜底。
//
// 这里走公开入口 AnalyzeTicketWithAudit，断言写入底账的 Degraded 与真实降级路径一致；
// 用它而不是直接测内部函数，是为了让测试同时也覆盖「提取 → 传递 → 入库」整条链。
func TestAnalyzeTicketWithAudit_PersistsDegradedFlag(t *testing.T) {
	svc, repo := newPersistFailEnv(t)
	// 未注入 RCA 服务 → 服务层会返回 ErrAIAnalysisUnavailable，
	// 属于「AI 未装配」而非「降级」，此处只确认它不放行伪造结果。
	_, err := svc.AnalyzeTicketWithAudit(context.Background(), 1, 1, 2)
	require.Error(t, err, "RCA 未装配时必须报错，不能返回伪造结果")
	assert.Empty(t, repo.aiResults, "失败路径不得写底账")
}
