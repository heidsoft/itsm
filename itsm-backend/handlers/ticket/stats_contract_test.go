package ticket

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestTicketStatsCountsOverdueWithinTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_stats_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	for _, tid := range []int{1, 2} {
		suffix := strconv.Itoa(tid)
		tenant := client.Tenant.Create().SetName(suffix).SetCode(suffix).SetDomain(suffix + ".test").SaveX(ctx)
		user := client.User.Create().SetUsername(suffix).SetEmail(suffix + "@example.com").SetName("test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
		for _, status := range []string{"open", "resolved", "closed", "cancelled"} {
			client.Ticket.Create().SetTitle(status).SetTicketNumber(suffix + status).SetStatus(status).SetRequesterID(user.ID).SetTenantID(tenant.ID).SetSLAResolutionDeadline(time.Now().Add(-time.Hour)).SaveX(ctx)
		}
	}
	h := NewHandler(NewService(NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar())), nil, zap.NewNop().Sugar()))
	r := gin.New()
	r.GET("/api/v1/tickets/stats", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		h.GetTicketStats(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/stats", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var response struct {
		Code int `json:"code"`
		Data struct {
			Overdue int `json:"overdue"`
			Total   int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, 1, response.Data.Overdue)
	require.Equal(t, 4, response.Data.Total)
}
