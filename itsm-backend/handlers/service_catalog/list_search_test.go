package service_catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scListData 走真实 GET 入口，返回信封里 data 的各字段，便于逐字段断言 camelCase 契约。
func scListData(t *testing.T, r *gin.Engine, path string) map[string]json.RawMessage {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, path, nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code    int                        `json:"code"`
		Message string                     `json:"message"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "body=%s", w.Body.String())
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", w.Body.String())
	return resp.Data
}

func scInt(t *testing.T, data map[string]json.RawMessage, key string) int {
	t.Helper()
	raw, ok := data[key]
	require.True(t, ok, "响应缺少 %s 字段", key)
	var v int
	require.NoError(t, json.Unmarshal(raw, &v))
	return v
}

func scNames(t *testing.T, data map[string]json.RawMessage) []string {
	t.Helper()
	var items []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(data["items"], &items))
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.Name)
	}
	return names
}

func scSetStatus(t *testing.T, r *gin.Engine, id int, status string) {
	t.Helper()
	resp := scDoReq(t, r, http.MethodPut, "/api/v1/service-catalogs/"+strconv.Itoa(id),
		map[string]any{"status": status})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
}

func scCreate(t *testing.T, r *gin.Engine, name, description string) int {
	t.Helper()
	resp := scDoReq(t, r, http.MethodPost, "/api/v1/service-catalogs",
		dto.CreateServiceCatalogRequest{Name: name, Category: "it_service", Description: description})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	return int(resp.Data.(map[string]any)["id"].(float64))
}

// TestHandler_ListKeywordFiltersServerSide 证明 q 由后端过滤，并且 total 是过滤后的命中数。
// 前端曾把「已加载的第一页」做客户端关键词过滤后当成全量结果，页外的匹配项会静默消失。
func TestHandler_ListKeywordFiltersServerSide(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	needle := "Kw" + uid

	nameHit := scCreate(t, r, needle+"-A", "无关描述")
	descHit := scCreate(t, r, needle+"-B", "交付"+needle+"的服务")
	disabledHit := scCreate(t, r, needle+"-C", "已停用目录")
	scSetStatus(t, r, disabledHit, "disabled")
	scCreate(t, r, "Noise-"+uid, "无关名称无关描述")

	q := "/api/v1/service-catalogs?q=" + url.QueryEscape(needle)
	data := scListData(t, r, q)
	assert.ElementsMatch(t, []string{needle + "-A", needle + "-B", needle + "-C"}, scNames(t, data),
		"名称和描述都要参与匹配，且不受分页影响")
	assert.Equal(t, 3, scInt(t, data, "total"), "total 必须是过滤后的命中数而不是全表条数")

	// 状态过滤与关键词叠加生效——这是 /search 做不到的（它固定只看已发布目录）。
	scSetStatus(t, r, nameHit, "disabled")
	scSetStatus(t, r, descHit, "disabled")
	data = scListData(t, r, q+"&status=disabled")
	assert.Len(t, scNames(t, data), 3, "disabled 过滤要能搜到已归档目录")
	data = scListData(t, r, q+"&status=enabled")
	assert.Empty(t, scNames(t, data), "全部转停用后 enabled 结果必须为空")
}

// TestHandler_ListKeywordKeepsPaginationBounded 证明关键词过滤后仍按请求页长返回，
// 不会一次序列化整个匹配集合。
func TestHandler_ListKeywordKeepsPaginationBounded(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	needle := "Paged" + uid

	for i := 0; i < 25; i++ {
		scCreate(t, r, needle+"-"+strconv.Itoa(i), "")
	}

	data := scListData(t, r, "/api/v1/service-catalogs?q="+url.QueryEscape(needle)+"&page=2&pageSize=10")
	assert.Equal(t, 25, scInt(t, data, "total"))
	assert.Len(t, scNames(t, data), 10, "第二页仍按请求页长返回")
	assert.Equal(t, 2, scInt(t, data, "page"))
	assert.Equal(t, 10, scInt(t, data, "pageSize"))
	assert.Equal(t, 3, scInt(t, data, "totalPages"))
}

// TestHandler_ListAssociationFiltersAreServerSide 证明管理页的 CI 类型/云服务筛选
// 由服务端生效：否则页面对「已加载的第一页」二次过滤，页外的关联目录会静默消失。
func TestHandler_ListAssociationFiltersAreServerSide(t *testing.T) {
	r, client, tenantID := scSetup(t)
	ctx := context.Background()
	vm, err := client.CIType.Create().SetName("VMType-" + scUID()).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	db, err := client.CIType.Create().SetName("DBType-" + scUID()).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	mw, err := client.CIType.Create().SetName("MWType-" + scUID()).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	cloud, err := client.CloudService.Create().
		SetProvider("aliyun").SetServiceCode("oss").SetServiceName("OSS").
		SetResourceTypeCode("bucket").SetResourceTypeName("Bucket").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	uid := scUID()
	needle := "Rel" + uid
	create := func(name string, ciTypeID int, cloudServiceID int) {
		resp := scDoReq(t, r, http.MethodPost, "/api/v1/service-catalogs",
			dto.CreateServiceCatalogRequest{
				Name: name, Category: "it_service", CITypeID: ciTypeID, CloudServiceID: cloudServiceID,
			})
		require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	}
	create(needle+"-vm", vm.ID, 0)
	create(needle+"-db", db.ID, 0)
	create(needle+"-none", 0, 0)
	create(needle+"-cloud", mw.ID, cloud.ID)

	data := scListData(t, r, "/api/v1/service-catalogs?q="+url.QueryEscape(needle)+"&ciTypeId="+strconv.Itoa(vm.ID))
	assert.Equal(t, []string{needle + "-vm"}, scNames(t, data))
	assert.Equal(t, 1, scInt(t, data, "total"), "total 要随关联筛选收窄")

	data = scListData(t, r, "/api/v1/service-catalogs?q="+url.QueryEscape(needle)+"&cloudServiceId="+strconv.Itoa(cloud.ID))
	assert.Equal(t, []string{needle + "-cloud"}, scNames(t, data), "只有真正关联该云服务的目录入选")
	assert.Equal(t, 1, scInt(t, data, "total"))

	data = scListData(t, r, "/api/v1/service-catalogs?q="+url.QueryEscape(needle)+"&cloudServiceId=9")
	assert.Empty(t, scNames(t, data), "没有任何目录关联该云服务")
	assert.Equal(t, 0, scInt(t, data, "total"))
}

// TestCreate_RejectsUnknownCITypeWithFieldMessage 证明引用校验失败给出中文字段提示，
// 而不是把仓储层的英文原文（或驱动错误）当成页面消息。
func TestCreate_RejectsUnknownCITypeWithFieldMessage(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()

	w, resp := scDoRaw(t, r, http.MethodPost, "/api/v1/service-catalogs",
		dto.CreateServiceCatalogRequest{
			Name: "BadRef-" + uid, Category: "it_service", CITypeID: 999_999,
		})
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(&resp))
	assert.Equal(t, "关联的 CI 类型不存在或未启用", resp.Message)
	assert.NotContains(t, w.Body.String(), "not found or inactive", "英文仓储错误不得透给页面")
}
