package controller

import (
	"io"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TicketAttachmentController struct {
	attachmentService *service.TicketAttachmentService
	logger            *zap.SugaredLogger
}

func NewTicketAttachmentController(attachmentService *service.TicketAttachmentService, logger *zap.SugaredLogger) *TicketAttachmentController {
	return &TicketAttachmentController{
		attachmentService: attachmentService,
		logger:            logger,
	}
}

// ListTicketAttachments 获取工单附件列表
// GET /api/v1/tickets/:id/attachments
func (tac *TicketAttachmentController) ListTicketAttachments(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	attachments, err := tac.attachmentService.ListAttachments(c.Request.Context(), ticketID, tenantID)
	if err != nil {
		tac.logger.Errorw("Failed to list ticket attachments", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.ListTicketAttachmentsResponse{
		Attachments: attachments,
		Total:       len(attachments),
	})
}

// UploadAttachment 上传附件
// POST /api/v1/tickets/:id/attachments
func (tac *TicketAttachmentController) UploadAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "请选择要上传的文件")
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		tac.logger.Errorw("Failed to open uploaded file", "error", err)
		common.Fail(c, common.InternalErrorCode, "文件打开失败")
		return
	}
	defer src.Close()

	// 创建文件头信息
	fileHeader := &service.FileHeader{
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		Reader:      src,
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	attachment, err := tac.attachmentService.UploadAttachment(c.Request.Context(), ticketID, fileHeader, userID, tenantID)
	if err != nil {
		tac.logger.Errorw("Failed to upload attachment", "error", err, "ticket_id", ticketID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, attachment)
}

// DownloadAttachment 下载附件
// GET /api/v1/tickets/:id/attachments/:attachment_id
func (tac *TicketAttachmentController) DownloadAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	attachmentID, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的附件ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	attachmentFile, err := tac.attachmentService.GetAttachmentFile(c.Request.Context(), ticketID, attachmentID, tenantID)
	if err != nil {
		tac.logger.Errorw("Failed to get attachment file", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	defer attachmentFile.File.Close()

	// 设置响应头
	mimeType := "application/octet-stream"
	if attachmentFile.MimeType != nil {
		mimeType = *attachmentFile.MimeType
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", `attachment; filename="`+attachmentFile.FileName+`"`)
	c.Header("Content-Length", strconv.FormatInt(attachmentFile.Size, 10))

	// 复制文件内容到响应
	_, err = io.Copy(c.Writer, attachmentFile.File)
	if err != nil {
		tac.logger.Errorw("Failed to copy file to response", "error", err)
		return
	}
}

// PreviewAttachment 预览附件
// GET /api/v1/tickets/:id/attachments/:attachment_id/preview
func (tac *TicketAttachmentController) PreviewAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	attachmentID, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的附件ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	attachmentFile, err := tac.attachmentService.GetAttachmentFile(c.Request.Context(), ticketID, attachmentID, tenantID)
	if err != nil {
		tac.logger.Errorw("Failed to get attachment file", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	defer attachmentFile.File.Close()

	// 设置响应头（预览模式）
	mimeType := "application/octet-stream"
	if attachmentFile.MimeType != nil {
		mimeType = *attachmentFile.MimeType
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", `inline; filename="`+attachmentFile.FileName+`"`)
	c.Header("Content-Length", strconv.FormatInt(attachmentFile.Size, 10))

	// 复制文件内容到响应
	_, err = io.Copy(c.Writer, attachmentFile.File)
	if err != nil {
		tac.logger.Errorw("Failed to copy file to response", "error", err)
		return
	}
}

// DeleteAttachment 删除附件
// DELETE /api/v1/tickets/:id/attachments/:attachment_id
func (tac *TicketAttachmentController) DeleteAttachment(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	attachmentID, err := strconv.Atoi(c.Param("attachment_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的附件ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	err = tac.attachmentService.DeleteAttachment(c.Request.Context(), ticketID, attachmentID, tenantID, userID)
	if err != nil {
		tac.logger.Errorw("Failed to delete attachment", "error", err, "ticket_id", ticketID, "attachment_id", attachmentID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

