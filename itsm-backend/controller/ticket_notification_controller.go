package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketNotificationController struct {
	notificationService *service.TicketNotificationService
	logger              *zap.SugaredLogger
}

func NewTicketNotificationController(notificationService *service.TicketNotificationService, logger *zap.SugaredLogger) *TicketNotificationController {
	return &TicketNotificationController{
		notificationService: notificationService,
		logger:              logger,
	}
}

// ListTicketNotifications 获取工单通知列表
// GET /api/v1/tickets/:id/notifications
func (tnc *TicketNotificationController) ListTicketNotifications(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	notifications, err := tnc.notificationService.ListTicketNotifications(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tnc.logger.Errorw("Failed to list ticket notifications", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListTicketNotificationsResponse{
		Notifications: notifications,
		Total:         len(notifications),
	})
}

// SendTicketNotification 发送工单通知
// POST /api/v1/tickets/:id/notifications
func (tnc *TicketNotificationController) SendTicketNotification(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req dto.SendTicketNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = tnc.notificationService.SendNotification(c.Request.Context(), ticketID, &req, tenantID)
	if err != nil {
		tnc.logger.Errorw("Failed to send ticket notification", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// ListUserNotifications 获取用户通知列表
// GET /api/v1/notifications
func (tnc *TicketNotificationController) ListUserNotifications(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	var read *bool
	if readStr := c.Query("read"); readStr != "" {
		readVal := readStr == "true"
		read = &readVal
	}

	notifications, total, err := tnc.notificationService.ListUserNotifications(
		c.Request.Context(),
		userID,
		tenantID,
		page,
		pageSize,
		read,
	)
	if err != nil {
		tnc.logger.Errorw("Failed to list user notifications", "error", err, "user_id", userID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// MarkNotificationRead 标记通知为已读
// PUT /api/v1/notifications/:id/read
func (tnc *TicketNotificationController) MarkNotificationRead(c *gin.Context) {
	notificationID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的通知ID")
		return
	}

	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err = tnc.notificationService.MarkNotificationRead(c.Request.Context(), notificationID, userID, tenantID)
	if err != nil {
		tnc.logger.Errorw("Failed to mark notification as read", "error", err, "notification_id", notificationID, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// MarkAllNotificationsRead 标记所有通知为已读
// PUT /api/v1/notifications/read-all
func (tnc *TicketNotificationController) MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID := c.GetInt("tenant_id")

	err := tnc.notificationService.MarkAllNotificationsRead(c.Request.Context(), userID, tenantID)
	if err != nil {
		tnc.logger.Errorw("Failed to mark all notifications as read", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetNotificationPreferences 获取用户通知偏好
// GET /api/v1/users/:id/notification-preferences
func (tnc *TicketNotificationController) GetNotificationPreferences(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的用户ID")
		return
	}

	// 只能查看自己的偏好
	currentUserID := c.GetInt("user_id")
	if userID != currentUserID {
		common.Fail(c, common.ForbiddenCode, "只能查看自己的通知偏好")
		return
	}

	preferences, err := tnc.notificationService.GetUserNotificationPreferences(c.Request.Context(), userID)
	if err != nil {
		tnc.logger.Errorw("Failed to get notification preferences", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, preferences)
}

// UpdateNotificationPreferences 更新用户通知偏好
// PUT /api/v1/users/:id/notification-preferences
func (tnc *TicketNotificationController) UpdateNotificationPreferences(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的用户ID")
		return
	}

	// 只能更新自己的偏好
	currentUserID := c.GetInt("user_id")
	if userID != currentUserID {
		common.Fail(c, common.ForbiddenCode, "只能更新自己的通知偏好")
		return
	}

	var req dto.UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	preferences, err := tnc.notificationService.UpdateUserNotificationPreferences(c.Request.Context(), userID, &req)
	if err != nil {
		tnc.logger.Errorw("Failed to update notification preferences", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, preferences)
}

