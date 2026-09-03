package cmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// cmdbTopologyRouter boots a router with auth/tenant middleware and only the
// routes needed for the topology + impact analysis tests. The tenant middleware
// reads the X-Test-Tenant header so each test can flip the tenant context.
func cmdbTopologyRouter(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:cmdb_topology?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	ciTypeService := service.NewCITypeService(client, logger)
	ciAttrSvc := service.NewCIAttributeDefinitionService(client, logger)
	ciHistorySvc := service.NewCIHistoryService(client, logger)
	ciTagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, ciHistorySvc, ciTagSvc)
	relSvc := service.NewCIRelationshipService(client, logger)

	ctrl := NewProductionService(
		logger,
		ciTypeService,
		ciAttrSvc,
		ciSvc,
		relSvc,
		ciHistorySvc,
		ciTagSvc,
		nil,
		nil,
	)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tenantID = v
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Next()
	})

	g := r.Group("/api/v1/cmdb")
	handler := NewHandler(NewService(nil, ctrl, logger))
	g.GET("/cis/:id/topology", handler.GetCITopology)
	g.GET("/cis/:id/impact-analysis", handler.GetCIImpactAnalysis)
	return r, client
}

func seedCITopologyFixture(t *testing.T, client *ent.Client, tenantID int) (rootID, childID int) {
	t.Helper()
	ctx := context.Background()
	// ConfigurationItem requires a CiType reference; create a ci_type row first.
	ciType, err := client.CIType.Create().
		SetName("server").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	root, err := client.ConfigurationItem.Create().
		SetName("root-server").
		SetCiType("server").
		SetCiTypeID(ciType.ID).
		SetStatus("active").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	child, err := client.ConfigurationItem.Create().
		SetName("child-app").
		SetCiType("application").
		SetCiTypeID(ciType.ID).
		SetStatus("active").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(child.ID).
		SetTargetCiID(root.ID).
		SetTenantID(tenantID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)
	return root.ID, child.ID
}

