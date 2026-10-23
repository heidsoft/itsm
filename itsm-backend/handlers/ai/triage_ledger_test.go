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
	"itsm-backend/dto"
	"itsm-backend/handlers/ai"
	"itsm-backend/service"
)

// ledgerCall 记录一次 RecordDecisionAudit 调用的入参。
type ledgerCall struct {
	TenantID      int
	UserID        int
	Scenario      string
	InputRef      string
	PromptVersion string
	Model         string
	Confidence    float64
	Suggestion    map[string]interface{}
}

// mockAuditLedger 服务端 AI 决策底账的测试替身：记录调用并返回固定行 ID。
type mockAuditLedger struct {
	calls []ledgerCall
	id    int64
	err   error
}

func (m *mockAuditLedger) RecordDecisionAudit(_ context.Context, tenantID, userID int, _, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error) {
	m.calls = append(m.calls, ledgerCall{
		TenantID: tenantID, UserID: userID, Scenario: scenario, InputRef: inputRef,
		PromptVersion: promptVersion, Model: model, Confidence: confidence, Suggestion: suggestion,
	})
	if m.err != nil {
		return 0, m.err
	}
	return m.id, nil
}

// setupTriageRouterWithLedger 构造带底账写口的分诊路由（keyword 链路，无需 LLM）。
func setupTriageRouterWithLedger(t *testing.T, ledger ai.AuditLedgerWriter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	triageService := service.NewTriageService(nil, zap.NewNop())
	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, triageService, nil, nil)
	if ledger != nil {
		svc.SetAuditLedger(ledger)
	}
	h := ai.NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/ai/triage", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 7)
		h.Triage(c)
	})
	return r
}

// 分诊产出建议时，服务端自动落一条 AI 决策底账（scenario=triage、溯源字段真实、
// confidence 与建议一致），响应携带 auditId + promptVersion + model。
// 这是反馈闭环的服务端底座：操作者采纳/拒绝直接 PATCH /ai/audit-logs/:id，
// 不再依赖前端伪造 model 二次 POST 自报。
func TestTriage_ServerSideAuditLedger(t *testing.T) {
	ledger := &mockAuditLedger{id: 42}
	router := setupTriageRouterWithLedger(t, ledger)

	body := bytes.NewBufferString(`{"title":"数据库连接超时","description":"应用无法连接 PostgreSQL"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AuditID       int64                  `json:"auditId"`
			PromptVersion string                 `json:"promptVersion"`
			Model         string                 `json:"model"`
			Suggestions   map[string]interface{} `json:"suggestions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, common.SuccessCode, resp.Code)

	// 响应携带底账 ID 与溯源字段
	assert.Equal(t, int64(42), resp.Data.AuditID)
	assert.Equal(t, "triage-v1", resp.Data.PromptVersion)
	assert.Contains(t, []string{"keyword", "guidance", "llm"}, resp.Data.Model)

	// 底账入参：真实操作者、真实溯源、与建议一致的置信度
	require.Len(t, ledger.calls, 1)
	call := ledger.calls[0]
	assert.Equal(t, 1, call.TenantID)
	assert.Equal(t, 7, call.UserID)
	assert.Equal(t, "triage", call.Scenario)
	assert.Contains(t, call.InputRef, "ticket:")
	assert.Equal(t, "triage-v1", call.PromptVersion)
	assert.InDelta(t, call.Confidence, resp.Data.Suggestions["confidence"].(float64), 0.0001)
	assert.NotContains(t, []string{"", "triage"}, call.Model, "底账 model 必须是真实决策链路，不得是前端伪造的 'triage'")
	assert.Contains(t, call.Suggestion, "category")
	assert.Contains(t, call.Suggestion, "priority")
}

// 底账写失败不阻塞建议返回（best-effort），只是响应不携带 auditId。
func TestTriage_LedgerFailureDoesNotBlockSuggestion(t *testing.T) {
	ledger := &mockAuditLedger{id: 42, err: assert.AnError}
	router := setupTriageRouterWithLedger(t, ledger)

	body := bytes.NewBufferString(`{"title":"打印机故障","description":"无法打印"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, common.SuccessCode, resp.Code)
	assert.Contains(t, resp.Data, "suggestions")
	assert.NotContains(t, resp.Data, "auditId")
}

// 未装配底账（如单测环境）时分诊照常工作，响应不携带 auditId。
func TestTriage_NoLedger_OmitsAuditId(t *testing.T) {
	router := setupTriageRouterWithLedger(t, nil)

	body := bytes.NewBufferString(`{"title":"邮箱无法登录","description":"密码错误锁定"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/triage", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotContains(t, resp.Data, "auditId")
	assert.Contains(t, resp.Data, "suggestions")
}

// POST /ai/audit 的 confidence 契约统一 0-1：越界/NaN 一律 400 ValidationError，
// 不再静默 clamp——非法值进入 evaluator 置信度校准分桶会把桶污染成"低置信"。
func TestRecordAudit_ConfidenceRangeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)
	r := gin.New()
	r.POST("/api/v1/ai/audit", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 7)
		h.RecordAudit(c)
	})

	cases := []struct {
		name       string
		confidence string
	}{
		{"负值", "-0.1"},
		{"超上界", "1.5"},
		{"远超上界", "100"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"scenario":"triage","inputRef":"ticket:1","confidence":` + tc.confidence + `,"suggestion":{"category":"general"}}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/audit", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			var resp struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, common.ValidationError, resp.Code)
		})
	}

	// 边界 0 与 1 合法：校验通过（后续落库依赖遥测服务，这里只断言不是校验失败）
	for _, boundary := range []string{"0", "1"} {
		payload := `{"scenario":"triage","inputRef":"ticket:1","confidence":` + boundary + `,"suggestion":{"category":"general"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/audit", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var resp struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEqual(t, common.ValidationError, resp.Code, "confidence=%s 应通过区间校验", boundary)
	}
}

// aiResultTrace 从 RCA/事件分析响应提取置信度与 prompt 版本（供 ai_analysis_results 持久化）。
func TestAIResultTrace(t *testing.T) {
	ticket := &dto.TicketRootCauseAnalysisResponse{ConfidenceScore: 0.82, PromptVersion: "ticket-rca-v1"}
	conf, ver, ok := ai.ExposeResultTraceForTest(ticket)
	require.True(t, ok)
	assert.InDelta(t, 0.82, conf, 0.0001)
	assert.Equal(t, "ticket-rca-v1", ver)

	incident := &dto.IncidentAIAnalysisResponse{ConfidenceScore: 0.65, PromptVersion: "incident-rca-v1"}
	conf, ver, ok = ai.ExposeResultTraceForTest(incident)
	require.True(t, ok)
	assert.InDelta(t, 0.65, conf, 0.0001)
	assert.Equal(t, "incident-rca-v1", ver)

	_, _, ok = ai.ExposeResultTraceForTest(map[string]interface{}{"confidenceScore": 0.9})
	assert.False(t, ok, "未知类型不得伪造置信度语义")
}
