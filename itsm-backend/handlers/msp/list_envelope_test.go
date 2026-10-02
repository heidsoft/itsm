package msp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 契约：MSP 的三个列表接口必须使用统一列表信封。
//   - GET /api/v1/msp/customers/:id/tickets 历史上返回 {tickets, total: len(tickets)}，
//     total 被当前页长度冒充，前端分页永远显示不出真实总量；数组元素还是未过
//     DTO Mapper 的领域对象（Go 导出字段直接序列化）。
//   - GET /api/v1/msp/reports/{customers,performance} 返回 {reports, total}。

func mspEnvelopeData(t *testing.T, r *gin.Engine, path string) map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

	var body struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	assert.Equal(t, 0, body.Code, body.Message)
	return body.Data
}

func TestGetCustomerTickets_PagedEnvelopeWithRealTotal(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_tickets_envelope?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	mspTenant, err := client.Tenant.Create().SetName("MSP-P").SetCode("msp-p").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	agent, err := client.User.Create().
		SetUsername("msp_agent_p").SetEmail("msp-p@example.com").SetName("MSP Agent P").
		SetPasswordHash("hash").SetTenantID(mspTenant.ID).Save(ctx)
	require.NoError(t, err)

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedTicket(t, client, 2, agent.ID, "CUST-A-1", base)
	seedTicket(t, client, 2, agent.ID, "CUST-A-2", base.Add(time.Minute))
	seedTicket(t, client, 2, agent.ID, "CUST-A-3", base.Add(2*time.Minute))
	seedTicket(t, client, 3, agent.ID, "CUST-B-1", base)

	mspCtx := &middleware.MSPContext{IsMSP: true, MSPUserID: agent.ID, AllowedCustomers: []int{2}}
	r := newMSPTestRouter(t, client, agent.ID, mspCtx)

	data := mspEnvelopeData(t, r, "/api/v1/msp/customers/2/tickets?page=1&pageSize=2")
	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, sortedDataKeys(data),
		"data 必须是标准列表信封，实际 %v", data)
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items 必须是数组，实际 %v", data["items"])
	assert.Len(t, items, 2)
	// total 是租户 2 的全量计数，不是当前页长度。
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(2), data["totalPages"])

	// 元素必须是 camelCase DTO；历史实现直接序列化 repository/ticket.Ticket，
	// 前端拿到的是 ID/TicketNumber/CreatedAt 这类 Go 导出字段名。
	first, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, first, "ticketNumber")
	assert.Contains(t, first, "createdAt")
	assert.NotContains(t, first, "TicketNumber")
}

func TestGetCustomerTickets_UnauthorizedCustomerStillRejected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_tickets_envelope_deny?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	mspCtx := &middleware.MSPContext{IsMSP: true, MSPUserID: 9, AllowedCustomers: []int{2}}
	r := newMSPTestRouter(t, client, 9, mspCtx)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/msp/customers/3/tickets", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMSPReports_UsesItemsKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_reports_envelope?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("MSP-R").SetCode("msp-r").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("msp_report_user").SetEmail("msp-r@example.com").SetName("MSP Report").
		SetPasswordHash("hash").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	seedTicket(t, client, tenant.ID, user.ID, "RPT-1", time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC))

	r := newMSPTestRouter(t, client, user.ID, &middleware.MSPContext{IsMSP: true, MSPUserID: user.ID})

	for _, path := range []string{
		"/api/v1/msp/reports/customers?startDate=2024-01-01&endDate=2024-12-31",
		"/api/v1/msp/reports/performance?startDate=2024-01-01&endDate=2024-12-31",
	} {
		data := mspEnvelopeData(t, r, path)
		// 报表是按日期区间的全量聚合，没有分页参数；信封只允许 items + total。
		assert.Equal(t, []string{"items", "total"}, sortedDataKeys(data), "%s 实际 %v", path, data)
		items, ok := data["items"].([]interface{})
		require.True(t, ok, "%s items 必须是数组，实际 %v", path, data["items"])
		assert.Equal(t, float64(len(items)), data["total"], "%s total 必须等于全量长度", path)

		// 行字段必须是 camelCase DTO；历史实现返回 total_tickets / status_summary
		// 这类数据库列名，前端按 totalTickets 读取永远是 undefined。
		first, ok := items[0].(map[string]interface{})
		require.True(t, ok, "%s 报表行必须是对象", path)
		assert.Contains(t, first, "totalTickets")
		assert.NotContains(t, first, "total_tickets")
	}
}

func sortedDataKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
