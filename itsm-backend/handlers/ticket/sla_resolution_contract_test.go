package ticket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	service "itsm-backend/internal/services"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"
	"itsm-backend/service/sla"
)

func TestResolveTicket_RecordsSLAResolutionInSameTx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:ticket_sla_resolve_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	logger := zap.NewNop().Sugar()

	tenant := client.Tenant.Create().SetName("sla-resolve").SetCode("sla-resolve").SetDomain("sla-resolve.test").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("resolver").SetName("Resolver").SetEmail("resolver@example.com").SetPasswordHash("hash").SetTenantID(tenant.ID).SetActive(true).SaveX(ctx)
	deadline := time.Now().Add(time.Hour)
	tk := client.Ticket.Create().
		SetTitle("SLA resolve ticket").
		SetTicketNumber("T-SLA-RESOLVE-1").
		SetDescription("desc").
		SetPriority("medium").
		SetType("incident").
		SetStatus("in_progress").
		SetRequesterID(user.ID).SetAssigneeID(user.ID).SetTenantID(tenant.ID).
		SetSLAResolutionDeadline(deadline).SaveX(ctx)
	state := client.SLAState.Create().
		SetTenantID(tenant.ID).SetAggregateType("ticket").SetAggregateID(tk.ID).
		SetResolutionDeadline(deadline).SetStatus("active").SaveX(ctx)

	slaMonitor := service.NewSLAMonitorService(client, logger)
	slaMonitor.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
	h := NewHandler(NewService(
		NewEntRepository(ticketrepo.NewEntRepository(client, logger), client),
		nil, slaMonitor, logger,
	))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", user.ID)
		c.Set("role", "admin")
	})
	r.POST("/api/v1/tickets/:id/resolve", h.ResolveTicket)
	r.PUT("/api/v1/tickets/:id/status", h.UpdateTicketStatus)

	t.Run("resolve endpoint", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/v1/tickets/%d/resolve", tk.ID),
			strings.NewReader(`{"resolution":"fixed"}`)))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		resolved := client.SLAState.GetX(ctx, state.ID)
		require.False(t, resolved.ResolvedAt.IsZero(), "工单解决必须回写 SLA 权威解决时间戳")
		require.Equal(t, "met", resolved.Status)
		require.Equal(t, "resolved", client.Ticket.GetX(ctx, tk.ID).Status)
	})

	t.Run("status endpoint", func(t *testing.T) {
		tk2 := client.Ticket.Create().
			SetTitle("SLA status ticket").SetTicketNumber("T-SLA-STATUS-1").
			SetDescription("desc").SetPriority("medium").SetType("incident").SetStatus("in_progress").
			SetRequesterID(user.ID).SetAssigneeID(user.ID).SetTenantID(tenant.ID).
			SetSLAResolutionDeadline(deadline).SaveX(ctx)
		state2 := client.SLAState.Create().
			SetTenantID(tenant.ID).SetAggregateType("ticket").SetAggregateID(tk2.ID).
			SetResolutionDeadline(deadline).SetStatus("active").SaveX(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut,
			fmt.Sprintf("/api/v1/tickets/%d/status", tk2.ID),
			strings.NewReader(`{"status":"resolved"}`)))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		resolved := client.SLAState.GetX(ctx, state2.ID)
		require.False(t, resolved.ResolvedAt.IsZero(), "PUT /status 写 resolved 也必须回写 SLA")
		require.Equal(t, "met", resolved.Status)
	})
}
