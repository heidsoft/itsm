package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// silentClockStub 模拟不填 CheckedAt 的探针（webhook / console 就是这种实现）。
type silentClockStub struct{}

func (s *silentClockStub) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                "silent-clock-stub",
		Version:             "1.0.0",
		Title:               "不报时间的探针",
		Provider:            "probe",
		Type:                connector.TypeCustom,
		Capabilities:        []connector.Capability{connector.CapHealthCheck},
		RequiredPermissions: []string{"connector.read"},
	}
}

func (s *silentClockStub) Init(_ context.Context, _ connector.Config) error   { return nil }
func (s *silentClockStub) Send(_ context.Context, _ *connector.Message) error { return nil }
func (s *silentClockStub) Close() error                                       { return nil }
func (s *silentClockStub) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}

func newSilentClockHandler(t *testing.T) *Handler {
	t.Helper()
	reg := connector.NewRegistry()
	reg.Register(func() connector.Connector { return &silentClockStub{} })
	mgr := connector.NewManager(reg, zap.NewNop().Sugar())
	if err := mgr.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "silent-clock-stub", Provider: "probe", Enabled: true,
	}); err != nil {
		t.Fatalf("provision silent-clock-stub: %v", err)
	}
	return NewHandler(mgr, reg, marketplace.New(), zap.NewNop().Sugar())
}

// 审计实测：`itsm --json connector health` 的 items[].checkedAt 渲染成
// 0001-01-01T00:00:00Z。接口对"未提供时间"的契约是省略字段，绝不能输出零值时间。
func TestHealth_CheckedAtNeverRendersZeroTime(t *testing.T) {
	h := newSilentClockHandler(t)
	w := callGET(t, h, 1, http.MethodGet, "/api/v1/connectors/health", h.Health)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "0001-01-01") {
		t.Fatalf("checkedAt leaked the Go zero time: %s", body)
	}

	var resp struct {
		Data dto.ConnectorHealthResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 health item, got %d", len(resp.Data.Items))
	}
	if resp.Data.Items[0].CheckedAt == nil || resp.Data.Items[0].CheckedAt.IsZero() {
		t.Fatalf("checkedAt must be a real check time, got %+v", resp.Data.Items[0].CheckedAt)
	}
}

// 同一零值时间也经由 lastCheckedAt 泄漏到市场与生命周期视图。
func TestMarketAndLifecycle_LastCheckedAtNeverRendersZeroTime(t *testing.T) {
	h := newSilentClockHandler(t)
	cases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"market", h.ListMarket},
		{"lifecycle", h.Lifecycle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := callGET(t, h, 1, http.MethodGet, "/api/v1/connectors", tc.fn)
			body := w.Body.String()
			if strings.Contains(body, "0001-01-01") {
				t.Fatalf("lastCheckedAt leaked the Go zero time: %s", body)
			}

			var resp struct {
				Data struct {
					Items []struct {
						Name          string     `json:"name"`
						LastCheckedAt *time.Time `json:"lastCheckedAt"`
					} `json:"items"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v body=%s", err, body)
			}
			var found bool
			for _, it := range resp.Data.Items {
				if it.Name != "silent-clock-stub" {
					continue
				}
				found = true
				if it.LastCheckedAt == nil || it.LastCheckedAt.IsZero() {
					t.Fatalf("lastCheckedAt must be a real check time, got %+v", it.LastCheckedAt)
				}
			}
			if !found {
				t.Fatalf("manifest silent-clock-stub missing from %s", body)
			}
		})
	}
}

// 边界防线：即使健康结果绕过 Manager（例如测试桩或未来的缓存来源），
// 零值时间也必须映射为 nil，而不是指针指向零值。
func TestHealthHelpers_TreatZeroCheckedAtAsUnknown(t *testing.T) {
	zero := map[string]connector.HealthStatus{
		"1/clockless/probe": {OK: true},
	}
	if _, at, _ := healthForManifest(zero, 1, "clockless"); at != nil {
		t.Fatalf("healthForManifest must map zero CheckedAt to nil, got %v", at)
	}
	cfg := connector.Config{TenantID: 1, Name: "clockless", Provider: "probe"}
	if _, at, _ := healthForConfig(zero, cfg); at != nil {
		t.Fatalf("healthForConfig must map zero CheckedAt to nil, got %v", at)
	}
}
