package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestLegacyProblemKnowledgeArticleListRouteRetired 锁死已退役的
// GET /api/v1/problem-knowledge-articles/problems/:id（2026-10-31）。
//
// 该端点忽略路径里的 problemID，返回租户内全部知识文章，既不表达「由该问题沉淀」，
// 也不校验问题可见性；正常调用方为 0（前端唯一消费者已删除），
// 反查统一由 GET /api/v1/problems/:id/knowledge 承担（按 source_problem_id 过滤 + 问题可见性）。
// 若有人重新注册回来，这里必须变红。
func TestLegacyProblemKnowledgeArticleListRouteRetired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// handler 不会被调用，nil 足以拿到路由表。
	SetupProblemInvestigationRoutes(r.Group("/api/v1"), nil)

	routes := make([]string, 0, len(r.Routes()))
	for _, rt := range r.Routes() {
		routes = append(routes, rt.Method+" "+rt.Path)
	}

	assert.NotContains(t, routes, "GET /api/v1/problem-knowledge-articles/problems/:id",
		"知识反查只保留 GET /api/v1/problems/:id/knowledge")
	assert.Contains(t, routes, "POST /api/v1/problem-knowledge-articles",
		"调查页的知识沉淀入口仍需保留")
}
