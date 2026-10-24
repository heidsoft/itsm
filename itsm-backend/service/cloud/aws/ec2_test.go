package aws

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

func testAccount(cred string) *ent.CloudAccount {
	return &ent.CloudAccount{Provider: Provider, CredentialRef: cred}
}

// newTestClient 造一个指向本地 httptest 服务器的 apiClient。
func newTestClient(t *testing.T, srv *httptest.Server, region string) *apiClient {
	t.Helper()
	cred, err := cloud.ResolveAWSCredential(context.Background(), testCred)
	if err != nil {
		t.Fatalf("解析测试凭据失败: %v", err)
	}
	cred.Endpoint = srv.URL
	return &apiClient{
		region: region,
		cred:   cred,
		hc:     srv.Client(),
		now:    func() time.Time { return vectorTime() },
	}
}

const testCred = `{"type":"keys","access_key_id":"AKIATEST","secret_access_key":"secret"}`

const regionsXML = `<?xml version="1.0" encoding="UTF-8"?>
<DescribeRegionsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-1</requestId>
  <regionInfo>
    <item><regionName>us-east-1</regionName><endpoint>ec2.us-east-1.amazonaws.com</endpoint><optInStatus>opt-in-not-required</optInStatus></item>
    <item><regionName>ap-northeast-1</regionName><endpoint>ec2.ap-northeast-1.amazonaws.com</endpoint><optInStatus>opt-in-not-required</optInStatus></item>
    <item><regionName></regionName><endpoint>x</endpoint></item>
  </regionInfo>
</DescribeRegionsResponse>`

const instancesXML = `<?xml version="1.0" encoding="UTF-8"?>
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-2</requestId>
  <reservationSet>
    <item>
      <reservationId>r-1</reservationId>
      <instancesSet>
        <item>
          <instanceId>i-0abc</instanceId>
          <instanceType>t3.micro</instanceType>
          <imageId>ami-1</imageId>
          <architecture>x86_64</architecture>
          <launchTime>2024-03-01T10:00:00.000Z</launchTime>
          <privateIpAddress>10.0.0.5</privateIpAddress>
          <publicIpAddress>203.0.113.9</publicIpAddress>
          <vpcId>vpc-1</vpcId>
          <subnetId>subnet-1</subnetId>
          <keyName>ops</keyName>
          <instanceState><code>16</code><name>running</name></instanceState>
          <placement><availabilityZone>us-east-1a</availabilityZone></placement>
          <tagSet>
            <item><key>Name</key><value>web-01</value></item>
            <item><key>env</key><value>prod</value></item>
          </tagSet>
        </item>
        <item>
          <instanceId>i-0def</instanceId>
          <instanceType>m5.large</instanceType>
          <instanceState><code>48</code><name>terminated</name></instanceState>
          <placement><availabilityZone>us-east-1b</availabilityZone></placement>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
  <nextToken>tok-2</nextToken>
</DescribeInstancesResponse>`

func TestCallSendsPublicParamsAndSignature(t *testing.T) {
	var authz, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(regionsXML))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "ap-northeast-1")
	if err := c.call(context.Background(), "DescribeRegions", nil, nil); err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(body, "Action=DescribeRegions") || !strings.Contains(body, "Version="+apiVersion) {
		t.Fatalf("表单体缺少公共参数: %q", body)
	}
	if !strings.Contains(authz, "AWS4-HMAC-SHA256") {
		t.Fatalf("未带 SigV4 签名头: %q", authz)
	}
	// 凭据范围必须含 region 与签名服务名：service 写成 "aws" 或 region 写错
	// 都会得到 403，而且是「签名算对了但服务端不认」那类最难查的错。
	if !strings.Contains(authz, "20150830/ap-northeast-1/ec2/aws4_request") {
		t.Fatalf("凭据范围不符: %s", authz)
	}
	if !strings.Contains(authz, "SignedHeaders=content-type;host;x-amz-date") {
		t.Fatalf("签名头列表不符: %s", authz)
	}
}

func TestListRegionsDropsEmptyRegionName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(regionsXML))
	}))
	defer srv.Close()

	cred := `{"type":"keys","access_key_id":"AKIATEST","secret_access_key":"secret","endpoint":"` + srv.URL + `"}`
	regions, err := NewAdapter(zap.NewNop().Sugar()).ListRegions(context.Background(), testAccount(cred))
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(regions) != 2 || regions[0] != "us-east-1" || regions[1] != "ap-northeast-1" {
		t.Fatalf("regions = %v, 期望两个非空 region", regions)
	}
}

