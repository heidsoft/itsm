package volcengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

const testCred = `{"type":"ak","access_key_id":"AKLTTEST","access_key_secret":"secret"}`

func testAccount(cred string) *ent.CloudAccount {
	return &ent.CloudAccount{Provider: Provider, CredentialRef: cred}
}

func newTestClient(t *testing.T, srv *httptest.Server, region string) *apiClient {
	t.Helper()
	cred, err := cloud.ResolveVolcengineCredential(context.Background(), testCred)
	if err != nil {
		t.Fatalf("解析测试凭据失败: %v", err)
	}
	return &apiClient{region: region, cred: cred, hc: srv.Client(), endpoint: srv.URL, now: func() time.Time { return fixedNow }}
}

const regionsJSON = `{"ResponseMetadata":{"RequestId":"r1","Action":"DescribeRegions","Version":"2020-04-01","Service":"ecs","Region":"cn-beijing"},
"Result":{"Regions":[{"RegionId":"cn-beijing","RegionName":"华北2（北京）"},{"RegionId":"cn-shanghai","RegionName":"华东2（上海）"},{"RegionId":""}]}}`

const instancesJSON = `{"ResponseMetadata":{"RequestId":"r2","Action":"DescribeInstances","Version":"2020-04-01","Service":"ecs","Region":"cn-beijing"},
"Result":{"TotalCount":150,"Instances":[
 {"InstanceId":"i-1","InstanceName":"web-01","InstanceType":"ecs.g1.large","Status":"RUNNING","ZoneId":"cn-beijing-b",
  "ImageId":"image-1","VpcId":"vpc-1","Hostname":"web-01","OsType":"Linux","Cpus":2,"MemorySize":8,
  "CreatedAt":"2024-03-01T10:00:00+08:00","UpdatedAt":"2024-03-02T10:00:00+08:00",
  "NetworkInterfaces":[{"NetworkInterfaceId":"eni-1","SubnetId":"subnet-1","PrimaryIpAddress":"192.168.1.10"}],
  "Tags":[{"Key":"env","Value":"prod"}]},
 {"InstanceId":"i-2","Status":"STOPPED","ZoneId":"cn-beijing-c"},
 {"InstanceId":"i-3","Status":"DELETED","ZoneId":"cn-beijing-d"}
]}}`

func TestCallPutsActionAndVersionInQueryString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("Action") != "DescribeRegions" {
			t.Errorf("Action 必须在 Query String: %v", q)
		}
		if q.Get("Version") != apiVersion {
			t.Errorf("Version = %q", q.Get("Version"))
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q", r.Method)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "HMAC-SHA256 Credential=AKLTTEST/20240301/cn-beijing/ecs/request") {
			t.Errorf("凭据范围不符: %s", authz)
		}
		_, _ = w.Write([]byte(regionsJSON))
	}))
	defer srv.Close()

	var out describeRegionsResponse
	if err := newTestClient(t, srv, "cn-beijing").call(context.Background(), "DescribeRegions", nil, &out); err != nil {
		t.Fatalf("call: %v", err)
	}
}

func TestListRegionsDropsEmptyRegionID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(regionsJSON))
	}))
	defer srv.Close()

	cred := `{"type":"ak","access_key_id":"AKLTTEST","access_key_secret":"secret","endpoint":"` + srv.URL + `"}`
	regions, err := NewAdapter(zap.NewNop().Sugar()).ListRegions(context.Background(), testAccount(cred))
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(regions) != 2 || regions[0] != "cn-beijing" || regions[1] != "cn-shanghai" {
		t.Fatalf("regions = %v, 期望两个非空地域", regions)
	}
}

func TestDiscoverRegionPaginatesByPageNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("PageSize") != "100" {
			t.Errorf("PageSize = %q", q.Get("PageSize"))
		}
		if q.Get("PageNumber") != "2" {
			t.Errorf("nextToken 未换算成 PageNumber: %q", q.Get("PageNumber"))
		}
		_, _ = w.Write([]byte(instancesJSON))
	}))
	defer srv.Close()

	page, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "cn-beijing", newTestClient(t, srv, "cn-beijing"), "2")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	// TotalCount=150、当前第 2 页（2*100=200 ≥ 150）→ 已到末尾。
	if page.NextToken != "" {
		t.Fatalf("NextToken = %q, 期望空", page.NextToken)
	}
	if len(page.Resources) != 3 {
		t.Fatalf("资源数 = %d", len(page.Resources))
	}
	first := page.Resources[0]
	if first.ResourceID != "i-1" || first.ResourceName != "web-01" || first.Region != "cn-beijing" {
		t.Fatalf("首个资源不符: %+v", first)
	}
	if first.Status != "active" || first.Zone != "cn-beijing-b" {
		t.Fatalf("状态/可用区不符: %+v", first)
	}
	if first.Tags["env"] != "prod" {
		t.Fatalf("Tags = %v", first.Tags)
	}
	if first.Extra["primary_ip"] != "192.168.1.10" || first.Extra["subnet_id"] != "subnet-1" {
		t.Fatalf("Extra 未取网卡信息: %v", first.Extra)
	}
	if page.Resources[1].Status != "inactive" || page.Resources[2].Status != "retired" {
		t.Fatalf("状态映射不符: %q / %q", page.Resources[1].Status, page.Resources[2].Status)
	}
	if page.Resources[1].ResourceName != "i-2" {
		t.Fatalf("无名实例应回落 ID, 得到 %q", page.Resources[1].ResourceName)
	}
}

func TestDiscoverRegionContinuesUntilTotalReached(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(instancesJSON))
	}))
	defer srv.Close()

	page, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "cn-beijing", newTestClient(t, srv, "cn-beijing"), "")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	// 第 1 页 100 条 < TotalCount 150 → 应继续。
	if page.NextToken != "2" {
		t.Fatalf("NextToken = %q, 期望 \"2\"", page.NextToken)
	}
}

func TestDiscoverRegionRejectsMalformedNextToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	if _, err := NewAdapter(zap.NewNop().Sugar()).DiscoverRegion(
		context.Background(), testAccount(testCred), "cn-beijing", newTestClient(t, srv, "cn-beijing"), "0"); err == nil {
		t.Fatal("PageNumber=0 应被拒绝（火山页码从 1 开始）")
	}
}

func TestBusinessErrorOnHTTP200IsNotTreatedAsSuccess(t *testing.T) {
	const errJSON = `{"ResponseMetadata":{"RequestId":"r","Error":{"Code":"InvalidAccessKey","Message":"invalid access key"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(errJSON))
	}))
	defer srv.Close()

	err := newTestClient(t, srv, "cn-beijing").call(context.Background(), "DescribeRegions", nil, nil)
	if err == nil {
		t.Fatal("200 + Error 必须报错")
	}
	if !strings.Contains(err.Error(), "code=InvalidAccessKey") {
		t.Fatalf("错误信息缺少 Code: %v", err)
	}
}

func TestInitClientsRequiresResolvableCredential(t *testing.T) {
	a := NewAdapter(zap.NewNop().Sugar())
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"unknown"}`), []string{"cn-beijing"}); err == nil {
		t.Fatal("未知凭据类型应失败")
	}
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"ak","access_key_secret":"x"}`), []string{"cn-beijing"}); err == nil {
		t.Fatal("缺 access_key_id 应失败")
	}
	if _, err := a.InitClients(context.Background(), testAccount(`{"type":"sts","access_key_id":"a","access_key_secret":"b"}`), []string{"cn-beijing"}); err == nil {
		t.Fatal("sts 缺 session_token 应失败")
	}
	clients, err := a.InitClients(context.Background(), testAccount(testCred), []string{"cn-beijing", "cn-shanghai"})
	if err != nil {
		t.Fatalf("InitClients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("客户端数 = %d", len(clients))
	}
}

func TestMapStatus(t *testing.T) {
	cases := map[string]string{
		"CREATING": "pending", "STARTING": "pending", "REBOOTING": "pending",
		"RUNNING": "active", "STOPPED": "inactive", "STOPPING": "inactive",
		"DELETED": "retired", "EXPIRED": "retired", "": "inactive",
	}
	for in, want := range cases {
		if got := mapStatus(in); got != want {
			t.Errorf("mapStatus(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
