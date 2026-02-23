package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupTestNotificationController(t *testing.T) (*gin.Engine, *ent.Client, *NotificationController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建服务
	notificationService := service.NewNotificationService(client)

	// 创建控制器
	notificationController := NewNotificationController(notificationService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r, client, notificationController
}

func TestNotificationController_GetNotifications(t *testing.T) {
	r, _, _ := setupTestNotificationController(t)

	tests := []struct {
		name string
	}{
		{
			name: "获取通知列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/notifications", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", 1)
			c.Set("user_id", 1)

			r.ServeHTTP(w, req)
		})
	}
}
