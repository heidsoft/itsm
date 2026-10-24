package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	knowledgeHandler "itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/problem"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// POST/GET /api/v1/problems/:id/knowledge 的真实契约（2026-10-22 知识沉淀闭环落地）：
// 正向把问题的现象/根因/解决方案组装为知识草稿（draft，发布须过知识评审流）；
// 反向列出由该问题沉淀的文章（含草稿），补齐调查摘要 KnowledgeArticles 恒空的断链。
//
// 红线：
//   - 无根因且无解决方案的问题沉淀必须 422 拒绝——空壳文章污染 RAG 语料；
//   - source_problem_id 是来源追溯的权威字段，响应必须回显；
//   - 跨租户问题在沉淀与反查两个方向都统一 404，不暴露 ID 可枚举信号；
//   - 知识仓储未装配时沉淀端点显式 503，绝不静默假成功。

func setupProblemKnowledgeRoute(t *testing.T) (*gin.Engine, *ent.Client, *ent.User, *ent.User, string) {
	t.Helper()
	dsn := fmt.Sprintf("file:router_problem_kb_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Kb A").SetCode("kb-a").SetDomain("kb-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Kb B").SetCode("kb-b").SetDomain("kb-b.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("kb-a-admin").SetEmail("kb-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("kb-b-admin").SetEmail("kb-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	kbRepo := knowledgeHandler.NewEntRepository(client)
	probSvc := problem.NewService(problem.NewEntRepository(client), client, nil, logger)
	probSvc.SetKnowledgeRepository(kbRepo)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "problem-kb-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:        secret,
		Logger:           logger,
		Client:           client,
		ProblemHandler:   problem.NewHandler(probSvc),
		KnowledgeHandler: knowledgeHandler.NewHandler(knowledgeHandler.NewService(kbRepo, logger)),
	})
	return r, client, userA, stranger, secret
}

