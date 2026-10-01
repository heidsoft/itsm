package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 列表信封收敛回归：GET /api/v1/knowledge/articles 只返回平铺的
// items/total/page/pageSize/totalPages。
//
// 修复前这里返回 {items,total,page,pageSize} 且 dto 曾同时输出 articles 别名，
// 与工单/事件/变更三套列表各自不同的键集合并存，前端只能按多字段 fallback 读取。
func TestListArticles_CanonicalEnvelope(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:knowledge_list_envelope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar())
	svc.SetEntClient(client)
	h := NewHandler(svc)

	for _, title := range []string{"runbook-a", "runbook-b", "runbook-c"} {
		_, err := svc.CreateArticle(ctx, &Article{
			Title: title, Content: "content", AuthorID: 1, TenantID: 7, IsPublished: true,
		})
		require.NoError(t, err)
	}
	// 租户 8 的文章不得出现在租户 7 的列表里。
	_, err := svc.CreateArticle(ctx, &Article{
		Title: "other-tenant", Content: "content", AuthorID: 1, TenantID: 8, IsPublished: true,
	})
	require.NoError(t, err)

	data := listArticlesData(t, h, 7, "page=1&pageSize=2")

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, keys)

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "data.items 缺失")
	assert.Len(t, items, 2, "pageSize=2 必须生效")
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(2), data["totalPages"])

	empty := listArticlesData(t, h, 99, "page=1&pageSize=20")
	assert.Equal(t, []interface{}{}, empty["items"], "空列表必须序列化为 []，不能是 null")
}

func listArticlesData(t *testing.T, h *Handler, tenantID int, query string) map[string]interface{} {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", tenantID) })
	r.GET("/api/v1/knowledge/articles", h.ListArticles)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/articles?"+query, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Code int `json:"code"`
		Data map[string]interface{}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, 0, body.Code, w.Body.String())
	return body.Data
}
