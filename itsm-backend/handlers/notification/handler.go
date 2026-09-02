package notification

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP handlers for notification domain
type Handler struct {
	notificationService        *service.NotificationService
	notificationPreferenceService *service.NotificationPreferenceService
	logger                     *zap.SugaredLogger
}

// NewHandler creates a new notification handler
func NewHandler(
	notificationService *service.NotificationService,
	notificationPreferenceService *service.NotificationPreferenceService,
	logger *zap.SugaredLogger,
) *Handler {
	return &Handler{
		notificationService:        notificationService,
		notificationPreferenceService: notificationPreferenceService,
		logger:                     logger,
	}
}

// GetNotifications 获取通知列表
// @Summary 获取通知列表
// @Description 获取当前用户的所有通知，支持分页和状态过滤
// @Tags 通知管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param is_read query bool false "已读状态"
// @Success 200 {object} common.Response
// @Router /api/v1/notifications [get]
func (h *Handler) GetNotifications(ctx *gin.Context) {
	var req dto.GetNotificationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	req.UserID = userID
	req.TenantID = tenantID

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	} else if req.Size > 100 {
		req.Size = 100
	}

	result, err := h.notificationService.GetNotifications(ctx, &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取通知失败: "+err.Error())
		return
	}

	common.Success(ctx, result)
}

// MarkNotificationRead 标记通知为已读
// @Summary 标记通知为已读
// @Description 将指定通知标记为已读状态
// @Tags 通知管理
// @Accept json
// @Produce json
// @Param id path int true "通知ID"
// @Success 200 {object} common.Response
// @Router /api/v1/notifications/{id}/read [post]
func (h *Handler) MarkNotificationRead(ctx *gin.Context) {
	notificationID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "通知ID格式错误: "+err.Error())
		return
	}

	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	req := &dto.MarkNotificationReadRequest{
		NotificationID: notificationID,
		UserID:         userID,
		TenantID:       tenantID,
	}

	err = h.notificationService.MarkNotificationRead(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记已读失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "标记已读成功"})
}

// MarkAllNotificationsRead 标记所有通知为已读
// @Summary 标记所有通知为已读
// @Description 将当前用户的所有未读通知标记为已读
// @Tags 通知管理
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/notifications/read-all [post]
func (h *Handler) MarkAllNotificationsRead(ctx *gin.Context) {
	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	req := &dto.MarkAllNotificationsReadRequest{
		UserID:   userID,
		TenantID: tenantID,
	}

	err = h.notificationService.MarkAllNotificationsRead(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记全部已读失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "标记全部已读成功"})
}

// DeleteNotification 删除通知
// @Summary 删除通知
// @Description 删除指定的通知
// @Tags 通知管理
// @Accept json
// @Produce json
// @Param id path int true "通知ID"
// @Success 200 {object} common.Response
// @Router /api/v1/notifications/{id} [delete]
func (h *Handler) DeleteNotification(ctx *gin.Context) {
	notificationID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "通知ID格式错误: "+err.Error())
		return
	}

	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	req := &dto.DeleteNotificationRequest{
		NotificationID: notificationID,
		UserID:         userID,
		TenantID:       tenantID,
	}

	err = h.notificationService.DeleteNotification(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除通知失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除通知成功"})
}

// GetUnreadCount 获取未读通知数量
func (h *Handler) GetUnreadCount(ctx *gin.Context) {
	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	count, err := h.notificationService.GetUnreadCount(ctx, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取未读数量失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"count": count})
}

// CreateNotification 创建通知（管理员功能）
func (h *Handler) CreateNotification(ctx *gin.Context) {
	var req dto.CreateNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "无法确认当前租户")
		return
	}
	req.TenantID = tenantID
	notification, err := h.notificationService.CreateNotification(ctx, &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建通知失败: "+err.Error())
		return
	}

	common.Success(ctx, notification)
}

