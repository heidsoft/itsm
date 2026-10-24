package kubernetes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// credWithServer 造一份指向测试服务的 K8s 凭据体。
func credWithServer(t *testing.T, apiServer string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"type":       "token",
		"token":      "tok-1",
		"api_server": apiServer,
	})
	if err != nil {
		t.Fatalf("marshal cred: %v", err)
	}
	return string(b)
}

func TestProviderIdentity(t *testing.T) {
	a := NewAdapter(nil)
	if a.Provider() != Provider || a.ServiceCode() != ServiceCode {
		t.Fatalf("unexpected identity: %s/%s", a.Provider(), a.ServiceCode())
	}
}

// TestResolveCredential_Defaults 凭据解析：api_server 缺省时回落到集群内地址，
// 且 token 缺失必须报错而不是拿到空凭据去请求。
func TestResolveCredential_Defaults(t *testing.T) {
	cred, err := cloud.ResolveKubernetesCredential(context.Background(), `{"type":"token","token":"t"}`)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cred.Endpoint != "https://kubernetes.default.svc" {
		t.Fatalf("unexpected default endpoint: %s", cred.Endpoint)
	}
	if _, err := cloud.ResolveKubernetesCredential(context.Background(), `{"type":"token"}`); err == nil {
		t.Fatal("expected error when token is missing")
	}
}

// TestDiscoverRegion_PaginationAndTransform 覆盖三段行为：分页参数透传、
// continue 回传、Pod → DiscoveredResource 的字段映射。
func TestDiscoverRegion_PaginationAndTransform(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"metadata": {"continue":"eyJ2IjoiMSJ9"},
			"items": [{
				"metadata": {"name":"web-1","uid":"uid-1","namespace":"prod","creationTimestamp":"2026-10-01T00:00:00Z","labels":{"app":"web"}},
				"spec": {"nodeName":"node-a"},
				"status": {"phase":"Running","podIP":"10.1.1.5","hostIP":"10.0.0.9"}
			}]
		}`))
	}))
	defer srv.Close()

	a := NewAdapter(nil)
	account := &ent.CloudAccount{CredentialRef: credWithServer(t, srv.URL)}
	clients, err := a.InitClients(context.Background(), account, []string{"prod"})
	if err != nil {
		t.Fatalf("InitClients: %v", err)
	}
	client, ok := clients["prod"]
	if !ok {
		t.Fatalf("expected a client for namespace prod, got %v", clients)
	}

	page, err := a.DiscoverRegion(context.Background(), account, "prod", client, "prev-token")
	if err != nil {
		t.Fatalf("DiscoverRegion: %v", err)
	}
	if gotAuth != "Bearer tok-1" {
		t.Fatalf("expected bearer token header, got %q", gotAuth)
	}
	for _, want := range []string{"/api/v1/namespaces/prod/pods", "limit=200", "continue=prev-token"} {
		if !strings.Contains(gotPath, want) {
			t.Fatalf("path %q missing %q", gotPath, want)
		}
	}
	if page.NextToken != "eyJ2IjoiMSJ9" {
		t.Fatalf("expected continue token to be propagated, got %q", page.NextToken)
	}
	if len(page.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(page.Resources))
	}
	r := page.Resources[0]
	// Pod 用 UID 做资源标识：重建后 UID 变化，不会被当成同一条 CI 覆盖。
	if r.ResourceID != "uid-1" || r.ResourceName != "web-1" {
		t.Fatalf("unexpected identity: %+v", r.BaseResource)
	}
	if r.Region != "prod" || r.Zone != "node-a" || r.Status != "Running" {
		t.Fatalf("unexpected mapping: %+v", r.BaseResource)
	}
	if r.CloudServiceCode != ServiceCode {
		t.Fatalf("unexpected service code: %s", r.CloudServiceCode)
	}
	if r.Extra["pod_ip"] != "10.1.1.5" {
		t.Fatalf("unexpected extra: %+v", r.Extra)
	}
}

// TestListRegions_ForbiddenFallsBackToDefault 覆盖：namespace-scoped 的
// ServiceAccount 列不了全集群 namespace（403）时，退化为 default 而不是让整个
// job 失败——RBAC 收紧是常态，403 不该被当成凭据错误。
func TestListRegions_ForbiddenFallsBackToDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"kind":"Status","reason":"Forbidden","message":"namespaces is forbidden"}`))
	}))
	defer srv.Close()

	a := NewAdapter(nil)
	regions, err := a.ListRegions(context.Background(), &ent.CloudAccount{CredentialRef: credWithServer(t, srv.URL)})
	if err != nil {
		t.Fatalf("expected graceful fallback, got %v", err)
	}
	if len(regions) != 1 || regions[0] != "default" {
		t.Fatalf("expected [default], got %v", regions)
	}
}

func TestListRegions_Namespaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"default"}},{"metadata":{"name":"prod"}}]}`))
	}))
	defer srv.Close()
	a := NewAdapter(nil)
	regions, err := a.ListRegions(context.Background(), &ent.CloudAccount{CredentialRef: credWithServer(t, srv.URL)})
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(regions) != 2 || regions[0] != "default" || regions[1] != "prod" {
		t.Fatalf("unexpected regions: %v", regions)
	}
}

// TestGetJSON_StatusErrorCarriesReason 回归：401（token 失效）与 404（namespace
// 不存在）处置完全不同，错误信息必须带上 APIServer 的 reason。
func TestGetJSON_StatusErrorCarriesReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"kind":"Status","reason":"Unauthorized","message":"Unauthorized"}`))
	}))
	defer srv.Close()
	c := &apiClient{baseURL: srv.URL, token: "bad", hc: srv.Client()}
	err := c.getJSON(context.Background(), "/api/v1/namespaces", &namespaceList{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http=401") || !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("error must carry status and reason, got: %v", err)
	}
}
