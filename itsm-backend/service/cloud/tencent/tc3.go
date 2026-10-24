package tencent

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"itsm-backend/service/cloud/signing"
)

const (
	// tc3Algorithm 与 tc3Terminator 由腾讯云「签名方法 v3」规范固定。
	tc3Algorithm  = "TC3-HMAC-SHA256"
	tc3Terminator = "tc3_request"
	// dateLayout 日期取 UTC，与规范里 Date 由时间戳按 UTC+0 换算一致。
	dateLayout = "2006-01-02"
)

// tc3Signer 计算腾讯云 TC3-HMAC-SHA256 签名。
type tc3Signer struct {
	secretID  string
	secretKey string
}

// canonicalRequest 拼出规范请求。
//
// 与 AWS SigV4 的差别：腾讯云的 API 一律 POST 到 "/"，没有查询串，所以
// CanonicalQueryString 恒为空；参与签名的头也只有 content-type / host /
// x-tc-action 三个（地域、版本、时间戳都不进签名）。
func canonicalRequest(method, uri string, hdrs [][2]string, payload []byte) string {
	return strings.Join([]string{
		method,
		uri,
		"", // CanonicalQueryString
		signing.CanonicalHeaders(hdrs),
		signing.SignedHeaderList(hdrs),
		signing.SHA256Hex(payload),
	}, "\n")
}

// stringToSign 拼出待签名串，并返回凭据范围。
func stringToSign(canonical, service string, timestamp int64) (sts string, scope string) {
	date := time.Unix(timestamp, 0).UTC().Format(dateLayout)
	scope = strings.Join([]string{date, service, tc3Terminator}, "/")
	return strings.Join([]string{
		tc3Algorithm,
		strconv.FormatInt(timestamp, 10),
		scope,
		signing.SHA256HexString(canonical),
	}, "\n"), scope
}

// signingKey 链式派生：kDate（给日期加 "TC3" 盐）→ kService → kSigning。
// 腾讯云比 SigV4 少一层：没有 region 这一环，因为地域走 X-TC-Region 头而不进签名。
func (s *tc3Signer) signingKey(timestamp int64, service string) []byte {
	date := time.Unix(timestamp, 0).UTC().Format(dateLayout)
	k := signing.HMACSHA256([]byte("TC3"+s.secretKey), date)
	k = signing.HMACSHA256(k, service)
	return signing.HMACSHA256(k, tc3Terminator)
}

// Sign 返回需要附加到请求上的头。host 必须与实际请求的 Host 一致。
func (s *tc3Signer) Sign(method, uri, host, action string, payload []byte, service string, timestamp int64) map[string]string {
	hdrs := [][2]string{
		{"content-type", "application/json; charset=utf-8"},
		{"host", host},
		{"x-tc-action", strings.ToLower(action)},
	}
	canonical := canonicalRequest(method, uri, hdrs, payload)
	sts, scope := stringToSign(canonical, service, timestamp)
	signature := signing.HMACSHA256Hex(s.signingKey(timestamp, service), sts)
	return map[string]string{
		"Authorization": fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			tc3Algorithm, s.secretID, scope, signing.SignedHeaderList(hdrs), signature),
	}
}

// SetHeaders 就地给请求打上签名头。
func (s *tc3Signer) SetHeaders(req *http.Request, action string, payload []byte, service string, timestamp int64) {
	uri := req.URL.Path
	if uri == "" {
		uri = "/"
	}
	for k, v := range s.Sign(req.Method, uri, req.URL.Host, action, payload, service, timestamp) {
		req.Header.Set(k, v)
	}
}
