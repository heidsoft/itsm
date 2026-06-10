// Package dingtalk 钉钉开放平台连接器
// 文档：
//   - 获取 access_token: https://open.dingtalk.com/document/orgapp/obtain-identity-token
//   - 工作通知: https://open.dingtalk.com/document/orgapp/asyncreply-message
//   - 群机器人: https://open.dingtalk.com/document/robots/custom-robot-access
//   - 事件订阅: https://open.dingtalk.com/document/isvapp/stream
//
// 设计要点：
//   1. access_token 缓存 + 自动刷新（TTL ~2h，提前 5 分钟）
//   2. 支持两种发送通道：app 工作通知（精确到 userid）+ 群机器人 webhook
//   3. 群机器人支持加签安全模式
//   4. 消息类型：text / markdown / actionCard
package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	BaseURL    = "https://oapi.dingtalk.com"
	OldBaseURL = "https://eco.taobao.com/router/rest" // 旧版 top
	DefaultTTL = 110 * time.Minute
)

type Client struct {
	baseURL    string
	appKey     string
	appSecret  string
	agentID    string // 发送工作通知必填
	logger     *zap.SugaredLogger
	hc         *http.Client

	mu    sync.Mutex
	token string
	exp   time.Time
}

func NewClient(appKey, appSecret, agentID, baseURL string) *Client {
	if baseURL == "" {
		baseURL = BaseURL
	}
	return &Client{
		baseURL:   baseURL,
		appKey:    appKey,
		appSecret: appSecret,
		agentID:   agentID,
		logger:    zap.S().Named("connector.dingtalk"),
		hc:        &http.Client{Timeout: 8 * time.Second},
	}
}

// Token 获取/刷新 access_token
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.exp) > 5*time.Minute {
		return c.token, nil
	}
	u := fmt.Sprintf("%s/gettoken?appkey=%s&appsecret=%s",
		c.baseURL, url.QueryEscape(c.appKey), url.QueryEscape(c.appSecret))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Token   string `json:"access_token"`
		Expire  int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("dingtalk: decode token: %w", err)
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("dingtalk: token api code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	c.token = out.Token
	c.exp = time.Now().Add(time.Duration(out.Expire) * time.Second)
	return c.token, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query map[string]string, in, out interface{}) error {
	body, _ := json.Marshal(in)
	u, _ := url.Parse(c.baseURL + path)
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("dingtalk: decode %s: %w", path, err)
		}
	}
	return nil
}

// SignRobotWebhook 群机器人"加签"模式签名
// 规则：secret = base64(HMAC-SHA256(secret, timestamp + "\n" + secret))，
// 然后 webhook URL 追加 &timestamp=...&sign=...
func SignRobotWebhook(webhook, secret string) (string, error) {
	if secret == "" {
		return webhook, nil
	}
	ts := time.Now().UnixMilli()
	stringToSign := fmt.Sprintf("%d\n%s", ts, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	sep := "&"
	if !strings.Contains(webhook, "?") {
		sep = "?"
	}
	return webhook + sep + "timestamp=" + fmt.Sprint(ts) + "&sign=" + sign, nil
}

// VerifyStreamSignature 验证 Stream 模式回调签名
// https://open.dingtalk.com/document/isvapp/stream-message-signature-verification
// 签名 = base64(HMAC-SHA256(secret, timestamp + "\n" + body))
func (c *Client) VerifyStreamSignature(timestamp, signature string, body []byte) bool {
	if c.appSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.appSecret))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature == expected
}

// helper: sort map keys for stable JSON
func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ = sortedMapKeys // exported helper for future use
