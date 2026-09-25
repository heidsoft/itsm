package skill_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/handlers/skill"
	"itsm-backend/service"
)

// doRequest 构造 HTTP 请求并返回 recorder。
func doRequest(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewBuffer(raw)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// responseEnvelope 解析 standard {code, message, data} 响应。
type responseEnvelope struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// parseData 解析响应包络并返回 data 字段。
func parseData(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var env responseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env.Data
}

// setupRouter 构造一个最小可用的 Gin 路由，把 SkillHandler 挂到 /api/v1 前缀。
// 同时关闭 RBAC 中间件（测试只关心 handler 行为；权限边界由 RequirePermission 在生产
// 路径中验证）。
func setupRouter(reg *service.SkillRegistry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	h := skill.NewHandler(reg, zap.NewNop().Sugar())
	v1 := r.Group("/api/v1")
	// 跳过 RequirePermission 中间件，专注于 handler 行为。
	v1.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	// 用专用 group 跳过 RBAC（auth 路径），直接挂 handler 的真实路由。
	markets := v1.Group("/skills")
	{
		markets.GET("", h.List)
		markets.GET("/:code", h.Get)
	}
	admin := v1.Group("/admin/skills")
	{
		admin.GET("", h.List)
		admin.GET("/:code", h.Get)
		admin.POST("", h.Create)
		admin.PUT("/:code", h.Update)
		admin.POST("/:code/promote", h.Promote)
		admin.DELETE("/:code", h.Delete)
		admin.POST("/:code/invoke", h.Invoke)
	}
	return r
}

// 1. 列出空注册表 → 200 + items: []
func TestList_EmptyRegistry(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	w := doRequest(t, r, http.MethodGet, "/api/v1/skills", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	data := parseData(t, w)
	assert.EqualValues(t, 0, data["total"])
}

// 2. 注册一个自定义 Skill → 200
func TestCreate_AndList(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	body := skill.SkillUpsertRequest{
		Code:                "user.demo",
		Version:             "v1",
		Title:               "Demo Skill",
		Description:         "用于测试",
		Category:            "pilot",
		Tags:                []string{"custom", "demo"},
		RequiredPermissions: []string{"skill:read"},
		Capabilities:        []string{"custom.demo"},
		Executor: map[string]interface{}{
			"target": "echo",
		},
	}

	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data := parseData(t, w)
	assert.Equal(t, "user.demo", data["code"])

	// 列表应包含该项
	w2 := doRequest(t, r, http.MethodGet, "/api/v1/skills", nil)
	assert.Equal(t, http.StatusOK, w2.Code)
	data2 := parseData(t, w2)
	assert.EqualValues(t, 1, data2["total"])
}

// 3. 重复注册 → 409 Conflict
func TestCreate_DuplicateReturns409(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	body := skill.SkillUpsertRequest{
		Code:                "user.duplicate",
		Version:             "v1",
		RequiredPermissions: []string{"skill:read"},
	}

	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

// 4. 缺的必填字段 → 400
func TestCreate_RequiresVersion(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	body := skill.SkillUpsertRequest{
		Code:                "user.noversion",
		RequiredPermissions: []string{"skill:read"},
		// 故意把 Version 留空
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5. 缺的 requiredPermissions → 400
func TestCreate_RequiresPermissions(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	body := skill.SkillUpsertRequest{
		Code:    "user.noperms",
		Version: "v1",
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 6. Get 详情 → 200 + manifest
func TestGet_AfterCreate(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.detail",
		Version:             "v1",
		Title:               "详情",
		RequiredPermissions: []string{"skill:read"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := doRequest(t, r, http.MethodGet, "/api/v1/skills/user.detail", nil)
	assert.Equal(t, http.StatusOK, w2.Code)
	data := parseData(t, w2)
	assert.Equal(t, "user.detail", data["code"])
}

// 7. 不存在的 Skill → 404
func TestGet_NotFound(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	w := doRequest(t, r, http.MethodGet, "/api/v1/skills/does.not.exist", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 8. 升级：pilot → ga
func TestPromote_PilotToGa(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.promote",
		Version:             "v1",
		Category:            "pilot",
		RequiredPermissions: []string{"skill:read"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills/user.promote/promote", nil)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 重新拉取，应为 ga
	w3 := doRequest(t, r, http.MethodGet, "/api/v1/skills/user.promote", nil)
	data := parseData(t, w3)
	assert.Equal(t, "ga", data["category"])
}

// 9. 禁用（DELETE）
func TestDelete_DisablesSkill(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.todelete",
		Version:             "v1",
		RequiredPermissions: []string{"skill:read"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := doRequest(t, r, http.MethodDelete, "/api/v1/admin/skills/user.todelete", nil)
	assert.Equal(t, http.StatusOK, w2.Code)

	w3 := doRequest(t, r, http.MethodGet, "/api/v1/skills/user.todelete", nil)
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

// 10. Invoke 成功
func TestInvoke_EchoSkill(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.echo",
		Version:             "v1",
		RequiredPermissions: []string{"skill:read"},
		Executor: map[string]interface{}{
			"target": "echo",
		},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills/user.echo/invoke",
		skill.SkillInvokeRequest{Input: map[string]interface{}{"hello": "world"}})
	assert.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	data := parseData(t, w2)
	assert.Equal(t, "user.echo", data["code"])
	// output != nil
	assert.NotNil(t, data["output"])
}

// 11. Invoke 不存在的 Skill → 404
func TestInvoke_NotFound(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills/missing/invoke", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// 12. Update 路径
func TestUpdate_VersionBump(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.bump",
		Version:             "v1",
		RequiredPermissions: []string{"skill:read"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	update := skill.SkillUpsertRequest{
		Code:                "user.bump",
		Version:             "v2",
		RequiredPermissions: []string{"skill:read"},
	}
	w2 := doRequest(t, r, http.MethodPut, "/api/v1/admin/skills/user.bump", update)
	assert.Equal(t, http.StatusOK, w2.Code)

	w3 := doRequest(t, r, http.MethodGet, "/api/v1/skills/user.bump", nil)
	data := parseData(t, w3)
	assert.Equal(t, "v2", data["version"])
}

// 13. 模糊搜索
func TestList_FilterByQuery(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	for _, code := range []string{"user.alpha", "user.beta"} {
		body := skill.SkillUpsertRequest{
			Code:                code,
			Version:             "v1",
			RequiredPermissions: []string{"skill:read"},
			Description:         code,
		}
		w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := doRequest(t, r, http.MethodGet, "/api/v1/skills?q=alpha", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w)
	assert.EqualValues(t, 1, data["total"])
}

// 14. Admin 列表：覆盖 bug #1 — GET /api/v1/admin/skills 必须存在
//
// 修复前的预期：404（路由未注册）。
// 修复后：返回 200 + items/total/page/pageSize/totalPages 完整列表。
func TestAdminList_ReturnsAllSkills(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	// 准备：先注册一个自定义 skill
	create := skill.SkillUpsertRequest{
		Code:                "user.admin_list",
		Version:             "v1",
		Title:               "Admin List Test",
		Category:            "pilot",
		RequiredPermissions: []string{"skill:read"},
		Capabilities:        []string{"custom.admin_list"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 调用 admin 列表
	w2 := doRequest(t, r, http.MethodGet, "/api/v1/admin/skills", nil)
	assert.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	data := parseData(t, w2)
	// 修复后的契约：{ items, total, page, pageSize, totalPages }
	assert.Contains(t, data, "items", "admin list response must include items")
	assert.Contains(t, data, "total")
	assert.Contains(t, data, "page")
	assert.Contains(t, data, "pageSize")
	assert.Contains(t, data, "totalPages")

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items must be an array")
	assert.GreaterOrEqual(t, len(items), 1, "admin list must include the registered skill")
}

// 15. Admin 详情：覆盖 bug #1 — GET /api/v1/admin/skills/:code 必须存在
func TestAdminGet_ReturnsFullEntry(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	create := skill.SkillUpsertRequest{
		Code:                "user.admin_detail",
		Version:             "v1",
		Title:               "Admin Detail",
		Category:            "pilot",
		RequiredPermissions: []string{"skill:read"},
		Capabilities:        []string{"custom.admin_detail"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code)

	// 修复前：404
	// 修复后：返回完整 entry
	w2 := doRequest(t, r, http.MethodGet, "/api/v1/admin/skills/user.admin_detail", nil)
	assert.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	data := parseData(t, w2)
	assert.Equal(t, "user.admin_detail", data["code"])
	assert.Equal(t, "v1", data["version"])
}

// 16. Admin 列表支持 page/pageSize 分页
func TestAdminList_Pagination(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	// 注册 3 个 skill
	for i := 0; i < 3; i++ {
		body := skill.SkillUpsertRequest{
			Code:                fmt.Sprintf("user.page_%d", i),
			Version:             "v1",
			RequiredPermissions: []string{"skill:read"},
			Capabilities:        []string{"custom.pagination"},
		}
		w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", body)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// pageSize=2
	w := doRequest(t, r, http.MethodGet, "/api/v1/admin/skills?page=1&pageSize=2", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w)
	assert.EqualValues(t, 2, data["pageSize"])
	assert.EqualValues(t, 1, data["page"])
	items, _ := data["items"].([]interface{})
	assert.Len(t, items, 2)
	assert.EqualValues(t, 3, data["total"])
	assert.EqualValues(t, 2, data["totalPages"])
}

// 17. Admin 详情找不到 → 404
func TestAdminGet_NotFound(t *testing.T) {
	reg := service.NewSkillRegistry()
	r := setupRouter(reg)

	w := doRequest(t, r, http.MethodGet, "/api/v1/admin/skills/does.not.exist", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
