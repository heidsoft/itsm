package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestRoleController(t *testing.T) (*gin.Engine, *ent.Client, *RoleController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	roleService := service.NewRoleService(client, logger)

	// 创建控制器
	roleController := NewRoleController(roleService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/roles", roleController.ListRoles)
	r.POST("/api/v1/roles", roleController.CreateRole)
	r.GET("/api/v1/roles/:id", roleController.GetRole)
	r.PUT("/api/v1/roles/:id", roleController.UpdateRole)
	r.DELETE("/api/v1/roles/:id", roleController.DeleteRole)

	return r, client, roleController
}

func createTestTenantForRole(t *testing.T, client *ent.Client) *ent.Tenant {
	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TESTROLE" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return tenant
}

func TestRoleController_ListRoles(t *testing.T) {
	r, client, _ := setupTestRoleController(t)
	defer client.Close()

	tenant := createTestTenantForRole(t, client)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "成功获取角色列表",
			queryParams: "",
		},
		{
			name:        "带分页参数",
			queryParams: "page=1&pageSize=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/roles"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestRoleController_CreateRole(t *testing.T) {
	r, client, _ := setupTestRoleController(t)
	defer client.Close()

	tenant := createTestTenantForRole(t, client)

	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "成功创建角色",
			payload: `{"code":"admin","name":"管理员","description":"系统管理员"}`,
		},
		{
			name:    "缺少必需字段",
			payload: `{"code":"","name":"管理员"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/roles", nil)
			require.NoError(t, err)
			req.Body = nil

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestRoleController_GetRole(t *testing.T) {
	r, client, _ := setupTestRoleController(t)
	defer client.Close()

	tenant := createTestTenantForRole(t, client)

	tests := []struct {
		name    string
		roleID  string
	}{
		{
			name:   "成功获取角色",
			roleID: "1",
		},
		{
			name:   "无效的角色ID",
			roleID: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/roles/"+tt.roleID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestRoleController_UpdateRole(t *testing.T) {
	r, client, _ := setupTestRoleController(t)
	defer client.Close()

	tenant := createTestTenantForRole(t, client)

	tests := []struct {
		name   string
		roleID string
	}{
		{
			name:   "成功更新角色",
			roleID: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("PUT", "/api/v1/roles/"+tt.roleID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestRoleController_DeleteRole(t *testing.T) {
	r, client, _ := setupTestRoleController(t)
	defer client.Close()

	tenant := createTestTenantForRole(t, client)

	tests := []struct {
		name   string
		roleID string
	}{
		{
			name:   "成功删除角色",
			roleID: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/roles/"+tt.roleID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
