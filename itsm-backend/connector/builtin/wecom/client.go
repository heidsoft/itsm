// Package wecom 企业微信连接器
// 文档：
//   - access_token: https://developer.work.weixin.qq.com/document/path/91039
//   - 应用消息: https://developer.work.weixin.qq.com/document/path/90236
//   - 群机器人: https://developer.work.weixin.qq.com/document/path/91770
//   - 事件订阅: https://developer.work.weixin.qq.com/document/path/90968
package wecom

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const BaseURL = "https://qyapi.weixin.qq.com"

type Client struct {
	baseURL   string
	corpID    string
	corpSecret string
	agentID   string
	logger    *zap.SugaredLogger
	hc        *http.Client

	mu    sync.Mutex
	token string
	exp   time.Time
}

func NewClient(corpID, corpSecret, agentID, baseURL string) *Client {
	if baseURL == "" {
		baseURL = BaseURL
	}
	return &Client{
		baseURL:    baseURL,
		corpID:     corpID,
		corpSecret: corpSecret,
		agentID:    agentID,
		logger:     zap.S().Named("connector.wecom"),
		hc:         &http.Client{Timeout: 8 * time.Second},
	}
}

// Token 获取/刷新 access_token
// 注意：企业微信的 access_token 是按"应用"维度，不同 agentid 用不同 corpsecret
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.exp) > 5*time.Minute {
		return c.token, nil
	}
	u := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		c.baseURL, c.corpID, c.corpSecret)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("wecom: decode token: %w", err)
	}
	if out.ErrCode != 0 {
		return "", fmt.Errorf("wecom: token api code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	c.token = out.AccessToken
	c.exp = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return c.token, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query map[string]string, in, out interface{}) error {
	body, _ := json.Marshal(in)
	u := c.baseURL + path
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for k := range(query) { keys = append(keys, k) }
		sort.Strings(keys)
		parts := make([]string, 0, len(query))
		for _, k := range keys { parts = append(parts, k+"="+query[k]) }
		u += "?" + strings.Join(parts, "&")
	}
	req, _ := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("wecom: decode %s: %w", path, err)
		}
	}
	return nil
}

// VerifyCallbackSignature 验证回调签名
// 签名 = SHA1(token, timestamp, nonce, msg_encrypt)
// token 在"接收消息 -> API 接收"页设置
func VerifyCallbackSignature(token, timestamp, nonce, msgEncrypt, signature string) bool {
	h := sha1.New()
	h.Write([]byte(strings.Join([]string{token, timestamp, nonce, msgEncrypt}, "")))
	expected := hex.EncodeToString(h.Sum(nil))
	return signature == expected
}
