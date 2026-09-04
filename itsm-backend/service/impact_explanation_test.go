package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"itsm-backend/dto"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// fakeLLMProvider 是 P1-4 影响分析解释层的最小 mock，用于离线单测。
type fakeLLMProvider struct {
	response string
	err      error
}

func (f *fakeLLMProvider) Chat(_ context.Context, _ string, _ []LLMMessage) (string, error) {
	return f.response, f.err
}

// alwaysAllowLimiter / nopObserver 满足 LLMGateway 构造约束
type alwaysAllowLimiter struct{}

func (alwaysAllowLimiter) Allow(nTokens int) bool { return true }

type nopObserver struct{}

func (nopObserver) Observe(_ string, _ string, _ int, _ time.Duration, _ error) {}

// TestImpactExplanation_FailOpenNoLLM 锁定回归：无 LLM 配置时 ExplainImpact 不报错、不 panic、返回 nil
func TestImpactExplanation_FailOpenNoLLM(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	// 不注入 LLMGateway（nil gateway 是 fail-open 路径）
	svc := NewImpactExplanationService(nil, "gpt-4o", nil, logger)

	impact := &dto.CIImpactAnalysisResponse{
		SourceCIID: 1,
		RiskLevel:  "medium",
		Summary:    "test impact",
	}
	got, err := svc.ExplainImpact(context.Background(), 1, 1, 3, impact)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestImpactExplanation_ParseMarkdownJSON 锁定回归：模型返回 ```json``` 块时正确剥离
func TestImpactExplanation_ParseMarkdownJSON(t *testing.T) {
	got, err := parseImpactExplanationJSON("```json\n" + `{"summary":"x","rootCauses":["a"],"slaRisks":["b"],"suggestions":["c"]}` + "\n```")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "x", got.Summary)
	require.Equal(t, []string{"a"}, got.RootCauses)
	require.Equal(t, []string{"b"}, got.SLARisks)
	require.Equal(t, []string{"c"}, got.Suggestions)
}

// TestImpactExplanation_ParsePlainJSON 锁定回归：直接 JSON 输入也能解析
func TestImpactExplanation_ParsePlainJSON(t *testing.T) {
	raw := `{"summary":"hello","rootCauses":["r1"],"slaRisks":["s1"],"suggestions":["sug1"]}`
	got, err := parseImpactExplanationJSON(raw)
	require.NoError(t, err)
	require.Equal(t, "hello", got.Summary)
}

// TestImpactExplanation_ParseMissingSummary 锁定回归：summary 缺失返回错误
func TestImpactExplanation_ParseMissingSummary(t *testing.T) {
	raw := `{"rootCauses":["r1"]}`
	got, err := parseImpactExplanationJSON(raw)
	require.Error(t, err)
	require.Nil(t, got)
}

// TestImpactExplanation_LLMResponse 端到端：LLM 返回 JSON → ExplainImpact 序列化 + 缓存写后 nil
func TestImpactExplanation_LLMResponse(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	mockLLM := &fakeLLMProvider{
		response: `{"summary":"数据库主库故障可能影响 3 个下游服务","rootCauses":["主库 IO 异常","网络抖动"],"slaRisks":["订单服务 critical 影响"],"suggestions":["先排查主库慢查询","切读从库"]}`,
	}
	llmGateway := NewLLMGateway(mockLLM, alwaysAllowLimiter{}, nopObserver{}, "mock")
	svc := NewImpactExplanationService(llmGateway, "mock-model", nil, logger)

	impact := &dto.CIImpactAnalysisResponse{
		SourceCIID: 42,
		RiskLevel:  "high",
		Summary:    "raw summary",
		UpstreamImpact: []dto.ImpactAnalysisItem{
			{Distance: 1, Direction: "incoming", RelationshipType: "depends_on", ImpactLevel: "high"},
		},
		DownstreamImpact: []dto.ImpactAnalysisItem{
			{Distance: 1, Direction: "outgoing", RelationshipType: "hosts", ImpactLevel: "critical"},
		},
	}
	got, err := svc.ExplainImpact(context.Background(), 1, 42, 3, impact)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "数据库主库故障可能影响 3 个下游服务", got.Summary)
	require.Len(t, got.RootCauses, 2)
	require.Len(t, got.SLARisks, 1)
	require.Len(t, got.Suggestions, 2)
	require.Equal(t, "mock-model", got.Model)
	require.False(t, got.GeneratedAt == "")

	// 缓存未注入（nil redis），第二次仍走 LLM：mock 仍返回 → 再次解析
	got2, err := svc.ExplainImpact(context.Background(), 1, 42, 3, impact)
	require.NoError(t, err)
	require.NotNil(t, got2)
	require.Equal(t, got.Summary, got2.Summary)
}

