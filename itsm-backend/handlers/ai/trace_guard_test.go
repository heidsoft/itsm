package ai

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/dto"
)

// 本文件守卫 AI 溯源的几条「形状」性质，重点是不该出现的东西：
//
//  1. 深度分析（SQL 聚合）**不得**出现 model 字段——它没有任何模型参与，
//     给它一个 model 就是伪造归因，客户端会以为这是可追溯的 AI 建议。
//  2. 预测/聊天的响应必须带口径版本与底账 ID。
//  3. 检索置信度必须是真实命中分，取不到时为 0（不是编造一个中间值）。

// TestDeepAnalyticsMustNotFakeAModel 锁住最重要的一条：聚合结果不得伪造溯源。
//
// 为什么值得单独一个测试：把 analytics 与 chat/摘要并列成「缺溯源字段」后，
// 最省事的补法是给它也加一个 model 字段——那样所有能力「看起来」都齐了，
// 但 analytics 的 model 必然是假的。假溯源比缺失更危险。
func TestDeepAnalyticsMustNotFakeAModel(t *testing.T) {
	typ := reflect.TypeOf(dto.DeepAnalyticsResponse{})

	// 不得有 model / promptVersion / confidence 这类「模型归因」字段。
	forbidden := []string{"model", "modelName", "promptVersion", "confidence"}
	fields := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			continue
		}
		name := tag
		if idx := len(tag); idx > 0 {
			for j := 0; j < idx; j++ {
				if tag[j] == ',' {
					name = tag[:j]
					break
				}
			}
		}
		fields[name] = true
	}
	for _, f := range forbidden {
		assert.False(t, fields[f],
			"DeepAnalyticsResponse 是 SQL 聚合，不得带 %q 字段：那是伪造的模型归因", f)
	}

	// 该有的是口径版本与方法标注——这才是聚合结果真正需要的溯源。
	assert.True(t, fields["metricVersion"], "聚合结果必须带口径版本 metricVersion")
	assert.True(t, fields["method"], "聚合结果必须带 method 标注（值固定为 aggregate）")
}

// TestPredictionResponseCarriesTraceFields 锁住预测响应的溯源字段：
// 预测值会被拿去做容量/风险决策，口径版本与底账 ID 缺一不可。
func TestPredictionResponseCarriesTraceFields(t *testing.T) {
	typ := reflect.TypeOf(dto.TrendPredictionResponse{})
	names := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		tag, ok := typ.Field(i).Tag.Lookup("json")
		if !ok {
			continue
		}
		name := tag
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				name = tag[:j]
				break
			}
		}
		names[name] = true
	}
	assert.True(t, names["promptVersion"], "预测响应必须带口径版本")
	assert.True(t, names["auditId"], "预测响应必须带底账 ID（供 PATCH 翻转采纳/拒绝）")
	assert.True(t, names["confidence"], "预测保留真实置信度（它有可计算的置信度）")
	assert.True(t, names["model"], "预测保留真实模型/算法标识")
}

// TestTopRetrievalScore 锁住检索置信度的语义：取命中片段里的最高真实分，
// 取不到时为 0——「库里没有可信命中」是实情，不能编一个中间值充数。
func TestTopRetrievalScore(t *testing.T) {
	cases := []struct {
		name  string
		items interface{}
		want  float64
	}{
		{"命中多片段取最高分", []map[string]any{
			{"score": 0.42}, {"score": 0.91}, {"score": 0.7},
		}, 0.91},
		{"无命中", []map[string]any{}, 0},
		{"片段无 score 字段", []map[string]any{{"title": "x"}}, 0},
		{"结构不符", map[string]any{"a": 1}, 0},
		{"nil", nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.want, topRetrievalScore(tc.items), 0.0001)
		})
	}
}

func TestHitCount(t *testing.T) {
	assert.Equal(t, 0, hitCount(nil))
	assert.Equal(t, 2, hitCount([]map[string]any{{"score": 0.1}, {"score": 0.2}}))
}

// TestTruncateForRef 底账 inputRef 要能定位又不膨胀：超长 query 截断，短的原样。
func TestTruncateForRef(t *testing.T) {
	assert.Equal(t, "短查询", truncateForRef("短查询", 120))
	long := ""
	for i := 0; i < 300; i++ {
		long += "字"
	}
	got := truncateForRef(long, 120)
	assert.Len(t, []rune(got), 121, "截断后应为 120 字符 + 省略号")
	assert.Contains(t, got, "…")
}

// TestChatWithoutRAGReturnsErrorNotPanic 锁住健壮性：RAG 未装配时返回错误，
// 而不是 nil 解引用崩溃。此前 s.rag.Ask(...) 会直接 panic（装配缺口的代价
// 不该是进程级 500）——handler 的降级分支能接住错误。
func TestChatWithoutRAGReturnsErrorNotPanic(t *testing.T) {
	svc := &Service{logger: zap.NewNop().Sugar()}
	require.NotPanics(t, func() {
		_, _, err := svc.Chat(context.Background(), 1, 1, "VPN 连不上怎么办", 5, 0)
		require.ErrorIs(t, err, ErrRAGUnavailable)
	})

	// 带溯源的入口同样不得 panic，而是把错误透传出去。
	require.NotPanics(t, func() {
		_, err := svc.ChatWithTrace(context.Background(), 1, 1, "VPN 连不上怎么办", 5, 0)
		require.ErrorIs(t, err, ErrRAGUnavailable)
	})
}

// TestForecastPromptVersionStable 锁住口径版本常量：预测与洞察共用同一个口径
// 版本，改算法时必须同步改它（否则底账里新旧口径无法区分）。
func TestForecastPromptVersionStable(t *testing.T) {
	assert.Equal(t, "sla-forecast-v1", slaForecastPromptVersion)
	assert.Equal(t, "deep-analytics-v1", analyticsMetricVersion)
	assert.Equal(t, "rag-ask-v1", ragAskPromptVersion)
}
