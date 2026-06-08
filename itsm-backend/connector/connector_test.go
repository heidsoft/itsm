package connector

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

// fakeConnector 测试用连接器
type fakeConnector struct {
	cfg      Config
	sent     int32
}

func (f *fakeConnector) Manifest() Manifest {
	return Manifest{Name: "fake", Title: "Fake", Type: TypeCustom, Capabilities: []Capability{CapSendMessage}}
}
func (f *fakeConnector) Init(_ context.Context, cfg Config) error { f.cfg = cfg; return nil }
func (f *fakeConnector) Send(_ context.Context, _ *Message) error {
	atomic.AddInt32(&f.sent, 1)
	return nil
}
func (f *fakeConnector) HealthCheck(_ context.Context) HealthStatus { return HealthStatus{OK: true} }
func (f *fakeConnector) Close() error                                { return nil }

type errorConnector struct{}

func (e *errorConnector) Manifest() Manifest {
	return Manifest{Name: "error-c", Title: "Err", Type: TypeCustom}
}
func (e *errorConnector) Init(_ context.Context, _ Config) error  { return nil }
func (e *errorConnector) Send(_ context.Context, _ *Message) error {
	return errors.New("injected")
}
func (e *errorConnector) HealthCheck(_ context.Context) HealthStatus {
	return HealthStatus{OK: false, Message: "always failing"}
}
func (e *errorConnector) Close() error { return nil }

func TestRegistry_RegisterAndList(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &fakeConnector{} })
	got, ok := r.Get("fake")
	if !ok {
		t.Fatal("expected fake to be registered")
	}
	if got().Manifest().Name != "fake" {
		t.Fatalf("unexpected manifest: %+v", got().Manifest())
	}
	if names := r.Names(); len(names) != 1 || names[0] != "fake" {
		t.Fatalf("Names() = %v", names)
	}
}

func TestRegistry_DuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &fakeConnector{} })
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register(func() Connector { return &fakeConnector{} })
}

func TestManager_SendError(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &errorConnector{} })
	mgr := NewManager(r, nil)
	cfg := Config{TenantID: 9, Name: "error-c", Enabled: true}
	if err := mgr.Provision(context.Background(), cfg); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := mgr.Send(context.Background(), 9, "failing", &Message{Channel: "c"}); err == nil {
		t.Fatal("expected send error")
	}
}

func TestManager_ProvisionAndSend(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &fakeConnector{} })
	mgr := NewManager(r, nil)
	cfg := Config{TenantID: 7, Name: "fake", Enabled: true, Credentials: map[string]string{}}
	if err := mgr.Provision(context.Background(), cfg); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := mgr.Send(context.Background(), 7, "fake", &Message{Channel: "c", Content: "hi"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// 另一个租户取不到
	if _, ok := mgr.Get(8, "fake"); ok {
		t.Fatal("expected no instance for tenant 8")
	}
}

func TestManager_Revoke(t *testing.T) {
	r := NewRegistry()
	r.Register(func() Connector { return &fakeConnector{} })
	mgr := NewManager(r, nil)
	cfg := Config{TenantID: 1, Name: "fake", Enabled: true}
	_ = mgr.Provision(context.Background(), cfg)
	mgr.Revoke(cfg)
	if _, ok := mgr.Get(1, "fake"); ok {
		t.Fatal("expected revoked instance to be gone")
	}
}

func TestManifest_RequiredFields(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty manifest name")
		}
	}()
	// 匿名 connector 故意不设置 Name
	NewRegistry().Register(func() Connector { return &emptyManifester{} })
}

type emptyManifester struct{ fakeConnector }

func (e *emptyManifester) Manifest() Manifest { return Manifest{} }


func TestRouter_Dedup(t *testing.T) {
	r := NewRouter(nil)
	var n int32
	r.Register(func(_ context.Context, _ *InboundMessage) error {
		// 累加
		_ = atomic.AddInt32(&n, 1)
		return nil
	})
	for i := 0; i < 3; i++ {
		_ = r.Dispatch(context.Background(), &InboundMessage{MessageID: "m1", Channel: "c1"})
	}
	if atomic.LoadInt32(&n) != 1 {
		t.Fatalf("expected handler called once due to dedup, got %d", n)
	}
}
