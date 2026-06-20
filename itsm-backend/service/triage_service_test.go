package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type MockLLMGateway struct {
	MockChat func(ctx context.Context, model string, messages []LLMMessage) (string, error)
}

func (m *MockLLMGateway) Chat(ctx context.Context, model string, messages []LLMMessage) (string, error) {
	return m.MockChat(ctx, model, messages)
}

type fakeGuidanceClient struct {
	result    *GuidanceTriageResponse
	err       error
	tenantID  int
	callCount int
}

func (f *fakeGuidanceClient) Triage(ctx context.Context, title, description string, tenantID int) (*GuidanceTriageResponse, error) {
	f.callCount++
	f.tenantID = tenantID
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestTriage_Suggest_Keyword_Database(t *testing.T) {
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
	result := svc.Suggest(context.Background(), "authentication attack detected", "")

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "critical", result.Priority)
	assert.Equal(t, 100, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_ChineseDatabase(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "生产数据库连接池耗尽", "业务查询大量超时")

	assert.Equal(t, "database", result.Category)
	assert.Equal(t, "high", result.Priority)
	assert.Equal(t, 101, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_ChineseNetwork(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "办公区网络中断", "交换机丢包，多个用户无法上网")

	assert.Equal(t, "network", result.Category)
	assert.Equal(t, "high", result.Priority)
	assert.Equal(t, 102, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_ChineseUserAccess(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "账号登录失败", "用户密码重置后仍无法访问系统")

	assert.Equal(t, "user_access", result.Category)
	assert.Equal(t, "medium", result.Priority)
	assert.Equal(t, 106, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_ChineseUserImpactEscalation(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "账号登录失败", "多个用户密码重置后仍无法访问系统")

	assert.Equal(t, "user_access", result.Category)
	assert.Equal(t, "high", result.Priority)
	assert.Equal(t, 106, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_StorageBeforeServerDisk(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "disk space alert", "backup volume capacity is almost full")

	assert.Equal(t, "storage", result.Category)
	assert.Equal(t, 105, result.AssigneeID)
}

func TestTriage_Suggest_Keyword_PriorityEscalation(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	result := svc.Suggest(context.Background(), "server down", "service unavailable")

	assert.Equal(t, "server", result.Category)
	assert.True(t, result.Priority == "high" || result.Priority == "critical",
		"expected priority high or critical, got %s", result.Priority)
}

func TestTriage_Suggest_LLM_Success(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
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

func TestTriage_Suggest_LLM_IgnoresModelAssigneeID(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return `{"category":"security","priority":"critical","confidence":0.9,"assignee_id":999,"explanation":"security incident"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	result := svc.Suggest(context.Background(), "发现恶意请求", "")

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, 100, result.AssigneeID)
}

func TestTriage_SuggestForTenant_PassesTenantToGuidance(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "database",
			Priority:   "high",
			Confidence: 0.8,
		},
	}
	svc.guidanceClient = guidance

	result := svc.SuggestForTenant(context.Background(), "数据库故障", "", 42)

	assert.Equal(t, 42, guidance.tenantID)
	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 101, result.AssigneeID)
}

func TestTriage_Suggest_LLM_ParseError_Fallback(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "not valid json", nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	result := svc.Suggest(context.Background(), "server is down", "")

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

	assert.Equal(t, "application", result.Category)
}

func TestTriage_Suggest_NoGateway_KeywordOnly(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
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
	assert.True(t, containsAny("mysql connection failed", "mysql", "database"))
	assert.False(t, containsAny("normal text", "mysql", "database"))
	assert.True(t, containsAny("hello world", "hello"))
	assert.False(t, containsAny("", "hello"))
	assert.False(t, containsAny("MySQL is down", "mysql"))
	assert.True(t, containsAny("mysql is down", "mysql"))
}

func TestTriage_Suggest_LLM_LowConfidence_FallbackToKeyword(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return `{"category":"general","priority":"medium","confidence":0.35,"explanation":"unclear ticket"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewTriageService(gateway, zap.NewNop())

	result := svc.Suggest(context.Background(), "mysql connection failed", "cannot connect to database")

	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 101, result.AssigneeID)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestTriage_Suggest_LLM_HighConfidence_NoKeywordFallback(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return `{"category":"network","priority":"high","confidence":0.85,"explanation":"wifi keywords detected"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	svc := NewTriageService(gateway, zap.NewNop())

	result := svc.Suggest(context.Background(), "WiFi connection issues", "")

	assert.Equal(t, "network", result.Category)
	assert.Equal(t, 102, result.AssigneeID)
	assert.Equal(t, 0.85, result.Confidence)
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

func TestTriage_Suggest_Guidance_NormalizesEmptyFields(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Explanation: "sidecar returned partial result",
		},
	}

	result := svc.Suggest(context.Background(), "无法判断", "")

	assert.Equal(t, "general", result.Category)
	assert.Equal(t, "medium", result.Priority)
	assert.Equal(t, 0, result.AssigneeID)
	assert.Equal(t, 0.6, result.Confidence)
	assert.Equal(t, "sidecar returned partial result", result.Explanation)
}

func TestTriage_Suggest_Guidance_NormalizesInvalidEnumsAndConfidence(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())
	svc.guidanceClient = &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:    "billing",
			Priority:    "p0",
			Confidence:  1.7,
			AssigneeID:  999,
			Explanation: "invalid enum values",
		},
	}

	result := svc.Suggest(context.Background(), "费用系统故障", "")

	assert.Equal(t, "general", result.Category)
	assert.Equal(t, "medium", result.Priority)
	assert.Equal(t, 0, result.AssigneeID)
	assert.Equal(t, 0.6, result.Confidence)
}

func TestTriage_BatchSuggest(t *testing.T) {
	svc := NewTriageService(nil, zap.NewNop())

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

// newTriageServiceWithGuidance creates a TriageService wired with a fake guidance client.
// Tests use this helper to exercise the guidance-failure / guidance-success paths
// without binding to localhost:8091.
func newTriageServiceWithGuidance(gateway *LLMGateway, guidance guidanceTriageClient) *TriageService {
	svc := NewTriageServiceWithGuidance(gateway, nil, zap.NewNop())
	svc.guidanceClient = guidance
	return svc
}

func TestTriage_Suggest_Guidance_Error_FallsBackToLLM(t *testing.T) {
	// Guidance sidecar errors out; service should log a warning and continue to LLM.
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return `{"category":"security","priority":"critical","confidence":0.88,"explanation":"attack signature"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	guidance := &fakeGuidanceClient{err: errors.New("sidecar unreachable")}

	svc := newTriageServiceWithGuidance(gateway, guidance)
	result := svc.Suggest(context.Background(), "发现恶意请求", "sql injection attempt")

	assert.Equal(t, 1, guidance.callCount, "guidance should be consulted exactly once")
	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "critical", result.Priority)
	assert.Equal(t, 0.88, result.Confidence)
	assert.Equal(t, 100, result.AssigneeID)
}

func TestTriage_Suggest_Guidance_Error_FallsBackToKeyword(t *testing.T) {
	// Both guidance and LLM fail; service should fall through to keyword fallback.
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "", errors.New("LLM provider timeout")
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")
	guidance := &fakeGuidanceClient{err: errors.New("sidecar 500")}

	svc := newTriageServiceWithGuidance(gateway, guidance)
	// "mysql" should trigger the database keyword bucket even when both upstream paths fail.
	result := svc.Suggest(context.Background(), "mysql connection failed", "database unreachable")

	assert.Equal(t, 1, guidance.callCount)
	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 101, result.AssigneeID)
}

func TestTriage_Suggest_Guidance_NormalizesEmptyCategory(t *testing.T) {
	// Guidance sidecar returns a successful response but with empty category.
	// The service must surface a normalized, non-empty category to the API contract.
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "",
			Priority:   "high",
			Confidence: 0.8,
		},
	}
	gateway := NewLLMGateway(nil, nil, nil, "test")
	svc := newTriageServiceWithGuidance(gateway, guidance)

	result := svc.Suggest(context.Background(), "anything", "anything")

	assert.Equal(t, "general", result.Category)
	assert.Equal(t, "high", result.Priority)
	assert.Equal(t, 0.8, result.Confidence)
}

func TestTriage_Suggest_Guidance_NormalizesInvalidCategory(t *testing.T) {
	// Guidance returns an enum value outside the contract; service must downgrade to general.
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "unknown_category",
			Priority:   "medium",
			Confidence: 0.7,
		},
	}
	gateway := NewLLMGateway(nil, nil, nil, "test")
	svc := newTriageServiceWithGuidance(gateway, guidance)

	result := svc.Suggest(context.Background(), "anything", "anything")

	assert.Equal(t, "general", result.Category)
}

func TestTriage_Suggest_Guidance_NormalizesOutOfRangeConfidence(t *testing.T) {
	// Confidence 1.5 is outside [0, 1]; service must clamp to the documented default.
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "network",
			Priority:   "high",
			Confidence: 1.5,
		},
	}
	gateway := NewLLMGateway(nil, nil, nil, "test")
	svc := newTriageServiceWithGuidance(gateway, guidance)

	result := svc.Suggest(context.Background(), "anything", "anything")

	assert.Equal(t, 0.6, result.Confidence)
	assert.Equal(t, 102, result.AssigneeID)
}

