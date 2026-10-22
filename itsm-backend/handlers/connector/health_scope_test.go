package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 连接器健康面的租户边界回归。
//
// 背景（真实缺陷）：`GET /connectors/health` 曾调用 Manager.HealthCheckAll —— 任何持
// connector.read 的租户都能读到**全部租户**连接器的名字/provider/在线状态，并且能借一次
// 请求让服务端去拨别家租户的 SMTP / 飞书 / 钉钉端点（跨租户连通性探针）。同一缺陷也存在于
// /connectors、/connectors/configs、/connectors/lifecycle 与 Provision 的响应里。
// 已写好的 TenantHealth / HealthCheckTenant 当时从未挂到路由上。
//
// 本文件用「计数器桩连接器」做负证明：租户 1 发请求时，租户 2 的实例**一次都不能被拨号**。

type healthProbeStub struct {
	mu     *sync.Mutex
	calls  *map[int]int
	tenant int
}

func (s *healthProbeStub) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                "health-probe-stub",
		Version:             "1.0.0",
		Title:               "健康探针桩",
		Provider:            "probe",
		Type:                connector.TypeCustom,
		Capabilities:        []connector.Capability{connector.CapHealthCheck},
		RequiredPermissions: []string{"connector.read"},
	}
}

func (s *healthProbeStub) Init(_ context.Context, cfg connector.Config) error {
	s.tenant = cfg.TenantID
	return nil
}

func (s *healthProbeStub) Send(_ context.Context, _ *connector.Message) error { return nil }

func (s *healthProbeStub) HealthCheck(_ context.Context) connector.HealthStatus {
	s.mu.Lock()
	(*s.calls)[s.tenant]++
	s.mu.Unlock()
	// 故意把租户号写进 message，用来确认响应里没有夹带别家的体检结论
	return connector.HealthStatus{
		OK:        s.tenant == 1,
		Message:   fmt.Sprintf("probe-tenant-%d", s.tenant),
		CheckedAt: time.Now(),
	}
}

func (s *healthProbeStub) Close() error { return nil }

// newTwoTenantFixture 装配两个租户各一个同名连接器实例，返回 handler 与拨号计数。
func newTwoTenantFixture(t *testing.T) (*Handler, *map[int]int) {
	t.Helper()
	calls := map[int]int{}
	mu := &sync.Mutex{}

	reg := connector.NewRegistry()
	reg.Register(func() connector.Connector {
		return &healthProbeStub{mu: mu, calls: &calls}
	})

	mgr := connector.NewManager(reg, zap.NewNop().Sugar())
	for _, tenant := range []int{1, 2} {
		if err := mgr.Provision(context.Background(), connector.Config{
			TenantID: tenant,
			Name:     "health-probe-stub",
			Provider: "probe",
			Enabled:  true,
			Settings: map[string]interface{}{},
		}); err != nil {
			t.Fatalf("provision tenant %d: %v", tenant, err)
		}
	}
	return NewHandler(mgr, reg, marketplace.New(), zap.NewNop().Sugar()), &calls
}

func callGET(t *testing.T, h *Handler, tenantID int, method string, path string, fn func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Set("tenant_id", tenantID)
	c.Set("user_id", 7)
	fn(c)
	return w
}

func TestHealth_OnlyProbesCurrentTenant(t *testing.T) {
	h, calls := newTwoTenantFixture(t)

	w := callGET(t, h, 1, http.MethodGet, "/api/v1/connectors/health", h.Health)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	// 负证明核心：租户 2 的实例没有被拨号
	if got := (*calls)[2]; got != 0 {
		t.Fatalf("tenant 2 connector must not be probed by tenant 1 request, got %d health checks", got)
	}
	if got := (*calls)[1]; got != 1 {
		t.Fatalf("expected exactly 1 health check for tenant 1, got %d", got)
	}
	if strings.Contains(body, "probe-tenant-2") {
		t.Fatalf("response leaked another tenant's health result: %s", body)
	}

	var resp struct {
		Code int                         `json:"code"`
		Data dto.ConnectorHealthResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, body)
	}
	if resp.Data.TenantID != 1 {
		t.Fatalf("expected tenantId=1, got %d", resp.Data.TenantID)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected exactly 1 item, got %d (%s)", len(resp.Data.Items), body)
	}
	item := resp.Data.Items[0]
	if item.Name != "health-probe-stub" || item.Provider != "probe" {
		t.Fatalf("unexpected item identity: %+v", item)
	}
	if !item.OK {
		t.Fatalf("expected tenant 1 instance healthy, got %+v", item)
	}
	// 响应不应再出现 "1/health-probe-stub/probe" 这类把租户号编进键的形态
	if strings.Contains(body, "1/health-probe-stub") {
		t.Fatalf("tenant id must not be baked into item keys: %s", body)
	}
}

// 同一缺陷在另外三个读面上同样存在，逐个锁住「只拨本租户」。
func TestConnectorReadEndpoints_OnlyProbeCurrentTenant(t *testing.T) {
	cases := []struct {
		name string
		path string
		fn   func(h *Handler) func(*gin.Context)
	}{
		{"market", "/api/v1/connectors", func(h *Handler) func(*gin.Context) { return h.ListMarket }},
		{"configs", "/api/v1/connectors/configs", func(h *Handler) func(*gin.Context) { return h.ListConfigs }},
		{"lifecycle", "/api/v1/connectors/lifecycle", func(h *Handler) func(*gin.Context) { return h.Lifecycle }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, calls := newTwoTenantFixture(t)
			w := callGET(t, h, 1, http.MethodGet, tc.path, tc.fn(h))
			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
			}
			if got := (*calls)[2]; got != 0 {
				t.Fatalf("%s probed tenant 2 %d times", tc.name, got)
			}
			if strings.Contains(w.Body.String(), "probe-tenant-2") {
				t.Fatalf("%s leaked tenant 2 health: %s", tc.name, w.Body.String())
			}
		})
	}
}

// 响应形状是前端取键的依据：items 必须带 name/provider，且排序稳定（便于快照/对比）。
func TestHealth_ItemsSortedAndIdentifiable(t *testing.T) {
	h, _ := newTwoTenantFixture(t)
	w := callGET(t, h, 1, http.MethodGet, "/api/v1/connectors/health", h.Health)

	var resp struct {
		Data dto.ConnectorHealthResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data.Items) < 2 {
		// 只有一个实例时也要能取到 name/provider；这里只校验字段非空
		for _, it := range resp.Data.Items {
			if it.Name == "" || it.Provider == "" {
				t.Fatalf("item missing identity fields: %+v", it)
			}
		}
		return
	}
	for i := 1; i < len(resp.Data.Items); i++ {
		prev, cur := resp.Data.Items[i-1], resp.Data.Items[i]
		if prev.Name > cur.Name || (prev.Name == cur.Name && prev.Provider > cur.Provider) {
			t.Fatalf("items not sorted: %+v then %+v", prev, cur)
		}
	}
}
