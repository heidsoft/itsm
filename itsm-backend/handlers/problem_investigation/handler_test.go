package problem_investigation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestHandler(t *testing.T) *gin.Engine {
	r, _, _ := setupTestHandlerWithService(t)
	return r
}

func setupTestHandlerWithService(t *testing.T) (*gin.Engine, *ent.Client, *service.ProblemInvestigationService) {
	gin.SetMode(gin.TestMode)

	// 每个测试用独立命名内存库；cache=shared 让 ent 客户端与 raw sql.DB 看到同一份数据
	// （否则两条连接各自打开一个空的内存库），独立命名保证测试之间不共享状态。
	dbName := "file:pi_test_" + t.Name() + "?mode=memory&cache=shared&_fk=1"

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", dbName)

	// 打开单独的 sql.DB 连接用于 ProblemInvestigationService
	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	invService := service.NewProblemInvestigationService(db, client, logger)
	h := NewHandler(logger, invService)

	r := gin.New()
	r.Use(gin.Recovery())

	// Tenant/user/role 上下文注入（模拟中间件）
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
		role := "admin"
		if h := c.GetHeader("X-Test-Role"); h != "" {
			role = h
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", role)
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

	return r, client, invService
}

// createTestProblem 造一条可被知识沉淀引用的问题单（raw SQL 仓储按 tenant_id 校验归属）。
func createTestProblem(t *testing.T, ctx context.Context, client *ent.Client, tenantID, createdBy int) *ent.Problem {
	t.Helper()
	now := time.Now()
	p, err := client.Problem.Create().
		SetTitle("磁盘告警排查").
		SetDescription("磁盘使用率持续告警").
		SetStatus("open").
		SetPriority("medium").
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	require.NoError(t, err)
	return p
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
	r, client, _ := setupTestHandlerWithService(t)
	ctx := context.Background()
	problem := createTestProblem(t, ctx, client, 1, 1)

	body := []byte(fmt.Sprintf(`{"problemId":%d,"articleTitle":"DNS 故障排查手册","articleContent":"步骤一…","articleType":"troubleshooting","tags":["dns","network"]}`, problem.ID))
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

	// 来源追溯：problemId 是必填项，必须落到 source_problem_id，否则沉淀文章
	// 不会出现在问题反查与调查摘要里。
	article, err := client.KnowledgeArticle.Query().
		Where(knowledgearticle.IDEQ(resp.Data.ArticleID)).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, article.SourceProblemID)
	assert.Equal(t, problem.ID, *article.SourceProblemID)
}

// 知识文章不能成为反查他人问题标题的通道：跨租户 problemId 与不存在同样返回 404，
// 且不得落库。
func TestHandler_CreateKnowledgeArticle_RejectsCrossTenantProblem(t *testing.T) {
	r, client, _ := setupTestHandlerWithService(t)
	ctx := context.Background()
	otherTenantProblem := createTestProblem(t, ctx, client, 2, 9)

	body := []byte(fmt.Sprintf(`{"problemId":%d,"articleTitle":"越权文章","articleContent":"内容","articleType":"runbook"}`, otherTenantProblem.ID))
	req, err := http.NewRequest("POST", "/api/v1/problem-knowledge-articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", "1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 4004, resp.Code)

	count, err := client.KnowledgeArticle.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "跨租户引用被拒时不得留下文章")
}

func TestHandler_MissingTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:pi_notenant?mode=memory&_fk=1")
	db, err := sql.Open("sqlite3", "file:pi_notenant?mode=memory&_fk=1")
	require.NoError(t, err)
	logger := zaptest.NewLogger(t).Sugar()
	invService := service.NewProblemInvestigationService(db, client, logger)
	h := NewHandler(logger, invService)

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
