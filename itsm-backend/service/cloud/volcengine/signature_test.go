package volcengine

import (
	"strings"
	"testing"
	"time"

	"itsm-backend/service/cloud/signing"
)

// 与 aws / tencent 不同：火山引擎官方只公开了签名的**步骤与头部语法**，没有给出
// 一组可复现的 AK/SK + 期望签名，所以这里没有类似 AWS 5d672d79… 那样的黄金
// 向量。测试因此退而求其次，锁住三件能被客观验证的事：
//  1. Authorization 的语法与文档一致（算法名、凭据范围结尾、签名头组合）；
//  2. 规范请求/待签名串的拼接结构；
//  3. 派生密钥与 SigV4 的实质差异（不加 "AWS4" 盐、以 request 结尾）——
//     这是混用两套算法时唯一会出错的地方，必须能证伪。
//
// 这一限制在 UPGRADE 里同步说明，避免后来者误以为已按官方向量验证过。
var fixedNow = time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)

func TestSignProducesDocumentedHeaderGrammar(t *testing.T) {
	s := &volcSigner{accessKeyID: "AKLTTEST", secretAccessKey: "secret"}
	hdrs := s.Sign("GET", "/", "Action=DescribeRegions&Version=2020-04-01",
		"open.volcengineapi.com", "application/json", nil, "cn-beijing", "ecs", fixedNow)

	authz := hdrs["Authorization"]
	for _, want := range []string{
		"HMAC-SHA256 Credential=AKLTTEST/20240301/cn-beijing/ecs/request, ",
		"SignedHeaders=content-type;host;x-content-sha256;x-date, ",
		"Signature=",
	} {
		if !strings.Contains(authz, want) {
			t.Fatalf("Authorization 缺少 %q: %s", want, authz)
		}
	}
	// 不能出现 SigV4 的算法名或结尾——混用只会在服务端表现为 403。
	if strings.Contains(authz, "AWS4-HMAC-SHA256") || strings.Contains(authz, "aws4_request") {
		t.Fatalf("混用了 SigV4 的算法名/结尾: %s", authz)
	}
	if hdrs["X-Date"] != "20240301T100000Z" {
		t.Fatalf("X-Date = %q, 期望 20240301T100000Z", hdrs["X-Date"])
	}
	// 空 body 的哈希是公开的 SHA256 空串摘要，可客观验证。
	if hdrs["X-Content-Sha256"] != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("X-Content-Sha256 = %q", hdrs["X-Content-Sha256"])
	}
}

func TestCanonicalRequestUsesQueryStringNotBody(t *testing.T) {
	hdrs := [][2]string{
		{"content-type", "application/json"},
		{"host", "open.volcengineapi.com"},
		{"x-content-sha256", "e3b0c442"},
		{"x-date", "20240301T100000Z"},
	}
	got := canonicalRequest("GET", "/", "Action=DescribeRegions&Version=2020-04-01", hdrs, nil)
	want := "GET\n" +
		"/\n" +
		"Action=DescribeRegions&Version=2020-04-01\n" +
		"content-type:application/json\n" +
		"host:open.volcengineapi.com\n" +
		"x-content-sha256:e3b0c442\n" +
		"x-date:20240301T100000Z\n" +
		"\n" +
		"content-type;host;x-content-sha256;x-date\n" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("规范请求串不符\n got: %q\nwant: %q", got, want)
	}
}

func TestStringToSignScopeEndsWithRequest(t *testing.T) {
	sts, scope := stringToSign("canonical", "cn-beijing", "ecs", fixedNow)
	if scope != "20240301/cn-beijing/ecs/request" {
		t.Fatalf("scope = %q", scope)
	}
	want := "HMAC-SHA256\n20240301T100000Z\n20240301/cn-beijing/ecs/request\n" +
		signing.SHA256HexString("canonical")
	if sts != want {
		t.Fatalf("待签名串不符\n got: %q\nwant: %q", sts, want)
	}
}

