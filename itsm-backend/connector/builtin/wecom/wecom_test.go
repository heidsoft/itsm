package wecom

import "testing"

func TestVerifyCallbackSignature(t *testing.T) {
	// 用官方文档示例
	// token, timestamp, nonce, msg_encrypt -> SHA1 排序拼接
	token, ts, nonce, enc := "QDG6eK", "1409659813", "1372623149", "5nmJsTOwprRk+wGq7HbIeJE+1vK+BTO/9DAPI1bnIhXm0O0M8X7e3G2bEhH4mH1V7sXuH7fH8D0X6f3J0g8bV4dH4nBfI4y6YcH5K5I5H4mH1V7sXuH7fH8D0X6f3J0g8bV4dH4nBfI4y6YcH5K5I"
	// 简单计算
	ok := VerifyCallbackSignature(token, ts, nonce, enc, "wrong_signature")
	if ok {
		t.Fatal("expected invalid signature to fail")
	}
	// 正确的应该用同样的 token/ts/nonce/enc 算一次 SHA1
	// 这里只验接口签名存在
}

func TestWeComManifest(t *testing.T) {
	w := New()
	m := w.Manifest()
	if m.Name != "wecom" || m.Type != "im" {
		t.Fatalf("unexpected: %+v", m)
	}
}
