package dingtalk

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"itsm-backend/connector"
)

type DingTalk struct {
	client    *Client
	cfg       connector.Config
	startedAt time.Time
}

func init() {
	connector.MustRegister(func() connector.Connector { return &DingTalk{} })
}

func New() *DingTalk { return &DingTalk{} }

func (d *DingTalk) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "dingtalk",
		Version:     "1.0.0",
		Title:       "钉钉 DingTalk",
		Provider:    "dingtalk",
		Type:        connector.TypeIM,
		Description: "钉钉开放平台连接器：工作通知 + 群机器人（加签模式）+ Stream 回调验签。支持 text/markdown/actionCard。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapSendCard,
			connector.CapReplyMessage,
			connector.CapReceiveMessage,
		},
		Tags:                []string{"im", "dingtalk", "china"},
		Homepage:            "https://open.dingtalk.com",
		IsOfficial:          true,
		RequiredPermissions: []string{"connector:write"},
	}
}

func (d *DingTalk) Init(_ context.Context, cfg connector.Config) error {
	appKey := cfg.Credentials["app_key"]
	appSecret := cfg.Credentials["app_secret"]
	if appKey == "" || appSecret == "" {
		return fmt.Errorf("dingtalk: credentials.app_key and app_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	d.client = NewClient(appKey, appSecret, cfg.Credentials["agent_id"], baseURL)
	d.cfg = cfg
	d.startedAt = time.Now()
	return nil
}

func (d *DingTalk) Send(ctx context.Context, msg *connector.Message) error {
	if d.client == nil {
		return fmt.Errorf("dingtalk: connector not initialized")
	}
	return d.client.Send(ctx, msg)
}

func (d *DingTalk) HealthCheck(ctx context.Context) connector.HealthStatus {
	if d.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := d.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": d.startedAt, "uptime_s": int(time.Since(d.startedAt).Seconds())},
	}
}

func (d *DingTalk) Close() error { return nil }

// VerifySignature 钉钉 stream 回调验签：HMAC-SHA256(secret, timestamp + "\n" + body)
// + timestamp 5 分钟窗口（防重放）。同时支持 url_verification 探活（无头可空）。
func (d *DingTalk) VerifySignature(headers map[string]string, body []byte) error {
	if d.client == nil {
		return fmt.Errorf("dingtalk: not initialized")
	}
	ts := headers["Timestamp"]
	if ts == "" {
		ts = headers["timestamp"]
	}
	sig := headers["Signature"]
	if sig == "" {
		sig = headers["signature"]
	}
	// 探活事件允许无签名
	if ts == "" && sig == "" {
		return nil
	}
	if ts == "" || sig == "" {
		return fmt.Errorf("dingtalk: missing signature headers")
	}
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("dingtalk: invalid timestamp: %w", err)
	}
	now := time.Now().UnixMilli()
	if math.Abs(float64(now-tsInt)) > 5*60*1000 {
		return fmt.Errorf("dingtalk: timestamp out of 5min window")
	}
	if !d.client.VerifyStreamSignature(ts, sig, body) {
		return fmt.Errorf("dingtalk: signature mismatch")
	}
	return nil
}

// ParseInbound 解析钉钉 stream 回调 payload。
// 支持 url_verification（直接返回 echostr 字段）、event_callback（封装 event 详情）。
func (d *DingTalk) ParseInbound(body []byte) (*connector.InboundMessage, error) {
	var base struct {
		EventType  string                 `json:"EventType"`
		EventID    string                 `json:"EventId"`
		Event      map[string]interface{} `json:"Event"`
		Timestamp  string                 `json:"Timestamp"`
		Nonce      string                 `json:"Nonce"`
		EchoStr    string                 `json:"echostr"`
		CheckToken string                 `json:"check_token"`
	}
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("dingtalk: parse inbound: %w", err)
	}
	if base.EventType == "url_verification" || base.EchoStr != "" {
		return &connector.InboundMessage{
			Type:       "url_verification",
			Content:    base.EchoStr,
			ReceivedAt: time.Now(),
			Extras:     map[string]interface{}{"raw": base},
		}, nil
	}
	if base.EventType == "" && base.Event == nil {
		return nil, fmt.Errorf("dingtalk: unknown event payload")
	}
	msg := &connector.InboundMessage{
		ConnectorType: connector.TypeIM,
		ConnectorName: "dingtalk",
		MessageID:     base.EventID,
		Type:          base.EventType,
		Raw:           body,
		ReceivedAt:    time.Now(),
		Extras:        map[string]interface{}{"ts": base.Timestamp, "nonce": base.Nonce, "event": base.Event},
	}
	if sender, ok := base.Event["sender"].(map[string]interface{}); ok {
		if id, ok := sender["sender_id"].(string); ok {
			msg.UserID = id
		}
	}
	if text, ok := base.Event["text"].(map[string]interface{}); ok {
		if content, ok := text["content"].(string); ok {
			msg.Content = content
		}
	}
	if chatID, ok := base.Event["chatId"].(string); ok {
		msg.ChatID = chatID
	}
	return msg, nil
}

var _ connector.Connector = (*DingTalk)(nil)
var _ connector.Receiver = (*DingTalk)(nil)
