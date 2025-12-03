package service

import (
	"context"
	"fmt"
	"io"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketattachment"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

type TicketAttachmentService struct {
	client      *ent.Client
	logger      *zap.SugaredLogger
	uploadDir   string
	maxFileSize int64 // 最大文件大小（字节），默认10MB
	allowedTypes []string // 允许的文件类型
}

func NewTicketAttachmentService(client *ent.Client, logger *zap.SugaredLogger) *TicketAttachmentService {
	uploadDir := "uploads/tickets"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Warnw("Failed to create upload directory", "error", err, "dir", uploadDir)
	}

	return &TicketAttachmentService{
		client:      client,
		logger:      logger,
		uploadDir:   uploadDir,
		maxFileSize: 10 * 1024 * 1024, // 10MB
		allowedTypes: []string{
			// 图片
			"image/jpeg", "image/png", "image/gif", "image/webp",
			// 文档
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"application/vnd.ms-powerpoint",
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
			// 文本
			"text/plain", "text/csv",
			// 压缩文件
			"application/zip", "application/x-rar-compressed",
		},
	}
}

// UploadAttachment 上传附件
func (s *TicketAttachmentService) UploadAttachment(
	ctx context.Context,
	ticketID int,
	fileHeader *FileHeader,
	userID, tenantID int,
) (*dto.TicketAttachmentResponse, error) {
	s.logger.Infow("Uploading attachment", "ticket_id", ticketID, "file_name", fileHeader.Filename, "user_id", userID)

	// 验证工单是否存在且属于当前租户
	ticketExists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket existence", "error", err)
		return nil, fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !ticketExists {
		return nil, fmt.Errorf("ticket not found")
	}

	// 验证文件大小
	if fileHeader.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d bytes)", s.maxFileSize)
	}

	// 验证文件类型
	mimeType := fileHeader.ContentType
	if mimeType == "" {
		// 尝试从文件扩展名推断
		ext := filepath.Ext(fileHeader.Filename)
		mimeType = mime.TypeByExtension(ext)
	}

	if !s.isAllowedType(mimeType) {
		return nil, fmt.Errorf("file type not allowed: %s", mimeType)
	}

	// 生成唯一文件名
	fileName := fmt.Sprintf("%d_%d_%s", ticketID, time.Now().Unix(), fileHeader.Filename)
	filePath := filepath.Join(s.uploadDir, fileName)

	// 保存文件
	if err := s.saveFile(fileHeader, filePath); err != nil {
		s.logger.Errorw("Failed to save file", "error", err)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 生成文件URL（相对路径，实际URL由前端或CDN提供）
	fileURL := fmt.Sprintf("/api/v1/tickets/%d/attachments/%s/download", ticketID, fileName)

	// 创建附件记录
	attachment, err := s.client.TicketAttachment.Create().
		SetTicketID(ticketID).
		SetFileName(fileHeader.Filename).
		SetFilePath(filePath).
		SetFileURL(fileURL).
		SetFileSize(int(fileHeader.Size)).
		SetFileType(mimeType).
		SetMimeType(mimeType).
		SetUploadedBy(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		// 如果数据库保存失败，删除已上传的文件
		os.Remove(filePath)
		s.logger.Errorw("Failed to create attachment record", "error", err)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	// 查询上传人信息
	uploader, err := s.client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Warnw("Failed to get uploader", "error", err, "user_id", userID)
		uploader = nil
	}

	return dto.ToTicketAttachmentResponse(attachment, uploader), nil
}

// ListAttachments 获取附件列表
func (s *TicketAttachmentService) ListAttachments(ctx context.Context, ticketID, tenantID int) ([]*dto.TicketAttachmentResponse, error) {
	s.logger.Infow("Listing attachments", "ticket_id", ticketID)

	// 验证工单是否存在且属于当前租户
	ticketExists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket existence", "error", err)
		return nil, fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !ticketExists {
		return nil, fmt.Errorf("ticket not found")
	}

	// 查询附件
	attachments, err := s.client.TicketAttachment.Query().
		Where(
			ticketattachment.TicketID(ticketID),
			ticketattachment.TenantID(tenantID),
		).
		Order(ent.Desc(ticketattachment.FieldCreatedAt)).
		WithUploader().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list attachments", "error", err)
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}

	// 转换为 DTO
	responses := make([]*dto.TicketAttachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		var uploader *ent.User
		if attachment.Edges.Uploader != nil {
			uploader = attachment.Edges.Uploader
		} else {
			uploader, _ = s.client.User.Get(ctx, attachment.UploadedBy)
		}
		responses = append(responses, dto.ToTicketAttachmentResponse(attachment, uploader))
	}

	return responses, nil
}

