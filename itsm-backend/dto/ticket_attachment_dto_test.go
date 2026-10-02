package dto

import (
	"encoding/json"
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
)

// fileUrl 必须由 (ticketId, id) 推导。库里存量行的 file_url 是
// /attachments/<文件名>/download —— 该路由从未注册过，前端点击恒 404。
// 如果 mapper 继续透传该列，修好路由也没用。
func TestToTicketAttachmentResponseDerivesDownloadUrlFromIdentity(t *testing.T) {
	resp := ToTicketAttachmentResponse(&ent.TicketAttachment{
		ID:       11,
		TicketID: 22,
		FileName: "report.pdf",
		FilePath: "uploads/tickets/22/report.pdf",
		FileURL:  "/api/v1/tickets/attachments/report.pdf/download",
	}, nil)

	assert.Equal(t, "/api/v1/tickets/22/attachments/11", resp.FileURL,
		"fileUrl 必须等于 router/ticket_routes.go 注册的下载路由，不得透传库里的死链")
}

func TestTicketAttachmentResponseUsesCamelCaseJSON(t *testing.T) {
	resp := TicketAttachmentResponse{
		ID:         1,
		TicketID:   2,
		FileName:   "report.pdf",
		FilePath:   "/tmp/report.pdf",
		FileURL:    "/api/v1/tickets/2/attachments/1",
		FileSize:   1234,
		FileType:   "application/pdf",
		MimeType:   "application/pdf",
		UploadedBy: 7,
		CreatedAt:  time.Unix(0, 0).UTC(),
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"ticketId":2`)
	assert.Contains(t, jsonStr, `"fileName":"report.pdf"`)
	assert.Contains(t, jsonStr, `"filePath":"/tmp/report.pdf"`)
	assert.Contains(t, jsonStr, `"fileUrl":"/api/v1/tickets/2/attachments/1"`)
	assert.Contains(t, jsonStr, `"fileSize":1234`)
	assert.Contains(t, jsonStr, `"fileType":"application/pdf"`)
	assert.Contains(t, jsonStr, `"mimeType":"application/pdf"`)
	assert.Contains(t, jsonStr, `"uploadedBy":7`)
	assert.Contains(t, jsonStr, `"createdAt"`)
	assert.NotContains(t, jsonStr, `"ticket_id"`)
	assert.NotContains(t, jsonStr, `"file_name"`)
	assert.NotContains(t, jsonStr, `"file_path"`)
	assert.NotContains(t, jsonStr, `"file_url"`)
	assert.NotContains(t, jsonStr, `"file_size"`)
	assert.NotContains(t, jsonStr, `"file_type"`)
	assert.NotContains(t, jsonStr, `"mime_type"`)
	assert.NotContains(t, jsonStr, `"uploaded_by"`)
	assert.NotContains(t, jsonStr, `"created_at"`)
}
