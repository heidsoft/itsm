package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupTestProjectController(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建控制器
	_ = NewProjectController(client)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r
}

func TestProjectController_ListProjects(t *testing.T) {
	r := setupTestProjectController(t)

	tests := []struct {
		name string
	}{
		{
			name: "获取项目列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/projects", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", 1)

			r.ServeHTTP(w, req)
		})
	}
}

func TestProjectController_CreateProject(t *testing.T) {
	r := setupTestProjectController(t)

	tests := []struct {
		name string
	}{
		{
			name: "创建项目",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/projects", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", 1)

			r.ServeHTTP(w, req)
		})
	}
}
