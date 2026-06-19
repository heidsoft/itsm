// Package feishu 飞书/Lark 开放平台连接器
// 文档：
//   - 发送消息：https://open.feishu.cn/document/server-docs/im-v1/message/create
//   - 事件订阅：https://open.feishu.cn/document/server-docs/event-subscription-guide/overview
//   - 签名校验：https://open.feishu.cn/document/server-docs/event-subscription-guide/signature-verification
//
// 设计要点：
//  1. tenant_access_token 统一缓存 + 自动刷新
//  2. 事件回调支持 URL Verification（首次握手）
//  3. 事件回调支持 HMAC-SHA256 签名校验
//  4. 卡片回调（card.action.trigger）支持 ActionHandler 注册
package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	BaseURLCN   = "https://open.feishu.cn"
	BaseURLIntl = "https://open.larksuite.com"
	DefaultTTL  = 110 * time.Minute // token 官方 TTL 2h，提前 10 分钟刷新
)

type Client struct {
	baseURL    string
	appID      string
	appSecret  string
	verifyTok  string
	encryptKey string
	logger     *zap.SugaredLogger
	hc         *http.Client

	mu     sync.Mutex
	token  string
	expire time.Time
}

// NewClient 创建飞书客户端
// 必填：app_id / app_secret
// 可选：verification_token（用于事件订阅 URL 校验） / encrypt_key（事件加密）
func NewClient(baseURL, appID, appSecret, verifyTok, encryptKey string) *Client {
	if baseURL == "" {
		baseURL = BaseURLCN
	}
	return &Client{
		baseURL:    baseURL,
		appID:      appID,
		appSecret:  appSecret,
		verifyTok:  verifyTok,
		encryptKey: encryptKey,
		logger:     zap.S().Named("connector.feishu"),
		hc:         &http.Client{Timeout: 8 * time.Second},
	}
}

// Token 获取当前可用的 tenant_access_token（自动刷新）
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.expire) > 5*time.Minute {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("feishu: decode token resp: %w", err)
	}
	if out.Code != 0 {
		return "", fmt.Errorf("feishu: token api error code=%d msg=%s", out.Code, out.Msg)
	}
	c.token = out.TenantAccessToken
	c.expire = time.Now().Add(time.Duration(out.Expire) * time.Second)
	return c.token, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out interface{}) error {
	body, _ := json.Marshal(in)
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("feishu: decode %s resp: %w", path, err)
		}
	}
	return nil
}

// VerifyEventSignature 验证事件订阅请求的签名
// https://open.feishu.cn/document/server-docs/event-subscription-guide/signature-verification
//
//	encrypt_key + timestamp + nonce 拼接后做 SHA256，与 header X-Lark-Signature 比对
func (c *Client) VerifyEventSignature(ts, nonce, signature string, body []byte) bool {
	if c.encryptKey == "" || signature == "" {
		return false
	}
	h := sha256.New()
	h.Write([]byte(c.encryptKey))
	h.Write([]byte(ts))
	h.Write([]byte(nonce))
	h.Write(body)
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyURLToken 校验首次 URL Verification 时的 token
func (c *Client) VerifyURLToken(token string) bool {
	return c.verifyTok != "" && c.verifyTok == token
}
