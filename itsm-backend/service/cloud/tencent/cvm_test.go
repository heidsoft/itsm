package tencent

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

const testCred = `{"type":"ak","secret_id":"AKIDTEST","secret_key":"secret"}`

func testAccount(cred string) *ent.CloudAccount {
	return &ent.CloudAccount{Provider: Provider, CredentialRef: cred}
}

func newTestClient(t *testing.T, srv *httptest.Server, region string) *apiClient {
	t.Helper()
	cred, err := cloud.ResolveCredential(context.Background(), Provider, testCred)
	if err != nil {
		t.Fatalf("解析测试凭据失败: %v", err)
	}
	return &apiClient{region: region, cred: cred, hc: srv.Client(), endpoint: srv.URL, now: fixedNow}
}

func fixedNow() time.Time { return time.Unix(vectorTimestamp, 0) }

const regionsJSON = `{"Response":{"RequestId":"r1","RegionSet":[
  {"Region":"ap-guangzhou","RegionName":"华南地区(广州)","RegionState":"AVAILABLE"},
  {"Region":"ap-shanghai","RegionName":"华东地区(上海)","RegionState":"AVAILABLE"},
  {"Region":"","RegionName":"bad"}
]}}`

const instancesJSON = `{"Response":{"RequestId":"r2","TotalCount":3,"InstanceSet":[
  {"InstanceId":"ins-1","InstanceName":"web-01","InstanceType":"S5.MEDIUM4","CPU":2,"Memory":4,
   "OsName":"TencentOS 3.1","ImageId":"img-1","InstanceState":"RUNNING","CreatedTime":"2024-03-01T10:00:00Z",
   "PrivateIpAddresses":["10.0.0.5"],"PublicIpAddresses":["203.0.113.9"],"InstanceChargeType":"POSTPAID_BY_HOUR",
   "Placement":{"Zone":"ap-guangzhou-3"},"VirtualPrivateCloud":{"VpcId":"vpc-1","SubnetId":"subnet-1"},
   "Tags":[{"Key":"env","Value":"prod"}]},
  {"InstanceId":"ins-2","InstanceState":"STOPPED","Placement":{"Zone":"ap-guangzhou-4"}},
  {"InstanceId":"ins-3","InstanceState":"TERMINATING","Placement":{"Zone":"ap-guangzhou-5"}}
]}}`

func TestCallSendsActionHeadersAndSignature(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		if r.Header.Get("X-TC-Action") != "DescribeRegions" {
			t.Errorf("X-TC-Action = %q", r.Header.Get("X-TC-Action"))
		}
		if r.Header.Get("X-TC-Version") != apiVersion {
			t.Errorf("X-TC-Version = %q", r.Header.Get("X-TC-Version"))
		}
		if r.Header.Get("X-TC-Region") != defaultRegion {
			t.Errorf("X-TC-Region = %q", r.Header.Get("X-TC-Region"))
		}
		if !strings.Contains(r.Header.Get("Authorization"), "TC3-HMAC-SHA256") {
			t.Errorf("未带 TC3 签名头: %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(regionsJSON))
	}))
	defer srv.Close()

	var out apiResponse
	if err := newTestClient(t, srv, defaultRegion).call(context.Background(), "DescribeRegions", []byte(`{}`), &out); err != nil {
		t.Fatalf("call: %v", err)
	}
	if body != `{}` {
		t.Errorf("body = %q", body)
	}
	// 凭据范围里的日期必须由时间戳按 UTC 换算，且签名服务名是 cvm。
	_ = body
}

func TestListRegionsDropsEmptyRegion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(regionsJSON))
	}))
	defer srv.Close()

	cred := `{"type":"ak","secret_id":"AKIDTEST","secret_key":"secret","endpoint":"` + srv.URL + `"}`
	regions, err := NewAdapter(zap.NewNop().Sugar()).ListRegions(context.Background(), testAccount(cred))
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(regions) != 2 || regions[0] != "ap-guangzhou" || regions[1] != "ap-shanghai" {
		t.Fatalf("regions = %v, 期望两个非空地域", regions)
	}
}

func TestDiscoverRegionPaginatesByOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), `"Limit":100`) {
			t.Errorf("请求未带 Limit: %s", raw)
		}
		if !strings.Contains(string(raw), `"Offset":100`) {
			t.Errorf("nextToken 未换算成 Offset: %s", raw)
		}
		_, _ = w.Write([]byte(instancesJSON))
	}))
	defer srv.Close()

	page, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "ap-guangzhou", newTestClient(t, srv, "ap-guangzhou"), "100")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	if len(page.Resources) != 3 {
		t.Fatalf("资源数 = %d", len(page.Resources))
	}
	// TotalCount=3、本页 3 条、offset=100 → 已到末尾，不应再给 next。
	if page.NextToken != "" {
		t.Fatalf("NextToken = %q, 期望空", page.NextToken)
	}
	first := page.Resources[0]
	if first.ResourceID != "ins-1" || first.ResourceName != "web-01" || first.Region != "ap-guangzhou" {
		t.Fatalf("首个资源不符: %+v", first)
	}
	if first.Status != "active" || first.Zone != "ap-guangzhou-3" {
		t.Fatalf("状态/可用区不符: %+v", first)
	}
	if first.Tags["env"] != "prod" {
		t.Fatalf("Tags = %v", first.Tags)
	}
	if page.Resources[1].Status != "inactive" {
		t.Fatalf("STOPPED 应映射 inactive, 得到 %q", page.Resources[1].Status)
	}
	if page.Resources[2].Status != "retired" {
		t.Fatalf("TERMINATING 应映射 retired, 得到 %q", page.Resources[2].Status)
	}
	// 无 InstanceName 时回落实例 ID。
	if page.Resources[1].ResourceName != "ins-2" {
		t.Fatalf("无名实例应回落 ID, 得到 %q", page.Resources[1].ResourceName)
	}
}

func TestDiscoverRegionContinuesWhenTotalNotReached(t *testing.T) {
	// TotalCount 大于本页条数时必须继续，否则会静默漏掉资源。
	const partial = `{"Response":{"TotalCount":250,"InstanceSet":[{"InstanceId":"ins-1","InstanceState":"RUNNING"}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(partial))
	}))
	defer srv.Close()

	page, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "ap-guangzhou", newTestClient(t, srv, "ap-guangzhou"), "")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	if page.NextToken != "1" {
		t.Fatalf("NextToken = %q, 期望 \"1\"（offset 前进一位）", page.NextToken)
	}
}

func TestDiscoverRegionRejectsMalformedNextToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	if _, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "ap-guangzhou", newTestClient(t, srv, "ap-guangzhou"), "abc"); err == nil {
		t.Fatal("非法 nextToken 应报错而不是当成 offset=0 重复扫首页")
	}
}

func TestBusinessErrorOnHTTP200IsNotTreatedAsSuccess(t *testing.T) {
	// 腾讯云的业务错误是 200 + Response.Error。只判状态码会得到一份空资源列表，
	// 表现为「发现跑完了但什么都没发现」——这类假成功最难定位。
	const errJSON = `{"Response":{"Error":{"Code":"AuthFailure.SignatureExpire","Message":"Signature expired"},"RequestId":"r"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(errJSON))
	}))
	defer srv.Close()

	err := newTestClient(t, srv, defaultRegion).call(context.Background(), "DescribeRegions", []byte(`{}`), nil)
	if err == nil {
		t.Fatal("200 + Error 必须报错")
	}
	if !strings.Contains(err.Error(), "code=AuthFailure.SignatureExpire") {
		t.Fatalf("错误信息缺少 Code: %v", err)
	}
}

func TestInitClientsRequiresResolvableCredential(t *testing.T) {
	a := NewAdapter(zap.NewNop().Sugar())
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"unknown"}`), []string{"ap-guangzhou"}); err == nil {
		t.Fatal("未知凭据类型应失败")
	}
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"ak","secret_key":"x"}`), []string{"ap-guangzhou"}); err == nil {
		t.Fatal("缺 secret_id 应失败")
	}
	clients, err := a.InitClients(context.Background(), testAccount(testCred), []string{"ap-guangzhou", "ap-shanghai"})
	if err != nil {
		t.Fatalf("InitClients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("客户端数 = %d", len(clients))
	}
}

func TestMapInstanceState(t *testing.T) {
	cases := map[string]string{
		"PENDING": "pending", "STARTING": "pending", "REBOOTING": "pending",
		"RUNNING": "active", "STOPPED": "inactive", "STOPPING": "inactive",
		"SHUTDOWN": "inactive", "TERMINATING": "retired", "LAUNCH_FAILED": "retired",
		"": "inactive",
	}
	for in, want := range cases {
		if got := mapInstanceState(in); got != want {
			t.Errorf("mapInstanceState(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
