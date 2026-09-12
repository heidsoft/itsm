package wecom

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/connector"
)

type WeCom struct {
	client    *Client
	cfg       connector.Config
	startedAt time.Time
	token     string // 回调 token；用于 VerifySignature
	aesKey    string // 回调加解密 aesKey（base64 编码）
}

func init() {
	connector.MustRegister(func() connector.Connector { return &WeCom{} })
}

func New() *WeCom { return &WeCom{} }

func (w *WeCom) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "wecom",
		Version:     "1.0.0",
		Title:       "企业微信 WeCom",
		Provider:    "wecom",
		Type:        connector.TypeIM,
		Description: "企业微信连接器：应用消息（精确到 UserID/部门/标签）+ 群机器人 Webhook + 回调验签与解密。支持 text/markdown/textcard/news。",
		Capabilities: []connector.Capability{
			connector.CapSendMessage,
			connector.CapSendCard,
			connector.CapReceiveMessage,
		},
		Tags:                []string{"im", "wecom", "wechat", "china"},
		Homepage:            "https://developer.work.weixin.qq.com",
		IsOfficial:          true,
		RequiredPermissions: []string{"connector:write"},
	}
}

func (w *WeCom) Init(_ context.Context, cfg connector.Config) error {
	corpID := cfg.Credentials["corp_id"]
	corpSecret := cfg.Credentials["corp_secret"]
	if corpID == "" || corpSecret == "" {
		return fmt.Errorf("wecom: credentials.corp_id and corp_secret are required")
	}
	baseURL, _ := cfg.Settings["base_url"].(string)
	w.client = NewClient(corpID, corpSecret, cfg.Credentials["agent_id"], baseURL)
	w.cfg = cfg
	w.token = cfg.Credentials["callback_token"]
	w.aesKey = cfg.Credentials["encoding_aes_key"]
	w.startedAt = time.Now()
	return nil
}

func (w *WeCom) Send(ctx context.Context, msg *connector.Message) error {
	if w.client == nil {
		return fmt.Errorf("wecom: connector not initialized")
	}
	return w.client.Send(ctx, msg)
}

func (w *WeCom) HealthCheck(ctx context.Context) connector.HealthStatus {
	if w.client == nil {
		return connector.HealthStatus{OK: false, Message: "not initialized"}
	}
	tok, err := w.client.Token(ctx)
	if err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	return connector.HealthStatus{
		OK:        tok != "",
		Message:   "access_token valid",
		CheckedAt: time.Now(),
		Extra:     map[string]interface{}{"started_at": w.startedAt, "uptime_s": int(time.Since(w.startedAt).Seconds())},
	}
}

func (w *WeCom) Close() error { return nil }

// VerifySignature 企业微信回调验签：
//
//	msg_signature = SHA1(token, timestamp, nonce, encrypt_or_echo_sorted)
//
// 支持 url_verification（只 echo echostr 时算签）与事件回调（带 msg_encrypt）。
func (w *WeCom) VerifySignature(headers map[string]string, body []byte) error {
	if w.client == nil {
		return fmt.Errorf("wecom: not initialized")
	}
	if w.token == "" {
		return fmt.Errorf("wecom: callback_token not configured")
	}
	ts := headers["timestamp"]
	nonce := headers["nonce"]
	sig := headers["msg_signature"]
	// url_verification 在 query 参数里
	if ts == "" {
		ts = headers["Timestamp"]
	}
	if nonce == "" {
		nonce = headers["Nonce"]
	}
	if sig == "" {
		sig = headers["msgSignature"]
	}
	if ts == "" || nonce == "" || sig == "" {
		return fmt.Errorf("wecom: missing signature headers")
	}
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("wecom: invalid timestamp: %w", err)
	}
	if mathabs(time.Now().Unix()-tsInt) > 5*60 {
		return fmt.Errorf("wecom: timestamp out of 5min window")
	}
	// 计算签名：用 body 中的 msg_encrypt 或 echostr（按字典序拼接）
	var payload struct {
		EchoStr    string `json:"echostr"`
		Encrypt    string `json:"Encrypt"`
		MsgEncrypt string `json:"msg_encrypt"`
	}
	_ = json.Unmarshal(body, &payload)
	parts := []string{payload.Encrypt, payload.MsgEncrypt, payload.EchoStr}
	nonEmpty := parts[:0]
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return fmt.Errorf("wecom: missing msg_encrypt/echostr")
	}
	sort.Strings(nonEmpty)
	expected := sha1Hex(append([]string{w.token, ts, nonce}, nonEmpty...)...)
	if !strings.EqualFold(expected, sig) {
		return fmt.Errorf("wecom: signature mismatch")
	}
	return nil
}

// sha1Hex 计算企业微信回调签名：
// 签名 = SHA1(sort([token, timestamp, nonce, msg_encrypt]))。
func sha1Hex(parts ...string) string {
	sorted := make([]string, len(parts))
	copy(sorted, parts)
	sort.Strings(sorted)
	h := sha1.New()
	h.Write([]byte(strings.Join(sorted, "")))
	return hex.EncodeToString(h.Sum(nil))
}

// ParseInbound 解析企业微信回调 body：保留 raw + msg_signature + echostr/encrypt。
// 真正的密文解密在 Handler 层（需要回调 token + aes_key 与 corp 配置一致的 PKCS#7）。
func (w *WeCom) ParseInbound(body []byte) (*connector.InboundMessage, error) {
	var base struct {
		ToUserName string `json:"ToUserName"`
		Encrypt    string `json:"Encrypt"`
		MsgEncrypt string `json:"msg_encrypt"`
		EchoStr    string `json:"echostr"`
	}
	if err := json.Unmarshal(body, &base); err != nil {
		return nil, fmt.Errorf("wecom: parse inbound: %w", err)
	}
	if base.EchoStr != "" {
		return &connector.InboundMessage{
			Type:       "url_verification",
			Content:    base.EchoStr,
			ReceivedAt: time.Now(),
			Extras:     map[string]interface{}{"raw": base},
		}, nil
	}
	if base.Encrypt == "" && base.MsgEncrypt == "" {
		return nil, fmt.Errorf("wecom: unknown event payload")
	}
	return &connector.InboundMessage{
		ConnectorType: connector.TypeIM,
		ConnectorName: "wecom",
		Type:          "encrypted_event",
		Content:       base.Encrypt + base.MsgEncrypt,
		Raw:           body,
		ReceivedAt:    time.Now(),
		Extras:        map[string]interface{}{"toUserName": base.ToUserName, "raw": base},
	}, nil
}

func mathabs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

var (
	_ connector.Connector = (*WeCom)(nil)
	_ connector.Receiver  = (*WeCom)(nil)
)
