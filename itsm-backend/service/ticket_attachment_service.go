package service

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketattachment"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type TicketAttachmentService struct {
	client       *ent.Client
	logger       *zap.SugaredLogger
	uploadDir    string
	maxFileSize  int64    // 最大文件大小（字节），默认10MB
	allowedTypes []string // 允许的文件类型
	virusScanner AttachmentVirusScanner
}

type AttachmentVirusScanner interface {
	Scan(context.Context, string) error
}
type noopAttachmentVirusScanner struct{}

func (noopAttachmentVirusScanner) Scan(context.Context, string) error { return nil }

func NewTicketAttachmentService(client *ent.Client, logger *zap.SugaredLogger) *TicketAttachmentService {
	uploadDir := "uploads/tickets"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
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
		virusScanner: noopAttachmentVirusScanner{},
	}
}

func (s *TicketAttachmentService) SetVirusScanner(scanner AttachmentVirusScanner) {
	if scanner != nil {
		s.virusScanner = scanner
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

	// 验证文件类型（Client-provided Content-Type 可被伪造，仅作初筛）
	mimeType := fileHeader.ContentType
	if mimeType == "" {
		// 尝试从文件扩展名推断
		ext := filepath.Ext(fileHeader.Filename)
		mimeType = mime.TypeByExtension(ext)
	}

	if !s.isAllowedType(mimeType) {
		return nil, fmt.Errorf("file type not allowed: %s", mimeType)
	}

	// 1) 文件名清洗：拒绝路径遍历/控制字符，限制长度，防止 XSS/覆盖/目录穿越
	safeName := sanitizeFilename(fileHeader.Filename)
	if safeName == "" {
		return nil, fmt.Errorf("invalid filename: empty after sanitization")
	}

	// 2) Magic bytes / 实际内容嗅探：避免 Content-Type/扩展名 与 真实内容不一致
	//    从文件头最多读取 512 字节，调用 net/http.DetectContentType。
	//    注意：fileHeader.Reader 通常是一次性的，因此需要将嗅探过的字节 prepend 回去以便后续 saveFile 读取。
	if fileHeader.Reader != nil {
		sniffBuf := make([]byte, 0, 512)
		tmp := make([]byte, 512)
		for len(sniffBuf) < 512 {
			n, rerr := fileHeader.Reader.Read(tmp)
			if n > 0 {
				sniffBuf = append(sniffBuf, tmp[:n]...)
			}
			if rerr != nil {
				break
			}
		}
		if len(sniffBuf) > 0 {
			detected := http.DetectContentType(sniffBuf)
			// 以 sniff 出的类型为准，校验是否仍在白名单内
			if !s.isAllowedType(detected) {
				return nil, fmt.Errorf("detected file type not allowed: %s (claimed: %s)", detected, mimeType)
			}
			mimeType = detected
			// 把嗅探过的字节塞回 Reader 的前面，保证 saveFile 读得到完整内容
			fileHeader.Reader = &prefixedReader{prefix: sniffBuf, r: fileHeader.Reader}
		}
	}

	// 生成唯一文件名（使用清洗后的文件名）
	fileName := fmt.Sprintf("%d_%d_%s", ticketID, time.Now().UnixNano(), safeName)
	filePath := filepath.Join(s.uploadDir, fileName)

	// 保存文件
	if err := s.saveFile(fileHeader, filePath); err != nil {
		s.logger.Errorw("Failed to save file", "error", err)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	if err := s.virusScanner.Scan(ctx, filePath); err != nil {
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("file rejected by malware scan")
	}

	// 生成文件URL（相对路径，实际URL由前端或CDN提供）
	fileURL := fmt.Sprintf("/api/v1/tickets/%d/attachments/%s/download", ticketID, fileName)

	// 创建附件记录
	attachment, err := s.client.TicketAttachment.Create().
		SetTicketID(ticketID).
		SetFileName(safeName).
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
func (s *TicketAttachmentService) ListAttachments(ctx context.Context, ticketID, tenantID, userID int) ([]*dto.TicketAttachmentResponse, error) {
	s.logger.Infow("Listing attachments", "ticket_id", ticketID)
	if err := s.authorizeTicketAttachmentAccess(ctx, ticketID, tenantID, userID); err != nil {
		return nil, err
	}

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
	ticketInfo, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
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
	err = s.client.TicketAttachment.DeleteOneID(attachmentID).
		Where(
			ticketattachment.TicketID(ticketID),
			ticketattachment.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete attachment", "error", err)
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

// GetAttachmentFile 获取附件文件（用于下载）
func (s *TicketAttachmentService) GetAttachmentFile(ctx context.Context, ticketID, attachmentID, tenantID, userID int) (*AttachmentFile, error) {
	if err := s.authorizeTicketAttachmentAccess(ctx, ticketID, tenantID, userID); err != nil {
		return nil, err
	}
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	written, err := io.Copy(dst, io.LimitReader(fileHeader.Reader, s.maxFileSize+1))
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	if written > s.maxFileSize {
		_ = os.Remove(filePath)
		return fmt.Errorf("file size exceeds maximum allowed size (%d bytes)", s.maxFileSize)
	}
	if fileHeader.Size >= 0 && written != fileHeader.Size {
		_ = os.Remove(filePath)
		return fmt.Errorf("uploaded file size does not match declared size")
	}

	return nil
}

func (s *TicketAttachmentService) authorizeTicketAttachmentAccess(ctx context.Context, ticketID, tenantID, userID int) error {
	if userID <= 0 {
		return fmt.Errorf("authentication required")
	}
	t, err := s.client.Ticket.Query().Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("ticket not found")
	}
	if t.RequesterID == userID || (t.AssigneeID > 0 && t.AssigneeID == userID) {
		return nil
	}
	u, err := s.client.User.Query().Where(user.ID(userID), user.TenantID(tenantID), user.Active(true)).Only(ctx)
	if err != nil {
		return fmt.Errorf("permission denied")
	}
	switch string(u.Role) {
	case "super_admin", "admin", "manager", "agent", "technician", "security":
		return nil
	}
	return fmt.Errorf("permission denied")
}

func SanitizeDownloadFilename(name string) string { return sanitizeFilename(name) }

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

// sanitizeFilename cleans an upload filename for safe on-disk + header usage.
// - disallows path separators, control chars, NUL, leading dots, relative segments
// - limits length to 200 runes
func sanitizeFilename(name string) string {
	if name == "" {
		return ""
	}
	// 路径遍历防御：剥掉任何目录部分
	name = filepath.Base(name)
	// 去掉 Windows 驱动器前缀和反斜杠路径段
	if strings.ContainsRune(name, '\\') {
		parts := strings.FieldsFunc(name, func(r rune) bool { return r == '\\' })
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	}
	// 剥掉控制字符、NUL、以及可能触发 shell/URL 二次解析的危险字符
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f:
			continue // control / DEL
		case r == '/' || r == '\\' || r == 0:
			continue
		case r == '%' || r == '`' || r == '|' || r == '&' || r == ';' || r == '>' || r == '<' || r == '"' || r == '\'' || r == '*' || r == '?':
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	out = strings.TrimLeft(out, ". ") // 防 ../ 和 dotfiles
	if out == "" || out == "." || out == ".." {
		return ""
	}
	// 截断到 200 runes
	runes := []rune(out)
	if len(runes) > 200 {
		out = string(runes[:200])
	}
	return out
}

// prefixedReader prepends sniffed bytes back onto the original reader so
// downstream consumers of fileHeader.Reader see the full stream.
type prefixedReader struct {
	prefix []byte
	off    int
	r      io.Reader
}

func (p *prefixedReader) Read(b []byte) (int, error) {
	if p.off < len(p.prefix) {
		n := copy(b, p.prefix[p.off:])
		p.off += n
		if n == len(b) {
			return n, nil
		}
		// 继续从底层 reader 填剩下的空间
		n2, err := p.r.Read(b[n:])
		return n + n2, err
	}
	return p.r.Read(b)
}
