package aws

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"itsm-backend/service/cloud/signing"
)

const (
	// sigV4Algorithm 与 sigV4Terminator 由 AWS Signature Version 4 规范固定。
	sigV4Algorithm   = "AWS4-HMAC-SHA256"
	sigV4Terminator  = "aws4_request"
	sigV4DateHeader  = "X-Amz-Date"
	sigV4TokenHeader = "X-Amz-Security-Token" // #nosec G101 -- HTTP 头名，非凭据
)

// sigv4Signer 计算 AWS Signature Version 4 签名。
type sigv4Signer struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
}

// canonicalRequest 拼出规范请求：
//
//	HTTPMethod \n CanonicalURI \n CanonicalQueryString \n
//	CanonicalHeaders \n SignedHeaders \n HexEncode(Hash(Payload))
//
// CanonicalHeaders 自身每行以 \n 结尾，所以它与 SignedHeaders 之间会出现一个
// 空行——这是规范要求的，不是拼接错误。
func canonicalRequest(method, uri, query string, hdrs [][2]string, payload []byte) string {
	return strings.Join([]string{
		method,
		uri,
		query,
		signing.CanonicalHeaders(hdrs),
		signing.SignedHeaderList(hdrs),
		signing.SHA256Hex(payload),
	}, "\n")
}

// stringToSign 拼出待签名串，并返回 Credential 用到的凭据范围。
func stringToSign(canonical string, region, service string, now time.Time) (sts string, scope string) {
	amzDate := amzTime(now)
	scope = strings.Join([]string{dateStamp(now), region, service, sigV4Terminator}, "/")
	return strings.Join([]string{
		sigV4Algorithm,
		amzDate,
		scope,
		signing.SHA256HexString(canonical),
	}, "\n"), scope
}

// signingKey 链式派生签名密钥：
// kDate → kRegion → kService → kSigning，其中首步给日期加 "AWS4" 盐。
func (s *sigv4Signer) signingKey(now time.Time, region, service string) []byte {
	k := signing.HMACSHA256([]byte("AWS4"+s.secretAccessKey), dateStamp(now))
	k = signing.HMACSHA256(k, region)
	k = signing.HMACSHA256(k, service)
	return signing.HMACSHA256(k, sigV4Terminator)
}

// Sign 返回需要附加到请求上的头（含 Authorization、X-Amz-Date，按需带
// X-Amz-Security-Token）。host 必须与实际请求的 Host 一致，否则服务端重算的
// 摘要不同——这是「签名看起来对但一直 403」最常见的原因。
func (s *sigv4Signer) Sign(method, uri, query, host string, extra [][2]string, payload []byte, region, service string, now time.Time) map[string]string {
	hdrs := make([][2]string, 0, len(extra)+3)
	hdrs = append(hdrs, [2]string{"host", host})
	hdrs = append(hdrs, extra...)
	hdrs = append(hdrs, [2]string{"x-amz-date", amzTime(now)})
	if s.sessionToken != "" {
		hdrs = append(hdrs, [2]string{"x-amz-security-token", s.sessionToken})
	}

	canonical := canonicalRequest(method, uri, query, hdrs, payload)
	sts, scope := stringToSign(canonical, region, service, now)
	signature := signing.HMACSHA256Hex(s.signingKey(now, region, service), sts)

	out := map[string]string{
		sigV4DateHeader: amzTime(now),
		"Authorization": fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			sigV4Algorithm, s.accessKeyID, scope, signing.SignedHeaderList(hdrs), signature),
	}
	if s.sessionToken != "" {
		out[sigV4TokenHeader] = s.sessionToken
	}
	return out
}

// SetHeaders 就地给请求打上签名头。
func (s *sigv4Signer) SetHeaders(req *http.Request, extra [][2]string, payload []byte, region, service string, now time.Time) {
	for k, v := range s.Sign(req.Method, canonicalURIOf(req), req.URL.RawQuery, req.URL.Host, extra, payload, region, service, now) {
		req.Header.Set(k, v)
	}
}

// canonicalURIOf 返回规范路径：空路径是 "/"，其余按段编码但保留 '/'.
func canonicalURIOf(req *http.Request) string {
	path := req.URL.Path
	if path == "" {
		return "/"
	}
	return signing.URIEncode(path, false)
}

func amzTime(now time.Time) string { return now.UTC().Format("20060102T150405Z") }

func dateStamp(now time.Time) string { return now.UTC().Format("20060102") }
