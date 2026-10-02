package msp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 契约：GET /api/v1/msp/reports/{customers,performance} 的查询参数是 camelCase
// startDate / endDate，聚合租户只来自认证上下文。历史实现读取 snake_case，而前端
// msp-api.ts 一直按 camelCase 发送，导致两个报表接口在真实页面上永远命中「必填参数缺失」。
//
// 同时覆盖 MSP 上下文缺失时的失败语义：必须 fail closed 返回 403，
// 而不是丢弃 exists 标志后直接解引用空指针。

func newMSPTestRouter(t *testing.T, client *ent.Client, userID int, mspCtx *middleware.MSPContext) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(nil, service.NewTicketServiceForTest(client, logger), logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("tenant_id", 1)
		if mspCtx != nil {
			// 与 MSPMiddleware 的写入方式保持一致（键名 + 指针类型）。
			c.Set(middleware.MSPContextKey, mspCtx)
		}
		c.Next()
	})
	// 路径与方法与 router/msp_routes.go 一致；Auth/RBAC/MSP 中间件链由 router 包覆盖。
	group := r.Group("/api/v1/msp")
	group.GET("/status", h.GetMSPStatus)
	group.GET("/context", h.GetMSPContext)
	group.GET("/customers/:customer_tenant_id/tickets", h.GetCustomerTickets)
	group.GET("/reports/customers", h.GetCustomerReports)
	group.GET("/reports/performance", h.GetPerformanceReports)
	return r
}

// seedTicket 在 tenantID 下创建一张 createdAt 时间的工单。
// 报表服务按认证上下文里的 tenant_id 聚合，fixture 必须让工单落在调用者租户内。
func seedTicket(t *testing.T, client *ent.Client, tenantID, requesterID int, number string, createdAt time.Time) {
	t.Helper()
	_, err := client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle("MSP 报表回归").
		SetDescription("用于验证 MSP 报表日期区间参数契约").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		SetCreatedAt(createdAt).
		Save(context.Background())
	require.NoError(t, err)
}

func doMSPRequest(t *testing.T, r *gin.Engine, path string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body
}

func reportTotalTickets(t *testing.T, body map[string]interface{}) int {
	t.Helper()
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是报表对象，实际响应 %v", body)
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items 必须是报表数组，实际响应 %v", data)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	count, ok := item["totalTickets"].(float64)
	require.True(t, ok, "报表缺少聚合计数字段，实际 %v", item)
	return int(count)
}

func TestGetCustomerReports_HonorsCamelCaseDateRange(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_report_customer?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("MSP-A").SetCode("msp-a").SetType("msp").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("msp_agent").SetEmail("msp-a@example.com").SetName("MSP Agent").
		SetPasswordHash("hash").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	r := newMSPTestRouter(t, client, user.ID, nil)

	// 修复前：handler 读 start_date/end_date，camelCase 请求恒返回 1001。
	status, body := doMSPRequest(t, r, "/api/v1/msp/reports/customers?startDate=2024-01-01&endDate=2024-12-31")
	require.Equal(t, http.StatusOK, status, body["message"])
	assert.Equal(t, float64(0), body["code"])
	assert.NotNil(t, body["data"])

	// 只发送 snake_case 时仍须报参数缺失——禁止新旧字段双读兼容。
	status, body = doMSPRequest(t, r, "/api/v1/msp/reports/customers?start_date=2024-01-01&end_date=2024-12-31")
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, float64(1001), body["code"])
}

func TestGetPerformanceReports_HonorsCamelCaseDateRange(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_report_perf?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("MSP-B").SetCode("msp-b").SetType("msp").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("msp_perf").SetEmail("msp-b@example.com").SetName("MSP Perf").
		SetPasswordHash("hash").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	seedTicket(t, client, tenant.ID, user.ID, "PERF-IN-1", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC))
	seedTicket(t, client, tenant.ID, user.ID, "PERF-IN-2", time.Date(2024, 4, 6, 0, 0, 0, 0, time.UTC))
	seedTicket(t, client, tenant.ID, user.ID, "PERF-OUT-1", time.Date(2025, 9, 9, 0, 0, 0, 0, time.UTC))

	mspCtx := &middleware.MSPContext{IsMSP: true, MSPUserID: user.ID}
	r := newMSPTestRouter(t, client, user.ID, mspCtx)

	// 修复前：handler 读 start_date/end_date，camelCase 请求恒返回 1001。
	status, body := doMSPRequest(t, r, "/api/v1/msp/reports/performance?startDate=2024-01-01&endDate=2024-12-31")
	require.Equal(t, http.StatusOK, status, body["message"])
	assert.Equal(t, float64(0), body["code"])
	// 日期区间必须真正下推到查询：区间外的工单不得计入。
	assert.Equal(t, 2, reportTotalTickets(t, body))

	// MSP 上下文缺失仍按授权失败 fail closed（与修复前一致，不放宽鉴权面）。
	r = newMSPTestRouter(t, client, user.ID, nil)
	status, body = doMSPRequest(t, r, "/api/v1/msp/reports/performance?startDate=2024-01-01&endDate=2024-12-31")
	require.Equal(t, http.StatusForbidden, status, body["message"])
	assert.Equal(t, float64(2003), body["code"])
	assert.Contains(t, body["message"], "非MSP用户")
}

