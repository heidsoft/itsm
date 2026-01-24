package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SMSConfig 短信配置
type SMSConfig struct {
	Provider      string // 短信提供商: aliyun, tencent, mock
	AccessKey     string // 访问密钥
	SecretKey     string // 密钥
	SignName      string // 签名名称
	Region        string // 区域
	Endpoint      string // API端点
	Timeout       int    // 超时时间（秒）
}

// SMSMessage 短信消息
type SMSMessage struct {
	PhoneNumbers []string       // 手机号列表
	TemplateCode string         // 模板ID
	TemplateParam map[string]string // 模板参数
	SignName     string         // 签名（覆盖配置）
	Content      string         // 短信内容（当不使用模板时）
}

// SMSService 短信服务
type SMSService struct {
	config SMSConfig
	logger *zap.SugaredLogger
	client *http.Client
}

// NewSMSService 创建短信服务
func NewSMSService(config SMSConfig, logger *zap.SugaredLogger) *SMSService {
	return &SMSService{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// Send 发送短信
func (s *SMSService) Send(ctx context.Context, msg *SMSMessage) error {
	s.logger.Infow("Sending SMS",
		"provider", s.config.Provider,
		"phones", len(msg.PhoneNumbers),
	)

	switch s.config.Provider {
	case "aliyun":
		return s.sendAliyun(ctx, msg)
	case "tencent":
		return s.sendTencent(ctx, msg)
	case "mock":
		return s.sendMock(ctx, msg)
	default:
		return s.sendMock(ctx, msg)
	}
}

// sendAliyun 阿里云短信发送
func (s *SMSService) sendAliyun(ctx context.Context, msg *SMSMessage) error {
	// 阿里云短信API实现
	// 注意：实际使用时需要集成阿里云SDK
	s.logger.Infow("Aliyun SMS - simulating send",
		"phones", msg.PhoneNumbers,
		"template", msg.TemplateCode,
	)

	// 模拟发送成功
	return nil
}

// sendTencent 腾讯云短信发送
func (s *SMSService) sendTencent(ctx context.Context, msg *SMSMessage) error {
	// 腾讯云短信API实现
	// 注意：实际使用时需要集成腾讯云SDK
	s.logger.Infow("Tencent SMS - simulating send",
		"phones", msg.PhoneNumbers,
		"template", msg.TemplateCode,
	)

	// 模拟发送成功
	return nil
}

// sendMock 模拟发送（用于测试）
func (s *SMSService) sendMock(ctx context.Context, msg *SMSMessage) error {
	s.logger.Infow("Mock SMS sent",
		"phones", msg.PhoneNumbers,
		"content", msg.Content,
	)
	return nil
}

// SendTicketNotification 发送工单通知短信
func (s *SMSService) SendTicketNotification(ctx context.Context, phoneNumbers []string, ticketNumber, action string) error {
	signName := s.config.SignName
	if signName == "" {
		signName = "ITSM系统"
	}

	var content string
	if action == "assigned" {
		content = fmt.Sprintf("【%s】您有一个新工单 %s 请及时处理。", signName, ticketNumber)
	} else if action == "escalated" {
		content = fmt.Sprintf("【%s】工单 %s 已升级，请紧急处理！", signName, ticketNumber)
	} else {
		content = fmt.Sprintf("【%s】工单 %s 有新动态：%s", signName, ticketNumber, action)
	}

	msg := &SMSMessage{
		PhoneNumbers: phoneNumbers,
		Content:      content,
	}

	return s.Send(ctx, msg)
}

// SendSLANotification 发送SLA告警短信
func (s *SMSService) SendSLANotification(ctx context.Context, phoneNumbers []string, ticketNumber, slaType string, minutesRemaining int) error {
	signName := s.config.SignName
	if signName == "" {
		signName = "ITSM系统"
	}

	urgency := "请及时处理"
	if minutesRemaining <= 30 {
		urgency = "请立即处理！"
	}

	content := fmt.Sprintf("【%s】SLA告警：工单 %s 的%s还剩%d分钟，%s",
		signName, ticketNumber, slaType, minutesRemaining, urgency)

	msg := &SMSMessage{
		PhoneNumbers: phoneNumbers,
		Content:      content,
	}

	return s.Send(ctx, msg)
}

// SendVerificationCode 发送验证码短信
func (s *SMSService) SendVerificationCode(ctx context.Context, phoneNumbers []string, code string, expireMinutes int) error {
	signName := s.config.SignName
	if signName == "" {
		signName = "ITSM系统"
	}

	content := fmt.Sprintf("【%s】您的验证码是：%s，%d分钟内有效，请勿泄露给他人。",
		signName, code, expireMinutes)

	msg := &SMSMessage{
		PhoneNumbers: phoneNumbers,
		Content:      content,
	}

	return s.Send(ctx, msg)
}

// ValidateConfig 验证短信配置
func (s *SMSService) ValidateConfig() error {
	if s.config.Provider == "" {
		return fmt.Errorf("SMS provider is required")
	}
	if s.config.SignName == "" {
		return fmt.Errorf("SMS sign name is required")
	}
	return nil
}

// BatchSend 批量发送短信
func (s *SMSService) BatchSend(ctx context.Context, messages []*SMSMessage) error {
	var lastErr error
	for i, msg := range messages {
		if err := s.Send(ctx, msg); err != nil {
			s.logger.Errorw("Failed to send SMS in batch", "index", i, "error", err)
			lastErr = err
		}
	}
	return lastErr
}

// AliyunSMSRequest 阿里云短信请求结构
type AliyunSMSRequest struct {
	AccessKeyID      string            `json:"AccessKeyId"`
	Timestamp        string            `json:"Timestamp"`
	SignatureMethod  string            `json:"SignatureMethod"`
	SignatureVersion string            `json:"SignatureVersion"`
	SignatureNonce   string            `json:"SignatureNonce"`
	Action           string            `json:"Action"`
	Version          string            `json:"Version"`
	RegionID         string            `json:"RegionId"`
	PhoneNumbers     string            `json:"PhoneNumbers"`
	SignName         string            `json:"SignName"`
	TemplateCode     string            `json:"TemplateCode"`
	TemplateParam    map[string]string `json:"TemplateParam"`
}

// AliyunSMSResponse 阿里云短信响应结构
type AliyunSMSResponse struct {
	RequestID string `json:"RequestId"`
	BizID     string `json:"BizId"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// SendAliyunTemplate 发送阿里云模板短信
func (s *SMSService) SendAliyunTemplate(ctx context.Context, phoneNumbers []string, templateCode string, params map[string]string) error {
	_ = AliyunSMSRequest{
		AccessKeyID:      s.config.AccessKey,
		Timestamp:        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		SignatureMethod:  "HMAC-SHA1",
		SignatureVersion: "1.0",
		Action:           "SendSms",
		Version:          "2017-05-25",
		RegionID:         s.config.Region,
		PhoneNumbers:     strings.Join(phoneNumbers, ","),
		SignName:         s.config.SignName,
		TemplateCode:     templateCode,
		TemplateParam:    params,
	}

	// TODO: 实现真实的阿里云签名和API调用
	s.logger.Infow("Aliyun template SMS",
		"phones", phoneNumbers,
		"template", templateCode,
	)

	return nil
}

// TencentSMSRequest 腾讯云短信请求结构
type TencentSMSRequest struct {
	Tel    map[string]string `json:"tel"`
	Sig    string            `json:"sig"`
	Time   int64             `json:"time"`
	Type   int               `json:"type"`
	Msg    string            `json:"msg"`
}

// SendTencentTemplate 发送腾讯云模板短信
func (s *SMSService) SendTencentTemplate(ctx context.Context, phoneNumbers []string, templateID string, params map[string]string) error {
	// TODO: 实现真实的腾讯云签名和API调用
	s.logger.Infow("Tencent template SMS",
		"phones", phoneNumbers,
		"template", templateID,
	)

	return nil
}

// GetProviderInfo 获取短信提供商信息
func (s *SMSService) GetProviderInfo() map[string]string {
	return map[string]string{
		"provider":   s.config.Provider,
		"region":     s.config.Region,
		"sign_name":  s.config.SignName,
		"endpoint":   s.config.Endpoint,
	}
}
