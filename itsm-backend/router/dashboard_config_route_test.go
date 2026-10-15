package router

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/handlers"

	"github.com/gin-gonic/gin"
)

// errStoreDown 模拟持久化层故障。
var errStoreDown = errors.New("store down")

// bytesReader 把 body 包装成 io.Reader。
func bytesReader(s string) io.Reader { return bytes.NewReader([]byte(s)) }

// contains 判断字符串是否含子串。
func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// fakeDashboardCfgStore 是 DashboardConfigStore 的内存实现。
type fakeDashboardCfgStore struct {
	configs  map[string]map[string]any
	layouts  map[string]map[string]any
	widgets  map[string][]map[string]any
	saveErr  error
	loadErr  error
	saveCall int
}

func newFakeDashboardCfgStore() *fakeDashboardCfgStore {
	return &fakeDashboardCfgStore{
		configs: map[string]map[string]any{},
		layouts: map[string]map[string]any{},
		widgets: map[string][]map[string]any{},
	}
}

func k(tenant, user int) string {
	return string(rune(tenant)) + ":" + string(rune(user))
}

func (f *fakeDashboardCfgStore) LoadDashboardConfig(_ context.Context, tid, uid int) (map[string]any, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.configs[k(tid, uid)], nil
}

func (f *fakeDashboardCfgStore) SaveDashboardConfig(_ context.Context, tid, uid int, cfg map[string]any) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saveCall++
	f.configs[k(tid, uid)] = cfg
	return nil
}

func (f *fakeDashboardCfgStore) LoadDashboardLayout(_ context.Context, tid, uid int) (map[string]any, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.layouts[k(tid, uid)], nil
}

func (f *fakeDashboardCfgStore) SaveDashboardLayout(_ context.Context, tid, uid int, l map[string]any) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saveCall++
	f.layouts[k(tid, uid)] = l
	return nil
}

func (f *fakeDashboardCfgStore) LoadDashboardWidgets(_ context.Context, tid, uid int) ([]map[string]any, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	return f.widgets[k(tid, uid)], nil
}

func (f *fakeDashboardCfgStore) SaveDashboardWidgets(_ context.Context, tid, uid int, w []map[string]any) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saveCall++
	f.widgets[k(tid, uid)] = w
	return nil
}

func (f *fakeDashboardCfgStore) DeleteDashboardWidgets(_ context.Context, tid, uid int) error {
	delete(f.widgets, k(tid, uid))
	return nil
}

// newDashboardTestRouter 装配只含 /dashboard 组的小路由，注入租户与用户上下文。
//
// 关于鉴权：Dashboard 路由上的 middleware.RequirePermission 需要 DB client
// （DBOnly 模式按 client 查角色授权），在无 DB 的单测里必然 401/500。
// 因此这里刻意**不挂**该中间件 —— 本文件验证的是 handler 层的持久化接线与
// 状态码口径，不重复验证 RBAC（那部分由 middleware 包与路由守卫测试负责）。
//
// DashboardHandler 传 nil service：绑定 h 方法的统计端点（stats/kpi/ticket-trend
// 等）在本文件里从不被调用，只是被注册。
func newDashboardTestRouter(store DashboardConfigStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tenant := r.Group("/api/v1/t")
	tenant.Use(func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("user_id", 42)
		c.Set("role", "super_admin")
		// RequirePermission 必须能取到 *ent.Client 键（类型断言用），
		// 但 super_admin 在 AuthorizeResource 里直接短路返回 true，
		// 不会真的拿它去查库 —— 因此 nil 是安全的。
		c.Set("client", (*ent.Client)(nil))
		c.Next()
	})
	SetupDashboardRoutes(tenant, handlers.NewDashboardHandler(nil, nil, nil, nil), nil, store)
	return r
}

// doReq 发请求并返回 recorder。
func doReq(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytesReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestDashboardConfigPersists 验证 POST /dashboard/config 真正落库并可读回。
//
// 修复前该端点只回显请求体（无持久化），表现为"保存成功"实则重启即丢。
func TestDashboardConfigPersists(t *testing.T) {
	store := newFakeDashboardCfgStore()
	r := newDashboardTestRouter(store)

	w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/config", `{"theme":"dark"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST config 期望 200，got %d body=%s", w.Code, w.Body.String())
	}
	if store.saveCall != 1 {
		t.Errorf("配置未落库：saveCall=%d，期望 1", store.saveCall)
	}
	if store.configs[k(7, 42)]["theme"] != "dark" {
		t.Errorf("落库内容不对: %+v", store.configs[k(7, 42)])
	}

	// 读回
	w = doReq(r, http.MethodGet, "/api/v1/t/dashboard/config", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET config 期望 200，got %d", w.Code)
	}
	if !contains(w.Body.String(), "dark") {
		t.Errorf("读回配置缺少已保存值: %s", w.Body.String())
	}
}

// TestDashboardLayoutPersists 验证布局写面落库。
func TestDashboardLayoutPersists(t *testing.T) {
	store := newFakeDashboardCfgStore()
	r := newDashboardTestRouter(store)

	w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/layout", `{"cols":6,"rows":12}`)
	if w.Code != http.StatusOK {
		t.Fatalf("POST layout 期望 200，got %d body=%s", w.Code, w.Body.String())
	}
	if got := store.layouts[k(7, 42)]["cols"]; got != float64(6) {
		t.Errorf("布局未正确落库: cols=%v", got)
	}
}

