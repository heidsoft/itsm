package cmdb

// 回归：GET /api/v1/cmdb/cis/:id/relationships 对未知（或跨租户）CI 曾返回
// ok:true + 空数组，调用方无法区分「该 CI 确实没有关系」与「CI 不存在 /
// ID 写错 / 跨租户」。现在走标准的 404 + 业务 code 4004。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type relationshipFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	ciTypeID int
}

func newRelationshipByCIFixture(t *testing.T, ctxTenantID int) relationshipFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:cmdb_rel_by_ci?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("CMDB 关系归属").
		SetCode("cmdb-rel-by-ci").
		SetDomain("cmdb-rel-by-ci.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	relSvc := service.NewCIRelationshipService(client, logger)
	production := NewProductionService(logger, nil, nil, nil, relSvc, nil, nil, nil, nil)
	h := NewHandler(NewService(nil, production, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: ctxTenantID})
		c.Next()
	})
	r.GET("/api/v1/cmdb/cis/:id/relationships", h.ListCIRelationshipsByCIID)
	return relationshipFixture{router: r, client: client, tenantID: tenant.ID, ciTypeID: ciType.ID}
}

func TestListCIRelationshipsByCIID_UnknownCIIs404(t *testing.T) {
	f := newRelationshipByCIFixture(t, 1)

	ci, err := f.client.ConfigurationItem.Create().
		SetName("known-ci").
		SetCiTypeID(f.ciTypeID).
		SetCiType("Server").
		SetStatus("active").
		SetTenantID(f.tenantID).
		Save(context.Background())
	require.NoError(t, err)

	// 已存在但没有关系的 CI：200 + 空集合（合法的空结果）。
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/cis/"+strconv.Itoa(ci.ID)+"/relationships", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var okBody struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &okBody))
	assert.Equal(t, 0, okBody.Code)
	assert.JSONEq(t, `[]`, string(okBody.Data))

	// 未知 CI：必须 404，不能伪装成空关系。
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/cis/99999/relationships", nil))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var errBody struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
	assert.Equal(t, 4004, errBody.Code)
	assert.NotEmpty(t, errBody.Msg)
}

func TestListCIRelationshipsByCIID_CrossTenantCIIs404(t *testing.T) {
	f := newRelationshipByCIFixture(t, 2)

	ci, err := f.client.ConfigurationItem.Create().
		SetName("tenant-1-ci").
		SetCiTypeID(f.ciTypeID).
		SetCiType("Server").
		SetStatus("active").
		SetTenantID(f.tenantID).
		Save(context.Background())
	require.NoError(t, err)

	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/cis/"+strconv.Itoa(ci.ID)+"/relationships", nil))
	require.Equal(t, http.StatusNotFound, w.Code, "跨租户 CI 不得暴露存在性；body=%s", w.Body.String())

	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, common.NotFoundCode, body.Code)
}
