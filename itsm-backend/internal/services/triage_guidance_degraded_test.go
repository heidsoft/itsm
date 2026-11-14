package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// E-P0-3 回归：AI sidecar 在 LLM 不可用时不报错，而是把关键词分类包成 HTTP 200
// 返回（method="guidance-fallback"、degraded=true）。后端此前硬编码
// withTrace(..., "guidance")，审计底账因此出现一条「模型决策」——事后无法区分
// 哪些结论来自模型、哪些来自正则。以下用例锁死「降级必须可识别」这条不变量。

// sidecar 明确降级时，不能直接采信为 guidance 决策：本地关键词启发式
// 分类枚举更完整，应优先于 sidecar 的降级结果。
func TestTriage_GuidanceDegraded_PrefersLocalKeyword(t *testing.T) {
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:    "general", // sidecar 关键词表较窄，命中不到
			Priority:    "medium",
			Confidence:  0.5,
			Method:      "guidance-fallback",
			Degraded:    true,
			Explanation: "keyword-based classification (LLM unavailable)",
		},
	}
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "MySQL连接失败", "数据库无法访问", 1)

	assert.Equal(t, 1, guidance.callCount)
	assert.Equal(t, "database", result.Category, "应回落到本地关键词启发式")
	assert.Equal(t, 0, result.AssigneeID, "关键词链路不再注入硬编码 assignee，0 = 无建议")
	assert.Equal(t, "keyword", result.Model)
	assert.True(t, result.Degraded, "关键词链路不是模型推理，必须标注降级")
	assert.NotEmpty(t, result.DegradedReason)
}

// 只提供 degraded 布尔位（不改 method）的新版 sidecar 也要被识别。
func TestTriage_GuidanceDegradedFlagOnly(t *testing.T) {
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "network",
			Priority:   "high",
			Confidence: 0.5,
			Method:     "", // 老 method 字段留空
			Degraded:   true,
		},
	}
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "应用部署失败", "deploy 报错", 1)

	assert.Equal(t, "application", result.Category, "不得采用 sidecar 的降级分类")
	assert.True(t, result.Degraded)
}

// 正常模型链路不得被误判为降级。
func TestTriage_GuidanceModelPath_NotDegraded(t *testing.T) {
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "database",
			Priority:   "high",
			Confidence: 0.86,
			Method:     "guidance",
		},
	}
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "数据库慢查询", "", 1)

	assert.Equal(t, "guidance", result.Model)
	assert.False(t, result.Degraded)
	assert.Empty(t, result.DegradedReason)
	assert.Equal(t, 0.86, result.Confidence)
}

// 老版本 sidecar 没有 method 字段：按 guidance 处理（不能因此把正常结果判成降级）。
func TestTriage_GuidanceEmptyMethod_TreatedAsModelPath(t *testing.T) {
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "storage",
			Priority:   "medium",
			Confidence: 0.72,
		},
	}
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "磁盘空间不足", "", 1)

	assert.Equal(t, "guidance", result.Model)
	assert.False(t, result.Degraded)
}

// sidecar 降级 + 本地全不可用（无网关、关键词兜底关闭）时，才回落到 sidecar
// 降级结果，并保留其自报 method 作为溯源——绝不伪装成模型链路。
func TestTriage_GuidanceDegraded_LastResortKeepsProvenance(t *testing.T) {
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "server",
			Priority:   "high",
			Confidence: 0.5,
			Method:     "guidance-fallback",
			Degraded:   true,
		},
	}
	svc := NewTriageServiceWithConfig(nil, zap.NewNop(), false)
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "主机负载过高", "", 1)

	assert.Equal(t, "server", result.Category)
	assert.Equal(t, "guidance-fallback", result.Model, "必须如实标注链路，不能写成 guidance")
	assert.True(t, result.Degraded)
	assert.Contains(t, result.DegradedReason, "guidance-fallback")
}

// 网关报错且关键词兜底关闭时，也不能丢掉 sidecar 的降级结果（有总比没有好），
// 但同样要标注来源。
func TestTriage_GatewayError_UsesSidecarDegradedAsLastResort(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "", errors.New("LLM provider unavailable")
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "security",
			Priority:   "critical",
			Confidence: 0.5,
			Method:     "guidance-fallback",
			Degraded:   true,
		},
	}
	svc := NewTriageServiceWithConfig(gateway, zap.NewNop(), false)
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "疑似入侵", "", 1)

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "guidance-fallback", result.Model)
	assert.True(t, result.Degraded)
}

// 关键词链路自身也必须标注降级：它是启发式，不是模型结论。
func TestTriage_KeywordPath_IsMarkedDegraded(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())

	result := svc.Suggest(context.Background(), "MySQL 连接失败", "")

	assert.Equal(t, "keyword", result.Model)
	assert.True(t, result.Degraded)
	assert.NotEmpty(t, result.DegradedReason)
}

// SuggestWithMetadata 的 fallback 分支同样要带降级标记（它也是审计底账的来源）。
func TestTriage_SuggestWithMetadata_FallbackMarkedDegraded(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())

	meta, err := svc.SuggestWithMetadata(context.Background(), "服务器内存不足", "Linux 主机负载过高")

	assert.NoError(t, err)
	assert.Equal(t, "fallback", meta.Method)
	assert.True(t, meta.Result.Degraded)
	assert.Equal(t, "keyword", meta.Result.Model)
	assert.NotZero(t, meta.Latency)
}
