package scenarios

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/connector"
	"itsm-backend/connector/builtin/dingtalk"
	"itsm-backend/connector/builtin/feishu"
	"itsm-backend/connector/builtin/wecom"
)

// Scenario 7: 连接器端到端（Feishu/DingTalk/WeCom 签名验证 + Manager 生命周期 + Router 去重）
//
// 覆盖：
//   - 飞书 HMAC-SHA256 事件签名校验（合法/非法/空参数）
//   - 钉钉 Stream 签名校验（合法/非法/5 分钟窗口/缺头拒绝）
//   - 企微 SHA1 签名校验（合法/非法/排序拼接）
//   - Manager Provision / Send / Revoke 生命周期
//   - Router 入站消息去重（5 分钟 TTL）

func TestScenario7_ConnectorE2E(t *testing.T) {
	ctx := context.Background()

	t.Run("feishu HMAC-SHA256 event signature", func(t *testing.T) {
		encryptKey := "test-encrypt-key-12345"
		client := feishu.NewClient("", "app_id", "app_secret", "verify_token", encryptKey)

		ts := strconv.FormatInt(time.Now().Unix(), 10)
		nonce := "test-nonce"
		body := []byte(`{"type":"event_callback"}`)

		mac := hmac.New(sha256.New, []byte(encryptKey))
		mac.Write([]byte(ts))
		mac.Write([]byte(nonce))
		mac.Write(body)
		validSig := hex.EncodeToString(mac.Sum(nil))

		if !client.VerifyEventSignature(ts, nonce, validSig, body) {
			t.Fatal("合法签名应通过校验")
		}

		if client.VerifyEventSignature(ts, nonce, "invalid-signature", body) {
			t.Fatal("非法签名应被拒绝")
		}

		if client.VerifyEventSignature("", nonce, validSig, body) {
			t.Fatal("空 timestamp 应被拒绝")
		}
		if client.VerifyEventSignature(ts, "", validSig, body) {
			t.Fatal("空 nonce 应被拒绝")
		}
		if client.VerifyEventSignature(ts, nonce, "", body) {
			t.Fatal("空 signature 应被拒绝")
		}

		wrongMac := hmac.New(sha256.New, []byte("wrong-key"))
		wrongMac.Write([]byte(ts))
		wrongMac.Write([]byte(nonce))
		wrongMac.Write(body)
		wrongSig := hex.EncodeToString(wrongMac.Sum(nil))
		if client.VerifyEventSignature(ts, nonce, wrongSig, body) {
			t.Fatal("错误密钥生成的签名应被拒绝")
		}
	})

	t.Run("feishu URL verification token", func(t *testing.T) {
		client := feishu.NewClient("", "app_id", "app_secret", "my-verify-token", "encrypt-key")

		if !client.VerifyURLToken("my-verify-token") {
			t.Fatal("正确的 verification token 应通过")
		}
		if client.VerifyURLToken("wrong-token") {
			t.Fatal("错误的 verification token 应被拒绝")
		}
		if client.VerifyURLToken("") {
			t.Fatal("空 token 应被拒绝")
		}
	})

	t.Run("dingtalk stream signature verification", func(t *testing.T) {
		dt := dingtalk.New()
		cfg := connector.Config{
			TenantID: 1, Name: "dingtalk", Provider: "dingtalk",
			Enabled: true,
			Credentials: map[string]string{
				"app_key":    "test-key",
				"app_secret": "test-secret-for-signing",
			},
		}
		if err := dt.Init(ctx, cfg); err != nil {
			t.Fatalf("init dingtalk: %v", err)
		}

		body := []byte(`{"EventType":"test_event"}`)
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

		mac := hmac.New(sha256.New, []byte("test-secret-for-signing"))
		mac.Write([]byte(ts))
		mac.Write([]byte("\n"))
		mac.Write(body)
		validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		headers := map[string]string{"Timestamp": ts, "Signature": validSig}
		if err := dt.VerifySignature(headers, body); err != nil {
			t.Fatalf("合法钉钉签名应通过: %v", err)
		}

		badHeaders := map[string]string{"Timestamp": ts, "Signature": "bad-sig"}
		if err := dt.VerifySignature(badHeaders, body); err == nil {
			t.Fatal("非法钉钉签名应被拒绝")
		}

		oldTs := strconv.FormatInt(time.Now().UnixMilli()-10*60*1000, 10)
		oldHeaders := map[string]string{"Timestamp": oldTs, "Signature": validSig}
		if err := dt.VerifySignature(oldHeaders, body); err == nil {
			t.Fatal("超过 5 分钟窗口的钉钉签名应被拒绝")
		}

		probeHeaders := map[string]string{}
		if err := dt.VerifySignature(probeHeaders, body); err == nil {
			t.Fatal("无签名头的探活事件应被拒绝（fail-closed），不得放行")
		}
	})

	t.Run("dingtalk parse inbound", func(t *testing.T) {
		dt := dingtalk.New()
		cfg := connector.Config{
			TenantID: 1, Name: "dingtalk", Provider: "dingtalk", Enabled: true,
			Credentials: map[string]string{"app_key": "k", "app_secret": "s"},
		}
		_ = dt.Init(ctx, cfg)

		urlVerify := []byte(`{"EventType":"url_verification","echostr":"hello-echo"}`)
		msg, err := dt.ParseInbound(urlVerify)
		if err != nil {
			t.Fatalf("解析 url_verification 失败: %v", err)
		}
		if msg.Type != "url_verification" || msg.Content != "hello-echo" {
			t.Fatalf("url_verification 解析不正确: type=%s content=%s", msg.Type, msg.Content)
		}

		eventBody, _ := json.Marshal(map[string]interface{}{
			"EventType": "event_callback",
			"EventId":   "evt-001",
			"Event": map[string]interface{}{
				"sender": map[string]interface{}{"sender_id": "user123"},
				"text":   map[string]interface{}{"content": "帮我查工单"},
				"chatId": "chat-abc",
			},
		})
		msg, err = dt.ParseInbound(eventBody)
		if err != nil {
			t.Fatalf("解析 event_callback 失败: %v", err)
		}
		if msg.UserID != "user123" || msg.Content != "帮我查工单" || msg.ChatID != "chat-abc" {
			t.Fatalf("event_callback 解析不正确: user=%s content=%s chat=%s", msg.UserID, msg.Content, msg.ChatID)
		}
	})

	t.Run("wecom SHA1 signature verification", func(t *testing.T) {
		wc := wecom.New()
		cfg := connector.Config{
			TenantID: 1, Name: "wecom", Provider: "wecom", Enabled: true,
			Credentials: map[string]string{
				"corp_id":        "corp123",
				"corp_secret":    "secret456",
				"agent_id":       "1001",
				"callback_token": "wc-token",
			},
		}
		if err := wc.Init(ctx, cfg); err != nil {
			t.Fatalf("init wecom: %v", err)
		}

		ts := strconv.FormatInt(time.Now().Unix(), 10)
		nonce := "wc-nonce"
		body, _ := json.Marshal(map[string]interface{}{
			"ToUserName": "corp123", "Encrypt": "encrypted-data",
		})

		parts := []string{"wc-token", ts, nonce, "encrypted-data"}
		sorted := make([]string, len(parts))
		copy(sorted, parts)
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		h := sha1.New()
		h.Write([]byte(joinStrings(sorted)))
		validSig := hex.EncodeToString(h.Sum(nil))

		headers := map[string]string{
			"timestamp":     ts,
			"nonce":         nonce,
			"msg_signature": validSig,
		}
		if err := wc.VerifySignature(headers, body); err != nil {
			t.Fatalf("合法企微签名应通过: %v", err)
		}

		badHeaders := map[string]string{
			"timestamp":     ts,
			"nonce":         nonce,
			"msg_signature": "bad-sig",
		}
		if err := wc.VerifySignature(badHeaders, body); err == nil {
			t.Fatal("非法企微签名应被拒绝")
		}

		oldTs := strconv.FormatInt(time.Now().Unix()-10*60, 10)
		oldHeaders := map[string]string{
			"timestamp":     oldTs,
			"nonce":         nonce,
			"msg_signature": validSig,
		}
		if err := wc.VerifySignature(oldHeaders, body); err == nil {
			t.Fatal("超过 5 分钟窗口的企微签名应被拒绝")
		}
	})

	t.Run("wecom parse inbound", func(t *testing.T) {
		wc := wecom.New()
		cfg := connector.Config{
			TenantID: 1, Name: "wecom", Provider: "wecom", Enabled: true,
			Credentials: map[string]string{
				"corp_id": "corp-1", "corp_secret": "s", "callback_token": "t",
				"encoding_aes_key": scenarioAESKey,
			},
		}
		if err := wc.Init(ctx, cfg); err != nil {
			t.Fatalf("init wecom: %v", err)
		}

		// echostr 是密文：解密后才是企微期望原样返回的明文 echo。
		echoBody, _ := json.Marshal(map[string]string{
			"echostr": sealWeCom(t, "echo-test", "corp-1"),
		})
		msg, err := wc.ParseInbound(echoBody)
		if err != nil {
			t.Fatalf("解析企微加密 echostr 失败: %v", err)
		}
		if msg.Type != "url_verification" || msg.Content != "echo-test" {
			t.Fatalf("企微 url_verification 应返回解密明文: type=%s content=%s", msg.Type, msg.Content)
		}

		// 事件回调密文必须解密成结构化消息，而非把密文当 encrypted_event 放行。
		const xml = `<xml><ToUserName><![CDATA[corp-1]]></ToUserName>` +
			`<FromUserName><![CDATA[user-9]]></FromUserName>` +
			`<CreateTime>1700000000</CreateTime>` +
			`<MsgType><![CDATA[text]]></MsgType>` +
			`<Content><![CDATA[报修：打印机卡纸]]></Content>` +
			`<MsgId>1234567890</MsgId></xml>`
		eventBody, _ := json.Marshal(map[string]string{
			"Encrypt": sealWeCom(t, xml, "corp-1"),
		})
		msg, err = wc.ParseInbound(eventBody)
		if err != nil {
			t.Fatalf("解析企微加密事件失败: %v", err)
		}
		if msg.Type != "text" || msg.Content != "报修：打印机卡纸" || msg.UserID != "user-9" {
			t.Fatalf("企微事件应解密为结构化字段: type=%s content=%s user=%s", msg.Type, msg.Content, msg.UserID)
		}

		// 未配置 encoding_aes_key 必须 fail-closed，绝不把密文当明文放行。
		wcNoKey := wecom.New()
		_ = wcNoKey.Init(ctx, connector.Config{
			TenantID: 1, Name: "wecom", Provider: "wecom", Enabled: true,
			Credentials: map[string]string{"corp_id": "corp-1", "corp_secret": "s", "callback_token": "t"},
		})
		if _, err := wcNoKey.ParseInbound(echoBody); err == nil {
			t.Fatal("未配置 encoding_aes_key 时解析密文应失败（fail-closed）")
		}
	})

	t.Run("connector manager provision/send/revoke lifecycle", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		registry := connector.NewRegistry()
		registry.Register(func() connector.Connector { return &mockConnector{} })

		mgr := connector.NewManager(registry, logger)
		defer mgr.CloseAll()

		cfg := connector.Config{
			TenantID: 1, Name: "mock", Provider: "mock", Enabled: true,
			Credentials: map[string]string{},
		}
		if err := mgr.Provision(ctx, cfg); err != nil {
			t.Fatalf("provision mock connector: %v", err)
		}

		c, ok := mgr.Get(1, "mock")
		if !ok || c == nil {
			t.Fatal("provision 后应能 Get 到连接器")
		}

		msg := &connector.Message{Channel: "user1", Type: "text", Content: "hello"}
		if err := mgr.Send(ctx, 1, "mock", msg); err != nil {
			t.Fatalf("send 消息失败: %v", err)
		}

		if err := mgr.Send(ctx, 999, "mock", msg); err == nil {
			t.Fatal("未 provision 的租户发送消息应失败")
		}

		mgr.Revoke(cfg)
		_, ok = mgr.Get(1, "mock")
		if ok {
			t.Fatal("revoke 后不应再 Get 到连接器")
		}
	})

	t.Run("connector manager disabled config revokes", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		registry := connector.NewRegistry()
		registry.Register(func() connector.Connector { return &mockConnector{} })

		mgr := connector.NewManager(registry, logger)
		defer mgr.CloseAll()

		cfg := connector.Config{
			TenantID: 2, Name: "mock", Provider: "mock", Enabled: true,
			Credentials: map[string]string{},
		}
		_ = mgr.Provision(ctx, cfg)

		cfg.Enabled = false
		_ = mgr.Provision(ctx, cfg)

		_, ok := mgr.Get(2, "mock")
		if ok {
			t.Fatal("Enabled=false 的 provision 应 revoke 实例")
		}
	})

	t.Run("connector manager unknown connector rejected", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		registry := connector.NewRegistry()
		mgr := connector.NewManager(registry, logger)
		defer mgr.CloseAll()

		cfg := connector.Config{
			TenantID: 1, Name: "nonexistent", Provider: "x", Enabled: true,
		}
		if err := mgr.Provision(ctx, cfg); err == nil {
			t.Fatal("未注册的连接器名称应被拒绝")
		}
	})

	t.Run("router dispatch with dedup", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		router := connector.NewRouter(logger)

		callCount := 0
		router.Register(func(ctx context.Context, msg *connector.InboundMessage) error {
			callCount++
			return nil
		})

		msg := &connector.InboundMessage{
			MessageID: "msg-001", Channel: "ch1",
			Content: "hello", ReceivedAt: time.Now(),
		}
		if err := router.Dispatch(ctx, msg); err != nil {
			t.Fatalf("首次 dispatch 失败: %v", err)
		}
		if callCount != 1 {
			t.Fatalf("首次 dispatch 应调用 handler 1 次, 实际 %d", callCount)
		}

		if err := router.Dispatch(ctx, msg); err != nil {
			t.Fatalf("重复 dispatch 失败: %v", err)
		}
		if callCount != 1 {
			t.Fatalf("重复消息应被去重, handler 仍应只调用 1 次, 实际 %d", callCount)
		}

		msg2 := &connector.InboundMessage{
			MessageID: "msg-002", Channel: "ch1",
			Content: "different", ReceivedAt: time.Now(),
		}
		if err := router.Dispatch(ctx, msg2); err != nil {
			t.Fatalf("不同消息 dispatch 失败: %v", err)
		}
		if callCount != 2 {
			t.Fatalf("不同消息应触发 handler, 实际调用 %d", callCount)
		}

		msgNoID := &connector.InboundMessage{
			Channel: "ch2", Content: "no-id", ReceivedAt: time.Now(),
		}
		_ = router.Dispatch(ctx, msgNoID)
		_ = router.Dispatch(ctx, msgNoID)
		if callCount != 4 {
			t.Fatalf("无 MessageID 的消息不应去重, 期望 4 次, 实际 %d", callCount)
		}
	})

	t.Run("registry manifest validation", func(t *testing.T) {
		m := connector.Manifest{Name: "", Version: "1.0.0", RequiredPermissions: []string{"x"}}
		if err := m.ValidateForRegistration(); err == nil {
			t.Fatal("空 name 应被拒绝")
		}

		m2 := connector.Manifest{Name: "test", Version: "", RequiredPermissions: []string{"x"}}
		if err := m2.ValidateForRegistration(); err == nil {
			t.Fatal("空 version 应被拒绝")
		}

		m3 := connector.Manifest{Name: "test", Version: "1.0.0", RequiredPermissions: nil}
		if err := m3.ValidateForRegistration(); err == nil {
			t.Fatal("空 required_permissions 应被拒绝")
		}

		m4 := connector.Manifest{Name: "test", Version: "1.0.0", RequiredPermissions: []string{"connector:write"}}
		if err := m4.ValidateForRegistration(); err != nil {
			t.Fatalf("合法 manifest 应通过: %v", err)
		}

		cs1 := m4.ComputeChecksum()
		cs2 := m4.ComputeChecksum()
		if cs1 != cs2 {
			t.Fatal("checksum 应是确定性的")
		}
		if cs1 == "" {
			t.Fatal("checksum 不应为空")
		}
	})
}

