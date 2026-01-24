package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string // SMTP服务器地址
	Port     int    // SMTP端口
	Username string // 用户名
	Password string // 密码
	From     string // 发件人地址
	FromName string // 发件人名称
}

// EmailService 邮件服务
type EmailService struct {
	config EmailConfig
	logger *zap.SugaredLogger
}

// EmailMessage 邮件消息
type EmailMessage struct {
	To          []string       // 收件人列表
	CC          []string       // 抄送人列表
	Subject     string         // 邮件主题
	Body        string         // 邮件正文（HTML）
	BodyText    string         // 邮件正文（纯文本）
	Attachments []EmailAttachment // 附件
}

// EmailAttachment 邮件附件
type EmailAttachment struct {
	Filename    string // 文件名
	ContentType string // 内容类型
	Data        []byte // 文件内容
}

// NewEmailService 创建邮件服务
func NewEmailService(config EmailConfig, logger *zap.SugaredLogger) *EmailService {
	return &EmailService{
		config: config,
		logger: logger,
	}
}

// Send 发送邮件
func (s *EmailService) Send(ctx context.Context, msg *EmailMessage) error {
	s.logger.Infow("Sending email",
		"to", msg.To,
		"subject", msg.Subject,
	)

	// 构建邮件内容
	var emailBody strings.Builder

	// MIME边界
	boundary := "Mixed_123456789"

	// 邮件头
	emailBody.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.From))
	emailBody.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ",")))
	if len(msg.CC) > 0 {
		emailBody.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ",")))
	}
	emailBody.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	emailBody.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	emailBody.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	emailBody.WriteString("\r\n")

	// 纯文本版本
	if msg.BodyText != "" {
		emailBody.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		emailBody.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		emailBody.WriteString("Content-Transfer-Encoding: base64\r\n")
		emailBody.WriteString("\r\n")
		emailBody.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.BodyText)))
		emailBody.WriteString("\r\n")
	}

	// HTML版本
	if msg.Body != "" {
		emailBody.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		emailBody.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		emailBody.WriteString("Content-Transfer-Encoding: base64\r\n")
		emailBody.WriteString("\r\n")
		emailBody.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.Body)))
		emailBody.WriteString("\r\n")
	}

	// 添加附件
	for _, att := range msg.Attachments {
		emailBody.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		emailBody.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename))
		emailBody.WriteString("Content-Transfer-Encoding: base64\r\n")
		emailBody.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
		emailBody.WriteString("\r\n")
		emailBody.WriteString(base64.StdEncoding.EncodeToString(att.Data))
		emailBody.WriteString("\r\n")
	}

	emailBody.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// 发送邮件
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	err := smtp.SendMail(addr, auth, s.config.From, msg.To, []byte(emailBody.String()))
	if err != nil {
		s.logger.Errorw("Failed to send email", "error", err, "to", msg.To)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Infow("Email sent successfully", "to", msg.To, "subject", msg.Subject)
	return nil
}

// SendTemplate 发送模板邮件
func (s *EmailService) SendTemplate(ctx context.Context, msg *EmailMessage, templateName string, data interface{}) error {
	s.logger.Infow("Sending template email", "template", templateName, "to", msg.To)

	// 解析模板
	tmpl, err := template.New(templateName).Parse(msg.Body)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	msg.Body = buf.String()

	return s.Send(ctx, msg)
}

// SendTicketNotification 发送工单通知邮件
func (s *EmailService) SendTicketNotification(ctx context.Context, to []string, ticketNumber, ticketTitle, action, content string) error {
	subject := fmt.Sprintf("[ITSM] 工单 %s - %s", ticketNumber, action)

	bodyText := fmt.Sprintf(`
工单通知
========

工单编号: %s
工单标题: %s
操作: %s

%s

---
此邮件由ITSM系统自动发送
发送时间: %s
`, ticketNumber, ticketTitle, action, content, time.Now().Format("2006-01-02 15:04:05"))

	bodyHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #333;">工单通知</h2>
        <table style="width: 100%%; border-collapse: collapse; margin: 20px 0;">
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">工单编号</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">工单标题</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">操作</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
        </table>
        <div style="background-color: #f5f5f5; padding: 15px; border-radius: 5px;">
            <p style="margin: 0;">%s</p>
        </div>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
        <p style="color: #888; font-size: 12px;">此邮件由ITSM系统自动发送</p>
    </div>
</body>
</html>
`, ticketNumber, ticketTitle, action, content)

	msg := &EmailMessage{
		To:       to,
		Subject:  subject,
		Body:     bodyHTML,
		BodyText: bodyText,
	}

	return s.Send(ctx, msg)
}

// SendSLANotification 发送SLA告警邮件
func (s *EmailService) SendSLANotification(ctx context.Context, to []string, ticketNumber, ticketTitle, slaType, deadline string) error {
	subject := fmt.Sprintf("[ITSM SLA告警] 工单 %s - %s 即将到期", ticketNumber, slaType)

	bodyText := fmt.Sprintf(`
SLA告警通知
===========

工单编号: %s
工单标题: %s
SLA类型: %s
截止时间: %s

请及时处理，避免SLA违规。

---
此邮件由ITSM系统自动发送
发送时间: %s
`, ticketNumber, ticketTitle, slaType, deadline, time.Now().Format("2006-01-02 15:04:05"))

	bodyHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #f59e0b;">⚠️ SLA告警通知</h2>
        <table style="width: 100%%; border-collapse: collapse; margin: 20px 0;">
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">工单编号</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">工单标题</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">SLA类型</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd;">%s</td>
            </tr>
            <tr>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; font-weight: bold;">截止时间</td>
                <td style="padding: 10px; border-bottom: 1px solid #ddd; color: #ef4444;">%s</td>
            </tr>
        </table>
        <div style="background-color: #fef3c7; padding: 15px; border-radius: 5px; border-left: 4px solid #f59e0b;">
            <p style="margin: 0; color: #92400e;">请及时处理，避免SLA违规！</p>
        </div>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
        <p style="color: #888; font-size: 12px;">此邮件由ITSM系统自动发送</p>
    </div>
</body>
</html>
`, ticketNumber, ticketTitle, slaType, deadline)

	msg := &EmailMessage{
		To:       to,
		Subject:  subject,
		Body:     bodyHTML,
		BodyText: bodyText,
	}

	return s.Send(ctx, msg)
}

// ValidateConfig 验证邮件配置
func (s *EmailService) ValidateConfig() error {
	if s.config.Host == "" {
		return fmt.Errorf("email host is required")
	}
	if s.config.Port == 0 {
		return fmt.Errorf("email port is required")
	}
	if s.config.Username == "" {
		return fmt.Errorf("email username is required")
	}
	if s.config.From == "" {
		return fmt.Errorf("email from address is required")
	}
	return nil
}