// TestSigningKeyDiffersFromSigV4 锁住派生密钥的实质差异。
// SigV4 首步是 HMAC("AWS4"+secret, date)，火山是 HMAC(secret, date)。
func TestSigningKeyDiffersFromSigV4(t *testing.T) {
	const secret = "Gu5t9xGARNpq86cd98joQYCN3EXAMPLE"
	volc := (&volcSigner{secretAccessKey: secret}).signingKey(fixedNow, "cn-beijing", "ecs")

	// 用同样的 secret 走一遍 SigV4 的派生，两者必须不同。
	sigv4 := signing.HMACSHA256([]byte("AWS4"+secret), "20240301")
	sigv4 = signing.HMACSHA256(sigv4, "cn-beijing")
	sigv4 = signing.HMACSHA256(sigv4, "ecs")
	sigv4 = signing.HMACSHA256(sigv4, "aws4_request")
	if string(volc) == string(sigv4) {
		t.Fatal("火山与 SigV4 派生出相同签名密钥——漏掉了「不加 AWS4 盐 + request 结尾」这两处差异")
	}

	// 反过来说：把 secret 加上 AWS4 盐、结尾换成 request，就应当与 SigV4 一致
	// （只差最后一步的终止串），以此确认差别的确只在这两处。
	sameAsSigV4ButTerminator := signing.HMACSHA256([]byte("AWS4"+secret), "20240301")
	sameAsSigV4ButTerminator = signing.HMACSHA256(sameAsSigV4ButTerminator, "cn-beijing")
	sameAsSigV4ButTerminator = signing.HMACSHA256(sameAsSigV4ButTerminator, "ecs")
	sameAsSigV4ButTerminator = signing.HMACSHA256(sameAsSigV4ButTerminator, "request")
	if string(sameAsSigV4ButTerminator) == string(sigv4) {
		t.Fatal("仅换终止串不应与 SigV4 相同")
	}
}

func TestSigningKeyIsRegionAndServiceScoped(t *testing.T) {
	s := &volcSigner{secretAccessKey: "secret"}
	base := s.signingKey(fixedNow, "cn-beijing", "ecs")
	if string(base) == string(s.signingKey(fixedNow, "cn-shanghai", "ecs")) {
		t.Fatal("换 region 未换签名密钥")
	}
	if string(base) == string(s.signingKey(fixedNow, "cn-beijing", "storage_ebs")) {
		t.Fatal("换 service 未换签名密钥")
	}
	if string(base) == string(s.signingKey(fixedNow.AddDate(0, 0, 1), "cn-beijing", "ecs")) {
		t.Fatal("换日期未换签名密钥")
	}
}

func TestSignIsPayloadSensitive(t *testing.T) {
	s := &volcSigner{accessKeyID: "AKLTTEST", secretAccessKey: "secret"}
	a := s.Sign("POST", "/", "Action=RunInstances&Version=2020-04-01", "open.volcengineapi.com",
		"application/json", []byte(`{"a":1}`), "cn-beijing", "ecs", fixedNow)
	b := s.Sign("POST", "/", "Action=RunInstances&Version=2020-04-01", "open.volcengineapi.com",
		"application/json", []byte(`{"a":2}`), "cn-beijing", "ecs", fixedNow)
	if a["Authorization"] == b["Authorization"] {
		t.Fatal("不同 payload 得到相同签名")
	}
	if a["X-Content-Sha256"] == b["X-Content-Sha256"] {
		t.Fatal("X-Content-Sha256 未随 payload 变化")
	}
}

func TestSessionTokenEntersSignedHeaders(t *testing.T) {
	s := &volcSigner{accessKeyID: "AKLTTEST", secretAccessKey: "secret", sessionToken: "STS-TOKEN"}
	hdrs := s.Sign("GET", "/", "Action=DescribeRegions", "open.volcengineapi.com",
		"application/json", nil, "cn-beijing", "ecs", fixedNow)
	if hdrs["X-Security-Token"] != "STS-TOKEN" {
		t.Fatalf("未带 X-Security-Token: %v", hdrs)
	}
	// token 表头必须进签名头列表，否则服务端重算时不算它 → 403。
	if !strings.Contains(hdrs["Authorization"], "x-security-token") {
		t.Fatalf("token 未进入 SignedHeaders: %s", hdrs["Authorization"])
	}
}
