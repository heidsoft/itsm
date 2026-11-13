package router

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/problem_investigation"
	"itsm-backend/internal/services"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 问题关联的单一所有者是 problem 域 /api/v1/problems/:id/associations（真实 Ent 边，
// 已校验租户）。调查域曾有一条假重复路径：POST /api/v1/problem-relationships 记日志即
// 返回成功却不落库，GET /api/v1/problems/:id/relationships 恒返回空列表。两条桩已退役。
//
// 回归：删桩前这两条假路由仍挂在路由树上（同时装配 Problem 与 ProblemInvestigation
// handler 即可复现），下面的 NotContains 断言会红；删桩后转绿，且真实 associations
// 三个方法必须仍在，证明收敛没有误删唯一所有者。
func TestProblemRelationshipsStubRetired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:router_problem_rel_retire_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	probSvc := problem.NewService(problem.NewEntRepository(client), client, nil, logger)
	invSvc := service.NewProblemInvestigationService(db, client, logger)

	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:                   "problem-rel-retire-secret",
		Logger:                      logger,
		Client:                      client,
		ProblemHandler:              problem.NewHandler(probSvc),
		ProblemInvestigationHandler: problem_investigation.NewHandler(logger, invSvc),
	})

	hasRoute := func(method, path string) bool {
		for _, ri := range r.Routes() {
			if ri.Method == method && ri.Path == path {
				return true
			}
		}
		return false
	}

	assert.False(t, hasRoute(http.MethodPost, "/api/v1/problem-relationships"),
		"假成功写桩必须退役，问题关联写入唯一入口是 /problems/:id/associations")
	assert.False(t, hasRoute(http.MethodGet, "/api/v1/problems/:id/relationships"),
		"恒空读桩必须退役，问题关联读取唯一入口是 /problems/:id/associations")

	assert.True(t, hasRoute(http.MethodGet, "/api/v1/problems/:id/associations"), "真实关联读端点必须保留")
	assert.True(t, hasRoute(http.MethodPost, "/api/v1/problems/:id/associations"), "真实关联写端点必须保留")
	assert.True(t, hasRoute(http.MethodDelete, "/api/v1/problems/:id/associations"), "真实关联删端点必须保留")
}
