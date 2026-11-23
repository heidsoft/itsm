package incident

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	service "itsm-backend/internal/services"
	"itsm-backend/service/sla"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolveIncident_RecordsSLAResolutionInSameTx(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:incident_sla_resolve_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	tenant := client.Tenant.Create().SetName("sla-resolve").SetCode("sla-resolve").SetDomain("sla-resolve.test").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("sla-resolve").SetName("Resolver").SetEmail("sla-resolve@example.com").SetPasswordHash("hash").SetTenantID(tenant.ID).SetActive(true).SaveX(ctx)
	inc := client.Incident.Create().SetTitle("SLA resolve incident").SetIncidentNumber("INC-SLA-RESOLVE-1").SetStatus("in_progress").SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)

	store := sla.NewStore(client, sla.NewEngine(nil))
	state := client.SLAState.Create().
		SetTenantID(tenant.ID).SetAggregateType("incident").SetAggregateID(inc.ID).
		SetStatus("active").SetResolutionDeadline(time.Now().Add(time.Hour)).SaveX(ctx)

	logger := zap.NewNop().Sugar()
	slaSvc := service.NewTicketSLAService(client, logger)
	slaSvc.SetSLAStore(store)
	production := service.NewIncidentService(client, logger, slaSvc)
	h := NewHandler(NewService(NewEntRepository(client), client, production, nil, nil, nil, nil, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("tenant_id", tenant.ID); c.Set("user_id", user.ID); c.Set("role", "agent") })
	r.POST("/api/v1/incidents/:id/resolve", h.Resolve)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/incidents/%d/resolve", inc.ID),
		strings.NewReader(`{"resolution":"fixed","rootCause":"bad config"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	resolved := client.SLAState.GetX(ctx, state.ID)
	require.False(t, resolved.ResolvedAt.IsZero(), "事件解决必须回写 SLA 权威解决时间戳")
	require.Equal(t, "met", resolved.Status)

	// 重复解决是幂等业务操作，SLA 首次解决时间戳不得改写。
	first := resolved.ResolvedAt
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/incidents/%d/resolve", inc.ID),
		strings.NewReader(`{"resolution":"fixed again"}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	current := client.SLAState.GetX(ctx, state.ID)
	require.True(t, first.Equal(current.ResolvedAt), "重复解决不得改写首次时间戳")
}
