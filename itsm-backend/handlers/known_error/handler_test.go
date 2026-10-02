package known_error

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 契约：known-errors 的边缘入口必须遵守标准信封与真实统计。
//
// 修复前的真实形态（本文件先跑红再转绿）：
//   - GET /stats 在 gin.H 里伪造 page=1、totalPages=1，让聚合统计看起来像分页集合；
//   - Service.GetStats 复用同一条 Ent query，而 Ent 的 Where 会就地追加谓词，
//     因此 status/severity 条件逐级 AND，resolved/deprecated/critical/... 恒为 0；
//   - GET /categories 只返回 {items}，前端按 {categories} 读取并配了 || [] 兜底，
//     分类下拉因此永远为空。

func newKnownErrorTestRouter(t *testing.T, client *ent.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(NewService(client), logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// RequirePermission 的上游注入：super_admin 在 AuthorizeResource 里短路放行。
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Set("role", "super_admin")
		c.Set("client", client)
		c.Next()
	})
	// 直接挂载生产 RegisterRoutes，路径与 RBAC 中间件链和 router 包一致。
	group := r.Group("/api/v1")
	h.RegisterRoutes(group)
	return r
}

func doKnownErrorRequest(t *testing.T, r *gin.Engine, path string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "path=%s body=%s", path, w.Body.String())
	return w.Code, envelope
}

// dataKeys 返回响应 data 的键集合，用于断言信封形状。
func dataKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func seedKnownError(t *testing.T, client *ent.Client, tenantID int, title, status, severity, category string) {
	t.Helper()
	_, err := client.KnownError.Create().
		SetTenantID(tenantID).
		SetTitle(title).
		SetStatus(status).
		SetSeverity(severity).
		SetCategory(category).
		SetCreatedBy(1).
		Save(t.Context())
	require.NoError(t, err)
}

// TestStats_UsesFullResponse 断言 stats 是聚合对象而非伪分页集合，
// 并且每个计数条件独立统计。
func TestStats_HonorsIndependentCountsAndNoFakePagination(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ke_stats?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedKnownError(t, client, 1, "KE-1", "active", "critical", "network")
	seedKnownError(t, client, 1, "KE-2", "active", "high", "network")
	seedKnownError(t, client, 1, "KE-3", "resolved", "medium", "storage")
	seedKnownError(t, client, 1, "KE-4", "deprecated", "low", "storage")
	// 其他租户不得进入统计
	seedKnownError(t, client, 2, "KE-OTHER", "active", "critical", "network")

	r := newKnownErrorTestRouter(t, client)
	status, envelope := doKnownErrorRequest(t, r, "/api/v1/known-errors/stats")
	require.Equal(t, http.StatusOK, status)

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok, "envelope=%v", envelope)

	assert.Equal(t, []string{
		"active", "critical", "deprecated", "high", "low", "medium", "resolved", "total",
	}, dataKeys(data), "stats 不应包含 page/pageSize/totalPages")

	assert.EqualValues(t, 4, data["total"])
	assert.EqualValues(t, 2, data["active"])
	// 修复前：这些值因谓词逐级 AND 恒为 0
	assert.EqualValues(t, 1, data["resolved"])
	assert.EqualValues(t, 1, data["deprecated"])
	assert.EqualValues(t, 1, data["critical"])
	assert.EqualValues(t, 1, data["high"])
	assert.EqualValues(t, 1, data["medium"])
	assert.EqualValues(t, 1, data["low"])
}

// TestCategories_UsesItemsEnvelope 断言分类列表使用 {items,total}。
func TestCategories_UsesItemsEnvelope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ke_categories?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedKnownError(t, client, 1, "KE-1", "active", "high", "network")
	seedKnownError(t, client, 1, "KE-2", "active", "high", "network")
	seedKnownError(t, client, 1, "KE-3", "resolved", "low", "storage")

	r := newKnownErrorTestRouter(t, client)
	status, envelope := doKnownErrorRequest(t, r, "/api/v1/known-errors/categories")
	require.Equal(t, http.StatusOK, status)

	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok, "envelope=%v", envelope)
	assert.Equal(t, []string{"items", "total"}, dataKeys(data))

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.ElementsMatch(t, []interface{}{"network", "storage"}, items)
	assert.EqualValues(t, 2, data["total"])
}

// TestListEnvelope_TotalsAreFullCounts 断言 list/search 用标准信封，
// total 是全量计数而不是当页长度。
func TestListEnvelope_TotalsAreFullCounts(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ke_list?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	for i := 1; i <= 3; i++ {
		seedKnownError(t, client, 1, "KE-"+string(rune('a'+i)), "active", "high", "network")
	}

	r := newKnownErrorTestRouter(t, client)
	for _, path := range []string{
		"/api/v1/known-errors?page=1&pageSize=2",
		"/api/v1/known-errors/search?q=&page=1&pageSize=2",
	} {
		status, envelope := doKnownErrorRequest(t, r, path)
		require.Equal(t, http.StatusOK, status, "path=%s", path)

		data, ok := envelope["data"].(map[string]interface{})
		require.True(t, ok, "path=%s envelope=%v", path, envelope)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, dataKeys(data), "path=%s", path)
		assert.EqualValues(t, 3, data["total"], "path=%s total 必须是全量计数", path)
		assert.EqualValues(t, 2, data["pageSize"])
		assert.EqualValues(t, 2, data["totalPages"])
		assert.Len(t, data["items"], 2)
	}
}
