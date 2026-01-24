package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NotificationPreferenceController 通知偏好控制器
type NotificationPreferenceController struct {
	preferenceService *service.NotificationPreferenceService
	logger            *zap.SugaredLogger
}

// NewNotificationPreferenceController 创建通知偏好控制器
func NewNotificationPreferenceController(preferenceService *service.NotificationPreferenceService, logger *zap.SugaredLogger) *NotificationPreferenceController {
	return &NotificationPreferenceController{
		preferenceService: preferenceService,
		logger:            logger,
	}
}

// ListPreferences 获取用户的所有通知偏好
func (c *NotificationPreferenceController) ListPreferences(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	prefs, err := c.preferenceService.GetUserPreferences(cxt.Request.Context(), userID, tenantID)
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
func (c *NotificationPreferenceController) GetPreference(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	eventType := cxt.Param("event_type")
	if eventType == "" {
		common.ParamError(cxt, "事件类型不能为空")
		return
	}

	pref, err := c.preferenceService.GetUserPreferenceByEventType(cxt.Request.Context(), userID, tenantID, eventType)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "获取通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, pref)
}

// CreateOrUpdatePreference 创建或更新通知偏好
func (c *NotificationPreferenceController) CreateOrUpdatePreference(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.NotificationPreferenceRequest
	if err := cxt.ShouldBindJSON(&req); err != nil {
		common.ParamError(cxt, "参数错误: "+err.Error())
		return
	}

	pref, err := c.preferenceService.CreateOrUpdatePreference(cxt.Request.Context(), userID, tenantID, &req)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "保存通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, pref)
}

// BulkUpdatePreferences 批量更新通知偏好
func (c *NotificationPreferenceController) BulkUpdatePreferences(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.BulkNotificationPreferenceRequest
	if err := cxt.ShouldBindJSON(&req); err != nil {
		common.ParamError(cxt, "参数错误: "+err.Error())
		return
	}

	prefs, err := c.preferenceService.BulkUpdatePreferences(cxt.Request.Context(), userID, tenantID, &req)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "批量更新通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{
		"preferences": prefs,
	})
}

// DeletePreference 删除通知偏好
func (c *NotificationPreferenceController) DeletePreference(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	eventType := cxt.Param("event_type")
	if eventType == "" {
		common.ParamError(cxt, "事件类型不能为空")
		return
	}

	err = c.preferenceService.DeletePreference(cxt.Request.Context(), userID, tenantID, eventType)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "删除通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"deleted": true})
}

// ResetPreferences 重置为默认偏好
func (c *NotificationPreferenceController) ResetPreferences(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	err = c.preferenceService.ResetToDefaults(cxt.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "重置通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"reset": true})
}

// InitializeDefaultPreferences 初始化默认通知偏好
func (c *NotificationPreferenceController) InitializeDefaultPreferences(cxt *gin.Context) {
	userID, err := middleware.GetUserID(cxt)
	if err != nil || userID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	tenantID, err := middleware.GetTenantID(cxt)
	if err != nil || tenantID == 0 {
		common.Fail(cxt, common.UnauthorizedCode, "未授权访问")
		return
	}

	err = c.preferenceService.InitializeDefaultPreferences(cxt.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(cxt, common.InternalErrorCode, "初始化通知偏好失败: "+err.Error())
		return
	}

	common.Success(cxt, gin.H{"initialized": true})
}

// ListEventTypes 获取所有通知事件类型
func (c *NotificationPreferenceController) ListEventTypes(cxt *gin.Context) {
	common.Success(cxt, dto.ListNotificationEventTypes())
}

// GetUserIDFromContext 从上下文获取用户ID（兼容处理）
func getUserIDFromContext(c *gin.Context) (int, error) {
	// 尝试从 JWT 声明中获取
	userID, exists := c.Get("user_id")
	if exists {
		if id, ok := userID.(int); ok {
			return id, nil
		}
		if idStr, ok := userID.(string); ok {
			return strconv.Atoi(idStr)
		}
	}

	// 尝试从查询参数获取（临时方案）
	if idStr := c.Query("user_id"); idStr != "" {
		return strconv.Atoi(idStr)
	}

	return 0, nil
}
