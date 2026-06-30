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
	"net/url"
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
	var body io.Reader
	if in != nil {
		raw, _ := json.Marshal(in)
		body = bytes.NewReader(raw)
	}
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
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

// FeishuTask represents a Feishu task
type FeishuTask struct {
	GUID        string                 `json:"guid,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	StartTime   int64                  `json:"start_time,omitempty"` // Unix timestamp in seconds
	DueTime     int64                  `json:"due_time,omitempty"`   // Unix timestamp in seconds
	Completed   bool                   `json:"completed,omitempty"`
	CompletedAt int64                  `json:"completed_at,omitempty"`
	CreatorID   string                 `json:"creator_id,omitempty"`
	Assignees   []string               `json:"assignees,omitempty"` // User open IDs
	Status      string                 `json:"status,omitempty"`    // not_started / in_progress / completed / canceled
	Priority    string                 `json:"priority,omitempty"`  // low / medium / high / urgent
	CreatedAt   int64                  `json:"created_at,omitempty"`
	UpdatedAt   int64                  `json:"updated_at,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// CreateTask creates a new Feishu task
func (c *Client) CreateTask(ctx context.Context, task *FeishuTask) (*FeishuTask, error) {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Task *FeishuTask `json:"task"`
		} `json:"data"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/open-apis/task/v2/tasks", task, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("feishu: create task failed code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Task, nil
}

// UpdateTask updates an existing Feishu task
func (c *Client) UpdateTask(ctx context.Context, taskGUID string, task *FeishuTask) (*FeishuTask, error) {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Task *FeishuTask `json:"task"`
		} `json:"data"`
	}
	err := c.doJSON(ctx, http.MethodPatch, fmt.Sprintf("/open-apis/task/v2/tasks/%s", taskGUID), task, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("feishu: update task failed code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Task, nil
}

// GetTask gets a Feishu task by GUID
func (c *Client) GetTask(ctx context.Context, taskGUID string) (*FeishuTask, error) {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Task *FeishuTask `json:"task"`
		} `json:"data"`
	}
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/open-apis/task/v2/tasks/%s", taskGUID), nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("feishu: get task failed code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Task, nil
}

// DeleteTask deletes a Feishu task by GUID
func (c *Client) DeleteTask(ctx context.Context, taskGUID string) error {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	err := c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/open-apis/task/v2/tasks/%s", taskGUID), nil, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("feishu: delete task failed code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// ListTasks lists Feishu tasks with optional filters
func (c *Client) ListTasks(ctx context.Context, pageToken string, pageSize int) ([]*FeishuTask, string, error) {
	path := fmt.Sprintf("/open-apis/task/v2/tasks?page_size=%d", pageSize)
	if pageToken != "" {
		path += fmt.Sprintf("&page_token=%s", pageToken)
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Tasks     []*FeishuTask `json:"tasks"`
			PageToken string        `json:"page_token"`
			HasMore   bool          `json:"has_more"`
		} `json:"data"`
	}
	err := c.doJSON(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, "", err
	}
	if resp.Code != 0 {
		return nil, "", fmt.Errorf("feishu: list tasks failed code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.Tasks, resp.Data.PageToken, nil
}

// OAuthTokenResponse represents the response from Feishu OAuth token endpoint
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	UserID       string `json:"user_id,omitempty"`
	OpenID       string `json:"open_id,omitempty"`
	UnionID      string `json:"union_id,omitempty"`
}

// GetOAuthAuthURL returns the OAuth authorization URL for Feishu
func (c *Client) GetOAuthAuthURL(redirectURI, state string) string {
	return fmt.Sprintf("%s/open-apis/authen/v1/authorize?app_id=%s&redirect_uri=%s&state=%s&response_type=code&scope=task:task:readonly,im:message:send,contact:user.base:readonly",
		c.baseURL, c.appID, url.QueryEscape(redirectURI), state)
}

// ExchangeOAuthCode exchanges an authorization code for an access token
func (c *Client) ExchangeOAuthCode(ctx context.Context, code string) (*OAuthTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/open-apis/authen/v1/access_token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	// For OAuth code exchange, we need to use app access token
	appToken, err := c.getAppAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+appToken)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code int                 `json:"code"`
		Msg  string              `json:"msg"`
		Data *OAuthTokenResponse `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("feishu: decode oauth token resp: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("feishu: oauth token api error code=%d msg=%s", out.Code, out.Msg)
	}
	return out.Data, nil
}

// RefreshOAuthToken refreshes an access token using a refresh token
func (c *Client) RefreshOAuthToken(ctx context.Context, refreshToken string) (*OAuthTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/open-apis/authen/v1/refresh_access_token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	appToken, err := c.getAppAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+appToken)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code int                 `json:"code"`
		Msg  string              `json:"msg"`
		Data *OAuthTokenResponse `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("feishu: decode refresh token resp: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("feishu: refresh token api error code=%d msg=%s", out.Code, out.Msg)
	}
	return out.Data, nil
}

// getAppAccessToken gets the app access token (used for OAuth operations)
func (c *Client) getAppAccessToken(ctx context.Context) (string, error) {
	// We can reuse the tenant access token for now, but if we need app-specific, we can implement it
	return c.Token(ctx)
}
