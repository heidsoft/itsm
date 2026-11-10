// Package ticket_attachment 是工单附件域的 HTTP handler 层（域切片架构）。
// 业务逻辑仍由 service.TicketAttachmentService 承载，本包只做参数解析与响应封装。
package ticket_attachment

import (
	"errors"
	"io"
	"mime"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 工单附件 HTTP handler
type Handler struct {
	attachmentService *service.TicketAttachmentService
	logger            *zap.SugaredLogger
}

// NewHandler 创建工单附件 handler 实例
func NewHandler(attachmentService *service.TicketAttachmentService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		attachmentService: attachmentService,
		logger:            logger,
	}
}

// tenantUserID 提取租户和用户 ID
func tenantUserID(c *gin.Context) (tenantID, userID int, ok bool) {
	tenantID = c.GetInt("tenant_id")
	userID = c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "Authentication information missing")
		return 0, 0, false
	}
	return tenantID, userID, true
}

// pathIDs 提取路径参数中的 ticketID 和 attachmentID
func pathIDs(c *gin.Context) (ticketID, attachmentID int, ok bool) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid ticket ID")
		return 0, 0, false
	}
	attachmentID, err = strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid attachment ID")
		return 0, 0, false
	}
	return ticketID, attachmentID, true
}

// failAttachmentAccess 把附件访问失败映射为稳定的 HTTP 语义。
// 所有失败都回 5001 时，「附件不存在」「跨租户」「磁盘读失败」在调用方眼里长得一样。
// 原始错误只进日志；不存在与无权统一按 404 对外表达，避免确认对方工单存在。
func (h *Handler) failAttachmentAccess(c *gin.Context, rawErr error, ticketID, tenantID int) {
	switch {
	case errors.Is(rawErr, service.ErrAttachmentAccessDenied):
		h.logger.Warnw("Attachment access denied", "error", rawErr, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.ForbiddenCode, "No permission to access this attachment")
	case errors.Is(rawErr, service.ErrAttachmentNotFound):
		h.logger.Warnw("Attachment not found", "error", rawErr, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "Attachment does not exist or does not have access rights")
	default:
		h.logger.Errorw("Failed to read attachment", "error", rawErr, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "Failed to read attachment")
	}
}

// ListTicketAttachments 获取工单附件列表
func (h *Handler) ListTicketAttachments(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid ticket ID")
		return
	}

	tenantID, userID, ok := tenantUserID(c)
	if !ok {
		return
	}

	attachments, err := h.attachmentService.ListAttachments(c.Request.Context(), ticketID, tenantID, userID)
	if err != nil {
		h.failAttachmentAccess(c, err, ticketID, tenantID)
		return
	}

	common.Success(c, dto.ListTicketAttachmentsResponse{
		Items: attachments,
		Total: len(attachments),
	})
}

// UploadAttachment 上传附件
func (h *Handler) UploadAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid ticket ID")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Please select the file to upload")
		return
	}

	src, err := file.Open()
	if err != nil {
		h.logger.Errorw("Failed to open uploaded file", "error", err)
		common.Fail(c, common.InternalErrorCode, "File opening failed")
		return
	}
	defer src.Close()

	fileHeader := &service.FileHeader{
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		Reader:      src,
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "Authentication information missing")
		return
	}

	attachment, err := h.attachmentService.UploadAttachment(c.Request.Context(), ticketID, fileHeader, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to upload attachment", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.ParamErrorCode, "Attachment upload failed, please check file type and size")
		return
	}

	common.Success(c, attachment)
}

// DownloadAttachment 下载附件
func (h *Handler) DownloadAttachment(c *gin.Context) {
	ticketID, attachmentID, ok := pathIDs(c)
	if !ok {
		return
	}

	tenantID, userID, ok := tenantUserID(c)
	if !ok {
		return
	}

	attachmentFile, err := h.attachmentService.GetAttachmentFile(c.Request.Context(), ticketID, attachmentID, tenantID, userID)
	if err != nil {
		h.failAttachmentAccess(c, err, ticketID, tenantID)
		return
	}
	defer attachmentFile.File.Close()

	mimeType := "application/octet-stream"
	if attachmentFile.MimeType != nil {
		mimeType = *attachmentFile.MimeType
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachmentFile.FileName}))
	c.Header("Content-Length", strconv.FormatInt(attachmentFile.Size, 10))

	_, err = io.Copy(c.Writer, attachmentFile.File)
	if err != nil {
		h.logger.Errorw("Failed to copy file to response", "error", err)
	}
}

// PreviewAttachment 预览附件
func (h *Handler) PreviewAttachment(c *gin.Context) {
	ticketID, attachmentID, ok := pathIDs(c)
	if !ok {
		return
	}

	tenantID, userID, ok := tenantUserID(c)
	if !ok {
		return
	}

	attachmentFile, err := h.attachmentService.GetAttachmentFile(c.Request.Context(), ticketID, attachmentID, tenantID, userID)
	if err != nil {
		h.failAttachmentAccess(c, err, ticketID, tenantID)
		return
	}
	defer attachmentFile.File.Close()

	mimeType := "application/octet-stream"
	if attachmentFile.MimeType != nil {
		mimeType = *attachmentFile.MimeType
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": attachmentFile.FileName}))
	c.Header("Content-Length", strconv.FormatInt(attachmentFile.Size, 10))

	_, err = io.Copy(c.Writer, attachmentFile.File)
	if err != nil {
		h.logger.Errorw("Failed to copy file to response", "error", err)
	}
}

// DeleteAttachment 删除附件
func (h *Handler) DeleteAttachment(c *gin.Context) {
	ticketID, attachmentID, ok := pathIDs(c)
	if !ok {
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if tenantID == 0 || userID == 0 {
		common.Fail(c, common.AuthFailedCode, "Authentication information missing")
		return
	}

	err := h.attachmentService.DeleteAttachment(c.Request.Context(), ticketID, attachmentID, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to delete attachment", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		// service 已把 ent.NotFoundError 以 %w 包装、权限拒绝为 Forbidden 业务码；
		// 不再一律 Fail(InternalErrorCode) 吞掉分类——not-found → 404、权限不足 → 403。
		common.FailWithErr(c, err, "Failed to delete attachment")
		return
	}

	common.Success(c, nil)
}
