package aliyun

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"itsm-backend/service/cloud/signing"
)

// 阿里云 RPC 风格 API（云监控、云安全中心）用 SignatureVersion 1.0 签名：
// HMAC-SHA1 + Base64，密钥是 secret + "&"。URL 编码规则与 SigV4 相同，
// 因此复用 signing 包的编码原语，差别只在待签名串和摘要算法。

// BuildRPCSignature 计算 RPC 签名。params 不含 Signature 本身。
func BuildRPCSignature(method string, params map[string]string, accessKeySecret string) string {
	canonical := signing.EncodeForm(params)
	stringToSign := method + "&" + signing.URIEncode("/", true) + "&" + signing.URIEncode(canonical, true)
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type rpcClient struct {
	endpoint        string
	version         string
	accessKeyID     string
	accessKeySecret string
	securityToken   string
	httpClient      *http.Client
	now             func() time.Time
}

// rpcCaller 便于用固定响应测试上层分页与解析逻辑。
type rpcCaller interface {
	Call(ctx context.Context, action string, params map[string]string, out any) error
}

var _ rpcCaller = (*rpcClient)(nil)

func newRPCClient(endpoint, version, accessKeyID, accessKeySecret, securityToken string) *rpcClient {
	return &rpcClient{
		endpoint:        endpoint,
		version:         version,
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		securityToken:   securityToken,
		httpClient:      &http.Client{Timeout: 20 * time.Second},
		now:             time.Now,
	}
}

// RPCError 保留阿里云返回的错误码与 RequestId，便于区分权限不足、限流和参数错误。
type RPCError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("aliyun rpc: http %d %s: %s (request id %s)", e.StatusCode, e.Code, e.Message, e.RequestID)
}

// Call 发起一次 RPC 请求并把 JSON 响应解析到 out。out 为 nil 时只校验成功。
func (c *rpcClient) Call(ctx context.Context, action string, params map[string]string, out any) error {
	form := make(map[string]string, len(params)+10)
	maps.Copy(form, params)
	form["Action"] = action
	form["Version"] = c.version
	form["Format"] = "JSON"
	form["AccessKeyId"] = c.accessKeyID
	form["SignatureMethod"] = "HMAC-SHA1"
	form["SignatureVersion"] = "1.0"
	form["SignatureNonce"] = randomNonce()
	form["Timestamp"] = c.now().UTC().Format("2006-01-02T15:04:05Z")
	if c.securityToken != "" {
		form["SecurityToken"] = c.securityToken
	}

	signature := BuildRPCSignature(http.MethodPost, form, c.accessKeySecret)
	body := signing.EncodeForm(form) + "&Signature=" + signing.URIEncode(signature, true)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+c.endpoint+"/", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("aliyun rpc %s: %w", action, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("aliyun rpc %s: read response: %w", action, err)
	}
	if resp.StatusCode != http.StatusOK {
		return parseRPCError(resp.StatusCode, payload)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("aliyun rpc %s: decode response: %w", action, err)
	}
	return nil
}

func parseRPCError(status int, payload []byte) error {
	// 错误响应的 Code 也可能是数字（部分网关错误），用 flexString 解码，
	// 否则 json.Unmarshal 报错被忽略后 Code 恒为空、错误被降级成 UnknownError。
	var e struct {
		Code      flexString `json:"Code"`
		Message   string     `json:"Message"`
		RequestID string     `json:"RequestId"`
	}
	_ = json.Unmarshal(payload, &e)
	if e.Code == "" {
		e.Code = "UnknownError"
	}
	if e.Message == "" {
		e.Message = http.StatusText(status)
	}
	return &RPCError{StatusCode: status, Code: string(e.Code), Message: e.Message, RequestID: e.RequestID}
}

func randomNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