func TestGetCITopology_ReturnsUnifiedGraph(t *testing.T) {
	r, client := cmdbTopologyRouter(t)
	defer client.Close()

	root, _ := seedCITopologyFixture(t, client, 1)

	req, _ := http.NewRequest("GET", "/api/v1/cmdb/cis/"+strconv.Itoa(root)+"/topology?depth=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, float64(root), resp.Data["rootCiId"])
	assert.Equal(t, float64(3), resp.Data["depth"])
	// Two nodes (root + child) and one edge (depends_on).
	assert.Equal(t, float64(2), resp.Data["totalNodes"])
	assert.Equal(t, float64(1), resp.Data["totalEdges"])
}

func TestGetCITopology_RejectsCrossTenant(t *testing.T) {
	r, client := cmdbTopologyRouter(t)
	defer client.Close()

	// CI lives in tenant 2.
	_, child := seedCITopologyFixture(t, client, 2)

	req, _ := http.NewRequest("GET", "/api/v1/cmdb/cis/"+strconv.Itoa(child)+"/topology?depth=3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(4004), resp["code"])
}

func TestGetCITopology_CapsDepthAt10(t *testing.T) {
	r, client := cmdbTopologyRouter(t)
	defer client.Close()

	root, _ := seedCITopologyFixture(t, client, 1)

	// Request depth=20; the service caps at maxCIImpactAnalysisDepth (=10).
	req, _ := http.NewRequest("GET", "/api/v1/cmdb/cis/"+strconv.Itoa(root)+"/topology?depth=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(10), resp.Data["depth"], "depth=20 must be capped to 10")
}

func TestGetCITopology_DefaultsDepthTo3(t *testing.T) {
	r, client := cmdbTopologyRouter(t)
	defer client.Close()

	root, _ := seedCITopologyFixture(t, client, 1)

	req, _ := http.NewRequest("GET", "/api/v1/cmdb/cis/"+strconv.Itoa(root)+"/topology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp.Data["depth"], "missing depth must default to 3")
}

func TestCIGraphLookupErrors(t *testing.T) {
	for _, endpoint := range []string{"topology", "impact-analysis"} {
		for _, scenario := range []string{"missing", "foreign", "database"} {
			t.Run(endpoint+"/"+scenario, func(t *testing.T) {
				r, client := cmdbTopologyRouter(t)
				defer client.Close()
				root, _ := seedCITopologyFixture(t, client, 2)
				id, status, code := root, http.StatusNotFound, 4004
				if scenario == "missing" {
					id = 999999
				}
				if scenario == "database" {
					require.NoError(t, client.Close())
					status, code = http.StatusInternalServerError, 5001
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/cis/"+strconv.Itoa(id)+"/"+endpoint, nil))
				require.Equal(t, status, w.Code, w.Body.String())
				var response struct {
					Code    int         `json:"code"`
					Message string      `json:"message"`
					Data    interface{} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				assert.Equal(t, code, response.Code)
				assert.Nil(t, response.Data)
				assert.NotContains(t, response.Message, "sql")
				assert.NotContains(t, response.Message, "ent:")
				assert.NotContains(t, response.Message, "root-server")
			})
		}
	}
}

// Imported or legacy relations may be inconsistent: every related resource
// must be scoped independently, even after the root CI passed tenant checks.
func TestCIImpactAnalysis_FiltersForeignAssociations(t *testing.T) {
	r, client := cmdbTopologyRouter(t)
	defer client.Close()
	ctx := context.Background()
	for _, id := range []int{1, 2} {
		suffix := strconv.Itoa(id)
		tenant := client.Tenant.Create().SetName("Impact tenant " + suffix).SetCode("impact-" + suffix).SetDomain("impact-" + suffix + ".example.com").SaveX(ctx)
		require.Equal(t, id, tenant.ID)
	}
	root, _ := seedCITopologyFixture(t, client, 1)
	foreignRoot, _ := seedCITopologyFixture(t, client, 2)
	client.CIRelationship.Create().SetSourceCiID(root).SetTargetCiID(foreignRoot).SetRelationshipType("impacts").SetTenantID(1).SetIsActive(true).SaveX(ctx)
	for _, tenant := range []int{1, 2} {
		suffix := strconv.Itoa(tenant)
		user := client.User.Create().SetUsername("impact-user-" + suffix).SetEmail("impact-" + suffix + "@example.com").SetName("test").SetPasswordHash("test-hash").SetTenantID(tenant).SaveX(ctx)
		ticket := client.Ticket.Create().SetTitle("ticket-tenant-" + suffix).SetTicketNumber("T-" + suffix).SetRequesterID(user.ID).SetTenantID(tenant).SaveX(ctx)
		incident := client.Incident.Create().SetTitle("incident-tenant-" + suffix).SetIncidentNumber("I-" + suffix).SetReporterID(user.ID).SetTenantID(tenant).SaveX(ctx)
		client.ConfigurationItem.UpdateOneID(root).AddTickets(ticket).AddIncidents(incident).ExecX(ctx)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/cis/"+strconv.Itoa(root)+"/impact-analysis", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		Code int                          `json:"code"`
		Data dto.CIImpactAnalysisResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Zero(t, response.Code)
	require.Len(t, response.Data.AffectedTickets, 1)
	assert.Equal(t, "ticket-tenant-1", response.Data.AffectedTickets[0].Title)
	require.Len(t, response.Data.AffectedIncidents, 1)
	assert.Equal(t, "incident-tenant-1", response.Data.AffectedIncidents[0].Title)
	require.NotNil(t, response.Data.Graph)
	for _, edge := range response.Data.Graph.Edges {
		assert.NotEqual(t, foreignRoot, edge.Source)
		assert.NotEqual(t, foreignRoot, edge.Target)
	}
	assert.NotContains(t, w.Body.String(), "tenant-2")
}
