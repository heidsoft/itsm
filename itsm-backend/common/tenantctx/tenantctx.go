// Package tenantctx provides tenant_id propagation via context.Context.
//
// Design goals:
//   1. Give downstream code (services, DB layer, RLS hooks) a single source
//      of truth for "which tenant is this request for".
//   2. Support explicit system-bypass for background jobs, migrations, and
//      MSP cross-tenant operations.
//   3. Coexist with the existing `c.Set("tenant_id", int)` gin convention
//      without breaking any current call site.
//
// This package uses `int` to match the codebase's tenant_id type (Ent schemas
// declare tenant_id as int). The database/rls package works with int64 for
// pg SQL parameter safety; conversion happens at the DB boundary.
package tenantctx

import (
	"context"
	"errors"
)

// key is the unexported type used as context key.
type key int

const (
	tenantKey       key = iota // stores int tenant_id
	systemBypassKey            // stores bool for system-privileged ops
)

// ErrNoTenant is returned when tenant scope is required but missing.
var ErrNoTenant = errors.New("tenantctx: no tenant_id in context and system bypass not set")

// WithTenantID returns a new context carrying tenant_id.
//
// The tenant_id must be a positive integer; zero or negative values are
// silently accepted here (input validation is a middleware concern), but
// the DB layer will reject them via NULLIF(...)::bigint casting.
func WithTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantID returns the tenant_id stored in ctx. The second value is false
// when the key is absent (e.g., background tasks that haven't opted in).
func TenantID(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(tenantKey).(int)
	return v, ok
}

// MustTenantID returns tenant_id, or panics if absent AND system bypass
// is not set. Intended for hot-path assertions inside services that know
// they must be tenant-scoped.
func MustTenantID(ctx context.Context) int {
	if IsSystemBypass(ctx) {
		return 0
	}
	v, ok := TenantID(ctx)
	if !ok {
		panic(ErrNoTenant)
	}
	return v
}

// WithSystemBypass marks ctx as authorized to cross tenant boundaries.
// Only use for: migrations, seed jobs, cron workers, MSP admin ops.
// Every call site should have a code review comment justifying use.
func WithSystemBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemBypassKey, true)
}

// IsSystemBypass reports whether ctx is authorized for cross-tenant ops.
func IsSystemBypass(ctx context.Context) bool {
	v, _ := ctx.Value(systemBypassKey).(bool)
	return v
}
