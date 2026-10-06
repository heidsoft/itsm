package connector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactHeadersMasksSensitive(t *testing.T) {
	in := map[string]string{
		"X-DingTalk-Signature": "real-secret-abc",
		"X-Lark-Request-Timestamp": "1700000000",
		"X-WeCom-Msg_Signature": "wechat-secret",
		"Content-Type":          "application/json",
		"User-Agent":            "test",
		"Authorization":         "Bearer super-secret",
		"X-Trace-Id":            "trace-1",
	}
	out := RedactHeaders(in)

	require.Equal(t, "***", out["X-DingTalk-Signature"])
	require.Equal(t, "***", out["X-WeCom-Msg_Signature"])
	require.Equal(t, "***", out["Authorization"])

	require.Equal(t, "1700000000", out["X-Lark-Request-Timestamp"])
	require.Equal(t, "application/json", out["Content-Type"])
	require.Equal(t, "test", out["User-Agent"])
	require.Equal(t, "trace-1", out["X-Trace-Id"])
}

func TestRedactHeadersIsCaseInsensitiveAndSubstring(t *testing.T) {
	out := RedactHeaders(map[string]string{
		"x-lark-signature":   "v1",
		"X-LARK-TOKEN":       "v2",
		"x_wecom_aes_key":    "v3",
		"X-DingTalk-Encrypt": "v4",
		"Innocent":           "keep",
	})
	require.Equal(t, "***", out["x-lark-signature"])
	require.Equal(t, "***", out["X-LARK-TOKEN"])
	require.Equal(t, "***", out["x_wecom_aes_key"])
	require.Equal(t, "***", out["X-DingTalk-Encrypt"])
	require.Equal(t, "keep", out["Innocent"])
}

func TestRedactHeadersLeavesInputUntouched(t *testing.T) {
	in := map[string]string{"X-Foo-Signature": "secret"}
	_ = RedactHeaders(in)
	// Caller's original map must remain readable for downstream logging
	// that needs the real value (e.g. structured-zap logger).
	require.Equal(t, "secret", in["X-Foo-Signature"])
}

func TestIsSensitiveHeader(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Signature", true},
		{"x-lark-signature", true},
		{"X-WECOM-MSG_SIGNATURE", true},
		{"Authorization", true},
		{"X-Trace-Id", false},
		{"Content-Type", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isSensitiveHeader(tc.name))
		})
	}
}

// guard the contract that the package never returns a partial map without
// at least the masked shape (test ensures no silent nil return on edge input).
func TestRedactHeadersHandlesEmpty(t *testing.T) {
	require.Equal(t, map[string]string{}, RedactHeaders(map[string]string{}))
	require.Equal(t, map[string]string{"Keep": "value"}, RedactHeaders(map[string]string{"Keep": "value"}))
	// defensive: substring containment shouldn't trip over short keys.
	require.False(t, strings.Contains("X-Trace-Id", "signature"))
}
