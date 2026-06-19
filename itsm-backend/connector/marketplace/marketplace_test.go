package marketplace

import (
	"context"
	"testing"

	"itsm-backend/connector"
)

type fakeC struct{}

func (f *fakeC) Manifest() connector.Manifest {
	return connector.Manifest{Name: "fake", Title: "Fake", Type: connector.TypeCustom}
}
func (f *fakeC) Init(_ context.Context, _ connector.Config) error   { return nil }
func (f *fakeC) Send(_ context.Context, _ *connector.Message) error { return nil }
func (f *fakeC) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (f *fakeC) Close() error { return nil }

func TestSearch(t *testing.T) {
	r := connector.NewRegistry()
	r.Register(func() connector.Connector { return &fakeC{} })
	m := New()
	_ = m.Refresh(r)
	if got := m.Search("", ""); len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got := m.Search("fak", ""); len(got) != 1 {
		t.Fatalf("search should match")
	}
	if got := m.Search("nothing", ""); len(got) != 0 {
		t.Fatalf("search should return 0")
	}
}
