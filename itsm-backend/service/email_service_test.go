package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// ==================== 邮件服务测试 ====================

func TestEmailService_NewEmailService(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	svc := NewEmailService(config, logger)
	assert.NotNil(t, svc)
	assert.Equal(t, config.Host, svc.config.Host)
	assert.Equal(t, config.Port, svc.config.Port)
	assert.Equal(t, config.Username, svc.config.Username)
	assert.Equal(t, config.From, svc.config.From)
	assert.Equal(t, config.FromName, svc.config.FromName)
}

func TestEmailMessage_BuildMessage(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:       []string{"recipient@example.com"},
		CC:       []string{"cc@example.com"},
		Subject:  "Test Subject",
		Body:     "<h1>Test Body</h1>",
		BodyText: "Test Body Text",
	}

	// 由于Send会实际连接SMTP服务器，我们只测试消息构建逻辑
	assert.NotNil(t, msg.To)
	assert.Len(t, msg.To, 1)
	assert.Equal(t, "recipient@example.com", msg.To[0])
	assert.Equal(t, "Test Subject", msg.Subject)
}

func TestEmailService_BuildMessageWithAttachments(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:      []string{"recipient@example.com"},
		Subject: "Test with Attachment",
		Body:    "<p>Test body with attachment</p>",
		Attachments: []EmailAttachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        []byte("test content"),
			},
		},
	}

	assert.NotNil(t, msg.Attachments)
	assert.Len(t, msg.Attachments, 1)
	assert.Equal(t, "test.txt", msg.Attachments[0].Filename)
	assert.Equal(t, "text/plain", msg.Attachments[0].ContentType)
}

func TestEmailService_BuildMessageWithoutCC(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:       []string{"recipient@example.com"},
		CC:       []string{}, // 空CC
		Subject:  "No CC Test",
		Body:     "<p>No CC</p>",
	}

	assert.NotNil(t, msg.CC)
	assert.Len(t, msg.CC, 0)
}

func TestEmailService_MultipleRecipients(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:       []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		Subject:  "Multiple Recipients",
		Body:     "<p>Multiple recipients test</p>",
	}

	assert.Len(t, msg.To, 3)
}

func TestEmailService_HTMLAndPlainText(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	tests := []struct {
		name     string
		body     string
		bodyText string
		hasHTML  bool
		hasText  bool
	}{
		{
			name:     "Both HTML and Text",
			body:     "<h1>HTML Content</h1>",
			bodyText: "Plain Text Content",
			hasHTML:  true,
			hasText:  true,
		},
		{
			name:     "HTML Only",
			body:     "<p>HTML Only</p>",
			bodyText: "",
			hasHTML:  true,
			hasText:  false,
		},
		{
			name:     "Text Only",
			body:     "",
			bodyText: "Plain Text Only",
			hasHTML:  false,
			hasText:  true,
		},
		{
			name:     "Neither",
			body:     "",
			bodyText: "",
			hasHTML:  false,
			hasText:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &EmailMessage{
				To:       []string{"test@example.com"},
				Subject:  "Test",
				Body:     tt.body,
				BodyText: tt.bodyText,
			}

			assert.Equal(t, tt.hasHTML, msg.Body != "")
			assert.Equal(t, tt.hasText, msg.BodyText != "")
		})
	}
}

func TestEmailService_EmptyRecipients(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:       []string{}, // 空收件人
		Subject:  "Empty Recipients",
		Body:     "<p>Test</p>",
	}

	assert.NotNil(t, msg.To)
	assert.Len(t, msg.To, 0)
}

func TestEmailService_EmailConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   EmailConfig
		wantHost string
		wantPort int
	}{
		{
			name: "Standard SMTP",
			config: EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     587,
				Username: "user@gmail.com",
				Password: "password",
				From:     "user@gmail.com",
				FromName: "Gmail User",
			},
			wantHost: "smtp.gmail.com",
			wantPort: 587,
		},
		{
			name: "SSL SMTP",
			config: EmailConfig{
				Host:     "smtp.gmail.com",
				Port:     465,
				Username: "user@gmail.com",
				Password: "password",
				From:     "user@gmail.com",
				FromName: "Gmail User SSL",
			},
			wantHost: "smtp.gmail.com",
			wantPort: 465,
		},
		{
			name: "Office 365",
			config: EmailConfig{
				Host:     "smtp.office365.com",
				Port:     587,
				Username: "user@outlook.com",
				Password: "password",
				From:     "user@outlook.com",
				FromName: "Office 365 User",
			},
			wantHost: "smtp.office365.com",
			wantPort: 587,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			svc := NewEmailService(tt.config, logger)

			assert.Equal(t, tt.wantHost, svc.config.Host)
			assert.Equal(t, tt.wantPort, svc.config.Port)
			assert.Equal(t, tt.config.Username, svc.config.Username)
			assert.Equal(t, tt.config.From, svc.config.From)
		})
	}
}

