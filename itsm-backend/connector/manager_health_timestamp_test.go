package connector

import (
	"context"
	"testing"
	"time"
)

// 真实缺陷：webhook / console 的 HealthCheck 不填 CheckedAt，API 直接把 Go 零值
// 时间当作"最后检查时间"返回（0001-01-01T00:00:00Z）。探针可以不报时间，但
// "刚刚检查过"这一事实由 Manager 掌握，必须在统一出口落戳。
func TestHealthCheckTenant_StampsCheckedAtWhenConnectorOmitsIt(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &fakeConnector{} })
	mgr := NewManager(r, nil)
	if err := mgr.Provision(context.Background(), Config{
		TenantID: 3, Name: "fake", Provider: "fake", Enabled: true,
	}); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	before := time.Now()
	statuses := mgr.HealthCheckTenant(context.Background(), 3)
	status, ok := statuses["3/fake/fake"]
	if !ok {
		t.Fatalf("missing health entry for tenant 3, got %v", statuses)
	}
	if status.CheckedAt.IsZero() {
		t.Fatal("manager must stamp CheckedAt when the connector leaves it zero")
	}
	if status.CheckedAt.Before(before) {
		t.Fatalf("CheckedAt %v predates the check that started at %v", status.CheckedAt, before)
	}
}
