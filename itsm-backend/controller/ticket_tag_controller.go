package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TicketTagController 工单标签控制器
type TicketTagController struct {
	tagService *service.TicketTagService
	logger     *zap.Logger
}

// NewTicketTagController 创建工单标签控制器
func NewTicketTagController(tagService *service.TicketTagService, logger *zap.Logger) *TicketTagController {
	return &TicketTagController{
		tagService: tagService,
		logger:     logger,
	}
}

// CreateTag 创建标签
func (ttc *TicketTagController) CreateTag(c *gin.Context) {
	var req service.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.TenantID = tenantID

	tag, err := ttc.tagService.CreateTag(c.Request.Context(), &req)
	if err != nil {
		ttc.logger.Error("Failed to create tag", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tag)
}

// UpdateTag 更新标签
func (ttc *TicketTagController) UpdateTag(c *gin.Context) {
	tagID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的标签ID")
		return
	}

	var req service.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	tag, err := ttc.tagService.UpdateTag(c.Request.Context(), tagID, &req, tenantID)
	if err != nil {
		ttc.logger.Error("Failed to update tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tag)
}

// DeleteTag 删除标签
func (ttc *TicketTagController) DeleteTag(c *gin.Context) {
	tagID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的标签ID")
		return
	}

	err = ttc.tagService.DeleteTag(c.Request.Context(), tagID)
	if err != nil {
		ttc.logger.Error("Failed to delete tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "标签删除成功"})
}

// GetTag 获取标签
func (ttc *TicketTagController) GetTag(c *gin.Context) {
	tagID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的标签ID")
		return
	}

	tag, err := ttc.tagService.GetTag(c.Request.Context(), tagID)
	if err != nil {
		ttc.logger.Error("Failed to get tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, tag)
}

// ListTags 获取标签列表
func (ttc *TicketTagController) ListTags(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	isActiveStr := c.Query("is_active")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var active *bool
	if isActiveStr != "" {
		if a, err := strconv.ParseBool(isActiveStr); err == nil {
			active = &a
		}
	}

	req := &service.ListTagsRequest{
		Page:     page,
		PageSize: pageSize,
		IsActive: active,
		TenantID: tenantID,
	}

	tags, total, err := ttc.tagService.ListTags(c.Request.Context(), req)
	if err != nil {
		ttc.logger.Error("Failed to list tags", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"tags":  tags,
		"total": total,
	})
}

// AssignTagsToTicket 为工单分配标签
func (ttc *TicketTagController) AssignTagsToTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticketId"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		TagIDs []int `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	err = ttc.tagService.AssignTagsToTicket(c.Request.Context(), ticketID, req.TagIDs)
	if err != nil {
		ttc.logger.Error("Failed to assign tags to ticket", zap.Error(err), zap.Int("ticket_id", ticketID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "标签分配成功"})
}

// RemoveTagsFromTicket 从工单移除标签
func (ttc *TicketTagController) RemoveTagsFromTicket(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticketId"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req struct {
		TagIDs []int `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	err = ttc.tagService.RemoveTagsFromTicket(c.Request.Context(), ticketID, req.TagIDs)
	if err != nil {
		ttc.logger.Error("Failed to remove tags from ticket", zap.Error(err), zap.Int("ticket_id", ticketID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "标签移除成功"})
}
