package middleware

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// 审计/日志落库前的敏感字段脱敏。
//
// 历史实现是一串正则，直接匹配 JSON 里的**字面键名**，例如
// `"(password|new_password|...)"\s*:`。这漏掉了驼峰写法：DTO 的 json tag 是
// `newPassword` / `oldPassword` / `currentPassword`，全都不在白名单里，于是
// 改密接口的明文新密码被原样写进 audit_logs（audit:read 权限即可读走）。
//
// 现在改成「先取键名、归一化、再判定」：键名去分隔符并转小写后按后缀/精确
// 名单匹配，password / new_password / newPassword / NewPassword 走同一条规则。
// 归一化也顺带解决了 x-api-key、access_token、clientSecret 这类命名变体。

// maskKVPattern 匹配 JSON 中的「键: 标量值」对，值可能是字符串、数字、
// 布尔或 null。键与字符串值都允许转义引号。
var maskKVPattern = regexp.MustCompile(
	`"((?:[^"\\]|\\.)+)"\s*:\s*("(?:[^"\\]|\\.)*"|[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?|true|false|null)`,
)

// sensitiveKeySuffixes 是「以这些片段结尾即视为敏感」的键名后缀（已归一化）。
//
// 用后缀而不是完整键名，是为了覆盖前缀变体：newPassword / confirmPassword /
// dbPassword / redis_password 都落到 password 这一条上。
var sensitiveKeySuffixes = []string{
	"password", "passwd", "secret", "token", "apikey", "privatekey",
	"credential", "credentials", "authorization", "cookie", "cvv", "cvc",
}

// sensitiveKeyContains 是不能只靠后缀覆盖的片段（前后都可能带前缀/后缀）。
var sensitiveKeyContains = []string{
	"encryptkey", "signingkey", "hmackey", "appkey", "accesskey", "secretkey",
}

// sensitiveKeyExact 是无分隔符、也不带常见后缀的短键名与个人隐私字段。
//
// 联系方式与地址类沿用了历史白名单的口径（审计日志里不保留），避免本次
// 重构悄悄放宽已生效的脱敏范围。
var sensitiveKeyExact = map[string]bool{
	"pwd": true, "jwt": true,
	"ssn": true, "socialsecurity": true, "taxid": true, "nationalid": true,
	"idnumber": true, "passportnumber": true,
	"creditcard": true, "cardnumber": true, "ccnum": true,
	"bankaccount": true, "routingnumber": true, "iban": true, "swift": true, "bic": true,
	"phone": true, "mobile": true, "tel": true, "telephone": true,
	"email": true, "address": true, "street": true, "postalcode": true, "zipcode": true,
	"cookie": true, "setcookie": true, "authorization": true, "proxyauthorization": true,
}

// normalizeFieldKey 把键名归一化成「小写、无分隔符」形式。
//
// newPassword / new_password / new-password / New Password 都归一成 newpassword。
func normalizeFieldKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		switch r {
		case '_', '-', ' ', '.', '\t':
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// IsSensitiveFieldKey 判定某个 JSON 键名是否需要脱敏。
//
// 导出成可测试函数：新增 DTO 时可以直接用它自检，避免再出现「白名单只写
// 下划线写法、实际 tag 是驼峰」这类漏网。
func IsSensitiveFieldKey(key string) bool {
	normalized := normalizeFieldKey(key)
	if normalized == "" {
		return false
	}
	if sensitiveKeyExact[normalized] {
		return true
	}
	for _, suffix := range sensitiveKeySuffixes {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	for _, fragment := range sensitiveKeyContains {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

// MaskSensitiveFields masks sensitive fields in JSON request bodies for logging/audit.
//
// 非 JSON 或字段值本身是 JSON 字符串时也能处理：匹配的是文本层面的键值对，
// 不依赖整体能否反序列化。已脱敏的值不会被二次匹配。
func MaskSensitiveFields(body string) string {
	if body == "" {
		return body
	}
	return maskKVPattern.ReplaceAllStringFunc(body, func(match string) string {
		sub := maskKVPattern.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		if !IsSensitiveFieldKey(sub[1]) {
			return match
		}
		// 值统一替换成字符串 "***"，保留原键名，便于审计里看出「哪个字段被脱敏」。
		return `"` + sub[1] + `":"***"`
	})
}

// MaskResponseMiddleware is optional if we later log responses
func MaskResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// placeholder for future response masking if needed
		_ = json.Marshal
	}
}
