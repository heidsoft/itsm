package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type SimpleNotificationController struct {
	notificationService *service.SimpleNotificationService
}

func NewSimpleNotificationController(notificationService *service.SimpleNotificationService) *SimpleNotificationController {
	return &SimpleNotificationController{
		notificationService: notificationService,
	}
}

// GetNotifications 获取通知列表
func (c *SimpleNotificationController) GetNotifications(ctx *gin.Context) {
	userID, err := c.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := c.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	notifications, err := c.notificationService.GetNotifications(ctx, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取通知失败: "+err.Error())
		return
	}

	common.Success(ctx, notifications)
}

// MarkNotificationRead 标记通知为已读
func (c *SimpleNotificationController) MarkNotificationRead(ctx *gin.Context) {
	notificationID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "通知ID格式错误: "+err.Error())
		return
	}

	userID, err := c.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := c.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	err = c.notificationService.MarkNotificationRead(ctx, notificationID, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记已读失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "标记已读成功"})
}

// MarkAllNotificationsRead 标记所有通知为已读
func (c *SimpleNotificationController) MarkAllNotificationsRead(ctx *gin.Context) {
	userID, err := c.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := c.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	err = c.notificationService.MarkAllNotificationsRead(ctx, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记全部已读失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "标记全部已读成功"})
}

// DeleteNotification 删除通知
func (c *SimpleNotificationController) DeleteNotification(ctx *gin.Context) {
	notificationID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "通知ID格式错误: "+err.Error())
		return
	}

	userID, err := c.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := c.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	err = c.notificationService.DeleteNotification(ctx, notificationID, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除通知失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除通知成功"})
}

// GetUnreadCount 获取未读通知数量
func (c *SimpleNotificationController) GetUnreadCount(ctx *gin.Context) {
	userID, err := c.notificationService.GetCurrentUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, "用户未登录: "+err.Error())
		return
	}

	tenantID, err := c.notificationService.GetCurrentTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "租户信息错误: "+err.Error())
		return
	}

	count, err := c.notificationService.GetUnreadCount(ctx, userID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取未读数量失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"count": count})
}

// CreateNotification 创建通知（管理员功能）
func (c *SimpleNotificationController) CreateNotification(ctx *gin.Context) {
	var req dto.CreateNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	notification, err := c.notificationService.CreateNotification(ctx, &req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建通知失败: "+err.Error())
		return
	}

	common.Success(ctx, notification)
}