type mockConnector struct {
	initialized bool
	closed      bool
}

func (m *mockConnector) Manifest() connector.Manifest {
	return connector.Manifest{
		Name: "mock", Version: "1.0.0", Provider: "mock",
		Type:                connector.TypeIM,
		Capabilities:        []connector.Capability{connector.CapSendMessage},
		RequiredPermissions: []string{"connector:write"},
	}
}

func (m *mockConnector) Init(_ context.Context, _ connector.Config) error {
	m.initialized = true
	return nil
}

func (m *mockConnector) Send(_ context.Context, _ *connector.Message) error {
	if !m.initialized {
		return fmt.Errorf("mock: not initialized")
	}
	return nil
}

func (m *mockConnector) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: m.initialized, CheckedAt: time.Now()}
}

func (m *mockConnector) Close() error {
	m.closed = true
	return nil
}

func joinStrings(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ""
		}
		result += p
	}
	return result
}

// scenarioAESKey 是 43 字符 EncodingAESKey（补 "=" 后 base64 解出 32 字节密钥）。
const scenarioAESKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

// sealWeCom 独立构造企微密文（16 随机 + 4 大端长度 + 消息 + receiveid，PKCS#7 块 32
// 填充，AES-256-CBC，IV=密钥前 16 字节），喂给生产解密路径，避免自加密自解密的同义反复。
func sealWeCom(t *testing.T, msg, receiveID string) string {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(scenarioAESKey + "=")
	if err != nil {
		t.Fatalf("decode aes key: %v", err)
	}
	raw := make([]byte, 0, 16+4+len(msg)+len(receiveID)+32)
	raw = append(raw, []byte("0123456789abcdef")...)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(msg)))
	raw = append(raw, lenBuf...)
	raw = append(raw, []byte(msg)...)
	raw = append(raw, []byte(receiveID)...)
	pad := 32 - len(raw)%32
	for i := 0; i < pad; i++ {
		raw = append(raw, byte(pad))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	out := make([]byte, len(raw))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(out, raw)
	return base64.StdEncoding.EncodeToString(out)
}
