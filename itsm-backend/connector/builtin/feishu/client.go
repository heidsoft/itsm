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
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	BaseURLCN   = "https://open.feishu.cn"
	BaseURLIntl = "https://open.larksuite.com"
	DefaultTTL  = 110 * time.Minute // token 官方 TTL 2h，提前 10 分钟刷新
	// tokenRefreshMargin 距过期多久就开始换新。官方 TTL 2h，留 5 分钟余量覆盖
	// 「换到 token 后网络抖动，实际生效晚于本地记录」的窗口。
	tokenRefreshMargin = 5 * time.Minute
	// maxResponseBytes 单个响应的读取上限，防止异常响应把内存吃满。
	maxResponseBytes = 4 << 20 // 4 MiB
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
//
// 刷新用互斥锁而不是 singleflight：飞书的 tenant_access_token 是租户级共享
// 凭据，N 个并发请求在过期窗口内各自去换会互相把对方刚换到的 token 挤掉，
// 表现为偶发 401。锁内串行化后，只有第一个请求真的去换，其余直接用新值。
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Until(c.expire) > tokenRefreshMargin {
		return c.token, nil
	}
	if err := c.refreshTenantToken(ctx); err != nil {
		return "", err
	}
	return c.token, nil
}

func (c *Client) refreshTenantToken(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	})
	raw, err := c.postRaw(ctx, "/open-apis/auth/v3/tenant_access_token/internal", body, "")
	if err != nil {
		return err
	}
	var out struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("feishu: decode token resp: %w", err)
	}
	if out.Code != 0 {
		return fmt.Errorf("feishu: token api error code=%d msg=%s", out.Code, out.Msg)
	}
	c.token = out.TenantAccessToken
	c.expire = time.Now().Add(time.Duration(out.Expire) * time.Second)
	return nil
}

// getAppAccessToken 获取 app_access_token。
//
// ⚠️ 原实现直接 `return c.Token(ctx)`（即拿 tenant_access_token 顶替），
// 注释里写着「先这样，需要时再实现」。但飞书的 OAuth 换码/刷新端点
// （/open-apis/authen/v1/access_token、refresh_access_token）只认
// **app_access_token**：用 tenant token 调用会返回 code=99991663 之类的鉴权
// 失败，而返回值又被当成「用户未授权」处理——OAuth 链路是整条断的，且错误
// 指向错误的方向。这里按官方端点补齐。
func (c *Client) getAppAccessToken(ctx context.Context) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	})
	raw, err := c.postRaw(ctx, "/open-apis/auth/v3/app_access_token/internal", body, "")
	if err != nil {
		return "", err
	}
	var out struct {
		Code            int    `json:"code"`
		Msg             string `json:"msg"`
		AppAccessToken  string `json:"app_access_token"`
		Expire          int    `json:"expire"`
		TenantAppAccess string `json:"tenant_app_access_token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("feishu: decode app token resp: %w", err)
	}
	if out.Code != 0 {
		return "", fmt.Errorf("feishu: app token api error code=%d msg=%s", out.Code, out.Msg)
	}
	return out.AppAccessToken, nil
}

// postRaw 发一个 JSON POST 并返回**已校验过状态码**的响应体。
//
// 关键点：必须先看 resp.StatusCode。飞书网关在限流/鉴权失败时返回的是带
// code 的 JSON（HTTP 200 之外的状态码也有），原实现不判状态码就把 body 往
// 结构体里 unmarshal，网关返回 HTML/空体时报的是「decode xxx resp: invalid
// character '<'」——错误信息指向 JSON 解析，实际是 401/429，排查方向完全错。
func (c *Client) postRaw(ctx context.Context, path string, body []byte, bearer string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("feishu: read %s resp: %w", req.URL.Path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 带上状态码与响应体片段：限流（429）与鉴权（401）在飞书侧都有明确
		// 语义，截断后一起报出来，调用方才知道该重试还是改配置。
		return nil, fmt.Errorf("feishu: %s %s http=%d body=%s", req.Method, req.URL.Path,
			resp.StatusCode, truncateForLog(raw, 256))
	}
	return raw, nil
}

func truncateForLog(raw []byte, n int) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out interface{}) error {
	var body []byte
	if in != nil {
		body, _ = json.Marshal(in)
	}
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+tok)
	raw, err := c.do(req)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("feishu: decode %s resp: %w", path, err)
	}
	return nil
}

// VerifyEventSignature 验证事件订阅请求的签名
// https://open.feishu.cn/document/server-docs/event-subscription-guide/signature-verification
//
//	以 encrypt_key 为密钥，对 timestamp + nonce + body 做 HMAC-SHA256，
//	与 header X-Lark-Signature 做常量时间比较。
func (c *Client) VerifyEventSignature(ts, nonce, signature string, body []byte) bool {
	if c.encryptKey == "" || ts == "" || nonce == "" || signature == "" {
		return false
	}
	h := hmac.New(sha256.New, []byte(c.encryptKey))
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
		c.baseURL, c.appID, url.QueryEscape(redirectURI), url.QueryEscape(state))
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
	raw, err := c.postRaw(ctx, "/open-apis/authen/v1/access_token", body, appToken)
	if err != nil {
		return nil, err
	}
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
	raw, err := c.postRaw(ctx, "/open-apis/authen/v1/refresh_access_token", body, appToken)
	if err != nil {
		return nil, err
	}
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
