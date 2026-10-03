package service_catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// scSeq 生成唯一后缀，避免跨用例的命名/编号冲突
var scSeq int64

func scUID() string {
	scSeq++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), scSeq)
}

// withTenant 注入 service_catalog handler 使用的 c.GetInt("tenant_id")
func withTenant(tid int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("tenant_id", tid)
		c.Next()
	}
}

// scDoReq 直接驱动 handler（不经过路由权限中间件），跳过鉴权头
func scDoReq(t *testing.T, r *gin.Engine, method, path string, body interface{}) *common.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

func scSetup(t *testing.T) (*gin.Engine, *ent.Client, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sc_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("SCTenant").
		SetCode("SC" + scUID()).
		SetDomain("sc.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	svc := NewService(repo, zaptest.NewLogger(t).Sugar())
	h := NewHandler(svc)

	r := gin.New()
	r.Use(withTenant(tenant.ID))
	r.GET("/api/v1/service-catalogs", h.List)
	r.POST("/api/v1/service-catalogs", h.Create)
	r.GET("/api/v1/service-catalogs/search", h.Search)
	r.GET("/api/v1/service-catalogs/stats", h.Stats)
	r.GET("/api/v1/service-catalogs/:id", h.Get)
	r.PUT("/api/v1/service-catalogs/:id", h.Update)
	r.DELETE("/api/v1/service-catalogs/:id", h.Delete)
	return r, client, tenant.ID
}

func TestHandler_Create_Success(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	req := dto.CreateServiceCatalogRequest{
		Name:     "Catalog-" + uid,
		Category: "hardware",
	}
	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", req)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	require.IsType(t, map[string]interface{}{}, resp.Data)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, req.Name, data["name"])
	assert.Equal(t, "enabled", data["status"]) // normalizeServiceCatalogRequest 默认 enabled
}

func TestHandler_Create_MissingName(t *testing.T) {
	r, _, _ := scSetup(t)
	req := dto.CreateServiceCatalogRequest{Category: "hardware"}
	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", req)
	assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(resp))
}

func TestHandler_Create_MissingCategory(t *testing.T) {
	r, _, _ := scSetup(t)
	req := dto.CreateServiceCatalogRequest{Name: "Catalog-X"}
	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", req)
	assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(resp))
}

func TestHandler_Create_CloudServiceRequiresCIType(t *testing.T) {
	r, _, _ := scSetup(t)
	req := dto.CreateServiceCatalogRequest{
		Name:           "Catalog-Cld",
		Category:       "cloud",
		CloudServiceID: 5, // CITypeID 缺省为 0
	}
	resp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", req)
	assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(resp))
}

func TestHandler_List(t *testing.T) {
	r, _, _ := scSetup(t)
	// 先创建一条
	uid := scUID()
	create := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "ListCat-" + uid, Category: "hw",
	})
	require.Equal(t, common.SuccessCode, create.Code)

	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	data := resp.Data.(map[string]interface{})
	assert.GreaterOrEqual(t, int(data["total"].(float64)), 1)
	// 平台五键信封：集合键 items（原 catalogs）、页长键 pageSize（原 size）、
	// 并带上 totalPages，调用方才能核对是否还有下一页。
	assert.ElementsMatch(t, []string{"items", "total", "page", "pageSize", "totalPages"}, keysOf(data))
	assert.IsType(t, []interface{}{}, data["items"])
	// HTTP 入口的默认页长是 20（common.GetPaginationFromQuery 里的平台值）。
	assert.Equal(t, float64(20), data["pageSize"])
}

// TestHandler_List_PageSizeAliasIsInert 锁住 E4-6f 的契约变更：
// 请求侧 size 不再是 pageSize 的别名，越界页长不再返回 1001 参数错误，
// 而是回落平台默认页长 20（唯一所有者 common.GetPaginationFromQuery）。
// 原 TestHandler_List_AcceptsSize1000 / _RejectsSize1001 断言的是「max=1000 放行、
// 1001 报 1001」那套第四页长真相，已随别名一起删除。
func TestHandler_List_PageSizeAliasIsInert(t *testing.T) {
	r, _, _ := scSetup(t)

	for _, query := range []string{"?size=1000", "?size=1001", "?pageSize=0", "?pageSize=300"} {
		resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs"+query, nil)
		require.Equal(t, common.SuccessCode, resp.Code, "%s 应为 200/code 0，body=%s", query, mustSC(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(20), data["pageSize"],
			"%s 必须回落默认页长 20，而不是按别名取 1000 条或报参数错误", query)
	}
}

func TestHandler_Get_Success(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	create := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "GetCat-" + uid, Category: "hw",
	})
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	assert.Equal(t, "GetCat-"+uid, resp.Data.(map[string]interface{})["name"])
}

