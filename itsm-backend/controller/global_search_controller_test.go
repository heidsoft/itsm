package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupGlobalSearchTest(t *testing.T) (*gin.Engine, *ent.Client, *ent.Tenant, *ent.User) {
	gin.SetMode(gin.TestMode)

	client := SetupTestDB(t)
	t.Cleanup(func() { client.Close() })

	tenant := CreateTestTenantWithID(t, client, "GSEARCH")
	user, err := client.User.Create().
		SetUsername("gsearch_user_" + uniqueTestID()).
		SetEmail("gsearch_" + uniqueTestID() + "@example.com").
		SetName("Global Search User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(t.Context())
	require.NoError(t, err)

	controller := NewGlobalSearchController(client)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{
			TenantID: tenant.ID,
			Tenant:   tenant,
		})
		c.Next()
	})
	controller.RegisterRoutes(r.Group("/api/v1"))

	return r, client, tenant, user
}

func TestGlobalSearch_SearchCaseInsensitiveAndNumber(t *testing.T) {
	r, client, tenant, user := setupGlobalSearchTest(t)

	_, err := client.Ticket.Create().
		SetTitle("VPN Access Failure").
		SetDescription("Cannot connect to gateway").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-GLOBAL-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		Save(t.Context())
	require.NoError(t, err)

	_, err = client.Incident.Create().
		SetTitle("Database latency alert").
		SetDescription("Primary database is slow").
		SetStatus("new").
		SetType("incident").
		SetPriority("high").
		SetSeverity("high").
		SetIncidentNumber("INC-GLOBAL-002").
		SetReporterID(user.ID).
		SetTenantID(tenant.ID).
		Save(t.Context())
	require.NoError(t, err)

	tests := []struct {
		name         string
		keyword      string
		expectedType string
		expectedNo   string
	}{
		{name: "case insensitive title", keyword: "vpn access", expectedType: "ticket", expectedNo: "TKT-GLOBAL-001"},
		{name: "case insensitive ticket number", keyword: "tkt-global", expectedType: "ticket", expectedNo: "TKT-GLOBAL-001"},
		{name: "case insensitive incident number", keyword: "inc-global", expectedType: "incident", expectedNo: "INC-GLOBAL-002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/global-search?keyword="+tt.keyword, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			require.Equal(t, common.SuccessCode, resp.Code)

			dataBytes, err := json.Marshal(resp.Data)
			require.NoError(t, err)

			var searchResp SearchResponse
			require.NoError(t, json.Unmarshal(dataBytes, &searchResp))
			require.NotEmpty(t, searchResp.Results)
			require.Equal(t, tt.expectedType, searchResp.Results[0].Type)
			require.Equal(t, tt.expectedNo, searchResp.Results[0].Number)
		})
	}
}
