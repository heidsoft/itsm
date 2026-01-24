package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/notificationpreference"

	"go.uber.org/zap"
)

// NotificationPreferenceService 通知偏好服务
type NotificationPreferenceService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewNotificationPreferenceService 创建通知偏好服务
func NewNotificationPreferenceService(client *ent.Client, logger *zap.SugaredLogger) *NotificationPreferenceService {
	return &NotificationPreferenceService{
		client: client,
		logger: logger,
	}
}

// GetUserPreferences 获取用户的所有通知偏好
func (s *NotificationPreferenceService) GetUserPreferences(ctx context.Context, userID, tenantID int) ([]*dto.NotificationPreferenceResponse, error) {
	prefs, err := s.client.NotificationPreference.Query().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询通知偏好失败: %w", err)
	}

	responses := make([]*dto.NotificationPreferenceResponse, len(prefs))
	for i, p := range prefs {
		responses[i] = s.toPreferenceResponse(p)
	}

	return responses, nil
}

// GetUserPreferenceByEventType 获取用户特定事件类型的通知偏好
func (s *NotificationPreferenceService) GetUserPreferenceByEventType(ctx context.Context, userID, tenantID int, eventType string) (*dto.NotificationPreferenceResponse, error) {
	pref, err := s.client.NotificationPreference.Query().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
			notificationpreference.EventType(eventType),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 返回默认偏好
			return s.getDefaultPreference(userID, tenantID, eventType), nil
		}
		return nil, fmt.Errorf("查询通知偏好失败: %w", err)
	}

	return s.toPreferenceResponse(pref), nil
}

// CreateOrUpdatePreference 创建或更新通知偏好
func (s *NotificationPreferenceService) CreateOrUpdatePreference(ctx context.Context, userID, tenantID int, req *dto.NotificationPreferenceRequest) (*dto.NotificationPreferenceResponse, error) {
	s.logger.Infow("Creating/updating notification preference", "user_id", userID, "event_type", req.EventType)

	// 检查是否已存在
	existing, err := s.client.NotificationPreference.Query().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
			notificationpreference.EventType(req.EventType),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("查询通知偏好失败: %w", err)
	}

	emailEnabled := true
	smsEnabled := false
	inAppEnabled := true
	pushEnabled := false
	frequency := "immediate"
	timezone := "UTC"

	if req.EmailEnabled != nil {
		emailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		smsEnabled = *req.SMSEnabled
	}
	if req.InAppEnabled != nil {
		inAppEnabled = *req.InAppEnabled
	}
	if req.PushEnabled != nil {
		pushEnabled = *req.PushEnabled
	}
	if req.Frequency != "" {
		frequency = req.Frequency
	}
	if req.Timezone != "" {
		timezone = req.Timezone
	}

	if existing != nil {
		// 更新现有偏好
		updated, err := existing.Update().
			SetEmailEnabled(emailEnabled).
			SetSmsEnabled(smsEnabled).
			SetInAppEnabled(inAppEnabled).
			SetPushEnabled(pushEnabled).
			SetFrequency(frequency).
			SetTimezone(timezone).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新通知偏好失败: %w", err)
		}
		return s.toPreferenceResponse(updated), nil
	}

	// 创建新偏好
	created, err := s.client.NotificationPreference.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		SetEventType(req.EventType).
		SetEmailEnabled(emailEnabled).
		SetSmsEnabled(smsEnabled).
		SetInAppEnabled(inAppEnabled).
		SetPushEnabled(pushEnabled).
		SetFrequency(frequency).
		SetTimezone(timezone).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建通知偏好失败: %w", err)
	}

	return s.toPreferenceResponse(created), nil
}