// GetAttachment 获取附件信息
func (s *TicketAttachmentService) GetAttachment(ctx context.Context, ticketID, attachmentID, tenantID int) (*dto.TicketAttachmentResponse, error) {
	attachment, err := s.client.TicketAttachment.Query().
		Where(
			ticketattachment.ID(attachmentID),
			ticketattachment.TicketID(ticketID),
			ticketattachment.TenantID(tenantID),
		).
		WithUploader().
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get attachment", "error", err)
		return nil, fmt.Errorf("attachment not found: %w", err)
	}

	var uploader *ent.User
	if attachment.Edges.Uploader != nil {
		uploader = attachment.Edges.Uploader
	} else {
		uploader, _ = s.client.User.Get(ctx, attachment.UploadedBy)
	}

	return dto.ToTicketAttachmentResponse(attachment, uploader), nil
}

// DeleteAttachment 删除附件
func (s *TicketAttachmentService) DeleteAttachment(ctx context.Context, ticketID, attachmentID, tenantID, userID int) error {
	s.logger.Infow("Deleting attachment", "ticket_id", ticketID, "attachment_id", attachmentID, "user_id", userID)

	// 查询附件
	attachment, err := s.client.TicketAttachment.Query().
		Where(
			ticketattachment.ID(attachmentID),
			ticketattachment.TicketID(ticketID),
			ticketattachment.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get attachment", "error", err)
		return fmt.Errorf("attachment not found: %w", err)
	}

	// 权限检查：只有上传人或工单处理人可以删除
	ticketInfo, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		s.logger.Errorw("Failed to get ticket", "error", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	canDelete := attachment.UploadedBy == userID ||
		(ticketInfo.AssigneeID > 0 && ticketInfo.AssigneeID == userID) ||
		ticketInfo.RequesterID == userID
	if !canDelete {
		return fmt.Errorf("permission denied: only uploader, ticket assignee or requester can delete")
	}

	// 删除文件
	if err := os.Remove(attachment.FilePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warnw("Failed to delete file", "error", err, "path", attachment.FilePath)
		// 继续删除数据库记录，即使文件删除失败
	}

	// 删除数据库记录
	err = s.client.TicketAttachment.DeleteOneID(attachmentID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete attachment", "error", err)
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

// GetAttachmentFile 获取附件文件（用于下载）
func (s *TicketAttachmentService) GetAttachmentFile(ctx context.Context, ticketID, attachmentID, tenantID int) (*AttachmentFile, error) {
	attachment, err := s.client.TicketAttachment.Query().
		Where(
			ticketattachment.ID(attachmentID),
			ticketattachment.TicketID(ticketID),
			ticketattachment.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("attachment not found: %w", err)
	}

	file, err := os.Open(attachment.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	mimeType := attachment.MimeType
	if mimeType == "" {
		mimeType = attachment.FileType
	}
	return &AttachmentFile{
		File:     file,
		FileName: attachment.FileName,
		MimeType: &mimeType,
		Size:     int64(attachment.FileSize),
	}, nil
}

// 辅助方法

// FileHeader 文件头信息
type FileHeader struct {
	Filename    string
	Size        int64
	ContentType string
	Reader      io.Reader
}

// AttachmentFile 附件文件
type AttachmentFile struct {
	File     *os.File
	FileName string
	MimeType *string
	Size     int64
}

// saveFile 保存文件
func (s *TicketAttachmentService) saveFile(fileHeader *FileHeader, filePath string) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, fileHeader.Reader); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// isAllowedType 检查文件类型是否允许
func (s *TicketAttachmentService) isAllowedType(mimeType string) bool {
	if mimeType == "" {
		return false
	}

	// 检查精确匹配
	for _, allowed := range s.allowedTypes {
		if mimeType == allowed {
			return true
		}
	}

	// 检查类型前缀（如 image/*, application/*）
	parts := strings.Split(mimeType, "/")
	if len(parts) == 2 {
		typePrefix := parts[0] + "/*"
		for _, allowed := range s.allowedTypes {
			if allowed == typePrefix {
				return true
			}
		}
	}

	return false
}

