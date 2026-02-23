package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupTestDepartmentController(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建控制器
	_ = NewDepartmentController(client)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r, client
}

func createTestTenantForDepartment(t *testing.T, client *ent.Client) *ent.Tenant {
	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TESTDEPT" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return tenant
}

func TestDepartmentController_CreateDepartment(t *testing.T) {
	r, client := setupTestDepartmentController(t)
	defer client.Close()

	tenant := createTestTenantForDepartment(t, client)

	tests := []struct {
		name string
	}{
		{
			name: "创建部门",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/departments", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestDepartmentController_GetDepartmentTree(t *testing.T) {
	r, client := setupTestDepartmentController(t)
	defer client.Close()

	tenant := createTestTenantForDepartment(t, client)

	tests := []struct {
		name string
	}{
		{
			name: "获取部门树",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/departments/tree", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
