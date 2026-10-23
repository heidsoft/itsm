package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/marketplaceitem"
	marketplaceService "itsm-backend/service/marketplace"

	// 真实连接器 provider：连接器类商品必须映射到已注册 manifest。
	_ "itsm-backend/connector/builtin/webhook"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 目录管理 HTTP 面的契约测试。
//
// 这里刻意不挂 RequirePermission：权限映射由 middleware 包的
// TestMarketplaceCatalogRoutesUsePlatformPermission 与 internal/authz 的
// TestMarketplaceCatalogWriteIsPlatformOnly 分别锁定（码 → 路由、码 → 角色）。
// 本文件只验证「业务拒绝时给出的状态码与文案是否可操作」——目录管理最常见的
// 失败是「点了上架没反应」，而原因（没有已发布版本 / 已上架不能删）本可以在
// 响应里说清楚。
type catalogFixture struct {
	router *gin.Engine
	client *ent.Client
	svc    *marketplaceService.Service
}

func newCatalogFixture(t *testing.T, dsn string) *catalogFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:"+dsn+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	svc := marketplaceService.NewService(client, zaptest.NewLogger(t).Sugar())
	h := NewHandler(svc)

	r := gin.New()
	g := r.Group("/api/v1/marketplace/catalog")
	g.GET("/items", h.AdminListItems)
	g.POST("/items", h.CreateCatalogItem)
	g.GET("/items/:id", h.GetCatalogItem)
	g.PATCH("/items/:id", h.UpdateCatalogItem)
	g.DELETE("/items/:id", h.DeleteCatalogItem)
	g.POST("/items/:id/publish", h.PublishCatalogItem)
	g.POST("/items/:id/unpublish", h.UnpublishCatalogItem)
	g.GET("/items/:id/versions", h.ListItemVersions)
	g.POST("/items/:id/versions", h.CreateItemVersion)
	g.POST("/items/:id/versions/:version/publish", h.PublishItemVersion)

	return &catalogFixture{router: r, client: client, svc: svc}
}

func (f *catalogFixture) do(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	var payload map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	}
	return rec.Code, payload
}

func TestCatalogHandler_CreateReturnsItemAndVersions(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_create")

	code, payload := f.do(t, http.MethodPost, "/api/v1/marketplace/catalog/items", map[string]any{
		"name": "webhook-connector",
		"type": "connector",
	})
	require.Equal(t, http.StatusOK, code, "%v", payload)

	data, ok := payload["data"].(map[string]any)
	require.True(t, ok, "data 缺失: %v", payload)
	item, ok := data["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "webhook-connector", item["name"])
	assert.Equal(t, "draft", item["status"])
	assert.Equal(t, "1.0.0", item["latestVersion"])
	// 版本内联返回：页面判断「能不能上架」的依据就是这里有没有 published。
	versions, ok := data["versions"].([]any)
	require.True(t, ok)
	require.Len(t, versions, 1)
	assert.Equal(t, "published", versions[0].(map[string]any)["status"])
}

// 未注册的连接器 provider 在创建时就拒绝，而不是等到租户点安装才失败。
func TestCatalogHandler_UnregisteredConnectorIsBadRequest(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_unregistered")

	code, payload := f.do(t, http.MethodPost, "/api/v1/marketplace/catalog/items", map[string]any{
		"name": "ghost-connector",
		"type": "connector",
	})
	require.Equal(t, http.StatusBadRequest, code, "%v", payload)
	msg, _ := payload["message"].(string)
	assert.Contains(t, msg, "ghost", "文案应指出是哪个 provider 没注册")
}

func TestCatalogHandler_PublishWithoutVersionIsConflictWithActionableMessage(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_publish_guard")
	ctx := context.Background()

	item, err := f.client.MarketplaceItem.Create().
		SetName("no-version-skill").
		SetType(marketplaceitem.TypeSkill).
		SetTitle("无版本技能").
		SetProvider("internal").
		SetLatestVersion("").
		Save(ctx)
	require.NoError(t, err)

	code, payload := f.do(t, http.MethodPost, "/api/v1/marketplace/catalog/items/"+strconv.Itoa(item.ID)+"/publish", nil)
	require.Equal(t, http.StatusConflict, code, "%v", payload)
	msg, _ := payload["message"].(string)
	assert.Contains(t, msg, "已发布版本", "文案必须告诉运营下一步做什么")
}

func TestCatalogHandler_DeletePublishedIsConflict(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_delete_published")
	ctx := context.Background()

	item, err := f.client.MarketplaceItem.Create().
		SetName("webhook-connector").
		SetType(marketplaceitem.TypeConnector).
		SetTitle("Webhook").
		SetProvider("generic").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)

	code, payload := f.do(t, http.MethodDelete, "/api/v1/marketplace/catalog/items/"+strconv.Itoa(item.ID), nil)
	require.Equal(t, http.StatusConflict, code, "%v", payload)
	msg, _ := payload["message"].(string)
	assert.Contains(t, msg, "先下架")
}

