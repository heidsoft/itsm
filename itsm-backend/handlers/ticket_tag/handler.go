// Package ticket_tag 是工单标签域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_tag_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketTagService 承载，本包只做参数解析与响应封装。
package ticket_tag

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
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

// failTagError 把标签写读失败映射为稳定语义。此前重复名、在用删除、跨租户取不到
// 全都回 500/5001「操作失败」，调用方分不清是自己给的名字重了还是后端故障。
// 原始错误只进日志，响应体固定为安全消息。
func (h *Handler) failTagError(c *gin.Context, rawErr error, opMsg string, tagID, tid int) {
	switch {
	case errors.Is(rawErr, service.ErrTicketTagNameExists):
		h.logger.Warnw("Ticket tag name conflict", "error", rawErr, "tenant_id", tid)
		common.Conflict(c, "标签名称已存在", nil)
	case errors.Is(rawErr, service.ErrTicketTagInUse):
		h.logger.Warnw("Ticket tag still in use", "error", rawErr, "tag_id", tagID, "tenant_id", tid)
		common.Conflict(c, "标签仍被工单使用，无法删除", nil)
	case errors.Is(rawErr, service.ErrTicketTagNameBlank):
		h.logger.Warnw("Ticket tag name blank", "error", rawErr, "tenant_id", tid)
		common.Fail(c, common.ParamErrorCode, "标签名称不能为空")
	case errors.Is(rawErr, service.ErrTicketTagNotFound), ent.IsNotFound(rawErr):
		// 查询都带租户谓词，所以跨租户与真实不存在同样表现为 not found。
		h.logger.Warnw("Ticket tag not found", "error", rawErr, "tag_id", tagID, "tenant_id", tid)
		common.Fail(c, common.NotFoundCode, "标签不存在或无权访问")
	default:
		h.logger.Errorw("Ticket tag operation failed",
			"error", rawErr, "operation", opMsg, "tag_id", tagID, "tenant_id", tid)
		common.Fail(c, common.InternalErrorCode, opMsg)
	}
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
		h.failTagError(c, err, "创建标签失败", 0, tid)
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
		h.failTagError(c, err, "更新标签失败", tagID, tid)
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
		h.failTagError(c, err, "删除标签失败", tagID, tid)
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
		h.failTagError(c, err, "获取标签失败", tagID, tid)
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

	pagination := common.GetPaginationFromQuery(c)
	page, pageSize := pagination.Page, pagination.PageSize
	isActiveStr := c.Query("isActive")

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
		h.failTagError(c, err, "获取标签列表失败", 0, tid)
		return
	}

	common.Success(c, gin.H{
		"items": dto.ToTicketTagResponseList(tags),
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
		common.Fail(c, common.ParamErrorCode, "tagIds 或 tags 必填")
		return
	}

	err := h.tagService.AssignTagsToTicket(c.Request.Context(), ticketID, tagIDs, tid)
	if err != nil {
		h.failTagError(c, err, "标签分配失败", 0, tid)
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
		common.Fail(c, common.ParamErrorCode, "tagIds 或 tags 必填")
		return
	}

	err := h.tagService.RemoveTagsFromTicket(c.Request.Context(), ticketID, tagIDs, tid)
	if err != nil {
		h.failTagError(c, err, "标签移除失败", 0, tid)
		return
	}

	common.Success(c, gin.H{"message": "标签移除成功"})
}
