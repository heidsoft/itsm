package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockLLMGateway implements LLMProvider for testing
type MockSummarizeLLMGateway struct {
	MockChat func(ctx context.Context, model string, messages []LLMMessage) (string, error)
}

func (m *MockSummarizeLLMGateway) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	return m.MockChat(ctx, model, messages)
}

func TestSummarize_TextShort_ReturnsAsIs(t *testing.T) {
	svc := NewSummarizeService(nil, zap.NewNop()) // nil gateway

	text := "短文本内容"
	result, err := svc.Summarize(context.Background(), text, 200)

	assert.NoError(t, err)
	assert.Equal(t, text, result)
}

func TestSummarize_WithLLM_Success(t *testing.T) {
	mockLLM := &MockSummarizeLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "这是LLM生成的摘要内容", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewSummarizeService(gateway, zap.NewNop())

	longText := strings.Repeat("a", 1000)
	result, err := svc.Summarize(context.Background(), longText, 200)

	assert.NoError(t, err)
	assert.Contains(t, result, "LLM生成的摘要")
}

func TestSummarize_WithLLM_TrimsOutput(t *testing.T) {
	mockLLM := &MockSummarizeLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "```json\n摘要内容\n```", nil // with markdown
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewSummarizeService(gateway, zap.NewNop())

	result, err := svc.Summarize(context.Background(), strings.Repeat("a", 1000), 200)

	assert.NoError(t, err)
	assert.NotContains(t, result, "```") // markdown should be cleaned
}

func TestSummarize_NoGateway_Fallback(t *testing.T) {
	svc := NewSummarizeService(nil, zap.NewNop()) // nil gateway

	longText := strings.Repeat("a", 500)
	result, err := svc.Summarize(context.Background(), longText, 100)

	assert.NoError(t, err)
	assert.Equal(t, 103, len(result)) // 100 + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestSummarize_WithContext(t *testing.T) {
	mockLLM := &MockSummarizeLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			// Verify context is included in prompt
			assert.Contains(t, messages[1].Content, "相关上下文信息")
			assert.Contains(t, messages[1].Content, "历史记录")
			return "基于上下文的摘要", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewSummarizeService(gateway, zap.NewNop())

	longText := strings.Repeat("a", 500)
	result, err := svc.SummarizeWithContext(context.Background(), longText, "历史记录：之前出现过类似问题", 200)

	assert.NoError(t, err)
	assert.Contains(t, result, "基于上下文的摘要")
}

func TestSummarize_GenerateActionItems(t *testing.T) {
	mockLLM := &MockSummarizeLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "- 重启服务器\n- 检查网络连接\n- 更新配置文件", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewSummarizeService(gateway, zap.NewNop())

	result, err := svc.GenerateActionItems(context.Background(), "服务器宕机需要处理")

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "重启服务器", result[0])
	assert.Equal(t, "检查网络连接", result[1])
	assert.Equal(t, "更新配置文件", result[2])
}

func TestSummarize_GenerateActionItems_NoGateway(t *testing.T) {
	svc := NewSummarizeService(nil, zap.NewNop()) // nil gateway

	result, err := svc.GenerateActionItems(context.Background(), "需要处理一些事情")

	assert.NoError(t, err)
	assert.Empty(t, result) // should return empty slice when no gateway
}

func TestSummarize_simpleTruncate(t *testing.T) {
	result := simpleTruncate("hello world", 5)
	assert.Equal(t, "hello...", result)

	result = simpleTruncate("hi", 10)
	assert.Equal(t, "hi", result) // shorter than maxLen, no truncation

	result = simpleTruncate("exactly", 7)
	assert.Equal(t, "exactly", result) // exactly maxLen, no truncation needed

	result = simpleTruncate("toolong", 4)
	assert.Equal(t, "tool...", result)
	assert.Equal(t, 7, len(result))
}

func TestSummarize_EmptyInput(t *testing.T) {
	svc := NewSummarizeService(nil, zap.NewNop())

	result, err := svc.Summarize(context.Background(), "", 200)

	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestSummarize_WithLLM_Error_Fallback(t *testing.T) {
	mockLLM := &MockSummarizeLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "", assert.AnError
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewSummarizeService(gateway, zap.NewNop())

	// Should fallback to simple truncation on error
	result, err := svc.Summarize(context.Background(), strings.Repeat("a", 500), 100)

	assert.NoError(t, err)
	assert.Equal(t, 103, len(result)) // 100 + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}