func TestHandler_Get_InvalidID(t *testing.T) {
	r, _, _ := scSetup(t)
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/notanint", nil)
	assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustSC(resp))
}

func TestHandler_Get_NotFound(t *testing.T) {
	r, _, _ := scSetup(t)
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/999999", nil)
	assert.Equal(t, common.NotFoundErrorCode, resp.Code, "body=%s", mustSC(resp))
}

func TestHandler_Update_Success(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	create := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "UpdCat-" + uid, Category: "hw",
	})
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	updated := "Renamed-" + uid
	disabled := "disabled"
	resp := scDoReq(t, r, "PUT", "/api/v1/service-catalogs/"+strconv.Itoa(id),
		dto.UpdateServiceCatalogRequest{Name: &updated, Status: &disabled})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	assert.Equal(t, updated, resp.Data.(map[string]interface{})["name"])
	assert.Equal(t, "disabled", resp.Data.(map[string]interface{})["status"])
}

func TestHandler_Delete_Success(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	create := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "DelCat-" + uid, Category: "hw",
	})
	require.Equal(t, common.SuccessCode, create.Code)
	id := int(create.Data.(map[string]interface{})["id"].(float64))

	del := scDoReq(t, r, "DELETE", "/api/v1/service-catalogs/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, del.Code, "body=%s", mustSC(del))

	// 删除采用归档禁用，保留目录历史
	after := scDoReq(t, r, "GET", "/api/v1/service-catalogs/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, after.Code, "body=%s", mustSC(after))
	assert.Equal(t, "disabled", after.Data.(map[string]interface{})["status"])
	search := scDoReq(t, r, "GET", "/api/v1/service-catalogs/search?q=DelCat", nil)
	require.Equal(t, common.SuccessCode, search.Code)
	assert.Zero(t, int(search.Data.(map[string]interface{})["total"].(float64)))
}

func TestHandler_CreateRejectsDuplicateNameAndInvalidDeliveryTime(t *testing.T) {
	r, _, _ := scSetup(t)
	name := "Duplicate-" + scUID()
	first := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: name, Category: "hardware",
	})
	require.Equal(t, common.SuccessCode, first.Code)
	duplicate := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: strings.ToUpper(name), Category: "hardware",
	})
	assert.Equal(t, common.ConflictCode, duplicate.Code)
	invalid := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Invalid Delivery " + scUID(), Category: "hardware", DeliveryTime: "tomorrow",
	})
	assert.Equal(t, common.ParamErrorCode, invalid.Code)
}

func TestHandler_CreateValidatesReferencedTenantResources(t *testing.T) {
	r, client, tenantID := scSetup(t)
	ctx := context.Background()
	otherTenant, err := client.Tenant.Create().
		SetName("Other SC Tenant").SetCode("OTHER-" + scUID()).SetDomain("other-sc.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	localType, err := client.CIType.Create().SetName("Local Type").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	foreignType, err := client.CIType.Create().SetName("Foreign Type").SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)
	foreignCloud, err := client.CloudService.Create().
		SetProvider("aliyun").SetServiceCode("ecs").SetServiceName("ECS").
		SetResourceTypeCode("instance").SetResourceTypeName("Instance").SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)

	foreignCI := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Foreign CI " + scUID(), Category: "cloud", CITypeID: foreignType.ID,
	})
	assert.Equal(t, common.ParamErrorCode, foreignCI.Code)
	foreignCloudResp := scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "Foreign Cloud " + scUID(), Category: "cloud",
		CITypeID: localType.ID, CloudServiceID: foreignCloud.ID,
	})
	assert.Equal(t, common.ParamErrorCode, foreignCloudResp.Code)
}

func TestHandler_Search(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "SearchCat-" + uid, Category: "hw",
	})
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/search?q=SearchCat", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	assert.IsType(t, []interface{}{}, resp.Data.(map[string]interface{})["items"])
}

func TestHandler_Stats(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "StatsCat-" + uid, Category: "hw",
	})
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/stats", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	data := resp.Data.(map[string]interface{})
	assert.Contains(t, data, "totalServices")
}

// keysOf 返回响应对象的键集合，用于断言信封是精确的五键而不是「包含某几个键」。
func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mustSC(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
