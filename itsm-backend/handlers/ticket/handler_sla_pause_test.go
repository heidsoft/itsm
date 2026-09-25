package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"
	"itsm-backend/service"
)

// 回归（2026-09-25 假成功收口）：PUT /tickets/:id/sla/pause|resume 此前直接返回
// 成功假象、完全不落库。必须真实调用 SLA 领域服务：暂停落库并记录原因、
// 重复暂停/未暂停恢复是 409 冲突、跨租户按 404 拒绝。
func TestSLAPauseResumeWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ticket_sla_pause?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zap.NewNop().Sugar()
	tenant := client.Tenant.Create().SetName("SLA").SetCode("sla-pause").SetDomain("sla.test").SaveX(ctx)
	user := client.User.Create().SetUsername("slau").SetEmail("slau@example.com").SetName("u").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	deadline := time.Now().Add(2 * time.Hour)
	tk := client.Ticket.Create().SetTitle("sla ticket").SetTicketNumber("T-SLA-1").SetStatus("open").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).
		SetSLAResponseDeadline(deadline).SetSLAResolutionDeadline(deadline.Add(2 * time.Hour)).
		SaveX(ctx)

	h := NewHandler(NewService(
		NewEntRepository(ticketrepo.NewEntRepository(client, logger)),
		nil,
		service.NewSLAMonitorService(client, logger),
		logger,
	))
	newRouter := func(tenantID int) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		})
		r.PUT("/api/v1/tickets/:id/sla/pause", h.PauseSLA)
		r.PUT("/api/v1/tickets/:id/sla/resume", h.ResumeSLA)
		return r
	}
	r := newRouter(tenant.ID)
	call := func(method, path, body string) (int, int) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		var response struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		return w.Code, response.Code
	}

	status, code := call(http.MethodPut, "/api/v1/tickets/"+strconv.Itoa(tk.ID)+"/sla/pause", `{"reason":"维护窗口"}`)
	require.Equal(t, 200, status, "暂停应成功")
	require.Equal(t, 0, code)
	paused := client.Ticket.GetX(ctx, tk.ID)
	require.Equal(t, "paused", paused.SLAStatus, "暂停必须落库")
	require.Equal(t, "维护窗口", paused.SLAPauseReason)
	require.False(t, paused.SLAPausedAt.IsZero())

	status, code = call(http.MethodPut, "/api/v1/tickets/"+strconv.Itoa(tk.ID)+"/sla/pause", `{"reason":"again"}`)
	require.Equal(t, 409, status, "重复暂停必须是冲突")
	require.Equal(t, 4090, code)

	status, code = call(http.MethodPut, "/api/v1/tickets/"+strconv.Itoa(tk.ID)+"/sla/resume", "")
	require.Equal(t, 200, status)
	require.Equal(t, 0, code)
	resumed := client.Ticket.GetX(ctx, tk.ID)
	require.Equal(t, "active", resumed.SLAStatus)
	require.True(t, resumed.SLAPausedAt.IsZero())
	require.False(t, resumed.SLAResponseDeadline.Before(deadline), "暂停时长应顺延截止时间")

	status, code = call(http.MethodPut, "/api/v1/tickets/"+strconv.Itoa(tk.ID)+"/sla/resume", "")
	require.Equal(t, 409, status, "未暂停时恢复必须是冲突")
	require.Equal(t, 4090, code)

	w := httptest.NewRecorder()
	newRouter(tenant.ID+1).ServeHTTP(w, httptest.NewRequest(
		http.MethodPut, "/api/v1/tickets/"+strconv.Itoa(tk.ID)+"/sla/pause", strings.NewReader(`{"reason":"x"}`),
	))
	require.Equal(t, 404, w.Code, "跨租户暂停必须 fail closed: %s", w.Body.String())
}
