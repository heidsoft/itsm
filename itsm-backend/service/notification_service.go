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
	// 验证用户是否存在，且必须属于同一租户（防止越权给别租户 user 发通知）
	if req.TenantID <= 0 {
		return nil, fmt.Errorf("tenant_id 不能为空")
	}
	_, err := s.client.User.Query().
		Where(user.ID(req.UserID)).
		Where(user.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("用户不存在或不属于当前租户: %w", err)
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
	if req.UserID <= 0 || req.TenantID <= 0 {
		return nil, fmt.Errorf("用户或租户信息无效")
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	} else if req.Size > 100 {
		req.Size = 100
	}
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
	_, err := s.client.Notification.UpdateOneID(req.NotificationID).
		Where(notification.UserID(req.UserID), notification.TenantID(req.TenantID)).
		SetRead(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("通知不存在或无权限")
		}
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
	err := s.client.Notification.DeleteOneID(req.NotificationID).
		Where(notification.UserID(req.UserID), notification.TenantID(req.TenantID)).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("通知不存在或无权限")
		}
		return fmt.Errorf("删除通知失败: %w", err)
	}

	return nil
}

// MarkNotificationsRead 批量标记已读，只影响当前租户的当前用户。
func (s *NotificationService) MarkNotificationsRead(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error) {
	ids := uniquePositiveIDs(notificationIDs)
	if len(ids) == 0 {
		return 0, fmt.Errorf("通知ID列表不能为空")
	}
	count, err := s.client.Notification.Update().
		Where(notification.IDIn(ids...), notification.UserID(userID), notification.TenantID(tenantID), notification.Read(false)).
		SetRead(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("批量标记已读失败: %w", err)
	}
	return count, nil
}

// DeleteNotifications 批量删除，只影响当前租户的当前用户。
func (s *NotificationService) DeleteNotifications(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error) {
	ids := uniquePositiveIDs(notificationIDs)
	if len(ids) == 0 {
		return 0, fmt.Errorf("通知ID列表不能为空")
	}
	count, err := s.client.Notification.Delete().
		Where(notification.IDIn(ids...), notification.UserID(userID), notification.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("批量删除通知失败: %w", err)
	}
	return count, nil
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
	ids := uniquePositiveIDs(userIDs)
	if tenantID <= 0 || len(ids) == 0 {
		return fmt.Errorf("租户和收件人不能为空")
	}
	validCount, err := s.client.User.Query().Where(user.IDIn(ids...), user.TenantID(tenantID), user.Active(true)).Count(ctx)
	if err != nil {
		return fmt.Errorf("校验通知收件人失败: %w", err)
	}
	if validCount != len(ids) {
		return fmt.Errorf("部分收件人不存在、不属于当前租户或已停用")
	}
	notifications := make([]*ent.NotificationCreate, len(ids))
	for i, userID := range ids {
		notifications[i] = s.client.Notification.Create().
			SetTitle(title).
			SetMessage(message).
			SetType(notificationType).
			SetUserID(userID).
			SetTenantID(tenantID)
	}

	_, err = s.client.Notification.CreateBulk(notifications...).Save(ctx)
	if err != nil {
		return fmt.Errorf("创建系统通知失败: %w", err)
	}

	return nil
}

func uniquePositiveIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
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
	id, ok := userID.(int)
	if !ok || id <= 0 {
		return 0, fmt.Errorf("用户ID无效")
	}
	return id, nil
}

// GetCurrentTenantID 从上下文获取当前租户ID
func (s *NotificationService) GetCurrentTenantID(c *gin.Context) (int, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("租户ID不存在")
	}
	id, ok := tenantID.(int)
	if !ok || id <= 0 {
		return 0, fmt.Errorf("租户ID无效")
	}
	return id, nil
}
