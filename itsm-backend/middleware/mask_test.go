package middleware

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// 审计脱敏回归锁。
//
// 缺陷（2026-11-05 白盒审查）：白名单只写下划线写法，而 DTO 的 json tag 是
// 驼峰 `newPassword`，于是改密请求的明文新密码原样写进 audit_logs。
// 下面按「命名变体 × 嵌套结构」逐类锁死，避免以后新增 DTO 再漏。

func TestMaskSensitiveFields_MasksPasswordCamelCase(t *testing.T) {
	// 这是 ChangePassword 接口的真实请求体形状（见 dto/user_dto.go:98-104）。
	body := `{"oldPassword":"OldPass2026!","newPassword":"NewPass2026!","confirmPassword":"NewPass2026!"}`

	masked := MaskSensitiveFields(body)

	require.NotContains(t, masked, "NewPass2026!")
	require.NotContains(t, masked, "OldPass2026!")
	require.Contains(t, masked, `"newPassword":"***"`)
	require.Contains(t, masked, `"oldPassword":"***"`)
}

func TestMaskSensitiveFields_MasksAllNamingVariants(t *testing.T) {
	cases := map[string]string{
		"snake_case": `{"new_password":"x1"}`,
		"kebab-case": `{"new-password":"x1"}`,
		"camelCase":  `{"newPassword":"x1"}`,
		"PascalCase": `{"NewPassword":"x1"}`,
		"upper snake": `{"NEW_PASSWORD":"x1"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			require.NotContains(t, MaskSensitiveFields(body), "x1", "键名变体必须同样被脱敏")
		})
	}
}

func TestMaskSensitiveFields_MasksCredentialKeys(t *testing.T) {
	values := map[string]string{
		"accessToken":    "tok-a",
		"refresh_token":  "tok-r",
		"csrfToken":      "csrf-v",
		"idToken":        "id-v",
		"clientSecret":   "sec-v",
		"app_secret":     "sec-a",
		"x-api-key":      "key-v",
		"apiKey":         "key-a",
		"privateKey":     "pk-v",
		"authorization":  "Bearer abc",
		"session_token":  "sess-v",
		"bot_token":      "bot-v",
		"db_password":    "db-v",
		"encrypt_key":    "enc-v",
		"feishu_token":   "fs-v",
	}
	for key, value := range values {
		t.Run(key, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{key: value})
			require.NoError(t, err)

			masked := MaskSensitiveFields(string(body))
			require.NotContains(t, masked, value, "键 %s 的值必须被脱敏", key)
		})
	}
}

func TestMaskSensitiveFields_MasksNestedAndArrays(t *testing.T) {
	body := `{"user":{"profile":{"newPassword":"deep-secret"},"contacts":[{"email":"a@b.com","phone":"13800138000"}]},"items":[{"accessToken":"arr-token"}]}`

	masked := MaskSensitiveFields(body)

	require.NotContains(t, masked, "deep-secret")
	require.NotContains(t, masked, "a@b.com")
	require.NotContains(t, masked, "13800138000")
	require.NotContains(t, masked, "arr-token")
	require.True(t, json.Valid([]byte(masked)), "脱敏后必须仍是合法 JSON：%s", masked)
}

func TestMaskSensitiveFields_LeavesOrdinaryFieldsAlone(t *testing.T) {
	body := `{"title":"无法登录","status":"open","priority":"high","ticketId":42,"count":3,"enabled":true,"note":null}`

	require.Equal(t, body, MaskSensitiveFields(body), "普通业务字段不应被改写，否则审计失去排查价值")
}

func TestMaskSensitiveFields_MasksNonStringScalars(t *testing.T) {
	// 少数接入方把 token 直接写成数字，字符串规则会漏掉。
	require.NotContains(t, MaskSensitiveFields(`{"cardNumber":4111111111111111}`), "4111111111111111")
	require.NotContains(t, MaskSensitiveFields(`{"password":123456}`), "123456")
}

func TestMaskSensitiveFields_HandlesEscapedQuotes(t *testing.T) {
	body := `{"password":"a\"b","title":"ok"}`

	masked := MaskSensitiveFields(body)

	require.NotContains(t, masked, `a\"b`)
	require.Contains(t, masked, `"title":"ok"`)
}

func TestMaskSensitiveFields_IsIdempotent(t *testing.T) {
	body := `{"newPassword":"x","title":"t"}`
	once := MaskSensitiveFields(body)

	require.Equal(t, once, MaskSensitiveFields(once), "重复脱敏不得继续变化")
}

func TestMaskSensitiveFields_NonJSONPassesThrough(t *testing.T) {
	body := "not a json body"
	require.Equal(t, body, MaskSensitiveFields(body))
	require.Equal(t, "", MaskSensitiveFields(""))
}

func TestIsSensitiveFieldKey(t *testing.T) {
	sensitive := []string{"password", "newPassword", "old_password", "passwd", "pwd", "accessToken", "jwt", "clientSecret", "x-api-key", "cookie", "Set-Cookie", "cvv", "idNumber"}
	for _, key := range sensitive {
		require.True(t, IsSensitiveFieldKey(key), "%s 应判定为敏感", key)
	}

	safe := []string{"title", "status", "priority", "tokenType", "passwordUpdatedAtHint", "tenantId", "userId", "description"}
	for _, key := range safe {
		require.False(t, IsSensitiveFieldKey(key), "%s 不应判定为敏感", key)
	}
}