// MarkNotificationsRead 批量标记通知为已读
func (h *Handler) MarkNotificationsRead(ctx *gin.Context) {
	var req dto.BatchNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请选择要标记的通知（最多100条）")
		return
	}
	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录")
		return
	}
	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "无法确认当前租户")
		return
	}
	count, err := h.notificationService.MarkNotificationsRead(ctx, req.NotificationIDs, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "批量标记已读失败")
		return
	}
	common.Success(ctx, gin.H{"updated": count})
}

// DeleteNotifications 批量删除通知
func (h *Handler) DeleteNotifications(ctx *gin.Context) {
	var req dto.BatchNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请选择要删除的通知（最多100条）")
		return
	}
	userID, err := h.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录")
		return
	}
	tenantID, err := h.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "无法确认当前租户")
		return
	}
	count, err := h.notificationService.DeleteNotifications(ctx, req.NotificationIDs, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "批量删除通知失败")
		return
	}
	common.Success(ctx, gin.H{"deleted": count})
}

// ListPreferences 获取用户的所有通知偏好
func (h *Handler) ListPreferences(cxt *gin.Context) {
	h.logger.Infow("ListPreferences called", "user_id", cxt.GetInt("user_id"))
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	prefs, err := h.notificationPreferenceService.GetUserPreferences(cxt.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "获取通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{
		"preferences": prefs,
		"event_types": dto.ListNotificationEventTypes(),
	})
}

// GetPreference 获取用户特定事件类型的通知偏好
func (h *Handler) GetPreference(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	eventType := cxt.Param("event_type")
	if eventType == "" {
		common.ParamError(cxt, "事件类型不能为空")
		return
	}

	pref, err := h.notificationPreferenceService.GetUserPreferenceByEventType(cxt.Request.Context(), userID, tenantID, eventType)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "获取通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, pref)
}

// CreateOrUpdatePreference 创建或更新通知偏好
func (h *Handler) CreateOrUpdatePreference(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.NotificationPreferenceRequest
	if err := cxt.ShouldBindJSON(&req); err != nil {
		common.ParamError(cxt, "参数错误: "+err.Error())
		return
	}

	pref, err := h.notificationPreferenceService.CreateOrUpdatePreference(cxt.Request.Context(), userID, tenantID, &req)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "保存通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, pref)
}

// BulkUpdatePreferences 批量更新通知偏好
func (h *Handler) BulkUpdatePreferences(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.BulkNotificationPreferenceRequest
	if err := cxt.ShouldBindJSON(&req); err != nil {
		common.ParamError(cxt, "参数错误: "+err.Error())
		return
	}

	prefs, err := h.notificationPreferenceService.BulkUpdatePreferences(cxt.Request.Context(), userID, tenantID, &req)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "批量更新通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{
		"preferences": prefs,
	})
}

// DeletePreference 删除通知偏好
func (h *Handler) DeletePreference(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	eventType := cxt.Param("event_type")
	if eventType == "" {
		common.ParamError(cxt, "事件类型不能为空")
		return
	}

	err := h.notificationPreferenceService.DeletePreference(cxt.Request.Context(), userID, tenantID, eventType)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "删除通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"deleted": true})
}

// ResetPreferences 重置为默认偏好
func (h *Handler) ResetPreferences(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	err := h.notificationPreferenceService.ResetToDefaults(cxt.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "重置通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"reset": true})
}

// InitializeDefaultPreferences 初始化默认通知偏好
func (h *Handler) InitializeDefaultPreferences(cxt *gin.Context) {
	userID := cxt.GetInt("user_id")
	tenantID := cxt.GetInt("tenant_id")
	if userID == 0 || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	err := h.notificationPreferenceService.InitializeDefaultPreferences(cxt.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "初始化通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"initialized": true})
}

// ListEventTypes 获取所有通知事件类型
func (h *Handler) ListEventTypes(cxt *gin.Context) {
	common.Success(cxt, dto.ListNotificationEventTypes())
}
