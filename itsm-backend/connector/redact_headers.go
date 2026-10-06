package connector

import "strings"

// sensitiveHeaderKeys enumerates header names whose values must never be
// persisted to audit_log or replayed in failure messages. Keys are matched
// case-insensitively and as substring, so the helper also catches suffixed
// variants like `X-Lark-Signature`, `X-DingTalk-Signature`,
// `X-WeCom-Msg_Signature`, etc.
var sensitiveHeaderKeys = []string{
	"signature",
	"msg_signature",
	"token",
	"secret",
	"access_token",
	"aes_key",
	"encrypt",
	"authorization",
	"cookie",
	"password",
}

// RedactHeaders returns a copy of headers with sensitive values masked.
// Non-sensitive keys are passed through unchanged so audit_log still
// records useful metadata (tenant, content-type, retry count, etc.).
//
// Use this when recording verify_signature or parse_inbound failures so the
// audit row does not leak the secret material the caller already failed to
// authenticate with.
func RedactHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if isSensitiveHeader(k) {
			out[k] = "***"
			continue
		}
		out[k] = v
	}
	return out
}

func isSensitiveHeader(name string) bool {
	lower := strings.ToLower(name)
	for _, key := range sensitiveHeaderKeys {
		if strings.Contains(lower, key) {
			return true
		}
	}
	return false
}