// TestDashboardWidgetsMergeByID 验证同 id 部件被覆盖而非重复追加。
func TestDashboardWidgetsMergeByID(t *testing.T) {
	store := newFakeDashboardCfgStore()
	r := newDashboardTestRouter(store)

	if w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/widgets", `{"id":"w1","title":"A"}`); w.Code != http.StatusOK {
		t.Fatalf("首次新增部件失败: %d %s", w.Code, w.Body.String())
	}
	if w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/widgets", `{"id":"w2","title":"B"}`); w.Code != http.StatusOK {
		t.Fatalf("新增第二部件失败: %d %s", w.Code, w.Body.String())
	}
	if got := len(store.widgets[k(7, 42)]); got != 2 {
		t.Fatalf("期望 2 个部件，实际 %d", got)
	}

	// 同 id 覆盖
	if w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/widgets", `{"id":"w1","title":"A2"}`); w.Code != http.StatusOK {
		t.Fatalf("覆盖部件失败: %d %s", w.Code, w.Body.String())
	}
	widgets := store.widgets[k(7, 42)]
	if len(widgets) != 2 {
		t.Errorf("同 id 覆盖不应改变部件总数，实际 %d: %+v", len(widgets), widgets)
	}
	for _, wdg := range widgets {
		if wdg["id"] == "w1" && wdg["title"] != "A2" {
			t.Errorf("w1 未被覆盖: %+v", wdg)
		}
	}
}

// TestDashboardWidgetDeleteRemoves 验证删除部件后清单真实变短。
func TestDashboardWidgetDeleteRemoves(t *testing.T) {
	store := newFakeDashboardCfgStore()
	r := newDashboardTestRouter(store)

	doReq(r, http.MethodPost, "/api/v1/t/dashboard/widgets", `{"id":"w1"}`)
	doReq(r, http.MethodPost, "/api/v1/t/dashboard/widgets", `{"id":"w2"}`)

	w := doReq(r, http.MethodDelete, "/api/v1/t/dashboard/widgets/w1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE 期望 200，got %d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.widgets[k(7, 42)]); got != 1 {
		t.Errorf("删除后应剩 1 个部件，实际 %d", got)
	}

	// 删不存在的应 404 而不是静默成功
	if w := doReq(r, http.MethodDelete, "/api/v1/t/dashboard/widgets/nope", ""); w.Code != http.StatusNotFound {
		t.Errorf("删除不存在部件应 404，实际 %d", w.Code)
	}
}

// TestDashboardWriteFailsLoudlyOnStoreError 验证存储故障时写面**显式报错**，
// 而不是回显请求体假装成功。
func TestDashboardWriteFailsLoudlyOnStoreError(t *testing.T) {
	store := newFakeDashboardCfgStore()
	store.saveErr = errStoreDown
	r := newDashboardTestRouter(store)

	w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/config", `{"theme":"dark"}`)
	if w.Code == http.StatusOK {
		t.Errorf("存储故障时不应返回 200（那是假成功），实际 %d body=%s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际 %d", w.Code)
	}
}

// TestDashboardNilStoreReadFallsBackWriteUnavailable 验证 store 未装配时：
// 读回落默认值（可用），写显式 503（而非假成功）。
func TestDashboardNilStoreReadFallsBackWriteUnavailable(t *testing.T) {
	r := newDashboardTestRouter(nil)

	if w := doReq(r, http.MethodGet, "/api/v1/t/dashboard/config", ""); w.Code != http.StatusOK {
		t.Errorf("未装配时读应回落默认值 200，实际 %d", w.Code)
	}
	w := doReq(r, http.MethodPost, "/api/v1/t/dashboard/config", `{"theme":"dark"}`)
	if w.Code == http.StatusOK {
		t.Errorf("未装配时写不应成功，实际 200")
	}
}

// TestDashboardReportEndpointsReturn501 验证报表生成类端点回 501，
// 不再返回 id=0 假对象 / downloadUrl="" / "not generated yet"。
func TestDashboardReportEndpointsReturn501(t *testing.T) {
	r := newDashboardTestRouter(newFakeDashboardCfgStore())

	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/t/dashboard/reports"},
		{http.MethodPost, "/api/v1/t/dashboard/reports/weekly"},
		{http.MethodGet, "/api/v1/t/dashboard/reports/r1/download"},
		{http.MethodPost, "/api/v1/t/dashboard/export"},
	}
	for _, tc := range cases {
		w := doReq(r, tc.method, tc.path, "")
		if w.Code != http.StatusNotImplemented {
			t.Errorf("%s %s 期望 501，实际 %d body=%s", tc.method, tc.path, w.Code, w.Body.String())
		}
		if contains(w.Body.String(), "not generated yet") {
			t.Errorf("%s %s 仍在返回占位文本", tc.method, tc.path)
		}
	}
}

// TestFakeEndpointsRemoved 锁定本轮摘除的虚构端点确实不存在。
// 用反证法：这些端点摘除前会返回 200 假数据，现在应 404。
func TestFakeEndpointsRemoved(t *testing.T) {
	r := newDashboardTestRouter(newFakeDashboardCfgStore())

	removed := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/t/dashboard/metrics/performance"},
		{http.MethodGet, "/api/v1/t/dashboard/metrics/usage"},
		{http.MethodGet, "/api/v1/t/dashboard/charts/bar"},
		{http.MethodGet, "/api/v1/t/dashboard/realtime/active"},
		{http.MethodGet, "/api/v1/t/dashboard/widgets/w1/data"},
		{http.MethodPost, "/api/v1/t/dashboard/widgets/w1/refresh"},
		{http.MethodPut, "/api/v1/t/dashboard/widgets/w1"},
	}
	for _, tc := range removed {
		w := doReq(r, tc.method, tc.path, "")
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s 期望已摘除(404)，实际 %d body=%s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}
