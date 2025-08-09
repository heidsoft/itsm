package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/notification"
	"itsm-backend/ent/user"

	"github.com/gin-gonic/gin"
)

type SimpleNotificationService struct {
	client *ent.Client
}

func NewSimpleNotificationService(client *ent.Client) *SimpleNotificationService {
	return &SimpleNotificationService{
		client: client,
	}
}

// GetNotifications 获取通知列表
func (s *SimpleNotificationService) GetNotifications(ctx context.Context, userID, tenantID int) ([]dto.Notification, error) {
	notifications, err := s.client.Notification.Query().
		Where(notification.UserID(userID)).
		Where(notification.TenantID(tenantID)).
		Order(ent.Desc(notification.FieldCreatedAt)).
		Limit(20).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取通知列表失败: %w", err)
	}

	// 转换为DTO
	notificationDTOs := make([]dto.Notification, len(notifications))
	for i, n := range notifications {
		notificationDTOs[i] = dto.Notification{
			ID:         n.ID,
			Title:      n.Title,
			Message:    n.Message,
			Type:       n.Type,
			Read:       n.Read,
			ActionURL:  &n.ActionURL,
			ActionText: &n.ActionText,
			UserID:     n.UserID,
			TenantID:   n.TenantID,
			CreatedAt:  n.CreatedAt,
			UpdatedAt:  n.UpdatedAt,
		}
	}

	return notificationDTOs, nil
}

// MarkNotificationRead 标记通知为已读
func (s *SimpleNotificationService) MarkNotificationRead(ctx context.Context, notificationID, userID, tenantID int) error {
	// 验证通知是否存在且属于该用户
	_, err := s.client.Notification.Query().
		Where(notification.ID(notificationID)).
		Where(notification.UserID(userID)).
		Where(notification.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("通知不存在或无权限: %w", err)
	}

	// 更新为已读
	_, err = s.client.Notification.UpdateOneID(notificationID).
		SetRead(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("标记已读失败: %w", err)
	}

	return nil
}

// MarkAllNotificationsRead 标记所有通知为已读
func (s *SimpleNotificationService) MarkAllNotificationsRead(ctx context.Context, userID, tenantID int) error {
	_, err := s.client.Notification.Update().
		Where(notification.UserID(userID)).
		Where(notification.TenantID(tenantID)).
		Where(notification.Read(false)).
		SetRead(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("标记全部已读失败: %w", err)
	}

	return nil
}

// DeleteNotification 删除通知
func (s *SimpleNotificationService) DeleteNotification(ctx context.Context, notificationID, userID, tenantID int) error {
	// 验证通知是否存在且属于该用户
	_, err := s.client.Notification.Query().
		Where(notification.ID(notificationID)).
		Where(notification.UserID(userID)).
		Where(notification.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("通知不存在或无权限: %w", err)
	}

	// 删除通知
	err = s.client.Notification.DeleteOneID(notificationID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除通知失败: %w", err)
	}

	return nil
}

// GetUnreadCount 获取未读通知数量
func (s *SimpleNotificationService) GetUnreadCount(ctx context.Context, userID, tenantID int) (int, error) {
	count, err := s.client.Notification.Query().
		Where(notification.UserID(userID)).
		Where(notification.TenantID(tenantID)).
		Where(notification.Read(false)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("获取未读数量失败: %w", err)
	}

	return count, nil
}

// CreateNotification 创建通知
func (s *SimpleNotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.Notification, error) {
	// 验证用户是否存在
	_, err := s.client.User.Query().Where(user.ID(req.UserID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 创建通知
	notification, err := s.client.Notification.Create().
		SetTitle(req.Title).
		SetMessage(req.Message).
		SetType(req.Type).
		SetUserID(req.UserID).
		SetTenantID(req.TenantID).
		SetNillableActionURL(req.ActionURL).
		SetNillableActionText(req.ActionText).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建通知失败: %w", err)
	}

	return &dto.Notification{
		ID:         notification.ID,
		Title:      notification.Title,
		Message:    notification.Message,
		Type:       notification.Type,
		Read:       notification.Read,
		ActionURL:  &notification.ActionURL,
		ActionText: &notification.ActionText,
		UserID:     notification.UserID,
		TenantID:   notification.TenantID,
		CreatedAt:  notification.CreatedAt,
		UpdatedAt:  notification.UpdatedAt,
	}, nil
}

// GetCurrentUserID 从上下文获取当前用户ID
func (s *SimpleNotificationService) GetCurrentUserID(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("用户ID不存在")
	}
	return userID.(int), nil
}

// GetCurrentTenantID 从上下文获取当前租户ID
func (s *SimpleNotificationService) GetCurrentTenantID(c *gin.Context) (int, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("租户ID不存在")
	}
	return tenantID.(int), nil
}
