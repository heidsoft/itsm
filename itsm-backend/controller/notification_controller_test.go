package controller

import (
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 重写原有弱测试：注册真实路由、注入租户/用户上下文、断言响应。
// 覆盖通知的创建（参数校验 + 持久化）与读取（租户作用域）主链路。
func setupNotificationController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "notification_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)

	tenantID, userID := seedTenantUser(t, client)
	svc := service.NewNotificationService(client)
	ctrl := NewNotificationController(svc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(tenantID, userID))
	r.GET("/api/v1/notifications", ctrl.GetNotifications)
	r.GET("/api/v1/notifications/unread-count", ctrl.GetUnreadCount)
	r.PUT("/api/v1/notifications/:id/read", ctrl.MarkNotificationRead)
	r.PUT("/api/v1/notifications/read-all", ctrl.MarkAllNotificationsRead)
	r.DELETE("/api/v1/notifications/:id", ctrl.DeleteNotification)
	r.POST("/api/v1/notifications", ctrl.CreateNotification)
	return r, client, tenantID, userID
}

func TestNotificationController_CreateNotification(t *testing.T) {
	r, _, tenantID, userID := setupNotificationController(t)

	t.Run("成功创建通知", func(t *testing.T) {
		body := dto.CreateNotificationRequest{
			Title:    "部署完成",
			Message:  "生产环境已发布 v1.2",
			Type:     "success",
			UserID:   userID,
			TenantID: tenantID,
		}
		resp := doReq(t, r, "POST", "/api/v1/notifications", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "部署完成", data["title"])
	})

	t.Run("缺少标题应返回参数错误", func(t *testing.T) {
		body := dto.CreateNotificationRequest{
			Message: "x", Type: "info", UserID: userID, TenantID: tenantID,
		}
		resp := doReq(t, r, "POST", "/api/v1/notifications", body, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("类型非法应返回参数错误", func(t *testing.T) {
		body := dto.CreateNotificationRequest{
			Title: "x", Message: "x", Type: "bogus", UserID: userID, TenantID: tenantID,
		}
		resp := doReq(t, r, "POST", "/api/v1/notifications", body, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestNotificationController_GetNotifications(t *testing.T) {
	r, _, _, _ := setupNotificationController(t)

	t.Run("列表返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/notifications", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
		data := resp.Data.(map[string]interface{})
		assert.Contains(t, data, "notifications")
		assert.Contains(t, data, "total")
	})
}

func TestNotificationController_GetUnreadCount(t *testing.T) {
	r, _, _, _ := setupNotificationController(t)
	resp := doReq(t, r, "GET", "/api/v1/notifications/unread-count", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestNotificationController_MarkAllRead(t *testing.T) {
	r, _, _, _ := setupNotificationController(t)
	resp := doReq(t, r, "PUT", "/api/v1/notifications/read-all", nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}

func TestNotificationController_DeleteNotification(t *testing.T) {
	r, _, tenantID, userID := setupNotificationController(t)

	created := doReq(t, r, "POST", "/api/v1/notifications", dto.CreateNotificationRequest{
		Title: "待删除通知", Message: "m", Type: "warning", UserID: userID, TenantID: tenantID,
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	resp := doReq(t, r, "DELETE", "/api/v1/notifications/"+strconv.Itoa(id), nil, false)
	assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
}
