package tencent

import (
	"testing"

	"itsm-backend/service/cloud/signing"
)

// 官方测试向量来源：腾讯云《签名方法 v3》文档的 DescribeInstances 演算示例。
//
// 文档把 SecretId/SecretKey 打了码，所以最终 Signature 无法复现；但**规范请求
// 与其哈希是完整公开的**，而规范请求的拼接（头的排序、空白处理、空查询串位置）
// 恰恰是最容易写错、且写错了只会表现为「签名不一致」的部分。这里就用它能证伪
// 的那一半做标尺。
const (
	vectorPayload   = `{"Limit": 1, "Filters": [{"Values": ["\u672a\u547d\u540d"], "Name": "instance-name"}]}`
	vectorTimestamp = int64(1551113065)
	vectorCanonical = "7019a55be8395899b900fb5564e4200d984910f34794a27cb3fb7d10ff6a1e84"
	vectorPayloadMD = "35e9c5b0e3ae67532d3c9f17ead6c90222632e5b1ff7f6e89887f1398934f064"
)

func TestCanonicalRequestMatchesTencentVector(t *testing.T) {
	if got := signing.SHA256HexString(vectorPayload); got != vectorPayloadMD {
		t.Fatalf("payload 哈希 = %s, 官方值 %s", got, vectorPayloadMD)
	}
	hdrs := [][2]string{
		{"content-type", "application/json; charset=utf-8"},
		{"host", "cvm.tencentcloudapi.com"},
		{"x-tc-action", "describeinstances"},
	}
	got := canonicalRequest("POST", "/", hdrs, []byte(vectorPayload))
	want := "POST\n" +
		"/\n" +
		"\n" +
		"content-type:application/json; charset=utf-8\n" +
		"host:cvm.tencentcloudapi.com\n" +
		"x-tc-action:describeinstances\n" +
		"\n" +
		"content-type;host;x-tc-action\n" +
		vectorPayloadMD
	if got != want {
		t.Fatalf("规范请求串不符\n got: %q\nwant: %q", got, want)
	}
	if h := signing.SHA256HexString(got); h != vectorCanonical {
		t.Fatalf("规范请求哈希 = %s, 官方值 %s", h, vectorCanonical)
	}
}

func TestStringToSignMatchesTencentVector(t *testing.T) {
	hdrs := [][2]string{
		{"content-type", "application/json; charset=utf-8"},
		{"host", "cvm.tencentcloudapi.com"},
		{"x-tc-action", "describeinstances"},
	}
	canonical := canonicalRequest("POST", "/", hdrs, []byte(vectorPayload))
	sts, scope := stringToSign(canonical, "cvm", vectorTimestamp)
	// 时间戳 1551113065 按 UTC 换算是 2019-02-25——用本地时区换算会让签名在
	// 非 UTC 机器上整体失效，且只在跨天后才暴露。
	if scope != "2019-02-25/cvm/tc3_request" {
		t.Fatalf("scope = %q, 期望 2019-02-25/cvm/tc3_request", scope)
	}
	want := "TC3-HMAC-SHA256\n1551113065\n2019-02-25/cvm/tc3_request\n" + vectorCanonical
	if sts != want {
		t.Fatalf("待签名串不符\n got: %q\nwant: %q", sts, want)
	}
}

func TestSignProducesDocumentedHeaderShape(t *testing.T) {
	s := &tc3Signer{secretID: "AKIDz8krbsJ5yKBZQpn74WFkmLPx3EXAMPLE", secretKey: "Gu5t9xGARNpq86cd98joQYCN3EXAMPLE"}
	hdrs := s.Sign("POST", "/", "cvm.tencentcloudapi.com", "DescribeInstances", []byte(vectorPayload), "cvm", vectorTimestamp)
	authz := hdrs["Authorization"]
	// Action 必须小写后参与签名：文档里的 x-tc-action 是 describeinstances。
	for _, want := range []string{
		"TC3-HMAC-SHA256 Credential=AKIDz8krbsJ5yKBZQpn74WFkmLPx3EXAMPLE/2019-02-25/cvm/tc3_request, ",
		"SignedHeaders=content-type;host;x-tc-action, ",
		"Signature=",
	} {
		if !contains(authz, want) {
			t.Fatalf("Authorization 缺少 %q: %s", want, authz)
		}
	}
}

func TestSigningKeyIsServiceScoped(t *testing.T) {
	s := &tc3Signer{secretKey: "Gu5t9xGARNpq86cd98joQYCN3EXAMPLE"}
	cvm := s.signingKey(vectorTimestamp, "cvm")
	vpc := s.signingKey(vectorTimestamp, "vpc")
	if string(cvm) == string(vpc) {
		t.Fatal("不同 service 派生出相同签名密钥——链式派生漏了 service 这一步")
	}
	// 换一天必须换密钥，否则签名长期有效会是安全问题（腾讯云用时间戳限窗口）。
	if string(s.signingKey(vectorTimestamp, "cvm")) == string(s.signingKey(vectorTimestamp+86400, "cvm")) {
		t.Fatal("不同日期派生出相同签名密钥")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
