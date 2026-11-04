package marketplace

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"itsm-backend/ent/itemversion"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 列表信封与安装路径参数的形状契约。
//
// tests/contract/list_envelope_ratchet_test.go 只解析 dto/*.go 里的 List 结构体，
// handler 里内联的 gin.H / common.SuccessWithList 信封不在它的视野内——市场这两个
// 接口正是从那个缝隙里漂移的：/items 声明了分页却漏掉 totalPages，/installations
// 直接返回裸数组。这里按真实注册路由锁形状，让同类回归立刻变红。

func decodeDataKeys(t *testing.T, body string) map[string]json.RawMessage {
	t.Helper()
	var payload struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &payload), body)
	require.Zero(t, payload.Code, body)
	return payload.Data
}

func keySet(data map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// TestRoutes_ItemsListReturnsFullPagingEnvelope 真分页的列表必须给满五元组。
// 只给 page/pageSize 而不给 totalPages 时，调用方无法判断是否还有下一页。
func TestRoutes_ItemsListReturnsFullPagingEnvelope(t *testing.T) {
	f := newHandlerFixture(t, "items_list_envelope")
	ctx := t.Context()
	for i := 0; i < 3; i++ {
		item := createRouteSkillItem(t, ctx, f.client, "envelope-skill-"+strconv.Itoa(i))
		addRouteSkillVersion(t, ctx, f.client, item, "2.0.0", "skill://ai.envelope", itemversion.StatusPublished)
	}

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/items?page=1&pageSize=2", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	data := decodeDataKeys(t, rec.Body.String())
	assert.ElementsMatch(t, []string{"items", "total", "page", "pageSize", "totalPages"}, keySet(data),
		"分页信封的键集合固定为 items/total/page/pageSize/totalPages")

	var total, page, pageSize, totalPages int
	require.NoError(t, json.Unmarshal(data["total"], &total))
	require.NoError(t, json.Unmarshal(data["page"], &page))
	require.NoError(t, json.Unmarshal(data["pageSize"], &pageSize))
	require.NoError(t, json.Unmarshal(data["totalPages"], &totalPages))
	assert.Equal(t, 1, page)
	assert.Equal(t, 2, pageSize)
	assert.GreaterOrEqual(t, total, 3, "样本至少要跨两页才能证明 totalPages 是真的算出来的")
	assert.Equal(t, (total+pageSize-1)/pageSize, totalPages, "totalPages 必须由 total/pageSize 推出")

	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(data["items"], &items))
	assert.Len(t, items, pageSize, "第一页应取满 pageSize 条")
}

// TestRoutes_InstallationListIsHonestUnpagedEnvelope 安装列表整表取回、service 没有
// Offset/Limit，因此只允许诚实的 {items,total}；补上 page/pageSize/totalPages 等于
// 伪造一个并不存在的分页协议（docs/api-reference.md「不分页的列表」）。
func TestRoutes_InstallationListIsHonestUnpagedEnvelope(t *testing.T) {
	f := newHandlerFixture(t, "installations_list_envelope")
	ctx := t.Context()
	item := createRouteSkillItem(t, ctx, f.client, "unpaged-skill")

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost,
		"/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/installations", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	data := decodeDataKeys(t, rec.Body.String())
	assert.ElementsMatch(t, []string{"items", "total"}, keySet(data),
		"不分页的列表只返回 {items,total}，集合键不得留在顶层裸数组")

	var rows []InstallationResponse
	require.NoError(t, json.Unmarshal(data["items"], &rows))
	require.Len(t, rows, 1)
	assert.Equal(t, item.ID, rows[0].ItemID)
}

// TestRoutes_InstallationDetailIsKeyedByItemID 锁定 /installations/:itemId 的语义：
// 路径参数是商品 ID，不是安装行主键。DTO 同时暴露 id 与 itemId，把 id 填进这条 URL
// 会命中另一条安装或 404——写错配置的后果比报错更糟，所以用行号≠商品号的样本锁死。
func TestRoutes_InstallationDetailIsKeyedByItemID(t *testing.T) {
	f := newHandlerFixture(t, "installations_item_id_param")
	ctx := t.Context()

	// 建 3 个商品但只装第 3 个：安装行主键为 1，itemId 为 3，两个数字不重合。
	for i := 1; i <= 2; i++ {
		createRouteSkillItem(t, ctx, f.client, "uninstalled-skill-"+strconv.Itoa(i))
	}
	item := createRouteSkillItem(t, ctx, f.client, "installed-skill")

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost,
		"/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	installed := decodeInstallation(t, rec.Body.String())
	require.NotEqual(t, installed.ID, installed.ItemID, "样本必须让行主键与商品 ID 不同，否则本用例证明不了任何事")

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet,
		"/api/v1/marketplace/installations/"+strconv.Itoa(installed.ItemID), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, installed.ItemID, decodeInstallation(t, rec.Body.String()).ItemID)

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet,
		"/api/v1/marketplace/installations/"+strconv.Itoa(installed.ID), "")
	assert.Equal(t, http.StatusNotFound, rec.Code,
		"安装行主键不是这条 URL 的标识符：%s", rec.Body.String())
}
