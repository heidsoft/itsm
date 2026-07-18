// Package tenantctx: system-privileged operation guidance.
//
// This file documents when and how to use WithSystemBypass, and provides
// a helper that emits a structured audit log for every bypass call.
//
// # When to use WithSystemBypass
//
//   YES – background jobs that operate across all tenants:
//     - SLA monitors, escalation workers, workflow schedulers
//     - Migration / bootstrap / seed
//     - MSP cross-tenant admin operations
//     - Consumer of internal event bus (no HTTP context)
//
//   NO – anything triggered by a user request. Even if the current user is
//   a "super admin", route through tenantctx.WithTenantID(their tenant).
//
// # How to use
//
//   ctx := tenantctx.SystemContext(ctx, "sla_monitor:tick", "escalate overdue tickets")
//   // ... run tenant-agnostic query ...
//
// Every SystemContext() call emits a zap log entry so it is auditable.
// If SystemContext is used inside an HTTP handler by mistake, code review
// (and a lint rule in future) should catch it.
package tenantctx

import (
	"context"

	"go.uber.org/zap"
)

// SystemContext returns ctx annotated with SystemBypass, and emits an audit
// log. `component` is the code region (e.g. "sla_monitor:tick"), `reason`
// is a short human-readable justification.
//
// Prefer this helper over calling WithSystemBypass directly: the log entry
// gives SRE a way to trace every bypass in production.
func SystemContext(ctx context.Context, component, reason string) context.Context {
	zap.L().Info("rls system bypass granted",
		zap.String("component", component),
		zap.String("reason", reason),
	)
	return WithSystemBypass(ctx)
}
