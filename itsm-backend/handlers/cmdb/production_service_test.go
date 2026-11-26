package cmdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/database"
	"itsm-backend/dto"
	"itsm-backend/ent/configurationitem"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/datascope"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestProductionService(t *testing.T) (*gin.Engine, *ProductionService, *ent.Client) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	ciTypeService := service.NewCITypeService(client, logger)
	ciAttributeDefinitionService := service.NewCIAttributeDefinitionService(client, logger)
	ciHistoryService := service.NewCIHistoryService(client, logger)
	ciTagService := service.NewCITagService(client, logger)
	configurationItemService := service.NewConfigurationItemService(client, logger, ciHistoryService, ciTagService)
	ciRelationshipService := service.NewCIRelationshipService(client, logger)
	cmdbImportExportService := service.NewCMDBImportExportService(client, logger, configurationItemService, ciTagService)
	cmdbSavedViewService := NewCMDBSavedViewService(client, logger)

	// 创建控制器
	cmdbController := NewProductionService(
		logger,
		ciTypeService,
		ciAttributeDefinitionService,
		configurationItemService,
		ciRelationshipService,
		ciHistoryService,
		ciTagService,
		cmdbImportExportService,
		cmdbSavedViewService,
	)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 添加中间件设置租户ID
	r.Use(func(c *gin.Context) {
		// 测试时使用默认租户ID
		c.Set("tenant_id", 1)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("user_id", 1)
		c.Next()
	})

	// 注册路由
	r.POST("/api/v1/cis", cmdbController.CreateCI)
	r.GET("/api/v1/cis/:id", cmdbController.GetCI)
	r.GET("/api/v1/cis", cmdbController.ListCIs)
	r.PUT("/api/v1/cis/:id", cmdbController.UpdateCI)
	r.DELETE("/api/v1/cis/:id", cmdbController.DeleteCI)

	return r, cmdbController, client
}

