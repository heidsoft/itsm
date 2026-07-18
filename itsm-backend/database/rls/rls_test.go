package rls

import (
	"context"
	"testing"
)

func TestWithTenantAndRoundTrip(t *testing.T) {
	ctx := WithTenant(context.Background(), 42)
	tid, ok := TenantFromContext(ctx)
	if !ok || tid != 42 {
		t.Fatalf("expected tenant 42, got %d ok=%v", tid, ok)
	}
}

func TestTenantMissing(t *testing.T) {
	_, ok := TenantFromContext(context.Background())
	if ok {
		t.Fatal("expected missing tenant")
	}
}

func TestSystemBypass(t *testing.T) {
	if IsSystemBypass(context.Background()) {
		t.Fatal("plain context should not be bypass")
	}
	ctx := WithSystemBypass(context.Background())
	if !IsSystemBypass(ctx) {
		t.Fatal("bypass context should be flagged")
	}
	// bypass and tenant can coexist; DB layer decides which to honor
	ctx = WithTenant(ctx, 1)
	if !IsSystemBypass(ctx) {
		t.Fatal("bypass should survive WithTenant chaining")
	}
}
