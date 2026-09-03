package incident

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

func TestLifecycleHTTPErrorContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident_lifecycle_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("Lifecycle").SetCode("lifecycle").SetDomain("lifecycle.test").SaveX(ctx)
	user := client.User.Create().SetUsername("lifecycle").SetName("test").SetEmail("lifecycle@example.com").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	inc := client.Incident.Create().SetTitle("closed incident").SetIncidentNumber("INC-review").SetStatus("closed").SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	h := NewHandler(NewService(nil, production, nil, nil, nil, zap.NewNop().Sugar()))
	for _, tc := range []struct {
		name, path           string
		tenant, status, code int
	}{
		{"transition", "acknowledge", tenant.ID, 409, 4090},
		{"foreign", "acknowledge", tenant.ID + 1, 404, 4004},
		{"missing identity", "acknowledge", 0, 401, 2001},
		{"sla unavailable", "sla/pause", tenant.ID, 503, 5003},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("tenant_id", tc.tenant); c.Set("user_id", user.ID) })
			r.POST("/api/v1/incidents/:id/acknowledge", h.Acknowledge)
			r.PUT("/api/v1/incidents/:id/sla/pause", h.PauseSLA)
			method := http.MethodPost
			if tc.path == "sla/pause" {
				method = http.MethodPut
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(method, "/api/v1/incidents/"+strconv.Itoa(inc.ID)+"/"+tc.path, nil))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			var response struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, tc.code, response.Code)
			require.NotEmpty(t, response.Message)
		})
	}
	require.Equal(t, "closed", client.Incident.GetX(ctx, inc.ID).Status)
}
