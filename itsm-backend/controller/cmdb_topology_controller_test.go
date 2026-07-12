package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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

	ctrl := NewCMDBController(
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
	g.GET("/cis/:id/topology", ctrl.GetCITopology)
	g.GET("/cis/:id/impact-analysis", ctrl.GetCIImpactAnalysis)
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

	// The handler uses InternalError on lookup failure → 500 with code 5001.
	assert.Equal(t, http.StatusInternalServerError, w.Code, "cross-tenant lookup must surface as 500, body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(5001), resp["code"])
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