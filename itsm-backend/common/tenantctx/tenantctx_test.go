package tenantctx

import (
	"context"
	"testing"
)

func TestTenantIDRoundTrip(t *testing.T) {
	ctx := WithTenantID(context.Background(), 42)
	got, ok := TenantID(ctx)
	if !ok || got != 42 {
		t.Fatalf("TenantID roundtrip: got=%d ok=%v, want 42/true", got, ok)
	}
}

func TestTenantIDAbsent(t *testing.T) {
	if _, ok := TenantID(context.Background()); ok {
		t.Fatalf("empty ctx should not have tenant_id")
	}
}

func TestSystemBypass(t *testing.T) {
	ctx := WithSystemBypass(context.Background())
	if !IsSystemBypass(ctx) {
		t.Fatalf("WithSystemBypass should set flag")
	}
	if IsSystemBypass(context.Background()) {
		t.Fatalf("empty ctx should not be bypass")
	}
}

func TestMustTenantIDWithBypass(t *testing.T) {
	ctx := WithSystemBypass(context.Background())
	if got := MustTenantID(ctx); got != 0 {
		t.Fatalf("bypass ctx should return 0, got %d", got)
	}
}

func TestMustTenantIDPanicsWithoutTenant(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when tenant absent and no bypass")
		}
	}()
	MustTenantID(context.Background())
}
