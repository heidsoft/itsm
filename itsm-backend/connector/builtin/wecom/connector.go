package wecom

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
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
	baseURL, _ := connector.SettingString(cfg.Settings, "base_url")
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

// ParseInbound 解析企业微信回调 body：先验签（Handler 已做），再解密。
//
// url_verification 的 echostr 与事件回调的 msg_encrypt 都是密文，必须用配置的
// EncodingAESKey 解密：前者还原出明文 echo 原样返回，后者解出 XML 再归一化为
// InboundMessage。解密时校验 receiveid 与配置 corpid 一致，拒绝跨企业伪造。
// 未配置 encoding_aes_key 或解密失败一律返回错误（fail-closed），绝不把密文当明文放行。
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

	aesKey, err := decodeAESKey(w.aesKey)
	if err != nil {
		return nil, err
	}
	corpID := w.cfg.Credentials["corp_id"]

	if base.EchoStr != "" {
		echo, receiveID, decErr := decryptMessage(aesKey, base.EchoStr)
		if decErr != nil {
			return nil, fmt.Errorf("wecom: decrypt echostr: %w", decErr)
		}
		if corpID != "" && receiveID != corpID {
			return nil, fmt.Errorf("wecom: echostr receiveid mismatch")
		}
		return &connector.InboundMessage{
			Type:       "url_verification",
			Content:    string(echo),
			ReceivedAt: time.Now(),
			Extras:     map[string]interface{}{"raw": base},
		}, nil
	}

	ciphertext := base.Encrypt
	if ciphertext == "" {
		ciphertext = base.MsgEncrypt
	}
	if ciphertext == "" {
		return nil, fmt.Errorf("wecom: unknown event payload")
	}
	plaintext, receiveID, err := decryptMessage(aesKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("wecom: decrypt event: %w", err)
	}
	if corpID != "" && receiveID != corpID {
		return nil, fmt.Errorf("wecom: event receiveid mismatch")
	}

	var event wecomEventXML
	if err := xml.Unmarshal(plaintext, &event); err != nil {
		return nil, fmt.Errorf("wecom: parse decrypted xml: %w", err)
	}
	msgType := event.MsgType
	if msgType == "" {
		msgType = "event"
	}
	messageID := event.MsgID
	if messageID == "" {
		// 事件回调通常没有 MsgId；用确定性字段合成去重键，避免重放绕过 Handler 去重。
		messageID = fmt.Sprintf("%d:%s:%s:%s", event.CreateTime, event.FromUserName, event.Event, event.EventKey)
	}
	return &connector.InboundMessage{
		ConnectorType: connector.TypeIM,
		ConnectorName: "wecom",
		Type:          msgType,
		Content:       event.Content,
		UserID:        event.FromUserName,
		MessageID:     messageID,
		Raw:           plaintext,
		ReceivedAt:    time.Now(),
		Extras: map[string]interface{}{
			"toUserName": base.ToUserName,
			"event":      event.Event,
			"eventKey":   event.EventKey,
			"agentId":    event.AgentID,
			"createTime": event.CreateTime,
		},
	}, nil
}

// wecomEventXML 是解密后消息体的 XML 结构（文本消息与事件回调共用）。
type wecomEventXML struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgID        string `xml:"MsgId"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
	AgentID      string `xml:"AgentID"`
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