func TestTriage_Suggest_Guidance_Success_PreservesAssignee(t *testing.T) {
	// When guidance returns a valid response, the service must preserve its
	// normalized category→assignee mapping without overriding it.
	guidance := &fakeGuidanceClient{
		result: &GuidanceTriageResponse{
			Category:   "database",
			Priority:   "high",
			AssigneeID: 999, // non-default assignee; should be kept when non-zero
			Confidence: 0.9,
		},
	}
	gateway := NewLLMGateway(nil, nil, nil, "test")
	svc := newTriageServiceWithGuidance(gateway, guidance)

	result := svc.Suggest(context.Background(), "anything", "anything")

	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 999, result.AssigneeID)
	assert.Equal(t, 0.9, result.Confidence)
}

func TestTriage_NormalizeResult_RejectsNegativeConfidence(t *testing.T) {
	// Guard against regression where negative confidence slipped through normalization.
	svc := NewTriageService(nil, zap.NewNop())
	got := svc.normalizeResult(TriageResult{
		Category:   "storage",
		Priority:   "medium",
		Confidence: -0.3,
	})
	assert.Equal(t, 0.6, got.Confidence)
	assert.Equal(t, "storage", got.Category)
	assert.Equal(t, 105, got.AssigneeID)
}

func TestTriage_Suggest_GuidanceError_FallsBackToLLM(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return `{"category":"security","priority":"critical","confidence":0.92,"explanation":"detected sql injection keywords"}`, nil
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	svc.guidanceClient = &fakeGuidanceClient{
		err: errors.New("Guidance sidecar network error"),
	}

	result := svc.Suggest(context.Background(), "SQL注入攻击检测", "")

	assert.Equal(t, "security", result.Category)
	assert.Equal(t, "critical", result.Priority)
	assert.Equal(t, 100, result.AssigneeID)
	assert.Equal(t, 0.92, result.Confidence)
}

func TestTriage_Suggest_GuidanceErrorAndLLMError_FallsBackToKeyword(t *testing.T) {
	mockLLM := &MockLLMGateway{
		MockChat: func(ctx context.Context, model string, messages []LLMMessage) (string, error) {
			return "", errors.New("LLM provider unavailable")
		},
	}
	gateway := NewLLMGateway(mockLLM, nil, nil, "test")

	svc := NewTriageService(gateway, zap.NewNop())
	svc.guidanceClient = &fakeGuidanceClient{
		err: errors.New("Guidance sidecar network error"),
	}

	result := svc.Suggest(context.Background(), "MySQL连接失败", "数据库无法访问")

	assert.Equal(t, "database", result.Category)
	assert.Equal(t, 101, result.AssigneeID)
	assert.Greater(t, result.Confidence, 0.5)
}
