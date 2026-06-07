package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
