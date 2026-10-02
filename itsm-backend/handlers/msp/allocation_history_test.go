package msp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/tenant"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 契约：GET /api/v1/msp/allocations/history 是真实接线的分配历史。
//
// 前端 msp-api.ts 一直在调用这个路径，但 router/msp_routes.go 从未注册它，
// 于是页签只能靠 PRODUCT_CAPABILITIES.mspAllocationHistory=false 藏着；
// 同时 dto.MSPAllocationHistory 声明了 deallocationReason / createdBy /
// customerName 这些表里根本没有列可供给的字段。
//
// 本文件锁定三件事：
//  1. 历史行包含已解除的记录，数据边界是调用者认证上下文里的 MSP 租户
//     （msp_allocations 没有 tenant_id 列，只能通过 msp_user 的租户收敛）；
//  2. 响应只含 MSPAllocationDTO 的真实 camelCase 字段，不再伪造原因/操作人；
//  3. 缺少租户或用户上下文必须 fail closed 返回 401，畸形过滤参数返回 400。

func newAllocationRouter(t *testing.T, client *ent.Client, userID, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	h := NewHandler(service.NewMSPAllocationService(client, logger), nil, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// 只有认证上下文里真实存在时才写入，用来覆盖 fail-closed 分支。
		if userID != 0 {
			c.Set("user_id", userID)
		}
		if tenantID != 0 {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	})
	group := r.Group("/api/v1/msp")
	group.GET("/allocations/history", h.GetAllocationHistory)
	group.POST("/allocations/deallocate", h.Deallocate)
	return r
}

func seedHistoryUser(t *testing.T, client *ent.Client, username string, tenantID int) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func seedHistoryTenant(t *testing.T, client *ent.Client, name, code string, tenantType tenant.Type) *ent.Tenant {
	t.Helper()
	tn, err := client.Tenant.Create().SetName(name).SetCode(code).SetType(tenantType).Save(context.Background())
	require.NoError(t, err)
	return tn
}

// seedHistoryAllocation 建一条分配；deassignedAt 为 nil 表示仍然活跃。
func seedHistoryAllocation(t *testing.T, client *ent.Client, mspUserID, customerTenantID int, assignedAt time.Time, deassignedAt *time.Time) *ent.MSPAllocation {
	t.Helper()
	builder := client.MSPAllocation.Create().
		SetMspUserID(mspUserID).
		SetCustomerTenantID(customerTenantID).
		SetRole("primary").
		SetAssignedAt(assignedAt)
	if deassignedAt != nil {
		builder.SetDeassignedAt(*deassignedAt)
	}
	a, err := builder.Save(context.Background())
	require.NoError(t, err)
	return a
}

func doAllocationRequest(t *testing.T, r *gin.Engine, method, path, body string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload), w.Body.String())
	return w.Code, payload
}

func historyEnvelope(t *testing.T, r *gin.Engine, query string) map[string]interface{} {
	t.Helper()
	status, body := doAllocationRequest(t, r, http.MethodGet, "/api/v1/msp/allocations/history"+query, "")
	require.Equal(t, http.StatusOK, status, "%v", body)
	require.Equal(t, float64(0), body["code"])
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是列表信封，实际 %v", body)
	return data
}

func TestGetAllocationHistory_IncludesClosedRowsWithinCallerTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_history_scope?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ownMSP := seedHistoryTenant(t, client, "MSP-Own", "msp-own-h", tenant.TypeMspProvider)
	foreignMSP := seedHistoryTenant(t, client, "MSP-Foreign", "msp-for-h", tenant.TypeMspProvider)
	custA := seedHistoryTenant(t, client, "Customer A", "cust-a-h", tenant.TypeCustomer)
	custB := seedHistoryTenant(t, client, "Customer B", "cust-b-h", tenant.TypeCustomer)

	agent := seedHistoryUser(t, client, "hist_agent", ownMSP.ID)
	foreignAgent := seedHistoryUser(t, client, "hist_foreign", foreignMSP.ID)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	closed := base.AddDate(0, 0, 12)
	// 本租户：一条活跃 + 一条已解除。
	seedHistoryAllocation(t, client, agent.ID, custA.ID, base, nil)
	seedHistoryAllocation(t, client, agent.ID, custB.ID, base.AddDate(0, 0, 10), &closed)
	// 另一个 MSP 租户服务同一个客户：绝不能出现在调用者的历史里。
	seedHistoryAllocation(t, client, foreignAgent.ID, custA.ID, base, nil)

	r := newAllocationRouter(t, client, agent.ID, ownMSP.ID)
	data := historyEnvelope(t, r, "")

	// 标准列表信封：集合只放在 items 下。
	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, sortedDataKeys(data))
	// 修复前这条路由根本不存在（404）；现在是本租户全量 2 条，跨租户那条被排除。
	assert.Equal(t, float64(2), data["total"], "total 必须是本租户的全量计数（含已解除）")

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 2)

	// assigned_at 倒序：第 10 天的记录在前。
	first := items[0].(map[string]interface{})
	assert.Equal(t, "Customer B", first["customerTenantName"])
	assert.Equal(t, "hist_agent", first["mspUsername"])
	assert.NotNil(t, first["deassignedAt"], "已解除记录必须带出结束时间")

	second := items[1].(map[string]interface{})
	assert.Equal(t, "Customer A", second["customerTenantName"])
	assert.NotContains(t, second, "deassignedAt", "活跃记录的 deassignedAt 必须缺省而非零值")

	// 表里没有 reason/created_by 列，契约就必须没有这些字段。
	for _, raw := range items {
		row := raw.(map[string]interface{})
		assert.NotContains(t, row, "deallocationReason")
		assert.NotContains(t, row, "createdBy")
		assert.NotContains(t, row, "createdByName")
		assert.NotContains(t, row, "customerName")
	}
}