func TestProblemKnowledgeRouteContract(t *testing.T) {
	r, client, userA, stranger, secret := setupProblemKnowledgeRoute(t)
	ctx := context.Background()

	do := func(t *testing.T, method, path, body string, user *ent.User) (int, problemEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, secret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var env problemEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, env
	}

	// tenant A 两个问题：一个带根因（可沉淀），一个只有现象（必须 422）。
	createProblem := func(t *testing.T, title, rootCause string) int {
		t.Helper()
		body := fmt.Sprintf(`{"title":%q,"description":"沉淀闭环契约验证用问题描述","priority":"medium","rootCause":%q}`, title, rootCause)
		code, env := do(t, http.MethodPost, "/api/v1/problems", body, userA)
		require.Equal(t, http.StatusOK, code, "body=%s", string(env.Data))
		// env.Data 已是 data 内容本身，不能再包一层 Data（envelope 二次解包陷阱）。
		var created struct {
			ID            int    `json:"id"`
			ProblemNumber string `json:"problemNumber"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created))
		return created.ID
	}
	problemWithRootCause := createProblem(t, "连接池耗尽导致服务雪崩", "连接池未设上限，慢查询堆积耗尽连接")
	problemBare := createProblem(t, "现象未定位的问题", "")

	t.Run("带根因的问题沉淀成功：draft+来源回显+内容分段", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/problems/%d/knowledge", problemWithRootCause)
		w, env := do(t, http.MethodPost, path, `{}`, userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))

		var article struct {
			ID              int      `json:"id"`
			Title           string   `json:"title"`
			Category        string   `json:"category"`
			Status          string   `json:"status"`
			SourceProblemID *int     `json:"sourceProblemId"`
			Tags            []string `json:"tags"`
			Content         string   `json:"content"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &article))
		require.NotNil(t, article.SourceProblemID, "来源追溯字段必须回显")
		assert.Equal(t, problemWithRootCause, *article.SourceProblemID)
		assert.Equal(t, "draft", article.Status, "沉淀产物是草稿，发布须过评审流")
		assert.Equal(t, "连接池耗尽导致服务雪崩", article.Title, "空 title 兜底为问题标题")
		assert.Equal(t, "故障排查", article.Category, "空 category 兜底为故障排查")
		assert.Contains(t, article.Tags, "问题沉淀")
		assert.Contains(t, article.Content, "## 根因分析")
		assert.Contains(t, article.Content, "连接池未设上限")

		// 沉淀产物真实落库，且反查端点能看到（含草稿）。
		count, err := client.KnowledgeArticle.Query().Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		wList, envList := do(t, http.MethodGet, path, "", userA)
		require.Equal(t, http.StatusOK, wList, "body=%s", string(envList.Data))
		var list struct {
			Items []struct {
				ID              int  `json:"id"`
				SourceProblemID *int `json:"sourceProblemId"`
			} `json:"items"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(envList.Data, &list))
		assert.Equal(t, 1, list.Total)
		require.Len(t, list.Items, 1)
		require.NotNil(t, list.Items[0].SourceProblemID)
		assert.Equal(t, problemWithRootCause, *list.Items[0].SourceProblemID)
	})

	t.Run("无根因且无解决方案的问题沉淀 422 拒绝", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/problems/%d/knowledge", problemBare)
		w, env := do(t, http.MethodPost, path, `{}`, userA)
		assert.Equal(t, http.StatusUnprocessableEntity, w, "body=%s", string(env.Data))
		assert.Equal(t, 4220, env.Code, "质量门用 UnprocessableEntity，不是参数校验错误")
	})

	t.Run("空列表也是 JSON 数组而非 null", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/problems/%d/knowledge", problemBare)
		w, env := do(t, http.MethodGet, path, "", userA)
		require.Equal(t, http.StatusOK, w)
		assert.Contains(t, string(env.Data), `"items":[]`, "items 必须是空数组，前端不做 null 防御")
	})

	t.Run("跨租户与不存在：沉淀与反查都统一 404/4004", func(t *testing.T) {
		// stranger(B) 探 tenant A 的问题。
		pathA := fmt.Sprintf("/api/v1/problems/%d/knowledge", problemWithRootCause)
		w, env := do(t, http.MethodPost, pathA, `{}`, stranger)
		assert.Equal(t, http.StatusNotFound, w, "跨租户不得沉淀他租问题")
		assert.Equal(t, 4004, env.Code)
		w, env = do(t, http.MethodGet, pathA, "", stranger)
		assert.Equal(t, http.StatusNotFound, w, "跨租户不得反查他租沉淀")
		assert.Equal(t, 4004, env.Code)

		w, env = do(t, http.MethodPost, "/api/v1/problems/987654/knowledge", `{}`, userA)
		assert.Equal(t, http.StatusNotFound, w)
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("知识仓储未装配时沉淀端点显式 503，不静默假成功", func(t *testing.T) {
		bare := gin.New()
		probSvcNoKb := problem.NewService(problem.NewEntRepository(client), client, nil, zaptest.NewLogger(t).Sugar())
		// 裸路由 + 注入鉴权上下文：本场景只验证「装配缺失」分支，
		// 不复刻完整 JWT/权限栈（那是其他用例的职责）。
		bare.Use(func(c *gin.Context) {
			c.Set("user_id", userA.ID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: userA.TenantID})
			c.Next()
		})
		bare.POST("/api/v1/problems/:id/knowledge", problem.NewHandler(probSvcNoKb).PromoteProblemKnowledge)

		token, err := middleware.GenerateAccessToken(userA.ID, userA.Username, string(userA.Role), userA.TenantID, secret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/problems/%d/knowledge", problemWithRootCause), strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		bare.ServeHTTP(w, req)

		var env problemEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		assert.Equal(t, http.StatusServiceUnavailable, w.Code, "body=%s", w.Body.String())
		assert.Equal(t, 5003, env.Code)
	})
}