// BulkUpdatePreferences 批量更新通知偏好
func (s *NotificationPreferenceService) BulkUpdatePreferences(ctx context.Context, userID, tenantID int, req *dto.BulkNotificationPreferenceRequest) ([]*dto.NotificationPreferenceResponse, error) {
	s.logger.Infow("Bulk updating notification preferences", "user_id", userID, "count", len(req.Preferences))

	responses := make([]*dto.NotificationPreferenceResponse, 0, len(req.Preferences))

	for _, pref := range req.Preferences {
		resp, err := s.CreateOrUpdatePreference(ctx, userID, tenantID, &pref)
		if err != nil {
			s.logger.Warnw("Failed to update preference", "event_type", pref.EventType, "error", err)
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// DeletePreference 删除通知偏好
func (s *NotificationPreferenceService) DeletePreference(ctx context.Context, userID, tenantID int, eventType string) error {
	s.logger.Infow("Deleting notification preference", "user_id", userID, "event_type", eventType)

	_, err := s.client.NotificationPreference.Delete().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
			notificationpreference.EventType(eventType),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除通知偏好失败: %w", err)
	}

	return nil
}

// ResetToDefaults 重置为默认偏好
func (s *NotificationPreferenceService) ResetToDefaults(ctx context.Context, userID, tenantID int) error {
	s.logger.Infow("Resetting notification preferences to defaults", "user_id", userID)

	// 删除所有现有偏好
	_, err := s.client.NotificationPreference.Delete().
		Where(
			notificationpreference.UserID(userID),
			notificationpreference.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("重置通知偏好失败: %w", err)
	}

	return nil
}

// InitializeDefaultPreferences 初始化默认通知偏好
func (s *NotificationPreferenceService) InitializeDefaultPreferences(ctx context.Context, userID, tenantID int) error {
	s.logger.Infow("Initializing default notification preferences", "user_id", userID)

	defaultPreferences := []dto.NotificationPreferenceRequest{
		{EventType: "ticket_created", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "ticket_assigned", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "ticket_updated", EmailEnabled: boolPtr(false), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "ticket_resolved", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "ticket_closed", EmailEnabled: boolPtr(false), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "sla_warning", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "sla_violated", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "comment_added", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "approval_required", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "mention", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "incident_created", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "incident_escalated", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(true), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(true), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "change_approved", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "change_rejected", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
		{EventType: "problem_identified", EmailEnabled: boolPtr(true), SMSEnabled: boolPtr(false), InAppEnabled: boolPtr(true), PushEnabled: boolPtr(false), Frequency: "immediate", Timezone: "UTC"},
	}

	for _, req := range defaultPreferences {
		_, err := s.CreateOrUpdatePreference(ctx, userID, tenantID, &req)
		if err != nil {
			s.logger.Warnw("Failed to create default preference", "event_type", req.EventType, "error", err)
		}
	}

	return nil
}

func (s *NotificationPreferenceService) toPreferenceResponse(pref *ent.NotificationPreference) *dto.NotificationPreferenceResponse {
	return &dto.NotificationPreferenceResponse{
		ID:              pref.ID,
		UserID:          pref.UserID,
		EventType:       pref.EventType,
		EmailEnabled:    pref.EmailEnabled,
		SmsEnabled:      pref.SmsEnabled,
		InAppEnabled:    pref.InAppEnabled,
		PushEnabled:     pref.PushEnabled,
		Frequency:       pref.Frequency,
		QuietHoursStart: pref.QuietHoursStart,
		QuietHoursEnd:   pref.QuietHoursEnd,
		Timezone:        pref.Timezone,
		CreatedAt:       pref.CreatedAt,
		UpdatedAt:       pref.UpdatedAt,
	}
}

func (s *NotificationPreferenceService) getDefaultPreference(userID, tenantID int, eventType string) *dto.NotificationPreferenceResponse {
	now := time.Now()
	return &dto.NotificationPreferenceResponse{
		ID:           0,
		UserID:       userID,
		EventType:    eventType,
		EmailEnabled: true,
		SmsEnabled:   false,
		InAppEnabled: true,
		PushEnabled:  false,
		Frequency:    "immediate",
		Timezone:     "UTC",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func boolPtr(b bool) *bool {
	return &b
}