func TestGetAllocationHistory_EndDateCoversWholeDay(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_history_dates?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	mspTenant := seedHistoryTenant(t, client, "MSP-D", "msp-d-h", tenant.TypeMspProvider)
	customer := seedHistoryTenant(t, client, "Cust D", "cust-d-h", tenant.TypeCustomer)
	agent := seedHistoryUser(t, client, "hist_dates", mspTenant.ID)

	seedHistoryAllocation(t, client, agent.ID, customer.ID,
		time.Date(2026, 5, 10, 13, 30, 0, 0, time.UTC), nil)
	seedHistoryAllocation(t, client, agent.ID, customer.ID,
		time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC), nil)

	r := newAllocationRouter(t, client, agent.ID, mspTenant.ID)

	// endDate 是日期（无时刻），必须覆盖当天全部时刻；否则当天 00:00 之后的分配会被静默丢掉。
	assert.Equal(t, float64(1), historyEnvelope(t, r, "?startDate=2026-05-10&endDate=2026-05-10")["total"])
	assert.Equal(t, float64(1), historyEnvelope(t, r, "?startDate=2026-05-11&endDate=2026-05-11")["total"])
	// 区间外仍要返回 0，证明日期过滤真的下推到查询而不是被忽略。
	assert.Equal(t, float64(0), historyEnvelope(t, r, "?startDate=2026-01-01&endDate=2026-01-31")["total"])
}

func TestGetAllocationHistory_FailsClosedWithoutAuthContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_history_fail?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	// 缺少 tenant_id：禁止回退到默认租户，也禁止改用 user_id 猜测租户。
	status, body := doAllocationRequest(t, newAllocationRouter(t, client, 1, 0),
		http.MethodGet, "/api/v1/msp/allocations/history", "")
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, float64(2002), body["code"])

	status, body = doAllocationRequest(t, newAllocationRouter(t, client, 0, 1),
		http.MethodGet, "/api/v1/msp/allocations/history", "")
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, float64(2002), body["code"])
}

func TestGetAllocationHistory_RejectsMalformedFilters(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_history_bad?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	r := newAllocationRouter(t, client, 1, 1)
	for _, query := range []string{
		"mspUserId=abc",
		"customerTenantId=abc",
		"startDate=20260510",
		"endDate=2026-13-45",
	} {
		status, body := doAllocationRequest(t, r, http.MethodGet,
			"/api/v1/msp/allocations/history?"+query, "")
		assert.Equal(t, http.StatusBadRequest, status, query)
		assert.Equal(t, float64(1001), body["code"], query)
	}
}

// Deactivate 必须报告它到底解除了没有：匹配不到活跃行时，历史实现照样返回成功，
// 调用方以为解除生效了。
func TestDeallocate_NoActiveAllocationReturnsNotFound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_deallocate_notfound?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	mspTenant := seedHistoryTenant(t, client, "MSP-N", "msp-n-h", tenant.TypeMspProvider)
	customer := seedHistoryTenant(t, client, "Cust N", "cust-n-h", tenant.TypeCustomer)
	agent := seedHistoryUser(t, client, "hist_dealloc", mspTenant.ID)

	closed := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	archived := seedHistoryAllocation(t, client, agent.ID, customer.ID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &closed)

	r := newAllocationRouter(t, client, agent.ID, mspTenant.ID)
	payload := `{"mspUserId":` + strconv.Itoa(agent.ID) +
		`,"customerTenantId":` + strconv.Itoa(customer.ID) + `}`

	status, body := doAllocationRequest(t, r, http.MethodPost,
		"/api/v1/msp/allocations/deallocate", payload)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, float64(4004), body["code"])

	row, err := client.MSPAllocation.Get(ctx, archived.ID)
	require.NoError(t, err)
	assert.True(t, row.DeassignedAt.Equal(closed),
		"归档行的 deassigned_at 必须保持 %v，实际 %v", closed, row.DeassignedAt)

	// 活跃分配存在时正常解除，证明 404 不是无条件失败。
	active := seedHistoryAllocation(t, client, agent.ID, customer.ID,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), nil)
	status, body = doAllocationRequest(t, r, http.MethodPost,
		"/api/v1/msp/allocations/deallocate", payload)
	assert.Equal(t, http.StatusOK, status, "%v", body)

	reloaded, err := client.MSPAllocation.Query().
		Where(mspallocation.IDEQ(active.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.False(t, reloaded.DeassignedAt.IsZero(), "解除必须写入结束时间")
}
