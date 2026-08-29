package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
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

// TestSummarizeTicket_RouteExists 验证 SummarizeTicket 方法存在
func TestSummarizeTicket_RouteExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := ai.NewService(nil, zap.L().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	// 验证方法存在（编译期保证）
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

func TestAnalyzeIncident_Handler_UnwiredServiceReturnsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := ai.NewService(nil, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)
	r := gin.New()
	r.POST("/api/v1/ai/incidents/:id/analyze", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		h.AnalyzeIncident(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/incidents/1/analyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.ServiceUnavailableCode, resp.Code)
	assert.Equal(t, "AI 事件分析服务尚未就绪", resp.Message)
}

// ==================== RAG 可见性过滤 HTTP 端到端测试 ====================
// 完整链路：gin 路由 → Handler.KnowledgeSearch → Service.SearchKnowledge →
// RAGService.Ask → Ent 查询。验证租户隔离、草稿排除、软删除排除在 HTTP 边界生效。

func setupKnowledgeSearchRouter(client *ent.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)

	rag := service.NewRAGService(client, nil, nil, zap.NewNop().Sugar(), service.RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   10,
	})
	svc := ai.NewService(nil, zap.NewNop().Sugar(), rag, nil, nil, nil, nil, nil, nil, nil, nil)
	h := ai.NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	// 中间件从 header 读取目标租户（模拟 RequirePermission 之后的 tenant_id）
	r.Use(func(c *gin.Context) {
		if tid, err := parseIntHeader(c.GetHeader("X-Test-Tenant")); err == nil {
			c.Set("req_tenant", tid)
		}
	})
	r.POST("/api/v1/ai/rag/search", func(c *gin.Context) {
		c.Set("tenant_id", c.GetInt("req_tenant"))
		h.KnowledgeSearch(c)
	})
	return r
}

func parseIntHeader(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// 端到端：租户1 检索只能看到本租户已发布文章，草稿与软删除文章不可见。
func TestKnowledgeSearch_VisibilityEndToEnd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:rag_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	defer client.Close()
	ctx := context.Background()

	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").SetCode("rag-http-1").SetDomain("t1.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").SetCode("rag-http-2").SetDomain("t2.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user1, err := client.User.Create().
		SetUsername("rag-http-u1").SetEmail("http1@test.com").SetName("U1").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)
	user2, err := client.User.Create().
		SetUsername("rag-http-u2").SetEmail("http2@test.com").SetName("U2").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	createArticle := func(tenantID, authorID int, title string, published bool) *ent.KnowledgeArticle {
		t.Helper()
		a, err := client.KnowledgeArticle.Create().
			SetTitle(title).SetContent("内容：" + title).SetCategory("故障处理").
			SetAuthorID(authorID).SetTenantID(tenantID).SetIsPublished(published).
			Save(ctx)
		require.NoError(t, err)
		return a
	}

	// 租户1：已发布 + 草稿 + 软删除（已发布后删除）
	pub1 := createArticle(tenant1.ID, user1.ID, "邮箱无法发送", true)
	createArticle(tenant1.ID, user1.ID, "邮箱草稿", false)
	del1 := createArticle(tenant1.ID, user1.ID, "邮箱旧文（已删）", true)
	_, err = client.KnowledgeArticle.UpdateOneID(del1.ID).SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	// 租户2：已发布同主题文章
	createArticle(tenant2.ID, user2.ID, "邮箱无法发送", true)

	router := setupKnowledgeSearchRouter(client)

	post := func(tenantID int, body string) (int, map[string]interface{}) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/rag/search", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Tenant", fmt.Sprintf("%d", tenantID))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return w.Code, resp
	}

	code, resp := post(tenant1.ID, `{"query":"邮箱","limit":10,"type":"kb"}`)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, float64(0), resp["code"], "响应码应为 0: %v", resp)

	data := resp["data"].(map[string]interface{})
	require.Equal(t, false, data["degraded"])
	results := data["results"].([]interface{})
	require.Len(t, results, 1, "租户1 只应看到自己的已发布文章")

	first := results[0].(map[string]interface{})
	assert.Equal(t, float64(pub1.ID), first["id"])
	assert.Equal(t, "邮箱无法发送", first["title"])
	assert.Equal(t, "kb", first["object_type"])
}

// 端到端：跨租户隔离——租户2 检索不到租户1 的任何文章。
func TestKnowledgeSearch_TenantIsolationEndToEnd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:rag_http_iso_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	defer client.Close()
	ctx := context.Background()

	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").SetCode("rag-http-i1").SetDomain("t1.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").SetCode("rag-http-i2").SetDomain("t2.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user1, err := client.User.Create().
		SetUsername("rag-http-i1").SetEmail("i1@test.com").SetName("U1").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// 租户1 有一篇完全匹配的文章，租户2 没有
	_, err = client.KnowledgeArticle.Create().
		SetTitle("内网DNS解析故障").SetContent("DNS 配置排查步骤").SetCategory("网络").
		SetAuthorID(user1.ID).SetTenantID(tenant1.ID).SetIsPublished(true).
		Save(ctx)
	require.NoError(t, err)

	router := setupKnowledgeSearchRouter(client)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/rag/search", bytes.NewBufferString(`{"query":"DNS","limit":10,"type":"kb"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", fmt.Sprintf("%d", tenant2.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Results  []interface{} `json:"results"`
			Degraded bool          `json:"degraded"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Empty(t, resp.Data.Results, "租户2 不得看到租户1 的文章")
	assert.False(t, resp.Data.Degraded)
}
