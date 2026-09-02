// Package ticket_notification 是工单通知域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_notification_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketNotificationService 承载，本包只做参数解析与响应封装。
package ticket_notification

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 工单通知 HTTP handler
type Handler struct {
	notificationService *service.TicketNotificationService
	logger              *zap.SugaredLogger
}

// NewHandler 创建工单通知 handler 实例
func NewHandler(notificationService *service.TicketNotificationService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		notificationService: notificationService,
		logger:              logger,
	}
}

// tenantUserID 提取租户和用户 ID
func tenantUserID(c *gin.Context) (tenantID, userID int) {
	return c.GetInt("tenant_id"), c.GetInt("user_id")
}

// pathID 提取路径参数 ID
func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的ID参数")
		return 0, false
	}
	return id, true
}

// ListTicketNotifications 获取工单通知列表
func (h *Handler) ListTicketNotifications(c *gin.Context) {
	ticketID, ok := pathID(c)
	if !ok {
		return
	}

	tenantID, _ := tenantUserID(c)
	notifications, err := h.notificationService.ListTicketNotifications(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket notifications", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ListTicketNotificationsResponse{
		Notifications: notifications,
		Total:         len(notifications),
	})
}

// SendTicketNotification 发送工单通知
func (h *Handler) SendTicketNotification(c *gin.Context) {
	ticketID, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.SendTicketNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, _ := tenantUserID(c)
	err := h.notificationService.SendNotification(c.Request.Context(), ticketID, &req, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to send ticket notification", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// ListUserNotifications 获取用户通知列表
func (h *Handler) ListUserNotifications(c *gin.Context) {
	userID, tenantID := tenantUserID(c)
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

	notifications, total, err := h.notificationService.ListUserNotifications(c.Request.Context(), userID, tenantID, page, pageSize, read)
	if err != nil {
		h.logger.Errorw("Failed to list user notifications", "error", err, "user_id", userID, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"pageSize":      pageSize,
	})
}

// MarkNotificationRead 标记通知为已读
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	notificationID, ok := pathID(c)
	if !ok {
		return
	}

	userID, tenantID := tenantUserID(c)
	err := h.notificationService.MarkNotificationRead(c.Request.Context(), notificationID, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to mark notification as read", "error", err, "notification_id", notificationID, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// MarkAllNotificationsRead 标记所有通知为已读
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID, tenantID := tenantUserID(c)
	err := h.notificationService.MarkAllNotificationsRead(c.Request.Context(), userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to mark all notifications as read", "error", err, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetNotificationPreferences 获取用户通知偏好
func (h *Handler) GetNotificationPreferences(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的用户ID")
		return
	}

	currentUserID := c.GetInt("user_id")
	if userID != currentUserID {
		common.Fail(c, common.ForbiddenCode, "只能查看自己的通知偏好")
		return
	}

	preferences, err := h.notificationService.GetUserNotificationPreferences(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to get notification preferences", "error", err, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, preferences)
}

// UpdateNotificationPreferences 更新用户通知偏好
func (h *Handler) UpdateNotificationPreferences(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的用户ID")
		return
	}

	currentUserID := c.GetInt("user_id")
	if userID != currentUserID {
		common.Fail(c, common.ForbiddenCode, "只能更新自己的通知偏好")
		return
	}

	var req dto.UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	preferences, err := h.notificationService.UpdateUserNotificationPreferences(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Errorw("Failed to update notification preferences", "error", err, "user_id", userID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, preferences)
}
