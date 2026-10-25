package knowledge

import (
	"bytes"
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

// 回归：知识检索曾用 map 手拼响应，item 字段写成 is_published /
// relevance_score / created_at / updated_at —— 把持久化列名泄漏成 API 字段，
// 调用方按 camelCase 契约读取全部落空。检索同时是 top-N（limit 上限 50），
// 必须显式告知是否被截断，否则截断结果会被当作全集。
func TestSearchArticles_CamelCaseItemsAndTruncation(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:knowledge_search_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar())
	svc.SetEntClient(client)
	h := NewHandler(svc)

	for _, title := range []string{"变更 发布流程", "变更 回滚手册"} {
		_, err := svc.CreateArticle(ctx, &Article{
			Title: title, Content: "变更内容说明", AuthorID: 1, TenantID: 7, IsPublished: true,
		})
		require.NoError(t, err)
	}

	data := searchArticles(t, h, 7, `{"query":"变更","pageSize":10}`)

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"items", "limit", "total", "truncated"}, keys,
		"检索信封必须是 items/total/limit/truncated，不得混入分页五键伪装成可翻页")

	items, ok := data["items"].([]interface{})
	require.True(t, ok, "data.items 缺失: %v", data)
	require.NotEmpty(t, items)

	first, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	itemKeys := make([]string, 0, len(first))
	for k := range first {
		itemKeys = append(itemKeys, k)
	}
	sort.Strings(itemKeys)
	assert.Equal(t, []string{
		"category", "createdAt", "id", "isPublished", "relevanceScore",
		"snippet", "tags", "title", "updatedAt",
	}, itemKeys, "item 字段必须是 camelCase 契约键")

	assert.Equal(t, false, data["truncated"], "未打满 limit 时不得声称截断")
	assert.Equal(t, float64(10), data["limit"])

	// limit=1 且命中 2 条 → 必须显式标记截断。
	truncated := searchArticles(t, h, 7, `{"query":"变更","pageSize":1}`)
	assert.Equal(t, true, truncated["truncated"])
	assert.Equal(t, float64(1), truncated["total"])
}

func searchArticles(t *testing.T, h *Handler, tenantID int, body string) map[string]interface{} {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		c.Set("role", "admin")
	})
	r.POST("/api/v1/knowledge/search", h.SearchArticles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var envelope struct {
		Code int `json:"code"`
		Data map[string]interface{}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Equal(t, 0, envelope.Code, w.Body.String())
	return envelope.Data
}
