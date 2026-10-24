// Package volcengine 是火山引擎的云发现适配器。
//
// 签名算法与 AWS SigV4 是同一套骨架，但有三处不同，混用会直接 403：
//  1. 算法名是 HMAC-SHA256，不是 AWS4-HMAC-SHA256；
//  2. 凭据范围以 request 结尾，不是 aws4_request；
//  3. 派生密钥时**不给日期加 "AWS4" 盐**（kDate = HMAC(kSecret, Date)）。
//
// 详见火山引擎《签名机制》。与 aws / kubernetes 一样不引官方 SDK：只读发现
// 只需要「签一次 GET」，SDK 带进来的依赖与这里的需求不成比例。
//
// 覆盖的接口（Action 与 Version 必须在 Query String 里）：
//   - DescribeRegions    → ListRegions
//   - DescribeInstances  → DiscoverRegion（PageNumber/PageSize 分页）
package volcengine

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"itsm-backend/service/cloud/signing"
)

const (
	// volcAlgorithm 与 volcTerminator 由火山引擎签名规范固定。
	volcAlgorithm  = "HMAC-SHA256"
	volcTerminator = "request"
	// volcDateFormat X-Date 的格式（ISO 8601 基本格式，UTC，精确到秒）。
	volcDateFormat = "20060102T150405Z"
	// volcDateFormatDay 凭据范围里的日期部分。
	volcDateFormatDay = "20060102"
)

// volcSigner 计算火山引擎签名。
type volcSigner struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
}

// canonicalRequest 拼出规范请求。
//
// 与 SigV4 形状一致（CanonicalQueryString 在第三行），差别在于火山把 Action /
// Version 放在 Query String 里，所以查询串永远非空。
func canonicalRequest(method, path, query string, hdrs [][2]string, payload []byte) string {
	return strings.Join([]string{
		method,
		path,
		query,
		signing.CanonicalHeaders(hdrs),
		signing.SignedHeaderList(hdrs),
		signing.SHA256Hex(payload),
	}, "\n")
}

// stringToSign 拼出待签名串，并返回凭据范围。
func stringToSign(canonical, region, service string, now time.Time) (sts string, scope string) {
	scope = strings.Join([]string{now.UTC().Format(volcDateFormatDay), region, service, volcTerminator}, "/")
	return strings.Join([]string{
		volcAlgorithm,
		now.UTC().Format(volcDateFormat),
		scope,
		signing.SHA256HexString(canonical),
	}, "\n"), scope
}

// signingKey 链式派生：**首步不加 "AWS4" 盐**，末步用 "request" 结尾。
// 这两处是与 SigV4 唯一的实质差异，写错不会报错、只会 403，所以单独注释。
func (s *volcSigner) signingKey(now time.Time, region, service string) []byte {
	k := signing.HMACSHA256([]byte(s.secretAccessKey), now.UTC().Format(volcDateFormatDay))
	k = signing.HMACSHA256(k, region)
	k = signing.HMACSHA256(k, service)
	return signing.HMACSHA256(k, volcTerminator)
}

// Sign 返回需要附加到请求上的头。host 必须与实际请求的 Host 一致。
//
// 参与签名的头固定为 content-type;host;x-content-sha256;x-date（ASCII 升序），
// 这是官方文档给出的组合。
func (s *volcSigner) Sign(method, path, query, host, contentType string, payload []byte, region, service string, now time.Time) map[string]string {
	xDate := now.UTC().Format(volcDateFormat)
	payloadHash := signing.SHA256Hex(payload)
	hdrs := [][2]string{
		{"content-type", contentType},
		{"host", host},
		{"x-content-sha256", payloadHash},
		{"x-date", xDate},
	}
	if s.sessionToken != "" {
		hdrs = append(hdrs, [2]string{"x-security-token", s.sessionToken})
	}
	canonical := canonicalRequest(method, path, query, hdrs, payload)
	sts, scope := stringToSign(canonical, region, service, now)
	signature := signing.HMACSHA256Hex(s.signingKey(now, region, service), sts)

	out := map[string]string{
		"X-Date":           xDate,
		"X-Content-Sha256": payloadHash,
		"Authorization": fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			volcAlgorithm, s.accessKeyID, scope, signing.SignedHeaderList(hdrs), signature),
	}
	if s.sessionToken != "" {
		out["X-Security-Token"] = s.sessionToken
	}
	return out
}

// SetHeaders 就地给请求打上签名头。
func (s *volcSigner) SetHeaders(req *http.Request, payload []byte, region, service string, now time.Time) {
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	for k, v := range s.Sign(req.Method, path, req.URL.RawQuery, req.URL.Host, contentType, payload, region, service, now) {
		req.Header.Set(k, v)
	}
}
