package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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
	if s.config.AccessKey == "" || s.config.SecretKey == "" {
		s.logger.Warnw("Aliyun SMS credentials not configured, using mock")
		return s.sendMock(ctx, msg)
	}

	// 构建请求参数
	params := make(map[string]string)
	params["AccessKeyId"] = s.config.AccessKey
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Action"] = "SendSms"
	params["Version"] = "2017-05-25"
	params["RegionId"] = s.config.Region
	if s.config.Region == "" {
		params["RegionId"] = "cn-hangzhou"
	}
	params["PhoneNumbers"] = strings.Join(msg.PhoneNumbers, ",")
	params["SignName"] = msg.SignName
	if msg.SignName == "" {
		params["SignName"] = s.config.SignName
	}
	if msg.TemplateCode != "" {
		params["TemplateCode"] = msg.TemplateCode
	}
	if len(msg.TemplateParam) > 0 {
		templateParamJSON := SafeMarshal(msg.TemplateParam)
		params["TemplateParam"] = string(templateParamJSON)
	}

	// 生成签名
	signature := s.generateAliyunSignature(params, "POST")
	params["Signature"] = signature

	// 发送请求
	endpoint := s.config.Endpoint
	if endpoint == "" {
		endpoint = "dysmsapi.aliyuncs.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://"+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 构建查询字符串
	queryParams := url.Values{}
	for k, v := range params {
		queryParams.Set(k, v)
	}
	req.URL.RawQuery = queryParams.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AliyunSMSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != "OK" {
		s.logger.Errorw("Aliyun SMS failed", "code", result.Code, "message", result.Message)
		return fmt.Errorf("SMS send failed: %s - %s", result.Code, result.Message)
	}

	s.logger.Infow("Aliyun SMS sent successfully", "bizId", result.BizID)
	return nil
}

// generateAliyunSignature 生成阿里云签名
func (s *SMSService) generateAliyunSignature(params map[string]string, method string) string {
	// 1. 排序参数键
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建规范化查询字符串
	var sortedParams []string
	for _, k := range keys {
		value := url.QueryEscape(params[k])
		sortedParams = append(sortedParams, fmt.Sprintf("%s=%s", url.QueryEscape(k), value))
	}
	canonicalizedQueryString := strings.Join(sortedParams, "&")

	// 3. 构建签名字符串
	stringToSign := fmt.Sprintf("%s&%s&%s",
		method,
		url.QueryEscape("/"),
		url.QueryEscape(canonicalizedQueryString))

	// 4. 计算 HMAC-SHA1 签名
	h := hmac.New(sha1.New, []byte(s.config.SecretKey+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// sendTencent 腾讯云短信发送
func (s *SMSService) sendTencent(ctx context.Context, msg *SMSMessage) error {
	if s.config.AccessKey == "" || s.config.SecretKey == "" {
		s.logger.Warnw("Tencent SMS credentials not configured, using mock")
		return s.sendMock(ctx, msg)
	}

	// 腾讯云短信需要先格式化手机号
	tel := make(map[string]string)
	if len(msg.PhoneNumbers) > 0 {
		tel["mobile"] = msg.PhoneNumbers[0]
		tel["nationcode"] = "86" // 默认中国
	}

	// 构建请求体
	reqBody := map[string]interface{}{
		"tel":    tel,
		"sdkappid": s.config.Region, // 使用 Region 字段存储 AppID
		"time":    time.Now().Unix(),
	}

	// 模板短信
	if msg.TemplateCode != "" {
		reqBody["tpl_id"] = msg.TemplateCode
		if len(msg.TemplateParam) > 0 {
			// 腾讯云模板参数是数组格式
			var params []string
			for _, v := range msg.TemplateParam {
				params = append(params, v)
			}
			reqBody["params"] = params
		}
		msgStr := msg.Content
		if msgStr == "" {
			msgStr = msg.SignName
		}
		reqBody["msg"] = msgStr
		reqBody["type"] = 0 // 0: 普通短信, 1: 营销短信
	} else {
		reqBody["msg"] = msg.Content
		reqBody["type"] = 0
	}

	// 生成签名
	sig := s.generateTencentSignature(tel["mobile"], time.Now().Unix(), reqBody)
	reqBody["sig"] = sig

	// 发送请求
	endpoint := s.config.Endpoint
	if endpoint == "" {
		endpoint = "sms.tencentcloudapi.com"
	}

	bodyBytes := SafeMarshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://"+endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result TencentSMSResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.BaseResponse.Result != 0 {
		s.logger.Errorw("Tencent SMS failed", "result", result.Response.BaseResponse.Result, "message", result.Response.BaseResponse.Msg)
		return fmt.Errorf("SMS send failed: %d - %s", result.Response.BaseResponse.Result, result.Response.BaseResponse.Msg)
	}

	s.logger.Infow("Tencent SMS sent successfully")
	return nil
}

// generateTencentSignature 生成腾讯云签名
func (s *SMSService) generateTencentSignature(mobile string, timestamp int64, body map[string]interface{}) string {
	// 腾讯云签名字符串格式: appkey+timestamp+mobile
	appkey := s.config.SecretKey

	// 将 body 按 key 排序
	var keys []string
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var bodyStr string
	for _, k := range keys {
		bodyStr += fmt.Sprintf("%s=%v&", k, body[k])
	}

	stringToSign := fmt.Sprintf("appkey=%s&time=%d&mobile=%s&%s",
		appkey, timestamp, mobile, bodyStr[:len(bodyStr)-1])

	// HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(appkey))
	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}

// TencentSMSResponse 腾讯云短信响应结构
type TencentSMSResponse struct {
	Response struct {
		BaseResponse struct {
			Result int    `json:"Result"`
			Msg    string `json:"Msg"`
		} `json:"BaseResponse"`
	} `json:"Response"`
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
	msg := &SMSMessage{
		PhoneNumbers:  phoneNumbers,
		TemplateCode:  templateCode,
		TemplateParam: params,
		SignName:      s.config.SignName,
	}
	return s.sendAliyun(ctx, msg)
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
	msg := &SMSMessage{
		PhoneNumbers:  phoneNumbers,
		TemplateCode:  templateID,
		TemplateParam: params,
		SignName:      s.config.SignName,
	}
	return s.sendTencent(ctx, msg)
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
