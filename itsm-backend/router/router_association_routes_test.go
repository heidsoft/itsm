package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTicketAssociationRoutes_ReadsURLParamNotContext verifies that the three
// ticket-association router handlers read the URL :id parameter via
// c.Param("id") instead of c.GetInt("id").  Earlier versions used
// c.GetInt("id") which reads from the gin context store (used for
// tenant_id/user_id/role).  The URL parameter is never in that store, so
// ticketID was always 0 and every request returned 400 "invalid ticket id".
//
// Regression coverage:
//   - GET /api/v1/tickets/:id/configuration-items
//   - GET /api/v1/tickets/:id/relations
//   - GET /api/v1/tickets/:id/relations/stats
func TestTicketAssociationRoutes_ReadsURLParamNotContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_assoc_urlparam?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant := client.Tenant.Create().
		SetName("Assoc Tenant").
		SetCode("assoc-urlparam").
		SetDomain("assoc.example.com").
		SetStatus("active").
		SaveX(ctx)
	requester := client.User.Create().
		SetUsername("assoc-requester").
		SetEmail("assoc-requester@example.com").
		SetName("Assoc Requester").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		SaveX(ctx)
	ciType := client.CIType.Create().
		SetName("server").
		SetTenantID(tenant.ID).
		SaveX(ctx)
	ticket := client.Ticket.Create().
		SetTicketNumber("TKT-ASSOC-URL-001").
		SetTitle("关联路由冒烟测试").
		SetDescription("验证工单关联路由正确读取 URL 参数").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(requester.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)
	ci := client.ConfigurationItem.Create().
		SetName("web-01").
		SetCiTypeID(ciType.ID).
		SetCiType("server").
		SetStatus("active").
		SetSerialNumber("SN-URL-001").
		SetTenantID(tenant.ID).
		SaveX(ctx)

	assoc := service.NewTicketAssociationService(client)
	require.NoError(t, assoc.AddConfigurationItem(ctx, ticket.ID, ci.ID))

	logger := zaptest.NewLogger(t).Sugar()
	const secret = "assoc-urlparam-secret"

	cfg := &RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketAssociationService: assoc,
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, cfg)

	token, err := middleware.GenerateAccessToken(requester.ID, requester.Username, "super_admin", tenant.ID, secret, time.Hour)
	require.NoError(t, err)

	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("configuration-items returns associated CIs", func(t *testing.T) {
		w := do(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/configuration-items", ticket.ID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var body struct {
			Code    int `json:"code"`
			Message string `json:"message"`
			Data    []struct {
				ID           int    `json:"id"`
				Name         string `json:"name"`
				CIType       string `json:"ciType"`
				Status       string `json:"status"`
				SerialNumber string `json:"serialNumber"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, 0, body.Code)
		require.Len(t, body.Data, 1, "ticket should have exactly the one associated CI")
		assert.Equal(t, ci.ID, body.Data[0].ID)
		assert.Equal(t, "web-01", body.Data[0].Name)
		assert.Equal(t, "server", body.Data[0].CIType)
	})

	t.Run("relations returns success with empty relation graph", func(t *testing.T) {
		w := do(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/relations", ticket.ID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var body struct {
			Code    int                    `json:"code"`
			Message string                 `json:"message"`
			Data    map[string]interface{} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, 0, body.Code)
		require.NotNil(t, body.Data, "relations payload must not be null")
		assert.Contains(t, body.Data, "parentChain")
		assert.Contains(t, body.Data, "childrenTree")
		assert.Contains(t, body.Data, "relatedTickets")
	})

	t.Run("relations/stats returns success with non-zero counts", func(t *testing.T) {
		w := do(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/relations/stats", ticket.ID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var body struct {
			Code int `json:"code"`
			Data struct {
				TotalRelations int `json:"totalRelations"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, 0, body.Code)
	})
}