package feishu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestDoJSON_HTTPStatusIsSurfaced 回归：非 2xx 必须报成 HTTP 错误，而不是落到
// JSON 解析错误里。
//
// 原实现不判 resp.StatusCode，网关返回 401（带 HTML 或空体）时报的会是
// 「decode /open-apis/im/v1/messages resp: invalid character '<'」，排查方向
// 从「鉴权失败」被误导成「响应结构变了」。
func TestDoJSON_HTTPStatusIsSurfaced(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/tenant_access_token/internal") {
			_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"t-1","expire":7200}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<html>unauthorized</html>`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "app", "sec", "", "")
	var out struct {
		Code int `json:"code"`
	}
	err := c.doJSON(context.Background(), http.MethodPost, "/open-apis/im/v1/messages", map[string]string{}, &out)
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if !strings.Contains(err.Error(), "http=401") {
		t.Fatalf("error must carry the HTTP status, got: %v", err)
	}
	if strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("error must not be reported as a JSON decode failure: %v", err)
	}
}

// TestGetAppAccessToken_UsesAppTokenEndpoint 回归：OAuth 换码/刷新必须拿
// app_access_token，不能拿 tenant_access_token 顶替（原实现 return c.Token()）。
func TestGetAppAccessToken_UsesAppTokenEndpoint(t *testing.T) {
	var hitTenant, hitApp int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			atomic.AddInt32(&hitApp, 1)
			_, _ = w.Write([]byte(`{"code":0,"app_access_token":"a-1","expire":7200}`))
		case "/open-apis/auth/v3/tenant_access_token/internal":
			atomic.AddInt32(&hitTenant, 1)
			_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"t-1","expire":7200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "app", "sec", "", "")
	tok, err := c.getAppAccessToken(context.Background())
	if err != nil {
		t.Fatalf("getAppAccessToken: %v", err)
	}
	if tok != "a-1" {
		t.Fatalf("expected app access token a-1, got %q", tok)
	}
	if atomic.LoadInt32(&hitApp) != 1 {
		t.Fatalf("expected exactly one app_access_token call, got %d", hitApp)
	}
	if atomic.LoadInt32(&hitTenant) != 0 {
		t.Fatalf("must not fall back to tenant_access_token, got %d calls", hitTenant)
	}
}

// TestToken_ConcurrentRefreshIsSerialized 回归：N 个并发调用在 token 过期时只
// 换一次。租户级 token 是共享凭据，并发各自去换会互相挤掉，表现为偶发 401。
func TestToken_ConcurrentRefreshIsSerialized(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"t-1","expire":7200}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "app", "sec", "", "")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Token(context.Background()); err != nil {
				t.Errorf("Token: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 token fetch for 8 concurrent callers, got %d", got)
	}
}

// TestHTTPHandler_ChunkedBodyDoesNotPanic 回归：回调读取曾写成
// `make([]byte, req.ContentLength)`，chunked 请求 ContentLength=-1 会直接
// panic；且 Read 不保证一次读满。改为限额 ReadAll 后两者都不成立。
func TestHTTPHandler_ChunkedBodyDoesNotPanic(t *testing.T) {
	f := New()
	f.client = NewClient("", "app", "sec", "tok", "")
	// 只验证「读取”这一段不炸、且能拿到完整 body」：签名校验会失败（401），
	// 但失败必须发生在读取之后。
	body := `{"type":"url_verification","challenge":"chal-1","token":"tok"}`
	req := httptest.NewRequest(http.MethodPost, "/feishu/webhook/i-1", strings.NewReader(body))
	req.ContentLength = -1 // chunked
	rec := httptest.NewRecorder()
	f.HTTPHandler(nil).ServeHTTP(rec, req)
	if rec.Body.String() == "" {
		t.Fatal("expected handler to have produced a response")
	}
}
