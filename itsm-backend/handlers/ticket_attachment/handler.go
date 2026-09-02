// Package ticket_attachment 是工单附件域的 HTTP handler 层（域切片架构）。
// 自 controller/ticket_attachment_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.TicketAttachmentService 承载，本包只做参数解析与响应封装。
package ticket_attachment

import (
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
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return 0, 0, false
	}
	return tenantID, userID, true
}

// pathIDs 提取路径参数中的 ticketID 和 attachmentID
func pathIDs(c *gin.Context) (ticketID, attachmentID int, ok bool) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return 0, 0, false
	}
	attachmentID, err = strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的附件ID")
		return 0, 0, false
	}
	return ticketID, attachmentID, true
}

// ListTicketAttachments 获取工单附件列表
func (h *Handler) ListTicketAttachments(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID, userID, ok := tenantUserID(c)
	if !ok {
		return
	}

	attachments, err := h.attachmentService.ListAttachments(c.Request.Context(), ticketID, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to list ticket attachments", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取附件列表失败")
		return
	}

	common.Success(c, dto.ListTicketAttachmentsResponse{
		Attachments: attachments,
		Total:       len(attachments),
	})
}

// UploadAttachment 上传附件
func (h *Handler) UploadAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "请选择要上传的文件")
		return
	}

	src, err := file.Open()
	if err != nil {
		h.logger.Errorw("Failed to open uploaded file", "error", err)
		common.Fail(c, common.InternalErrorCode, "文件打开失败")
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
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	attachment, err := h.attachmentService.UploadAttachment(c.Request.Context(), ticketID, fileHeader, userID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to upload attachment", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.ParamErrorCode, "附件上传失败，请检查文件类型和大小")
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
		h.logger.Errorw("Failed to get attachment file", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "附件不存在或无法访问")
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
		h.logger.Errorw("Failed to get attachment file", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "附件不存在或无法访问")
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
		common.Fail(c, common.AuthFailedCode, "认证信息缺失")
		return
	}

	err := h.attachmentService.DeleteAttachment(c.Request.Context(), ticketID, attachmentID, tenantID, userID)
	if err != nil {
		h.logger.Errorw("Failed to delete attachment", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "删除附件失败")
		return
	}

	common.Success(c, nil)
}
