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

type NotificationService struct {
	client *ent.Client
}

func NewNotificationService(client *ent.Client) *NotificationService {
	return &NotificationService{
		client: client,
	}
}

// CreateNotification 创建通知
func (s *NotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.Notification, error) {
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

	return s.convertToDTO(notification), nil
}

// GetNotifications 获取通知列表
func (s *NotificationService) GetNotifications(ctx context.Context, req *dto.GetNotificationsRequest) (*dto.NotificationListResponse, error) {
	query := s.client.Notification.Query().
		Where(notification.UserID(req.UserID)).
		Where(notification.TenantID(req.TenantID))

	// 添加过滤条件
	if req.Type != "" {
		query = query.Where(notification.Type(req.Type))
	}
	if req.Read != nil {
		query = query.Where(notification.Read(*req.Read))
	}

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取通知总数失败: %w", err)
	}

	// 分页查询
	notifications, err := query.
		Order(ent.Desc(notification.FieldCreatedAt)).
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取通知列表失败: %w", err)
	}

	// 转换为DTO
	notificationDTOs := make([]dto.Notification, len(notifications))
	for i, n := range notifications {
		notificationDTOs[i] = *s.convertToDTO(n)
	}

	return &dto.NotificationListResponse{
		Notifications: notificationDTOs,
		Total:         total,
		Page:          req.Page,
		Size:          req.Size,
	}, nil
}

// MarkNotificationRead 标记通知为已读
func (s *NotificationService) MarkNotificationRead(ctx context.Context, req *dto.MarkNotificationReadRequest) error {
	// 验证通知是否存在且属于该用户
	_, err := s.client.Notification.Query().
		Where(notification.ID(req.NotificationID)).
		Where(notification.UserID(req.UserID)).
		Where(notification.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("通知不存在或无权限: %w", err)
	}

	// 更新为已读
	_, err = s.client.Notification.UpdateOneID(req.NotificationID).
		SetRead(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("标记已读失败: %w", err)
	}

	return nil
}

// MarkAllNotificationsRead 标记所有通知为已读
func (s *NotificationService) MarkAllNotificationsRead(ctx context.Context, req *dto.MarkAllNotificationsReadRequest) error {
	_, err := s.client.Notification.Update().
		Where(notification.UserID(req.UserID)).
		Where(notification.TenantID(req.TenantID)).
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
func (s *NotificationService) DeleteNotification(ctx context.Context, req *dto.DeleteNotificationRequest) error {
	// 验证通知是否存在且属于该用户
	_, err := s.client.Notification.Query().
		Where(notification.ID(req.NotificationID)).
		Where(notification.UserID(req.UserID)).
		Where(notification.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("通知不存在或无权限: %w", err)
	}

	// 删除通知
	err = s.client.Notification.DeleteOneID(req.NotificationID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除通知失败: %w", err)
	}

	return nil
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID, tenantID int) (int, error) {
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

// CreateSystemNotification 创建系统通知
func (s *NotificationService) CreateSystemNotification(ctx context.Context, title, message, notificationType string, userIDs []int, tenantID int) error {
	notifications := make([]*ent.NotificationCreate, len(userIDs))
	for i, userID := range userIDs {
		notifications[i] = s.client.Notification.Create().
			SetTitle(title).
			SetMessage(message).
			SetType(notificationType).
			SetUserID(userID).
			SetTenantID(tenantID)
	}

	_, err := s.client.Notification.CreateBulk(notifications...).Save(ctx)
	if err != nil {
		return fmt.Errorf("创建系统通知失败: %w", err)
	}

	return nil
}

// convertToDTO 转换为DTO
func (s *NotificationService) convertToDTO(n *ent.Notification) *dto.Notification {
	notification := &dto.Notification{
		ID:        n.ID,
		Title:     n.Title,
		Message:   n.Message,
		Type:      n.Type,
		Read:      n.Read,
		UserID:    n.UserID,
		TenantID:  n.TenantID,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}

	// 处理可选字段
	if n.ActionURL != "" {
		notification.ActionURL = &n.ActionURL
	}
	if n.ActionText != "" {
		notification.ActionText = &n.ActionText
	}

	return notification
}

// GetCurrentUserID 从上下文获取当前用户ID
func (s *NotificationService) GetCurrentUserID(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("用户ID不存在")
	}
	return userID.(int), nil
}

// GetCurrentTenantID 从上下文获取当前租户ID
func (s *NotificationService) GetCurrentTenantID(c *gin.Context) (int, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("租户ID不存在")
	}
	return tenantID.(int), nil
}