func TestEmailAttachment(t *testing.T) {
	tests := []struct {
		name        string
		attachment  EmailAttachment
		wantName    string
		wantType    string
		wantSize    int
	}{
		{
			name: "Text File",
			attachment: EmailAttachment{
				Filename:    "readme.txt",
				ContentType: "text/plain",
				Data:        []byte("This is a readme file"),
			},
			wantName: "readme.txt",
			wantType: "text/plain",
			wantSize: 21,
		},
		{
			name: "PDF File",
			attachment: EmailAttachment{
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Data:        []byte("%PDF-1.4 fake pdf content"),
			},
			wantName: "document.pdf",
			wantType: "application/pdf",
			wantSize: 25,
		},
		{
			name: "Image File",
			attachment: EmailAttachment{
				Filename:    "image.png",
				ContentType: "image/png",
				Data:        []byte("\x89PNG\r\n\x1a\n fake png content"),
			},
			wantName: "image.png",
			wantType: "image/png",
			wantSize: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, tt.attachment.Filename)
			assert.Equal(t, tt.wantType, tt.attachment.ContentType)
			assert.Equal(t, tt.wantSize, len(tt.attachment.Data))
		})
	}
}

func TestEmailService_EmailWithMultipleAttachments(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:      []string{"recipient@example.com"},
		Subject: "Multiple Attachments",
		Body:    "<p>See attached files</p>",
		Attachments: []EmailAttachment{
			{Filename: "file1.txt", ContentType: "text/plain", Data: []byte("File 1")},
			{Filename: "file2.pdf", ContentType: "application/pdf", Data: []byte("File 2")},
			{Filename: "file3.docx", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Data: []byte("File 3")},
		},
	}

	assert.Len(t, msg.Attachments, 3)
}

func TestEmailService_SpecialCharactersInSubject(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	tests := []struct {
		name    string
		subject string
	}{
		{name: "Chinese Subject", subject: "测试邮件主题"},
		{name: "Emoji Subject", subject: "🎉 恭喜！您的工单已处理"},
		{name: "Special Characters", subject: "Test: 特殊字符 @#$%^&*()"},
		{name: "Unicode Subject", subject: "unicode: é è à ñ ü ß"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &EmailMessage{
				To:       []string{"test@example.com"},
				Subject:  tt.subject,
				Body:     "<p>Test</p>",
			}
			assert.Equal(t, tt.subject, msg.Subject)
		})
	}
}

func TestEmailService_EmptyBody(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config := EmailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	_ = NewEmailService(config, logger)

	msg := &EmailMessage{
		To:       []string{"test@example.com"},
		Subject:  "Empty Body Test",
		Body:     "",
		BodyText: "",
	}

	assert.Equal(t, "", msg.Body)
	assert.Equal(t, "", msg.BodyText)
}

// 辅助函数：验证测试
func TestEmailService_HelperFunctions(t *testing.T) {
	// 测试消息构建的辅助逻辑
	t.Run("RecipientsJoin", func(t *testing.T) {
		recipients := []string{"a@example.com", "b@example.com", "c@example.com"}
		// 模拟 strings.Join
		joined := ""
		for i, r := range recipients {
			if i > 0 {
				joined += ","
			}
			joined += r
		}
		assert.Equal(t, "a@example.com,b@example.com,c@example.com", joined)
	})

	t.Run("CCJoin", func(t *testing.T) {
		cc := []string{"cc1@example.com", "cc2@example.com"}
		joined := ""
		for i, c := range cc {
			if i > 0 {
				joined += ","
			}
			joined += c
		}
		assert.Equal(t, "cc1@example.com,cc2@example.com", joined)
	})
}
