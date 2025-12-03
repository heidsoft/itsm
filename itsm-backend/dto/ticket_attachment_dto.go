package dto

import (
	"itsm-backend/ent"
	"time"
)

// TicketAttachmentResponse 工单附件响应
type TicketAttachmentResponse struct {
	ID          int       `json:"id"`
	TicketID    int       `json:"ticket_id"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	FileURL     string    `json:"file_url"`
	FileSize    int       `json:"file_size"`
	FileType    string    `json:"file_type"`
	MimeType    string    `json:"mime_type"`
	UploadedBy  int       `json:"uploaded_by"`
	Uploader    *UserInfo `json:"uploader,omitempty"` // 上传人信息
	CreatedAt   time.Time `json:"created_at"`
}

// ListTicketAttachmentsResponse 工单附件列表响应
type ListTicketAttachmentsResponse struct {
	Attachments []*TicketAttachmentResponse `json:"attachments"`
	Total       int                         `json:"total"`
}

// ToTicketAttachmentResponse 将 Ent 实体转换为 DTO
func ToTicketAttachmentResponse(attachment *ent.TicketAttachment, uploader *ent.User) *TicketAttachmentResponse {
	resp := &TicketAttachmentResponse{
		ID:         attachment.ID,
		TicketID:   attachment.TicketID,
		FileName:   attachment.FileName,
		FilePath:   attachment.FilePath,
		FileSize:   attachment.FileSize,
		FileType:   attachment.FileType,
		UploadedBy: attachment.UploadedBy,
		CreatedAt:  attachment.CreatedAt,
		FileURL:    attachment.FileURL,
		MimeType:   attachment.MimeType,
	}

	// 设置上传人信息
	if uploader != nil {
		resp.Uploader = &UserInfo{
			ID:         uploader.ID,
			Username:   uploader.Username,
			Name:       uploader.Name,
			Email:      uploader.Email,
			Role:       string(uploader.Role),
			Department: uploader.Department,
			TenantID:   uploader.TenantID,
		}
	}

	return resp
}

