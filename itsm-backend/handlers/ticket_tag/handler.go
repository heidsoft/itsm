// Package ticket_tag 是工单标签域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_tag_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketTagService 承载，本包只做参数解析与响应封装。
package ticket_tag

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 工单标签 HTTP handler
type Handler struct {
	tagService *service.TicketTagService
	logger     *zap.SugaredLogger
}

// NewHandler 创建工单标签 handler 实例
func NewHandler(tagService *service.TicketTagService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		tagService: tagService,
		logger:     logger,
	}
}

// tenantID 提取租户上下文
func tenantID(c *gin.Context) (int, bool) {
	tid := c.GetInt("tenant_id")
	if tid == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return 0, false
	}
	return tid, true
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

// CreateTag 创建标签
func (h *Handler) CreateTag(c *gin.Context) {
	var req service.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}
	req.TenantID = tid

	tag, err := h.tagService.CreateTag(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create tag", zap.Error(err), zap.Int("tenant_id", tid))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketTagResponse(tag))
}

// UpdateTag 更新标签
func (h *Handler) UpdateTag(c *gin.Context) {
	tagID, ok := pathID(c)
	if !ok {
		return
	}

	var req service.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	tag, err := h.tagService.UpdateTag(c.Request.Context(), tagID, &req, tid)
	if err != nil {
		h.logger.Error("Failed to update tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketTagResponse(tag))
}

// DeleteTag 删除标签
func (h *Handler) DeleteTag(c *gin.Context) {
	tagID, ok := pathID(c)
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	err := h.tagService.DeleteTag(c.Request.Context(), tagID, tid)
	if err != nil {
		h.logger.Error("Failed to delete tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "标签删除成功"})
}

// GetTag 获取标签
func (h *Handler) GetTag(c *gin.Context) {
	tagID, ok := pathID(c)
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	tag, err := h.tagService.GetTag(c.Request.Context(), tagID, tid)
	if err != nil {
		h.logger.Error("Failed to get tag", zap.Error(err), zap.Int("tag_id", tagID))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, dto.ToTicketTagResponse(tag))
}

// ListTags 获取标签列表
func (h *Handler) ListTags(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	isActiveStr := c.Query("is_active")

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
		TenantID: tid,
	}

	tags, total, err := h.tagService.ListTags(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to list tags", zap.Error(err), zap.Int("tenant_id", tid))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"tags":  dto.ToTicketTagResponseList(tags),
		"total": total,
	})
}

// AssignTagsToTicket 为工单分配标签
func (h *Handler) AssignTagsToTicket(c *gin.Context) {
	ticketID, ok := pathID(c)
	if !ok {
		return
	}

	var req struct {
		TagIDs []int    `json:"tagIds"`
		Tags   []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	tagIDs := req.TagIDs
	if len(tagIDs) == 0 && len(req.Tags) > 0 {
		resolved, resolveErr := h.tagService.ResolveTagIDsByNames(c.Request.Context(), req.Tags, tid, true)
		if resolveErr != nil {
			common.ParamErrorWithErr(c, resolveErr, "请求参数错误")
			return
		}
		tagIDs = resolved
	}
	if len(tagIDs) == 0 {
		common.Fail(c, common.ParamErrorCode, "tag_ids 或 tags 必填")
		return
	}

	err := h.tagService.AssignTagsToTicket(c.Request.Context(), ticketID, tagIDs, tid)
	if err != nil {
		h.logger.Error("Failed to assign tags to ticket", zap.Error(err), zap.Int("ticket_id", ticketID))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "标签分配成功"})
}

// RemoveTagsFromTicket 从工单移除标签
func (h *Handler) RemoveTagsFromTicket(c *gin.Context) {
	ticketID, ok := pathID(c)
	if !ok {
		return
	}

	var req struct {
		TagIDs []int    `json:"tagIds"`
		Tags   []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	tagIDs := req.TagIDs
	if len(tagIDs) == 0 && len(req.Tags) > 0 {
		resolved, resolveErr := h.tagService.ResolveTagIDsByNames(c.Request.Context(), req.Tags, tid, false)
		if resolveErr != nil {
			common.ParamErrorWithErr(c, resolveErr, "请求参数错误")
			return
		}
		tagIDs = resolved
	}
	if len(tagIDs) == 0 {
		common.Fail(c, common.ParamErrorCode, "tag_ids 或 tags 必填")
		return
	}

	err := h.tagService.RemoveTagsFromTicket(c.Request.Context(), ticketID, tagIDs, tid)
	if err != nil {
		h.logger.Error("Failed to remove tags from ticket", zap.Error(err), zap.Int("ticket_id", ticketID))
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "标签移除成功"})
}
