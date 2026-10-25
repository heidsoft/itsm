package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 未匹配路径/方法必须返回与全站一致的 JSON 信封，而不是 gin 默认的
// text/plain "404 page not found"。
//
// 背景：客户端（前端 httpClient / itsm-cli / MCP）统一按 {code,message,data}
// 解析响应。纯文本 404 会让它们走进"响应结构变更"分支，把「接口不存在」
// 误报成解析失败——与 CHANGELOG 中飞书 doJSON 不判状态码、把 401/429
// 伪装成 decode 失败是同一类问题。
//
// 本测试锁三件事：
//  1. 未知路径 → HTTP 404 + code 4004 + 可解析 JSON
//  2. 已知路径用错动词 → HTTP 405 + code 4050（而不是同样伪装成 404）
//  3. 业务码与 HTTP 状态双向映射唯一（4004↔404、4050↔405 不串位）
func TestSetupRoutes_UnmatchedRouteReturnsJSONEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:router_noroute?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret: "noroute-contract-secret",
		Logger:    logger,
		Client:    client,
	})

	// --- 1. 未知路径：404 + 4004 + JSON ---
	req := httptest.NewRequest(http.MethodGet, "/api/v1/__definitely_not_a_route__", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "未知路径必须返回 HTTP 404")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json",
		"未知路径必须返回 JSON，不能是 text/plain")

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body),
		"响应体必须可被 JSON 解析，实际: %q", w.Body.String())
	assert.Equal(t, 4004, body.Code, "业务码应为 4004（资源/接口不存在）")
	assert.NotEmpty(t, body.Message, "message 不应为空，调用方需要可读原因")

	// --- 2. 已知路径 + 错误动词：405 + 4050 ---
	// /api/v1/health 只注册了 GET；POST 应命中 NoMethod 而不是 NoRoute。
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusMethodNotAllowed, w2.Code,
		"路径存在但方法不允许时应返回 405，而不是伪装成 404")
	assert.Contains(t, w2.Header().Get("Content-Type"), "application/json")

	var body2 struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body2),
		"405 响应体必须可被 JSON 解析，实际: %q", w2.Body.String())
	assert.Equal(t, 4050, body2.Code, "业务码应为 4050（方法不允许）")
	assert.NotEmpty(t, body2.Message)
}
