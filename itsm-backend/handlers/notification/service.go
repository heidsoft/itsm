package notification

import (
	"context"

	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

// NotificationService 定义通知域业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 service.NotificationService 提供。
type NotificationService interface {
	CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.Notification, error)
	GetNotifications(ctx context.Context, req *dto.GetNotificationsRequest) (*dto.NotificationListResponse, error)
	MarkNotificationRead(ctx context.Context, req *dto.MarkNotificationReadRequest) error
	MarkAllNotificationsRead(ctx context.Context, req *dto.MarkAllNotificationsReadRequest) error
	DeleteNotification(ctx context.Context, req *dto.DeleteNotificationRequest) error
	MarkNotificationsRead(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error)
	DeleteNotifications(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error)
	GetUnreadCount(ctx context.Context, userID, tenantID int) (int, error)
	GetCurrentUserID(c *gin.Context) (int, error)
	GetCurrentTenantID(c *gin.Context) (int, error)
}

// NotificationPreferenceService 定义通知偏好域业务接口（依赖倒置）。
type NotificationPreferenceService interface {
	GetUserPreferences(ctx context.Context, userID, tenantID int) ([]*dto.NotificationPreferenceResponse, error)
	GetUserPreferenceByEventType(ctx context.Context, userID, tenantID int, eventType string) (*dto.NotificationPreferenceResponse, error)
	CreateOrUpdatePreference(ctx context.Context, userID, tenantID int, req *dto.NotificationPreferenceRequest) (*dto.NotificationPreferenceResponse, error)
	BulkUpdatePreferences(ctx context.Context, userID, tenantID int, req *dto.BulkNotificationPreferenceRequest) ([]*dto.NotificationPreferenceResponse, error)
	DeletePreference(ctx context.Context, userID, tenantID int, eventType string) error
	ResetToDefaults(ctx context.Context, userID, tenantID int) error
	InitializeDefaultPreferences(ctx context.Context, userID, tenantID int) error
}