package aws

import (
	"testing"
	"time"

	"itsm-backend/service/cloud/signing"
)

// 官方测试向量来源：AWS General Reference《Signature Version 4 signing process》
// 的示例（AKIDEXAMPLE / wJalrXUtnFEMI，GET iam.amazonaws.com ListUsers）。
//
// 之所以把这些值写死而不是「自己算一遍对一遍」：自己算两遍用的是同一份对规范
// 的理解，错了也不会被发现。官方向量是唯一能证伪实现的外部标尺。
const (
	vectorAccessKey = "AKIDEXAMPLE"
	vectorSecretKey = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	vectorCanonical = "f536975d06c0309214f805bb90ccff089219ecd68b2577efef23edd43b7e1a59"
	vectorSignature = "5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
	vectorAuthz     = "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, SignedHeaders=content-type;host;x-amz-date, Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
)

func vectorTime() time.Time { return time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC) }

func TestCanonicalRequestMatchesAWSVector(t *testing.T) {
	hdrs := [][2]string{
		{"content-type", "application/x-www-form-urlencoded; charset=utf-8"},
		{"host", "iam.amazonaws.com"},
		{"x-amz-date", "20150830T123600Z"},
	}
	got := canonicalRequest("GET", "/", "Action=ListUsers&Version=2010-05-08", hdrs, nil)
	want := "GET\n" +
		"/\n" +
		"Action=ListUsers&Version=2010-05-08\n" +
		"content-type:application/x-www-form-urlencoded; charset=utf-8\n" +
		"host:iam.amazonaws.com\n" +
		"x-amz-date:20150830T123600Z\n" +
		"\n" +
		"content-type;host;x-amz-date\n" +
		signing.SHA256HexString("")
	if got != want {
		t.Fatalf("规范请求串不符\n got: %q\nwant: %q", got, want)
	}
	if h := signing.SHA256HexString(got); h != vectorCanonical {
		t.Fatalf("规范请求哈希 = %s, 官方值 %s", h, vectorCanonical)
	}
}

func TestStringToSignMatchesAWSVector(t *testing.T) {
	hdrs := [][2]string{
		{"content-type", "application/x-www-form-urlencoded; charset=utf-8"},
		{"host", "iam.amazonaws.com"},
		{"x-amz-date", "20150830T123600Z"},
	}
	canonical := canonicalRequest("GET", "/", "Action=ListUsers&Version=2010-05-08", hdrs, nil)
	sts, scope := stringToSign(canonical, "us-east-1", "iam", vectorTime())
	want := "AWS4-HMAC-SHA256\n20150830T123600Z\n20150830/us-east-1/iam/aws4_request\n" + vectorCanonical
	if sts != want {
		t.Fatalf("待签名串不符\n got: %q\nwant: %q", sts, want)
	}
	if scope != "20150830/us-east-1/iam/aws4_request" {
		t.Fatalf("scope = %q", scope)
	}
}

func TestSignMatchesAWSVector(t *testing.T) {
	s := &sigv4Signer{accessKeyID: vectorAccessKey, secretAccessKey: vectorSecretKey}
	got := s.Sign(
		"GET", "/", "Action=ListUsers&Version=2010-05-08", "iam.amazonaws.com",
		[][2]string{{"content-type", "application/x-www-form-urlencoded; charset=utf-8"}},
		nil, "us-east-1", "iam", vectorTime(),
	)
	if got["Authorization"] != vectorAuthz {
		t.Fatalf("Authorization 不符\n got: %s\nwant: %s", got["Authorization"], vectorAuthz)
	}
	if got[sigV4DateHeader] != "20150830T123600Z" {
		t.Fatalf("X-Amz-Date = %s", got[sigV4DateHeader])
	}
}

func TestSignIncludesSessionTokenWhenPresent(t *testing.T) {
	s := &sigv4Signer{accessKeyID: vectorAccessKey, secretAccessKey: vectorSecretKey, sessionToken: "FQoDYXdzE"}
	got := s.Sign("POST", "/", "", "ec2.us-east-1.amazonaws.com",
		[][2]string{{"content-type", "application/x-www-form-urlencoded; charset=utf-8"}},
		[]byte("Action=DescribeRegions"), "us-east-1", "ec2", vectorTime())
	if got[sigV4TokenHeader] != "FQoDYXdzE" {
		t.Fatalf("临时凭据未带 security token: %v", got)
	}
	// token 必须出现在签名头列表里，否则服务端重算时不把它算进去 → 403。
	authz := got["Authorization"]
	if !contains(authz, "x-amz-security-token") {
		t.Fatalf("security token 未进入 SignedHeaders: %s", authz)
	}
}

func TestSigningKeyDerivationIsRegionScoped(t *testing.T) {
	now := vectorTime()
	a := &sigv4Signer{secretAccessKey: vectorSecretKey}
	usEast := a.signingKey(now, "us-east-1", "ec2")
	euWest := a.signingKey(now, "eu-west-1", "ec2")
	if string(usEast) == string(euWest) {
		t.Fatal("不同 region 派生出相同签名密钥——链式派生漏了 region 这一步")
	}
	other := &sigv4Signer{secretAccessKey: vectorSecretKey + "x"}
	if string(other.signingKey(now, "us-east-1", "ec2")) == string(usEast) {
		t.Fatal("不同 secret 派生出相同签名密钥")
	}
}

func TestCanonicalRequestIsPayloadSensitive(t *testing.T) {
	hdrs := [][2]string{{"host", "ec2.us-east-1.amazonaws.com"}}
	a := canonicalRequest("POST", "/", "", hdrs, []byte("Action=DescribeRegions"))
	b := canonicalRequest("POST", "/", "", hdrs, []byte("Action=DescribeInstances"))
	if a == b {
		t.Fatal("不同 payload 得到相同规范请求——payload 哈希没进规范请求")
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
