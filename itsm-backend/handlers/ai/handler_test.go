package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/handlers/ai"
	"itsm-backend/service"
)

type mockTriageLLMProvider struct {
	response string
	err      error
}

func (m mockTriageLLMProvider) Chat(ctx context.Context, model string, messages []service.LLMMessage) (string, error) {
	return m.response, m.err
}

func setupTriageHandlerRouter(triageService *service.TriageService, withTenant bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, triageService, nil, nil)
	h := ai.NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/ai/triage", func(c *gin.Context) {
		if withTenant {
			c.Set("tenant_id", 1)
		}
		h.Triage(c)
	})
	return r
}

// TestCreateTicketByAI_Handler 验证 CreateTicketByAI Handler 接口响应
func TestCreateTicketByAI_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 使用构造函数初始化 Service（Logger 通过参数传入，避免访问未导出字段）
	svc := ai.NewService(nil, zap.L().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	r := gin.New()
	r.POST("/api/v1/ai/ticket/create", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		h.CreateTicketByAI(c)
	})

	body := map[string]interface{}{
		"description": "我的网络无法连接",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/ticket/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 期望返回 200（Service 没有 triageService，会走 fallback 分支）
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "draft", resp["status"])
}

// TestSummarizeTicketPost_RouteExists 验证 SummarizeTicketPost 方法存在
func TestSummarizeTicketPost_RouteExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := ai.NewService(nil, zap.L().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	// 验证方法存在（编译期保证）
	assert.NotNil(t, h.SummarizeTicketPost)
	assert.NotNil(t, h.SummarizeTicket)
	assert.NotNil(t, h.CreateTicketByAI)
}

func TestTriage_Handler_NormalizesLLMResultContract(t *testing.T) {
	gateway := service.NewLLMGateway(mockTriageLLMProvider{
		response: `{"category":"","priority":"urgent","assignee_id":999,"confidence":1.4,"explanation":"invalid enum from model"}`,
	}, nil, nil, "test")
	triageService := service.NewTriageService(gateway, zap.NewNop())
	router := setupTriageHandlerRouter(triageService, true)

	body := bytes.NewBufferString(`{"title":"需要人工判断","description":"模型返回了异常枚举"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Title       string                 `json:"title"`
			Description string                 `json:"description"`
			Suggestions map[string]interface{} `json:"suggestions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, "需要人工判断", resp.Data.Title)
	assert.Equal(t, "模型返回了异常枚举", resp.Data.Description)
	assert.Equal(t, "general", resp.Data.Suggestions["category"])
	assert.Equal(t, "medium", resp.Data.Suggestions["priority"])
	assert.Equal(t, float64(0.6), resp.Data.Suggestions["confidence"])
	assert.Equal(t, "invalid enum from model", resp.Data.Suggestions["reasoning"])
	assert.Equal(t, "medium", resp.Data.Suggestions["urgency"])
}

func TestTriage_Handler_RequiresTenant(t *testing.T) {
	router := setupTriageHandlerRouter(nil, false)

	body := bytes.NewBufferString(`{"title":"需要人工判断","description":"缺少租户上下文"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, common.AuthFailedCode, resp.Code)
	assert.Equal(t, "租户信息缺失", resp.Message)
}

func TestTriage_Handler_RequiresTitle(t *testing.T) {
	router := setupTriageHandlerRouter(nil, true)

	body := bytes.NewBufferString(`{"description":"缺少标题"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, common.ParamErrorCode, resp.Code)
	assert.Contains(t, resp.Message, "Title")
}