func TestCatalogHandler_UpdateNameIsConflict(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_update_name")

	code, payload := f.do(t, http.MethodPost, "/api/v1/marketplace/catalog/items", map[string]any{
		"name": "webhook-connector",
		"type": "connector",
	})
	require.Equal(t, http.StatusOK, code, "%v", payload)
	itemID := payload["data"].(map[string]any)["item"].(map[string]any)["id"].(float64)

	code, payload = f.do(t, http.MethodPatch, "/api/v1/marketplace/catalog/items/"+strconv.Itoa(int(itemID)), map[string]any{
		"name": "other-connector",
	})
	require.Equal(t, http.StatusConflict, code, "%v", payload)
	msg, _ := payload["message"].(string)
	assert.Contains(t, msg, "不可修改")
}

func TestCatalogHandler_NotFoundAndInvalidID(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_not_found")

	code, payload := f.do(t, http.MethodGet, "/api/v1/marketplace/catalog/items/999", nil)
	require.Equal(t, http.StatusNotFound, code, "%v", payload)

	code, _ = f.do(t, http.MethodGet, "/api/v1/marketplace/catalog/items/abc", nil)
	require.Equal(t, http.StatusBadRequest, code)
}

func TestCatalogHandler_VersionLifecycle(t *testing.T) {
	f := newCatalogFixture(t, "catalog_http_versions")

	code, payload := f.do(t, http.MethodPost, "/api/v1/marketplace/catalog/items", map[string]any{
		"name":     "excel-export-skill",
		"type":     "skill",
		"title":    "Excel 导出",
		"provider": "internal",
		"version":  "1.0.0",
	})
	require.Equal(t, http.StatusOK, code, "%v", payload)
	itemID := int(payload["data"].(map[string]any)["item"].(map[string]any)["id"].(float64))
	base := "/api/v1/marketplace/catalog/items/" + strconv.Itoa(itemID)

	code, payload = f.do(t, http.MethodPost, base+"/publish", nil)
	require.Equal(t, http.StatusOK, code, "%v", payload)

	code, payload = f.do(t, http.MethodPost, base+"/versions", map[string]any{"version": "1.1.0"})
	require.Equal(t, http.StatusOK, code, "%v", payload)
	assert.Equal(t, "draft", payload["data"].(map[string]any)["status"])

	code, payload = f.do(t, http.MethodGet, base+"/versions", nil)
	require.Equal(t, http.StatusOK, code, "%v", payload)
	assert.Len(t, payload["data"], 2)

	code, payload = f.do(t, http.MethodPost, base+"/versions/1.1.0/publish", nil)
	require.Equal(t, http.StatusOK, code, "%v", payload)
	assert.Equal(t, "1.1.0", payload["data"].(map[string]any)["item"].(map[string]any)["latestVersion"])

	// 重复版本号：409 且指出冲突的是版本号，而不是笼统的「新增失败」。
	code, payload = f.do(t, http.MethodPost, base+"/versions", map[string]any{"version": "1.1.0"})
	require.Equal(t, http.StatusConflict, code, "%v", payload)
}
