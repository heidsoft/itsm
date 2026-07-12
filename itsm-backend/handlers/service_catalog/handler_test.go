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
	assert.IsType(t, []interface{}{}, data["catalogs"])
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
	// repo 返回错误 → handler 透传 5001
	assert.Equal(t, common.InternalErrorCode, resp.Code, "body=%s", mustSC(resp))
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
	resp := scDoReq(t, r, "PUT", "/api/v1/service-catalogs/"+strconv.Itoa(id),
		dto.UpdateServiceCatalogRequest{Name: updated, Status: "disabled"})
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

	// 删除后再次查询应失败
	after := scDoReq(t, r, "GET", "/api/v1/service-catalogs/"+strconv.Itoa(id), nil)
	assert.Equal(t, common.InternalErrorCode, after.Code, "body=%s", mustSC(after))
}

func TestHandler_Search(t *testing.T) {
	r, _, _ := scSetup(t)
	uid := scUID()
	scDoReq(t, r, "POST", "/api/v1/service-catalogs", dto.CreateServiceCatalogRequest{
		Name: "SearchCat-" + uid, Category: "hw",
	})
	resp := scDoReq(t, r, "GET", "/api/v1/service-catalogs/search?q=SearchCat", nil)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustSC(resp))
	assert.IsType(t, []interface{}{}, resp.Data.(map[string]interface{})["catalogs"])
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

func mustSC(resp *common.Response) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
