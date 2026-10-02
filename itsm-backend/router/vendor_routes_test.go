package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	vendorHandler "itsm-backend/handlers/vendor"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E2-1）：/api/v1/vendors 原本把后端故障和跨租户访问
// 都伪装成成功，且列表信封用非 items 键：
//   - ListVendors 用 `total, _ :=` / `vendors, _ :=` 吞掉 Count/All 的错误，
//     数据库故障时返回 200 空列表，调用方无法区分「没有供应商」和「查询挂了」。
//   - GetVendor 把所有错误压成 code 404，而 404 不在 common.Fail 的 status 映射里，
//     实际返回 HTTP 200 + 业务码 404。
//   - DeleteVendor 忽略条件删除的影响行数，删不存在/别人的记录也回成功。
//   - 列表用 {list,total,page}，与全站 items/total/page/pageSize/totalPages 契约不一致。
// 必须从真实 Router 入口证明修复后的契约，并覆盖跨租户拒绝。

const vendorsPath = "/api/v1/vendors"

type vendorRouteFixture struct {
	engine   *gin.Engine
	secret   string
	client   *ent.Client
	userA    *ent.User // tenant A super_admin
	stranger *ent.User // tenant B super_admin，用于跨租户探测
	vendorA  *ent.Vendor
}

func setupVendorRouteTest(t *testing.T) vendorRouteFixture {
	t.Helper()
	return setupVendorRouteTestOn(t, fmt.Sprintf("file:router_vendor_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
}

func setupVendorRouteTestOn(t *testing.T, dsn string) vendorRouteFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Vendor A").SetCode("vendor-a").SetDomain("vendor-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Vendor B").SetCode("vendor-b").SetDomain("vendor-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("vendor-a-admin").SetEmail("vendor-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("vendor-b-admin").SetEmail("vendor-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	vendorA := client.Vendor.Create().SetName("华为").SetCode("hw-001").SetVendorType("硬件供应商").
		SetContactName("张三").SetContactEmail("zhangsan@example.com").SetContactPhone("13800000000").
		SetAddress("深圳市").SetWebsite("https://example.com").SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "vendor-route-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     secret,
		Logger:        logger,
		Client:        client,
		VendorHandler: vendorHandler.NewHandler(service.NewVendorService(client, logger), logger),
	})

	return vendorRouteFixture{engine: r, secret: secret, client: client, userA: userA, stranger: stranger, vendorA: vendorA}
}

func (f vendorRouteFixture) do(t *testing.T, method, path string, body *string, user *ent.User) (int, vendorEnvelope) {
	t.Helper()

	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(*body)
	}
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, f.secret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)

	var env vendorEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return w.Code, env
}

// vendorEnvelope 是 { code, message, data } 的解析目标；data 保持 RawMessage，
// 由各子用例按自己的契约形状解码。
type vendorEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (f vendorRouteFixture) vendorDTO(t *testing.T, env vendorEnvelope) map[string]any {
	t.Helper()
	var v map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &v), "data=%s", env.Data)
	return v
}

