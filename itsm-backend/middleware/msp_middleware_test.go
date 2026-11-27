package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"itsm-backend/common"
	"itsm-backend/ent/enttest"
)

// TestMSPMiddleware_Gate verifies that SetMSPEnabled(false) makes the
// /api/v1/msp/* route family answer with an explicit capability-disabled
// error instead of 404, even when no auth or DB state would otherwise block
// them. This is the private-deployment mode gate.
func TestMSPMiddleware_Gate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:msp_gate?mode=memory&_fk=1")
	defer client.Close()

	// enabled by default — middleware should NOT abort on the gate.
	t.Run("enabled passes through", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/msp/customers", nil)
		MSPMiddleware(client)(c)
		// Without a real user the middleware will respond 401 inside, not the
		// gate's 503. We only care the gate did not short-circuit.
		assert.NotEqual(t, http.StatusServiceUnavailable, w.Code, "gate should not report capability disabled when enabled")
	})

	t.Run("disabled reports capability unavailable instead of 404", func(t *testing.T) {
		prev := mspEnabled
		SetMSPEnabled(false)
		defer SetMSPEnabled(prev)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/msp/customers", nil)
		MSPMiddleware(client)(c)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.True(t, c.IsAborted())

		var body struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Capability string `json:"capability"`
				State      string `json:"state"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, common.ServiceUnavailableCode, body.Code)
		assert.Equal(t, "msp", body.Data.Capability)
		assert.Equal(t, "disabled", body.Data.State)
		assert.NotEmpty(t, body.Message)
	})
}

func TestMSPMiddlewareLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, status                          string
		expired, provider, header, unassigned bool
		want                                  int
		message                               string
	}{
		{name: "active", status: "active", want: 200},
		{name: "customer_suspended", status: "suspended", want: 200},
		{name: "customer_expired", status: "active", expired: true, want: 200},
		{name: "provider_suspended", status: "suspended", provider: true, want: 403, message: "租户已被暂停或过期"},
		{name: "provider_expired", status: "active", expired: true, provider: true, want: 403, message: "租户已过期"},
		{name: "header_suspended", status: "suspended", header: true, want: 403, message: "客户租户已被暂停或过期"},
		{name: "header_expired", status: "active", expired: true, header: true, want: 403, message: "客户租户已过期"},
		{name: "header_unassigned", status: "active", header: true, unassigned: true, want: 403, message: "MSP员工无权访问此客户租户"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:lifecycle_"+tc.name+"?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			provider, err := client.Tenant.Create().SetName("P").SetCode("p").SetType("msp_provider").Save(ctx)
			require.NoError(t, err)
			customer, err := client.Tenant.Create().SetName("C").SetCode("c").SetType("customer").SetMspProviderID(provider.ID).Save(ctx)
			require.NoError(t, err)
			id := customer.ID
			if tc.provider {
				id = provider.ID
			}
			update := client.Tenant.UpdateOneID(id).SetStatus(tc.status)
			if tc.expired {
				update.SetExpiresAt(time.Now().Add(-time.Hour))
			}
			require.NoError(t, update.Exec(ctx))
			actor, err := client.User.Create().SetUsername("actor").SetEmail("actor@example.com").SetName("Actor").SetPasswordHash("hash").SetTenantID(provider.ID).SetMspRole("provider_admin").Save(ctx)
			require.NoError(t, err)
			if !tc.unassigned {
				require.NoError(t, client.MSPAllocation.Create().SetMspUserID(actor.ID).SetCustomerTenantID(customer.ID).SetRole("primary").SetAssignedAt(time.Now()).Exec(ctx))
			}
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("user_id", actor.ID); c.Next() }, MSPMiddleware(client))
			r.GET("/api/v1/msp/context", func(c *gin.Context) { m, _ := GetMSPContext(c); common.Success(c, m.AllowedCustomers) })
			req := httptest.NewRequest("GET", "/api/v1/msp/context", nil)
			if tc.header {
				req.Header.Set("X-Customer-Tenant-ID", strconv.Itoa(customer.ID))
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			var body struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			if tc.want == 403 {
				require.Equal(t, common.ForbiddenCode, body.Code)
				require.Equal(t, tc.message, body.Message)
			} else {
				require.Zero(t, body.Code)
				var ids []int
				require.NoError(t, json.Unmarshal(body.Data, &ids))
				if tc.status == "active" && !tc.expired {
					require.Equal(t, []int{customer.ID}, ids)
				} else {
					require.Empty(t, ids)
				}
			}
		})
	}
}