func TestDiscoverRegionParsesInstancesAndPropagatesNextToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		if !strings.Contains(body, "MaxResults=100") {
			t.Errorf("请求未带 MaxResults: %q", body)
		}
		if !strings.Contains(body, "NextToken=prev") {
			t.Errorf("nextToken 未传给上游: %q", body)
		}
		_, _ = w.Write([]byte(instancesXML))
	}))
	defer srv.Close()

	page, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "us-east-1", newTestClient(t, srv, "us-east-1"), "prev")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	if page.NextToken != "tok-2" {
		t.Fatalf("NextToken = %q, 期望 tok-2", page.NextToken)
	}
	if len(page.Resources) != 2 {
		t.Fatalf("资源数 = %d, 期望 2", len(page.Resources))
	}
	first, second := page.Resources[0], page.Resources[1]
	if first.ResourceID != "i-0abc" || first.Region != "us-east-1" || first.Zone != "us-east-1a" {
		t.Fatalf("首个资源字段不符: %+v", first)
	}
	// 控制台口径：实例名取 Name 标签。
	if first.ResourceName != "web-01" {
		t.Fatalf("ResourceName = %q, 期望取自 Name 标签", first.ResourceName)
	}
	if first.Status != "active" {
		t.Fatalf("running 应映射 active, 得到 %q", first.Status)
	}
	if first.Tags["env"] != "prod" {
		t.Fatalf("Tags = %v", first.Tags)
	}
	if second.Status != "retired" {
		t.Fatalf("terminated 应映射 retired, 得到 %q", second.Status)
	}
	// 无 Name 标签时回落实例 ID，而不是留空——CI 表里一片无名资源没法认。
	if second.ResourceName != "i-0def" {
		t.Fatalf("无名实例应回落 ID, 得到 %q", second.ResourceName)
	}
	if first.Extra["instance_type"] != "t3.micro" || first.Extra["public_ip"] != "203.0.113.9" {
		t.Fatalf("Extra 字段不符: %v", first.Extra)
	}
}

func TestErrorResponseCarriesEC2Code(t *testing.T) {
	const errXML = `<?xml version="1.0"?><Response><Errors><Error><Code>AuthFailure</Code><Message>AWS was not able to validate the provided access credentials</Message></Error></Errors><RequestID>r</RequestID></Response>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(errXML))
	}))
	defer srv.Close()

	err := newTestClient(t, srv, "us-east-1").call(context.Background(), "DescribeRegions", nil, nil)
	if err == nil {
		t.Fatal("401 应返回错误")
	}
	// Code 才是处置依据：AuthFailure（凭据错）与 OptInRequired（地域未开通）
	// 修法完全不同，只报状态码等于让人重新猜一遍。
	if !strings.Contains(err.Error(), "code=AuthFailure") || !strings.Contains(err.Error(), "http=401") {
		t.Fatalf("错误信息缺少 code/http: %v", err)
	}
}

func TestBaseURLUsesChinaDomainForCNRegions(t *testing.T) {
	if got := (&apiClient{region: "cn-north-1", cred: &cloud.ResolvedCredential{}}).baseURL(); got != "https://ec2.cn-north-1.amazonaws.com.cn" {
		t.Fatalf("cn-north-1 端点 = %q", got)
	}
	if got := (&apiClient{region: "us-east-1", cred: &cloud.ResolvedCredential{}}).baseURL(); got != "https://ec2.us-east-1.amazonaws.com" {
		t.Fatalf("us-east-1 端点 = %q", got)
	}
}

func TestInitClientsRequiresResolvableCredential(t *testing.T) {
	a := NewAdapter(zap.NewNop().Sugar())
	// iam_role 此前会返回一份 AK/SK 皆空的凭据，SigV4 照签不误——现在必须
	// 显式失败，而不是拿空凭据去发必然失败的请求。
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"iam_role"}`), []string{"us-east-1"}); err == nil {
		t.Fatal("iam_role 应 fail closed")
	}
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"keys","secret_access_key":"x"}`), []string{"us-east-1"}); err == nil {
		t.Fatal("缺 access_key_id 应失败")
	}
	clients, err := a.InitClients(context.Background(), testAccount(testCred), []string{"us-east-1", "eu-west-1"})
	if err != nil {
		t.Fatalf("InitClients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("客户端数 = %d, 期望 2", len(clients))
	}
}

func TestMapInstanceState(t *testing.T) {
	cases := map[string]string{
		"pending": "pending", "running": "active", "stopping": "inactive",
		"stopped": "inactive", "shutting-down": "inactive", "terminated": "retired",
		"": "inactive",
	}
	for in, want := range cases {
		if got := mapInstanceState(in); got != want {
			t.Errorf("mapInstanceState(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