func TestVendorRoutes(t *testing.T) {
	fx := setupVendorRouteTest(t)

	t.Run("创建供应商取认证上下文租户并返回 camelCase DTO", func(t *testing.T) {
		// 请求体自报 tenantId=999 必须被忽略，租户只能来自认证中间件。
		w, env := fx.do(t, http.MethodPost, vendorsPath,
			jsonBody(`{"name":"中兴","code":"zt-001","vendorType":"IT服务","contactEmail":"ops@example.com","tenantId":999}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.Equal(t, 0, env.Code)

		v := fx.vendorDTO(t, env)
		assert.Equal(t, "中兴", v["name"])
		assert.Equal(t, "zt-001", v["code"])
		assert.Equal(t, "IT服务", v["vendorType"], "响应字段必须是 camelCase")
		assert.NotContains(t, v, "vendor_type", "不得同时返回 snake_case 键")
		assert.Equal(t, fx.vendorA.TenantID, int(v["tenantId"].(float64)), "租户必须来自认证上下文")
	})

	t.Run("编码重复是 409 冲突而不是后端故障", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, vendorsPath,
			jsonBody(`{"name":"华为重复","code":"hw-001","vendorType":"硬件供应商"}`), fx.userA)
		assert.Equal(t, http.StatusConflict, w, "body=%s", string(env.Data))
		assert.Equal(t, 4090, env.Code)
		assert.Contains(t, env.Message, "供应商编码已存在")
	})

	t.Run("缺少必填字段是 400 参数错误", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, vendorsPath, jsonBody(`{"vendorType":"IT服务"}`), fx.userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code)
	})

	t.Run("列表信封为 items/total/page/pageSize/totalPages 且按租户隔离", func(t *testing.T) {
		w, env := fx.do(t, http.MethodGet, vendorsPath+"?page=1&pageSize=20", nil, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.Equal(t, 0, env.Code)

		var list struct {
			Items      []map[string]any `json:"items"`
			Total      int              `json:"total"`
			Page       int              `json:"page"`
			PageSize   int              `json:"pageSize"`
			TotalPages int              `json:"totalPages"`
			List       []map[string]any `json:"list"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &list))
		assert.Len(t, list.Items, 2, "列表键必须是 items")
		assert.Equal(t, 2, list.Total)
		assert.Equal(t, 1, list.Page)
		assert.Equal(t, 20, list.PageSize)
		assert.Equal(t, 1, list.TotalPages)
		assert.Empty(t, list.List, "data 不得同时返回 items 和旧的 list 键")

		// tenant B 只能看到自己的范围。
		w, env = fx.do(t, http.MethodGet, vendorsPath, nil, fx.stranger)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.NoError(t, json.Unmarshal(env.Data, &list))
		assert.Zero(t, list.Total, "跨租户不得看到 tenant A 的供应商")
		assert.Empty(t, list.Items)
	})

	t.Run("取不存在的供应商是 404/4004 而不是 HTTP 200", func(t *testing.T) {
		w, env := fx.do(t, http.MethodGet, vendorsPath+"/987654", nil, fx.userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("路径 ID 非法是 400 参数错误", func(t *testing.T) {
		w, env := fx.do(t, http.MethodGet, vendorsPath+"/abc", nil, fx.userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code)
	})

	t.Run("跨租户读与删一律 404，且记录未被改掉", func(t *testing.T) {
		path := fmt.Sprintf("%s/%d", vendorsPath, fx.vendorA.ID)

		w, env := fx.do(t, http.MethodGet, path, nil, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		w, env = fx.do(t, http.MethodDelete, path, nil, fx.stranger)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		// tenant A 的记录仍然存在。
		w, env = fx.do(t, http.MethodGet, path, nil, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.Equal(t, "hw-001", fx.vendorDTO(t, env)["code"])
	})

	t.Run("本租户删除成功后再删是 404，不再伪装成功", func(t *testing.T) {
		w, env := fx.do(t, http.MethodPost, vendorsPath, jsonBody(`{"name":"待删","code":"del-001"}`), fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		id := int(fx.vendorDTO(t, env)["id"].(float64))

		path := fmt.Sprintf("%s/%d", vendorsPath, id)
		w, env = fx.do(t, http.MethodDelete, path, nil, fx.userA)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.Equal(t, 0, env.Code)

		w, env = fx.do(t, http.MethodDelete, path, nil, fx.userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)
	})
}

// TestVendorListRouteFailsClosedOnDBError 锁死「数据库故障伪装成空成功」这一类缺陷：
// 改动前 ListVendors 吞掉 Count/All 的错误，查询失败时仍然返回 200 + 空 items；
// GetVendor 则把同一个故障压成「不存在」。
// 这里用同一份共享内存库的第二条连接删掉 vendors 表，制造只有供应商查询会失败、
// 认证链路仍然正常的故障，避免整库关闭导致 401 掩盖真实语义。
func TestVendorListRouteFailsClosedOnDBError(t *testing.T) {
	dsn := fmt.Sprintf("file:router_vendor_broken_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	fx := setupVendorRouteTestOn(t, dsn)

	raw, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer raw.Close()
	_, err = raw.ExecContext(context.Background(), "DROP TABLE vendors")
	require.NoError(t, err, "夹具必须能制造 vendors 查询故障")

	w, env := fx.do(t, http.MethodGet, vendorsPath, nil, fx.userA)
	assert.Equal(t, http.StatusInternalServerError, w, "数据库故障不得返回 200 空列表: body=%s", string(env.Data))
	assert.Equal(t, 5001, env.Code)
	assert.NotContains(t, strings.ToLower(env.Message), "sql", "响应体不得泄漏原始数据库错误")

	w, env = fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", vendorsPath, fx.vendorA.ID), nil, fx.userA)
	assert.Equal(t, http.StatusInternalServerError, w, "单条读取同样不得伪装成 404: body=%s", string(env.Data))
	assert.Equal(t, 5001, env.Code)
}
