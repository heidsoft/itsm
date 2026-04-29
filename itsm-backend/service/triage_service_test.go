package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockLLMGateway implements LLMProvider for testing
type MockLLMGateway struct {
	MockChat func(ctx context.Context, model string, messages []LLMMessage) (string, error)
}

func (m *MockLLMGateway) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	return m.MockChat(ctx, model, messages)
}

func TestTriage_Suggest_Keyword_Database(t *testing.T) {
	// nil gateway forces keyword-based fallback
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "MySQL connection failed", "database cannot be accessed")

	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 101, result.AssigneeID)
	assert.Greater(t, result.Confidence, 0.6)
}

func TestTriage_Suggest_Keyword_Network(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "WiFi cannot connect", "router no response")

	assert.Equal(t, "network", result.Category)
	assert.Equal(t, 102, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_Security(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	// Use text that only matches security keywords (not SQL which is also database)
	result := svc.Suggest(context.Background(), "authentication attack detected", "")

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "critical", result.Priority)
	assert.Equal(t, 100, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_PriorityEscalation(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	// "server" matches category, "down" triggers priority escalation
	result := svc.Suggest(context.Background(), "server down", "service unavailable")

	assert.Equal(t, "server", result.Category)
	assert.True(t, result.Priority == "high" || result.Priority == "critical",
		"expected priority high or critical, got %s", result.Priority)
}

func TestTriage_Suggest_LLM_Success(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			// Verify the prompt contains the ticket title
			assert.True(t, strings.Contains(messages[1].Content, "发现恶意请求"))
			return `{"category":"security","priority":"critical","confidence":0.9,"explanation":"sql injection"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	result := svc.Suggest(context.Background(), "发现恶意请求", "")

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "critical", result.Priority)
	assert.Equal(t, 100, result.AssigneeID)
	assert.Equal(t, 0.9, result.Confidence)
}

func TestTriage_Suggest_LLM_ParseError_Fallback(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "not valid json", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	// Use text that matches keyword "server" for fallback verification
	result := svc.Suggest(context.Background(), "server is down", "")

	// Should fallback to keyword, matching "server"
	assert.Equal(t, "server", result.Category)
}

func TestTriage_Suggest_LLM_Error_Fallback(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "", errors.New("LLM provider error")
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	result := svc.Suggest(context.Background(), "应用部署失败", "deploy报错")

	// Should fallback to keyword, matching "application"
	assert.Equal(t, "application", result.Category)
}

func TestTriage_Suggest_NoGateway_KeywordOnly(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop()) // gateway=nil
	result := svc.Suggest(context.Background(), "应用部署失败", "deploy报错")

	assert.Equal(t, "application", result.Category)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestTriage_Suggest_EmptyInput(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "", "")

	assert.Equal(t, "general", result.Category)
	assert.Equal(t, "medium", result.Priority)
	assert.Equal(t, 0, result.AssigneeID)
}

func TestTriage_containsAny(t *testing.T) {
	// containsAny is case-sensitive
	assert.True(t, containsAny("mysql connection failed", "mysql", "database"))
	assert.False(t, containsAny("normal text", "mysql", "database"))
	assert.True(t, containsAny("hello world", "hello"))
	assert.False(t, containsAny("", "hello"))
	// case-sensitive: "MySQL" does NOT contain "mysql"
	assert.False(t, containsAny("MySQL is down", "mysql"))
	assert.True(t, containsAny("mysql is down", "mysql"))
}

func TestTriage_Suggest_LLM_JSONWithMarkdown(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "```json\n{\"category\":\"storage\",\"priority\":\"high\",\"confidence\":0.85,\"explanation\":\"disk space issue\"}\n```", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	result := svc.Suggest(context.Background(), "磁盘空间不足", "")

	assert.Equal(t, "storage", result.Category)
	assert.Equal(t, "high", result.Priority)
	assert.Equal(t, 105, result.AssigneeID)
}

func TestTriage_BatchSuggest(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop()) // use keyword-only for predictable results

	tickets := []struct {
		ID          int
		Title       string
		Description string
	}{
		{ID: 1, Title: "WiFi cannot connect", Description: "cannot上网"},
		{ID: 2, Title: "mysql connection failed", Description: "database cannot respond"},
	}

	results, err := svc.BatchSuggest(context.Background(), tickets)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 1, results[0].ID)
	assert.Equal(t, "network", results[0].Result.Category)
	assert.Equal(t, 2, results[1].ID)
	assert.Equal(t, "database", results[1].Result.Category)
}
