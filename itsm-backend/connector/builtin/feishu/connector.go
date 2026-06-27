package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/connector"
)

// Feishu 飞书连接器实现
// 复用 package 内 Client 以享受 tenant_access_token 缓存
type Feishu struct {
	client    *Client
	cfg       connector.Config
	startedAt time.Time
}

// ActionHandler 卡片按钮/回调事件
type ActionHandler func(ctx context.Context, event *CardActionEvent) (map[string]interface{}, error)

// CardActionEvent 飞书卡片交互事件（精简版）
type CardActionEvent struct {
	OpenID    string                 `json:"open_id"`
	UserID    string                 `json:"user_id"`
	ChatID    string                 `json:"chat_id"`
	Action    map[string]interface{} `json:"action"`
	Raw       map[string]interface{} `json:"raw"`
	Token     string                 `json:"token"`
	Timestamp string                 `json:"ts"`
}

func init() {
	connector.MustRegister(func() connector.Connector { return &Feishu{} })
}

func New() *Feishu { return &Feishu{} }

func (f *Feishu) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "feishu",
		Version:     "1.0.0",
		Title:       "飞书 / Lark",
		Provider:    "feishu",
		Type:        connector.TypeIM,
		Description: "飞书/Lark 开放平台连接器：发送/接收消息、卡片回调、签名校验。覆盖中国大陆及海外版本。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapReceiveMessage,
			connector.CapSendCard,
			connector.CapReplyMessage,
			connector.CapCreateTicket,
		},
		Tags:     []string{"im", "feishu", "lark", "china"},
		Homepage: "https://open.feishu.cn",
	}
}

func (f *Feishu) Init(_ context.Context, cfg connector.Config) error {
	appID := cfg.Credentials["app_id"]
	appSecret := cfg.Credentials["app_secret"]
	if appID == "" || appSecret == "" {
		return fmt.Errorf("feishu: credentials.app_id and app_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	if baseURL == "" {
		// 海外版判定
		if region, _ := cfg.Settings["region"].(string); region == "intl" {
			baseURL = BaseURLIntl
		}
	}
	f.client = NewClient(baseURL, appID, appSecret, cfg.Credentials["verification_token"], cfg.Credentials["encrypt_key"])
	f.cfg = cfg
	f.startedAt = time.Now()
	return nil
}

func (f *Feishu) Send(ctx context.Context, msg *connector.Message) error {
	if f.client == nil {
		return fmt.Errorf("feishu: connector not initialized")
	}
	return f.client.Send(ctx, msg)
}

func (f *Feishu) HealthCheck(ctx context.Context) connector.HealthStatus {
	if f.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := f.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "tenant_access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": f.startedAt, "uptime_s": int(time.Since(f.startedAt).Seconds())},
	}
}

func (f *Feishu) Close() error { return nil }

// VerifySignature 满足 connector.Receiver 接口
func (f *Feishu) VerifySignature(headers map[string]string, body []byte) error {
	if f.client == nil {
		return fmt.Errorf("feishu: not initialized")
	}
	ts := headers["X-Lark-Request-Timestamp"]
	nonce := headers["X-Lark-Request-Nonce"]
	sig := headers["X-Lark-Signature"]
	if ts == "" || sig == "" {
		return fmt.Errorf("feishu: missing signature headers")
	}
	if !f.client.VerifyEventSignature(ts, nonce, sig, body) {
		return fmt.Errorf("feishu: signature mismatch")
	}
	return nil
}

// ParseInbound 解析飞书事件回调
// 支持：url_verification / event_callback / card.action.trigger
func (f *Feishu) ParseInbound(body []byte) (*connector.InboundMessage, error) {
	var base struct {
		UUID      string `json:"uuid"`
		Token     string `json:"token"`
		Type      string `json:"type"`
		TS        string `json:"ts"`
		Challenge string `json:"challenge"`
		Header    *struct {
			AppID     string `json:"app_id"`
			TenantKey string `json:"tenant_key"`
			EventType string `json:"event_type"`
		} `json:"header"`
		Event map[string]interface{} `json:"event"`
	}
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("feishu: parse inbound: %w", err)
	}
	// URL Verification：原样返回 challenge
	if base.Type == "url_verification" {
		return &connector.InboundMessage{
			Type:       "url_verification",
			Content:    base.Challenge,
			ReceivedAt: time.Now(),
			Extras:     map[string]interface{}{"raw": base},
		}, nil
	}
	// 事件订阅
	if base.Header != nil {
		evType := base.Header.EventType
		msg := &connector.InboundMessage{
			ConnectorType: connector.TypeIM,
			ConnectorName: "feishu",
			MessageID:     base.UUID,
			Type:          evType,
			Raw:           body,
			ReceivedAt:    time.Now(),
			Extras:        map[string]interface{}{"ts": base.TS, "event": base.Event},
		}
		// 常见字段提取（im.message.receive_v1）
		if sender, ok := base.Event["sender"].(map[string]interface{}); ok {
			if sID, ok := sender["sender_id"].(map[string]interface{}); ok {
				if open, ok := sID["open_id"].(string); ok {
					msg.UserID = open
				}
			}
		}
		if msg2, ok := base.Event["message"].(map[string]interface{}); ok {
			if c, ok := msg2["content"].(map[string]interface{}); ok {
				if txt, ok := c["text"].(string); ok {
					msg.Content = txt
				}
			}
			if chID, ok := msg2["chat_id"].(string); ok {
				msg.ChatID = chID
			}
			if ct, ok := msg2["chat_type"].(string); ok {
				msg.ChatType = ct
			}
			if mID, ok := msg2["message_id"].(string); ok {
				msg.MessageID = mID
			}
		}
		// 卡片回调
		if evType == "card.action.trigger" {
			if act, ok := base.Event["action"].(map[string]interface{}); ok {
				msg.Type = "card_action"
				msg.Content = fmt.Sprintf("%v", act)
			}
		}
		// 应用机器人被加入/移除
		if evType == "im.chat.member.bot.added_v1" || evType == "im.chat.member.bot.deleted_v1" {
			msg.Type = evType
		}
		return msg, nil
	}
	return nil, fmt.Errorf("feishu: unknown event type=%s", base.Type)
}
