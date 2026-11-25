package ai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/invocationid"
)

func TestRevalidateApprovedToolUsesStoredPrincipal(t *testing.T) {
	f := newDryRunFixture(t, "approved_principal")
	ctx := authz.WithPrincipal(context.Background(), authz.PrincipalHuman)
	// The invocation has already become approved. Only its fingerprint identifies
	// the agent: the worker/approver context must not turn it into a human.
	registry := authz.NewRegistry(f.client)
	def, err := registry.Lookup(ctx, f.tenantID, "ticket:write")
	require.NoError(t, err)
	// 无信封的存量记录走兼容解码（非空 policyVersion → agent）。
	allowed, _, _, err := f.svc.RevalidateApprovedTool(ctx, f.tenantID, f.userID, "super_admin", "probe_write_ticket", "", def.PolicyVersion())
	require.NoError(t, err)
	require.True(t, allowed)

	// 显式 agent 信封：worker context 说 human 也不改变主体判定。
	agentArgs, err := invocationid.Encode(map[string]interface{}{}, authz.PrincipalAgent)
	require.NoError(t, err)
	allowed, _, _, err = f.svc.RevalidateApprovedTool(authz.WithPrincipal(ctx, authz.PrincipalHuman), f.tenantID, f.userID, "super_admin", "probe_write_ticket", agentArgs, def.PolicyVersion())
	require.NoError(t, err)
	require.True(t, allowed)

	// Human 信封即使 worker context 说 agent 也仍是 human（空 policyVersion）。
	humanArgs, err := invocationid.Encode(map[string]interface{}{}, authz.PrincipalHuman)
	require.NoError(t, err)
	allowed, _, _, err = f.svc.RevalidateApprovedTool(authz.WithPrincipal(ctx, authz.PrincipalAgent), f.tenantID, f.userID, "super_admin", "probe_write_ticket", humanArgs, "")
	require.NoError(t, err)
	require.True(t, allowed)

	// 指纹读不出仍拒绝。
	allowed, _, _, err = f.svc.RevalidateApprovedTool(ctx, f.tenantID, f.userID, "super_admin", "probe_write_ticket", humanArgs, authz.PolicyVersionUnknown)
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
			allowed, _, _, err := f.svc.RevalidateApprovedTool(context.Background(), f.tenantID, f.userID, "super_admin", "probe_write_ticket", "", "")
			require.Error(t, err)
			require.False(t, allowed)
			require.Zero(t, f.catalog.execCount)
		})
	}
}
