package ai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/internal/authz"
)

func TestRevalidateApprovedToolUsesStoredPrincipal(t *testing.T) {
	f := newDryRunFixture(t, "approved_principal")
	ctx := authz.WithPrincipal(context.Background(), authz.PrincipalHuman)
	// The invocation has already become approved. Only its fingerprint identifies
	// the agent: the worker/approver context must not turn it into a human.
	registry := authz.NewRegistry(f.client)
	def, err := registry.Lookup(ctx, f.tenantID, "ticket:write")
	require.NoError(t, err)
	allowed, _, err := f.svc.RevalidateApprovedTool(ctx, f.tenantID, f.userID, "super_admin", "probe_write_ticket", def.PolicyVersion())
	require.NoError(t, err)
	require.True(t, allowed)

	// Human snapshots remain human even if the worker context says agent.
	allowed, _, err = f.svc.RevalidateApprovedTool(authz.WithPrincipal(ctx, authz.PrincipalAgent), f.tenantID, f.userID, "super_admin", "probe_write_ticket", "")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, _, err = f.svc.RevalidateApprovedTool(ctx, f.tenantID, f.userID, "super_admin", "probe_write_ticket", authz.PolicyVersionUnknown)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Zero(t, f.catalog.execCount)
}

func TestRevalidateApprovedToolRejectsIncompleteDefinition(t *testing.T) {
	for _, missing := range []string{"tool", "resource", "action"} {
		t.Run(missing, func(t *testing.T) {
			f := newDryRunFixture(t, "approved_missing_"+missing)
			definition := f.catalog.defs["probe_write_ticket"]
			switch missing {
			case "tool":
				delete(f.catalog.defs, "probe_write_ticket")
			case "resource":
				definition.Resource = ""
				f.catalog.defs["probe_write_ticket"] = definition
			case "action":
				definition.Action = ""
				f.catalog.defs["probe_write_ticket"] = definition
			}
			allowed, _, err := f.svc.RevalidateApprovedTool(context.Background(), f.tenantID, f.userID, "super_admin", "probe_write_ticket", "")
			require.Error(t, err)
			require.False(t, allowed)
			require.Zero(t, f.catalog.execCount)
		})
	}
}
