// Package signing 云厂商签名算法的公共原语。
//
// AWS Signature Version 4 与火山引擎的签名算法是同一套骨架：把请求规范化成
// 一个字符串、算哈希、再用「链式 HMAC」从 Secret Access Key 派生签名密钥。
// 两者差别只在算法名（AWS4-HMAC-SHA256 / HMAC-SHA256）、凭据范围结尾
// （aws4_request / request）和派生密钥时给日期加不加盐（"AWS4" 前缀有无）。
//
// 骨架相同意味着：一旦出现两份各自实现，差异一定会落在这些细节上，而且两边
// 的 bug 不会互相暴露——这份包就是为了让「规范请求怎么拼」只有一处实现。
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SHA256Hex 返回字节切片的 SHA256 十六进制摘要。
func SHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// SHA256HexString 是 SHA256Hex 的字符串便捷形式。
func SHA256HexString(s string) string { return SHA256Hex([]byte(s)) }

// HMACSHA256 HMAC-SHA256 的原始摘要。链式派生密钥需要把上一步的结果直接当
// 下一步的 key，所以暴露字节而不是十六进制。
func HMACSHA256(key []byte, msg string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

// HMACSHA256Hex HMAC-SHA256 的十六进制结果。
func HMACSHA256Hex(key []byte, msg string) string {
	return hex.EncodeToString(HMACSHA256(key, msg))
}

// URIEncode 按 SigV4 的 URI 编码规则编码。
//
// 规则比 url.QueryEscape 严格：只有 A-Za-z0-9-_.~ 原样保留，其余一律 %XX
// （大写十六进制）。QueryEscape 会把空格编码成 '+'，而 SigV4 要求 %20——
// 这是历史上一类「本地算得过、服务端 403」的经典差异。
//
// encodeSlash 控制 '/' 是否编码：规范路径不编码 '/'，查询串与表单体里的值
// 必须编码为 %2F。
func URIEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		case c == '/':
			if encodeSlash {
				b.WriteString("%2F")
			} else {
				b.WriteByte('/')
			}
		default:
			b.WriteByte('%')
			b.WriteByte(upperHex[c>>4])
			b.WriteByte(upperHex[c&0x0F])
		}
	}
	return b.String()
}

// upperHex 避免每次编码一个字节都做一次 hex.EncodeToString + ToUpper 分配。
const upperHex = "0123456789ABCDEF"

// CanonicalHeaders 把参与签名的头按 SigV4 规则串成规范形式：
// 名字小写、值去首尾空白、按名字 ASCII 升序、每行以 '\n' 结尾（含最后一行）。
func CanonicalHeaders(hdrs [][2]string) string {
	sorted := sortedHeaders(hdrs)
	var b strings.Builder
	for _, h := range sorted {
		b.WriteString(strings.ToLower(strings.TrimSpace(h[0])))
		b.WriteByte(':')
		b.WriteString(strings.TrimSpace(h[1]))
		b.WriteByte('\n')
	}
	return b.String()
}

// SignedHeaderList 返回排序后的签名头名列表（分号连接）。
// 顺序必须与 CanonicalHeaders 一致，否则服务端拼出的摘要与本地不同。
func SignedHeaderList(hdrs [][2]string) string {
	sorted := sortedHeaders(hdrs)
	names := make([]string, 0, len(sorted))
	for _, h := range sorted {
		names = append(names, strings.ToLower(strings.TrimSpace(h[0])))
	}
	return strings.Join(names, ";")
}

func sortedHeaders(hdrs [][2]string) [][2]string {
	sorted := make([][2]string, len(hdrs))
	copy(sorted, hdrs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(sorted[i][0])) <
			strings.ToLower(strings.TrimSpace(sorted[j][0]))
	})
	return sorted
}

// EncodeForm 把参数编码成表单体 / 查询串：按参数名 ASCII 升序，键与值都做
// URI 编码（'/' 编码为 %2F）。
//
// 排序是签名的一部分——服务端按同样顺序重算，顺序不一致就是 403。
func EncodeForm(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, URIEncode(k, true)+"="+URIEncode(params[k], true))
	}
	return strings.Join(parts, "&")
}