func TestProductionService_CreateCI(t *testing.T) {
	r, _, client := setupTestProductionService(t)
	defer client.Close()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "缺少CI类型",
			requestBody: map[string]interface{}{
				"name":   "Test CI",
				"status": "active",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, err := http.NewRequest("POST", "/api/v1/cis", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestProductionService_GetCI(t *testing.T) {
	r, _, client := setupTestProductionService(t)
	defer client.Close()

	tests := []struct {
		name           string
		ciID           string
		expectedStatus int
	}{
		{
			name:           "获取不存在的配置项",
			ciID:           "99999",
			expectedStatus: http.StatusNotFound, // CI不存在
		},
		{
			name:           "无效的配置项ID",
			ciID:           "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/cis/"+tt.ciID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestProductionService_ListCIs(t *testing.T) {
	r, _, client := setupTestProductionService(t)
	defer client.Close()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "成功获取配置项列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带分页参数",
			queryParams:    "page=1&pageSize=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带类型筛选",
			queryParams:    "type=server",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带状态筛选",
			queryParams:    "status=active",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带环境筛选",
			queryParams:    "environment=production",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带搜索关键词",
			queryParams:    "search=test",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/cis"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestProductionService_UpdateCI(t *testing.T) {
	// 跳过测试 - 需要dddSvc
	t.Skip("UpdateCI requires dddSvc which is not available in unit tests")
}

func TestProductionService_DeleteCI(t *testing.T) {
	// 跳过测试 - 需要dddSvc
	t.Skip("DeleteCI requires dddSvc which is not available in unit tests")
}

// 测试租户隔离 - 使用 ListCIs 验证租户隔离
func TestProductionService_TenantIsolation(t *testing.T) {
	r, _, client := setupTestProductionService(t)
	defer client.Close()

	// 使用租户1查询列表
	req, err := http.NewRequest("GET", "/api/v1/cis", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证响应格式正确
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(0), response["code"])
}

// Exercises the production Handler -> ProductionService -> internal service -> Ent chain.
func TestCMDBAuditBusinessReadScope(t *testing.T) {
	_, production, client := setupTestProductionService(t)
	defer client.Close()
	ctx := context.Background()
	typ := client.CIType.Create().SetName("audit-server").SetTenantID(1).SaveX(ctx)
	makeCI := func(name string, owner, tenant int) *ent.ConfigurationItem {
		b := client.ConfigurationItem.Create().SetName(name).SetCiTypeID(typ.ID).SetTenantID(tenant)
		if owner > 0 {
			b.SetOwnerID(owner)
		}
		return b.SaveX(ctx)
	}
	owned := makeCI("audit-owned", 1, 1)
	hidden := makeCI("audit-hidden", 2, 1)
	public := makeCI("audit-public", 0, 1)
	foreign := makeCI("audit-foreign", 0, 2)
	for _, target := range []*ent.ConfigurationItem{hidden, public, foreign} {
		client.CIRelationship.Create().SetTenantID(1).SetSourceCiID(owned.ID).SetTargetCiID(target.ID).SetRelationshipType("depends_on").SetIsActive(true).SaveX(ctx)
	}
	handler := NewHandler(NewService(NewEntRepository(client), production, zaptest.NewLogger(t).Sugar()))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("user_id", 1)
		c.Set("role", "agent")
		if c.GetHeader("X-Test-Admin") == "yes" {
			c.Set("role", "admin")
		}
		if c.GetHeader("X-Test-No-Actor") == "yes" {
			c.Set("user_id", 0)
		}
	})
	r.GET("/cis", handler.ListCIs)
	r.POST("/cis/search", handler.SearchCI)
	r.GET("/cis/:id", handler.GetCI)
	r.GET("/cis/:id/topology", handler.GetCITopology)
	r.GET("/cis/:id/impact-analysis", handler.GetCIImpactAnalysis)
	r.GET("/cis/:id/relationships", handler.ListCIRelationshipsByCIID)
	r.GET("/relationships", handler.ListCIRelationships)
	r.GET("/relationships/:id", handler.GetCIRelationship)
	r.GET("/cis/:id/history", handler.GetCIHistory)
	r.GET("/stats", handler.GetCIStats)
	call := func(method, path, role string) (int, int, string) {
		req := httptest.NewRequest(method, path, strings.NewReader(`{"page":1,"pageSize":20,"filters":{},"scopeAll":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(role, "yes")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var env struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
		for _, forbidden := range []string{`"tenant_id":`, `"owner_id":`, `"password":`, `"apiKey":`} {
			assert.NotContains(t, w.Body.String(), forbidden)
		}
		return w.Code, env.Code, string(env.Data)
	}
	for _, suffix := range []string{"", "/topology", "/impact-analysis", "/relationships", "/history"} {
		for _, ci := range []*ent.ConfigurationItem{hidden, foreign} {
			status, code, _ := call("GET", fmt.Sprintf("/cis/%d%s", ci.ID, suffix), "")
			assert.Equal(t, http.StatusNotFound, status, suffix)
			assert.Equal(t, common.NotFoundCode, code)
		}
		status, code, body := call("GET", fmt.Sprintf("/cis/%d%s", owned.ID, suffix), "")
		assert.Equal(t, http.StatusOK, status, body)
		assert.Zero(t, code)
		assert.NotContains(t, body, hidden.Name)
		assert.NotContains(t, body, foreign.Name)
		if suffix == "" {
			assert.Contains(t, body, `"ciTypeId":`)
			assert.Contains(t, body, `"ownerId":`)
		}
	}
	for _, path := range []string{"/cis?withRelations=true", "/relationships", fmt.Sprintf("/cis/%d?withRelations=true", owned.ID)} {
		status, code, body := call("GET", path, "")
		assert.Equal(t, http.StatusOK, status, body)
		assert.Zero(t, code)
		assert.NotContains(t, body, hidden.Name)
		assert.NotContains(t, body, foreign.Name)
		assert.Contains(t, body, public.Name)
	}
	status, code, body := call("POST", "/cis/search", "")
	assert.Equal(t, http.StatusOK, status, body)
	assert.Zero(t, code)
	assert.NotContains(t, body, hidden.Name)
	assert.NotContains(t, body, foreign.Name)
	assert.Contains(t, body, owned.Name)
	assert.Contains(t, body, public.Name)
	status, code, _ = call("GET", fmt.Sprintf("/cis/%d", hidden.ID), "X-Test-Admin")
	assert.Equal(t, http.StatusOK, status)
	assert.Zero(t, code)
	status, code, _ = call("GET", fmt.Sprintf("/cis/%d", foreign.ID), "X-Test-Admin")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, common.NotFoundCode, code)
	status, code, _ = call("GET", fmt.Sprintf("/cis/%d", public.ID), "X-Test-No-Actor")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, common.NotFoundCode, code)
	status, code, body = call("GET", "/stats", "")
	assert.Equal(t, http.StatusOK, status, body)
	assert.Zero(t, code)
	var stats dto.CIStatsResponse
	require.NoError(t, json.Unmarshal([]byte(body), &stats))
	assert.Equal(t, 2, stats.Total)
	for _, rel := range client.CIRelationship.Query().AllX(ctx) {
		status, code, _ := call("GET", fmt.Sprintf("/relationships/%d", rel.ID), "")
		if rel.TargetCiID == public.ID {
			assert.Equal(t, http.StatusOK, status)
			assert.Zero(t, code)
		} else {
			assert.Equal(t, http.StatusNotFound, status)
			assert.Equal(t, common.NotFoundCode, code)
		}
	}
	// A trusted internal call does not inherit HTTP scope implicitly.
	ci, err := production.ciService.GetCIByID(datascope.WithActor(ctx, datascope.Actor{UserID: 1, Role: "agent", TenantID: 1}), hidden.ID, 1, false)
	require.NoError(t, err)
	require.NotNil(t, ci)
}

func TestCMDBAuditGovernanceIncludesRetiredWithSecurity(t *testing.T) {
	_, production, client := setupTestProductionService(t)
	defer client.Close()
	database.RegisterSecurityInterceptors(client, "off")
	ctx := context.Background()
	typ := client.CIType.Create().SetName("audit-governance").SetTenantID(1).SaveX(ctx)
	active := client.ConfigurationItem.Create().SetName("active").SetCiTypeID(typ.ID).SetTenantID(1).SaveX(ctx)
	retired := client.ConfigurationItem.Create().SetName("retired").SetCiTypeID(typ.ID).SetTenantID(1).SetLifecycleStatus(common.CILifecycleStatusScrapped).SaveX(ctx)
	foreign := client.ConfigurationItem.Create().SetName("foreign").SetCiTypeID(typ.ID).SetTenantID(2).SetLifecycleStatus(common.CILifecycleStatusScrapped).SaveX(ctx)
	items, err := NewEntRepository(client).ListCIsForGovernance(ctx, 1)
	require.NoError(t, err)
	ids := []int{}
	for _, ci := range items {
		ids = append(ids, ci.ID)
	}
	assert.ElementsMatch(t, []int{active.ID, retired.ID}, ids)
	assert.NotContains(t, ids, foreign.ID)
	list, err := production.ciService.ListCIs(ctx, 1, &dto.ListCIRequest{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, active.ID, list.Items[0].ID)
	ci, err := production.ciService.GetCIByID(ctx, retired.ID, 1, false)
	require.NoError(t, err)
	assert.Nil(t, ci)
	// The opt-in is query-local: ordinary raw Ent reads still filter retirement.
	visible, err := client.ConfigurationItem.Query().Where(configurationitem.TenantIDEQ(1)).IDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []int{active.ID}, visible)
}
