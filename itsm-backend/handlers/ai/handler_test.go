package ai_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"itsm-backend/handlers/ai"
)

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
