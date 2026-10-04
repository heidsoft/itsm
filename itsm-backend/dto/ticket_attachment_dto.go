package dto

import (
	"fmt"
	"time"

	"itsm-backend/ent"
)

// TicketAttachmentResponse 工单附件响应
type TicketAttachmentResponse struct {
	ID         int       `json:"id"`
	TicketID   int       `json:"ticketId"`
	FileName   string    `json:"fileName"`
	FilePath   string    `json:"filePath"`
	FileURL    string    `json:"fileUrl"`
	FileSize   int       `json:"fileSize"`
	FileType   string    `json:"fileType"`
	MimeType   string    `json:"mimeType"`
	UploadedBy int       `json:"uploadedBy"`
	Uploader   *UserInfo `json:"uploader,omitempty"` // 上传人信息
	CreatedAt  time.Time `json:"createdAt"`
}

// ListTicketAttachmentsResponse 工单附件列表响应
//
// 该端点实测不分页（handler 用 Total: len(attachments)），只保留诚实的 {items,total}。
type ListTicketAttachmentsResponse struct {
	Items []*TicketAttachmentResponse `json:"items"`
	Total int                         `json:"total"`
}

// ToTicketAttachmentResponse 将 Ent 实体转换为 DTO
//
// fileUrl 由 (ticketId, id) 推导，不读 attachment.FileURL：库里存量值是
// /attachments/<文件名>/download 这种从未注册过的死链，下载入口的唯一真相是
// router/ticket_routes.go 注册的路由。
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
		FileURL:    fmt.Sprintf("/api/v1/tickets/%d/attachments/%d", attachment.TicketID, attachment.ID),
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
