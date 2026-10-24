package signing

import "testing"

func TestURIEncodeKeepsUnreservedCharacters(t *testing.T) {
	if got := URIEncode("aZ0-_.~", false); got != "aZ0-_.~" {
		t.Fatalf("未保留无保留字符: %q", got)
	}
	// QueryEscape 会把空格编成 '+'，SigV4 要求 %20——这是「本地算得过、
	// 服务端 403」的经典差异，必须锁住。
	if got := URIEncode("a b", true); got != "a%20b" {
		t.Fatalf("空格应编成 %%20, 得到 %q", got)
	}
	if got := URIEncode("a/b", true); got != "a%2Fb" {
		t.Fatalf("encodeSlash=true 时 '/' 应编成 %%2F, 得到 %q", got)
	}
	if got := URIEncode("a/b", false); got != "a/b" {
		t.Fatalf("encodeSlash=false 时应保留 '/', 得到 %q", got)
	}
	// 百分号必须是大写十六进制。
	if got := URIEncode("\xff", true); got != "%FF" {
		t.Fatalf("十六进制应大写: %q", got)
	}
}

func TestEncodeFormSortsAndEncodes(t *testing.T) {
	got := EncodeForm(map[string]string{
		"Version":    "2016-11-15",
		"Action":     "DescribeInstances",
		"MaxResults": "100",
		"NextToken":  "a/b",
	})
	want := "Action=DescribeInstances&MaxResults=100&NextToken=a%2Fb&Version=2016-11-15"
	if got != want {
		t.Fatalf("EncodeForm = %q, 期望 %q（键须 ASCII 升序且值 URI 编码）", got, want)
	}
}

func TestCanonicalHeadersAndSignedHeaderListAgree(t *testing.T) {
	hdrs := [][2]string{
		{"X-Amz-Date", " 20240101T000000Z "},
		{"host", "ec2.us-east-1.amazonaws.com"},
		{"content-type", "application/x-www-form-urlencoded"},
	}
	canonical := CanonicalHeaders(hdrs)
	want := "content-type:application/x-www-form-urlencoded\n" +
		"host:ec2.us-east-1.amazonaws.com\n" +
		"x-amz-date:20240101T000000Z\n"
	if canonical != want {
		t.Fatalf("规范头不符\n got: %q\nwant: %q", canonical, want)
	}
	// 名字列表与规范头的顺序必须一致，否则服务端拼出的摘要不同。
	if got := SignedHeaderList(hdrs); got != "content-type;host;x-amz-date" {
		t.Fatalf("签名头列表 = %q", got)
	}
}

func TestCanonicalHeadersDoesNotMutateInput(t *testing.T) {
	hdrs := [][2]string{{"X-B", "2"}, {"x-a", "1"}}
	_ = CanonicalHeaders(hdrs)
	if hdrs[0][0] != "X-B" || hdrs[1][0] != "x-a" {
		t.Fatalf("排序污染了调用方的切片: %v", hdrs)
	}
}

func TestHMACSHA256IsChainable(t *testing.T) {
	// 链式派生要求上一步的输出直接当下一步的 key，所以必须能拿字节。
	k1 := HMACSHA256([]byte("secret"), "20240101")
	k2 := HMACSHA256(k1, "us-east-1")
	if len(k1) != 32 || len(k2) != 32 {
		t.Fatalf("HMAC-SHA256 摘要长度应为 32 字节, 得到 %d/%d", len(k1), len(k2))
	}
	if string(k1) == string(k2) {
		t.Fatal("不同消息得到相同摘要")
	}
	if HMACSHA256Hex([]byte("secret"), "x") != HMACSHA256Hex([]byte("secret"), "x") {
		t.Fatal("同输入应得到同输出")
	}
}