func TestGetPerformanceReports_MSPUserIDFilterIsRejected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_report_perf_badid?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("MSP-C").SetCode("msp-c").SetType("msp").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("msp_perf_c").SetEmail("msp-c@example.com").SetName("MSP Perf C").
		SetPasswordHash("hash").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	r := newMSPTestRouter(t, client, user.ID, &middleware.MSPContext{IsMSP: true, MSPUserID: user.ID})

	// mspUserId 表达的「按员工过滤绩效」从未实现：旧实现把这个 user ID 当成
	// tenant_id 传给查询。现在无论传什么值都显式 400，不得静默忽略后返回租户级汇总。
	for _, value := range []string{"abc", strconv.Itoa(user.ID)} {
		status, body := doMSPRequest(t, r,
			"/api/v1/msp/reports/performance?startDate=2024-01-01&endDate=2024-12-31&mspUserId="+value)
		require.Equal(t, http.StatusBadRequest, status, "%s 返回 %v", value, body)
		assert.Equal(t, float64(1001), body["code"])
		// 错误提示必须点名 camelCase 参数并说明原因，不能让调用方以为值格式有问题而已。
		assert.Contains(t, body["message"], "mspUserId")
		assert.Contains(t, body["message"], "未实现")
	}
}

// 报表聚合维度是 tenant，租户只能来自认证上下文。修复前 handler 把认证上下文里的
// user_id 传给了按 tenant_id 过滤的服务方法，于是调用者读到的是「ID 恰好等于自己
// user_id 的那个租户」的工单数——这是一份跨租户读数。
func TestMSPReports_TenantScopeComesFromAuthContextNotUserID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_report_tenant_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	ownTenant, err := client.Tenant.Create().SetName("MSP-Own").SetCode("msp-own").SetType("msp").Save(ctx)
	require.NoError(t, err)
	foreignTenant, err := client.Tenant.Create().SetName("Other").SetCode("other").SetType("customer").Save(ctx)
	require.NoError(t, err)

	// 先建一个不属于本租户的用户占住 ID 1，再建自己的调用者，
	// 使 callerUserID == foreignTenant.ID，正好命中旧实现的混淆条件。
	_, err = client.User.Create().
		SetUsername("other_user").SetEmail("other@example.com").SetName("Other").
		SetPasswordHash("hash").SetTenantID(foreignTenant.ID).Save(ctx)
	require.NoError(t, err)
	caller, err := client.User.Create().
		SetUsername("msp_caller").SetEmail("msp-own@example.com").SetName("MSP Caller").
		SetPasswordHash("hash").SetTenantID(ownTenant.ID).Save(ctx)
	require.NoError(t, err)
	require.Equal(t, foreignTenant.ID, caller.ID, "fixture 需要 user_id 与 foreign tenant id 相等")

	seedTicket(t, client, ownTenant.ID, caller.ID, "OWN-1", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC))
	seedTicket(t, client, foreignTenant.ID, 1, "FOR-1", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC))
	seedTicket(t, client, foreignTenant.ID, 1, "FOR-2", time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC))

	// 认证上下文：tenant = ownTenant（ID 1），user = caller（ID 2 == foreignTenant.ID）。
	r := newMSPTestRouter(t, client, caller.ID, &middleware.MSPContext{IsMSP: true, MSPUserID: caller.ID})

	for _, path := range []string{
		"/api/v1/msp/reports/customers?startDate=2024-01-01&endDate=2024-12-31",
		"/api/v1/msp/reports/performance?startDate=2024-01-01&endDate=2024-12-31",
	} {
		status, body := doMSPRequest(t, r, path)
		require.Equal(t, http.StatusOK, status, "%s 返回 %v", path, body)
		// 只统计本租户的 1 张工单；修复前这里读到的是 foreign tenant 的 2 张。
		assert.Equal(t, 1, reportTotalTickets(t, body), "%s 的聚合范围必须是认证租户", path)
	}
}

// ==================== MSP context 缺失不得 panic ====================
//
// MSPMiddleware 在 gin.Context 缺少 user_id 时直接 c.Next() 且不写入 MSPContextKey。
// 历史实现用 `mspCtx, _ := GetMSPContext(c)` 丢弃 exists 后立即解引用，
// 使 GetCustomerTickets / GetPerformanceReports 成为必然 500 panic 的路由。

func TestGetCustomerTickets_FailsClosedWithoutMSPContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_cust_tickets_noctx?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newMSPTestRouter(t, client, 1, nil)
	status, body := doMSPRequest(t, r, "/api/v1/msp/customers/9/tickets")

	require.Equal(t, http.StatusForbidden, status,
		"MSP 上下文缺失必须按授权失败返回 403，历史实现 panic 后由 Recovery 返回 500")
	assert.Equal(t, float64(2003), body["code"])
	assert.Contains(t, body["message"], "非MSP用户")
}

func TestGetMSPStatusAndContext_WithoutMSPContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_status_nocxt?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newMSPTestRouter(t, client, 1, nil)
	for _, path := range []string{"/api/v1/msp/status", "/api/v1/msp/context"} {
		status, body := doMSPRequest(t, r, path)
		require.Equal(t, http.StatusOK, status, "%s 返回 %v", path, body)
		assert.Equal(t, float64(0), body["code"], path)
		data, ok := body["data"].(map[string]interface{})
		require.True(t, ok, path)
		assert.Equal(t, false, data["isMsp"], "%s 必须把缺失 MSP 上下文表达为非 MSP", path)
	}
}

func TestGetCustomerTickets_WithMSPContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_cust_tickets_ok?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	customer, err := client.Tenant.Create().SetName("Cust").SetCode("cust-1").SetType("customer").Save(ctx)
	require.NoError(t, err)

	r := newMSPTestRouter(t, client, 1, &middleware.MSPContext{
		IsMSP:            true,
		MSPUserID:        42,
		Role:             "provider_agent",
		AllowedCustomers: []int{customer.ID},
	})
	status, body := doMSPRequest(t, r, "/api/v1/msp/customers/"+strconv.Itoa(customer.ID)+"/tickets")

	require.Equal(t, http.StatusOK, status, body["message"])
	assert.Equal(t, float64(0), body["code"])
}
