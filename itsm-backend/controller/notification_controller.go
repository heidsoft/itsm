package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService *service.NotificationService
}

func NewNotificationController(notificationService *service.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
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
func (c *NotificationController) GetNotifications(ctx *gin.Context) {
	var req dto.GetNotificationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取用户ID和租户ID
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

	req.UserID = userID
	req.TenantID = tenantID

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	result, err := c.notificationService.GetNotifications(ctx, &req)
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
func (c *NotificationController) MarkNotificationRead(ctx *gin.Context) {
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

	req := &dto.MarkNotificationReadRequest{
		NotificationID: notificationID,
		UserID:         userID,
		TenantID:       tenantID,
	}

	err = c.notificationService.MarkNotificationRead(ctx, req)
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
func (c *NotificationController) MarkAllNotificationsRead(ctx *gin.Context) {
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

	req := &dto.MarkAllNotificationsReadRequest{
		UserID:   userID,
		TenantID: tenantID,
	}

	err = c.notificationService.MarkAllNotificationsRead(ctx, req)
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
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
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

	req := &dto.DeleteNotificationRequest{
		NotificationID: notificationID,
		UserID:         userID,
		TenantID:       tenantID,
	}

	err = c.notificationService.DeleteNotification(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除通知失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "删除通知成功"})
}

// GetUnreadCount 获取未读通知数量
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
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
func (c *NotificationController) CreateNotification(ctx *gin.Context) {
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
