package probleminvestigation

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestHandler(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// 每个测试用独立命名内存库，避免 cache=shared 跨测试状态泄漏
	dbName := "file:pi_test_" + t.Name() + "?mode=memory&_fk=1"

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", dbName)

	// 打开单独的 sql.DB 连接用于 ProblemInvestigationService
	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	invService := service.NewProblemInvestigationService(db, logger)
	h := NewHandler(logger, invService, client)

	r := gin.New()
	r.Use(gin.Recovery())

	// Tenant/user 上下文注入（模拟中间件）
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tenantID = v
			}
		}
		userID := 1
		if h := c.GetHeader("X-Test-User"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				userID = v
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})

	// 注册路由 - mirror router.go 契约 /problem-investigation/*。
	r.POST("/api/v1/problem-investigation/investigations", h.CreateProblemInvestigation)
	r.GET("/api/v1/problem-investigation/investigations/:id", h.GetProblemInvestigation)
	r.PUT("/api/v1/problem-investigation/investigations/:id", h.UpdateProblemInvestigation)
	r.POST("/api/v1/problem-investigation/steps", h.CreateInvestigationStep)
	r.PUT("/api/v1/problem-investigation/steps/:id", h.UpdateInvestigationStep)
	r.GET("/api/v1/problem-investigation/investigations/:id/steps", h.GetInvestigationSteps)
	r.POST("/api/v1/problem-investigation/root-cause-analysis", h.CreateRootCauseAnalysis)
	r.PUT("/api/v1/problem-investigation/root-cause-analysis/:id", h.UpdateRootCauseAnalysis)
	r.POST("/api/v1/problem-investigation/solutions", h.CreateProblemSolution)
	r.PUT("/api/v1/problem-investigation/solutions/:id", h.UpdateProblemSolution)
	r.GET("/api/v1/problem-investigation/problems/:id/solutions", h.GetProblemSolutions)
	r.GET("/api/v1/problem-investigation/problems/:id/summary", h.GetProblemInvestigationSummary)
	r.POST("/api/v1/problem-relationships", h.CreateProblemRelationship)
	r.GET("/api/v1/problems/:id/relationships", h.GetProblemRelationships)
	r.POST("/api/v1/problem-knowledge-articles", h.CreateKnowledgeArticle)
	r.GET("/api/v1/problem-knowledge-articles/problems/:id", h.GetProblemKnowledgeArticles)

	return r
}

func TestHandler_CreateProblemInvestigation_BadRequest(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"findings":"missing problem_id"}`)
	req, err := http.NewRequest("POST", "/api/v1/problem-investigation/investigations", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_GetProblemInvestigation_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/problem-investigation/investigations/99999", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "service 层对不存在记录返回错误，HTTP 映射 500（与旧契约一致）")
}

func TestHandler_GetProblemInvestigation_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/problem-investigation/investigations/invalid", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateProblemInvestigation_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"findings":"updated"}`)
	req, err := http.NewRequest("PUT", "/api/v1/problem-investigation/investigations/99999", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateProblemInvestigation_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"findings":"test"}`)
	req, err := http.NewRequest("PUT", "/api/v1/problem-investigation/investigations/invalid", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateProblemRelationship_StubContract(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"problemId":1,"relatedType":"incident","relatedId":5,"relationshipType":"caused_by"}`)
	req, err := http.NewRequest("POST", "/api/v1/problem-relationships", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Message     string `json:"message"`
			ProblemID   int    `json:"problemId"`
			RelatedType string `json:"relatedType"`
			RelatedID   int    `json:"relatedId"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, 1, resp.Data.ProblemID)
	assert.Equal(t, "incident", resp.Data.RelatedType)
	assert.Equal(t, 5, resp.Data.RelatedID)
}

func TestHandler_GetProblemRelationships_EmptyList(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/problems/1/relationships", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ProblemID     int           `json:"problemId"`
			Relationships []interface{} `json:"relationships"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Empty(t, resp.Data.Relationships)
}

func TestHandler_CreateKnowledgeArticle(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"problemId":1,"articleTitle":"DNS 故障排查手册","articleContent":"步骤一…","articleType":"troubleshooting","tags":["dns","network"]}`)
	req, err := http.NewRequest("POST", "/api/v1/problem-knowledge-articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ArticleID int `json:"articleId"`
			Article   struct {
				Title    string `json:"title"`
				Category string `json:"category"`
				Tags     string `json:"tags"`
				AuthorID int    `json:"authorId"`
			} `json:"article"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Greater(t, resp.Data.ArticleID, 0)
	assert.Equal(t, "DNS 故障排查手册", resp.Data.Article.Title)
	assert.Equal(t, "dns,network", resp.Data.Article.Tags)
	assert.Equal(t, 1, resp.Data.Article.AuthorID, "未指定作者时应回填当前用户")
}

func TestHandler_GetProblemKnowledgeArticles(t *testing.T) {
	r := setupTestHandler(t)

	// 先创建一篇文章
	body := []byte(`{"problemId":1,"articleTitle":"文章A","articleContent":"内容","articleType":"runbook"}`)
	req, err := http.NewRequest("POST", "/api/v1/problem-knowledge-articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// 再查询列表
	req2, err := http.NewRequest("GET", "/api/v1/problem-knowledge-articles/problems/1", nil)
	require.NoError(t, err)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "body=%s", w2.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ProblemID         int           `json:"problemId"`
			KnowledgeArticles []interface{} `json:"knowledgeArticles"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data.KnowledgeArticles, 1)
}

func TestHandler_MissingTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:pi_notenant?mode=memory&_fk=1")
	db, err := sql.Open("sqlite3", "file:pi_notenant?mode=memory&_fk=1")
	require.NoError(t, err)
	logger := zaptest.NewLogger(t).Sugar()
	invService := service.NewProblemInvestigationService(db, logger)
	h := NewHandler(logger, invService, client)

	r := gin.New()
	r.Use(gin.Recovery())
	// 不注入 tenant_id → RequireTenantID 应返回 401
	r.GET("/api/v1/problem-investigation/investigations/:id", h.GetProblemInvestigation)

	req, err := http.NewRequest("GET", "/api/v1/problem-investigation/investigations/1", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "缺 tenant 上下文应 401（handlerctx 契约）")
}

var _ = ent.Desc // 保持 import（Order(ent.Desc) 在 routes.go 中使用）