// TestImpactExplanation_LLMError 锁定回归：LLM 调用失败时返回 nil 不向上抛
func TestImpactExplanation_LLMError(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	mockLLM := &fakeLLMProvider{err: context.DeadlineExceeded}
	llmGateway := NewLLMGateway(mockLLM, alwaysAllowLimiter{}, nopObserver{}, "mock")
	svc := NewImpactExplanationService(llmGateway, "mock-model", nil, logger)

	impact := &dto.CIImpactAnalysisResponse{SourceCIID: 1, RiskLevel: "low"}
	got, err := svc.ExplainImpact(context.Background(), 1, 1, 3, impact)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestImpactExplanation_NilImpact 锁定回归：impact 入参 nil 必须报错（防御）
func TestImpactExplanation_NilImpact(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewImpactExplanationService(nil, "mock", nil, logger)
	got, err := svc.ExplainImpact(context.Background(), 1, 1, 3, nil)
	require.Error(t, err)
	require.Nil(t, got)
}

// TestImpactExplanation_BuildPrompt 验证 prompt 包含必要字段（便于 LLM 理解）
func TestImpactExplanation_BuildPrompt(t *testing.T) {
	impact := &dto.CIImpactAnalysisResponse{
		SourceCIID: 99,
		RiskLevel:  "high",
		Summary:    "raw-summary",
		UpstreamImpact: []dto.ImpactAnalysisItem{
			{Distance: 1, Direction: "incoming", RelationshipType: "depends_on", ImpactLevel: "high"},
		},
		DownstreamImpact: []dto.ImpactAnalysisItem{
			{Distance: 2, Direction: "outgoing", RelationshipType: "hosts", ImpactLevel: "medium"},
		},
	}
	prompt := buildImpactPrompt(impact)
	require.Contains(t, prompt, "源 CI ID: 99")
	require.Contains(t, prompt, "整体风险等级: high")
	require.Contains(t, prompt, "上游影响 (1)")
	require.Contains(t, prompt, "下游影响 (1)")
	require.Contains(t, prompt, "distance=1")
	require.Contains(t, prompt, "direction=incoming")
}

// TestImpactExplanation_JSONRoundtrip 验证 ImpactExplanation 可序列化（用于缓存/对外 API）
func TestImpactExplanation_JSONRoundtrip(t *testing.T) {
	ie := &ImpactExplanation{
		Summary:     "test",
		RootCauses:  []string{"a", "b"},
		SLARisks:    []string{"s"},
		Suggestions: []string{"x", "y", "z"},
	}
	raw, err := json.Marshal(ie)
	require.NoError(t, err)
	var back ImpactExplanation
	require.NoError(t, json.Unmarshal(raw, &back))
	require.Equal(t, ie.Summary, back.Summary)
	require.Equal(t, ie.RootCauses, back.RootCauses)
}

// TestImpactExplanation_TruncateLog 锁定 truncate 函数边界（200 字符）
func TestImpactExplanation_TruncateLog(t *testing.T) {
	short := "x"
	require.Equal(t, short, truncateForLog(short))
	long := strings.Repeat("y", 250)
	out := truncateForLog(long)
	require.Equal(t, 203, len(out)) // 200 + "..."
	require.True(t, strings.HasSuffix(out, "..."))
}