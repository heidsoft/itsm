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
//
// 阿里云 RPC 风格接口的鉴权失败、限流、参数错误**可能以 HTTP 200 返回**，
// 错误信息只放在响应体的 Code/Message 里。因此「状态码 200」不等于成功：
// 这里统一解一次错误信封，非成功码一律转成 *RPCError。少了这道闸，一次
// 限流（Throttling）就会被上层解成全零值，变成「云上一条漏洞都没有」，
// 进而把存量未修复漏洞全部误标为已修复——安全上比漏报危险得多。
func (c *rpcClient) Call(ctx context.Context, action string, params map[string]string, out any) error {
	raw, err := c.callRaw(ctx, action, params)
	if err != nil {
		return err
	}
	if raw.Status != http.StatusOK {
		return parseRPCError(raw.Status, raw.Body)
	}
	if err := detectRPCBusinessError(action, raw); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw.Body, out); err != nil {
		return fmt.Errorf("aliyun rpc %s: decode response: %w", action, err)
	}
	return nil
}

// rpcEnvelope 是阿里云 RPC 响应里可能携带的业务错误信封。
//
// 只有错误分支才带 Code（成功分支一般只有数据字段），例外是云监控
// DescribeAlertLogList 成功时显式回填 Code="200" + Success=true。
type rpcEnvelope struct {
	Code      flexString `json:"Code"`
	Message   string     `json:"Message"`
	RequestID string     `json:"RequestId"`
	Success   *bool      `json:"Success"`
}

// rpcSuccessCodes 是各产品在成功响应里可能回填的 Code 取值（统一转小写比较）。
// 阿里云的错误码是 Throttling / Forbidden.RAM / InvalidParameter 这类字符串，
// 不会与这些成功标记重叠。
var rpcSuccessCodes = map[string]bool{
	"":        true,
	"200":     true,
	"0":       true,
	"ok":      true,
	"success": true,
	"true":    true,
}

// detectRPCBusinessError 判定一个 HTTP 200 响应是否其实是业务错误。
//
// 判定规则（宁严不宽，但不会误伤合法成功响应）：
//   - 响应体不是 JSON 对象 → 不是信封，交给后续 out 解码处理；
//   - Success 显式为 false → 业务错误；
//   - Code 非空且不在成功码集合内 → 业务错误；
//   - 其余视为成功。
//
// 只读 Describe* 接口的成功响应不带业务错误码，因此这条规则不会拦掉正常
// 数据；而限流、鉴权、参数错误必然带 Code，全部会被拦下。
func detectRPCBusinessError(action string, raw rpcRawResponse) error {
	if len(raw.Body) == 0 {
		return nil
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw.Body, &env); err != nil {
		// 非 JSON（XML 错误体、HTML 网关页等）交给上层解码报错。
		return nil
	}
	code := strings.TrimSpace(string(env.Code))
	requestID := firstNonEmpty(env.RequestID, raw.RequestID)

	if env.Success != nil && !*env.Success {
		if rpcSuccessCodes[strings.ToLower(code)] {
			code = "UnknownError"
		}
		return &RPCError{
			StatusCode: raw.Status,
			Code:       code,
			Message:    firstNonEmpty(env.Message, "阿里云接口返回 Success=false"),
			RequestID:  requestID,
		}
	}
	if rpcSuccessCodes[strings.ToLower(code)] {
		return nil
	}
	return &RPCError{
		StatusCode: raw.Status,
		Code:       code,
		Message:    firstNonEmpty(env.Message, fmt.Sprintf("aliyun rpc %s: 业务错误", action)),
		RequestID:  requestID,
	}
}

// rpcRawResponse 是一次 RPC 调用的原始结果。
type rpcRawResponse struct {
	Status    int
	Body      []byte
	RequestID string
}

// callRaw 发起一次 RPC 请求并返回原始状态码、响应体与请求 ID。
//
// 抽出来是为了让只读探测工具复用**完全相同**的签名与编码路径：探测跑通的
// 签名就是生产用的签名，不是两份实现各自通过。同时它保留原始体，让探测能
// 检查「HTTP 200 + 响应体里带业务错误码」这类调用方必须自行判定的形态
// （CMS 线上就出现过 Code 为字符串 "Throttling" 的错误分支）。
func (c *rpcClient) callRaw(ctx context.Context, action string, params map[string]string) (rpcRawResponse, error) {
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
		return rpcRawResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return rpcRawResponse{}, fmt.Errorf("aliyun rpc %s: %w", action, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return rpcRawResponse{}, fmt.Errorf("aliyun rpc %s: read response: %w", action, err)
	}
	return rpcRawResponse{
		Status:    resp.StatusCode,
		Body:      payload,
		RequestID: resp.Header.Get("X-Acs-Request-Id"),
	}, nil
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
