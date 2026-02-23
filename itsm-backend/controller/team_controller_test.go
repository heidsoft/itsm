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

func setupTestTeamController(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建控制器
	_ = NewTeamController(client)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r, client
}

func createTestTenantForTeam(t *testing.T, client *ent.Client) *ent.Tenant {
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TESTTEAM").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return tenant
}

func TestTeamController_ListTeams(t *testing.T) {
	r, client := setupTestTeamController(t)
	defer client.Close()

	tenant := createTestTenantForTeam(t, client)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "成功获取团队列表",
			queryParams: "",
		},
		{
			name:        "带分页参数",
			queryParams: "page=1&pageSize=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/teams"
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

func TestTeamController_CreateTeam(t *testing.T) {
	r, client := setupTestTeamController(t)
	defer client.Close()

	tenant := createTestTenantForTeam(t, client)

	tests := []struct {
		name string
	}{
		{
			name: "创建团队",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/teams", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
