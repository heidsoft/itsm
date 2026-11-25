package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/dto"
	"itsm-backend/handlers/mcp"
)

type visibilityFunc func(context.Context, string, string, string, int) bool

func (f visibilityFunc) HasPermission(ctx context.Context, role, resource, action string, tenantID int) bool {
	return f(ctx, role, resource, action, tenantID)
}

// Exercise the production protocol dispatcher, including projection and entitlement.
func TestToolsList_RBACVisibility(t *testing.T) {
	checker := visibilityFunc(func(_ context.Context, role, resource, action string, tenantID int) bool {
		require.Equal(t, 42, tenantID)
		return (role == "reader" || role == "audit_admin") && resource == "ticket" && action == "read"
	})
	for _, tc := range []struct {
		name, role, principal string
		visibility            mcp.VisibilityChecker
		entitled              bool
		want                  []string
	}{
		{"human_allowed", "reader", "human", checker, true, []string{"ticket_read"}},
		{"human_denied", "guest", "human", checker, true, []string{}},
		{"missing_checker", "reader", "human", nil, true, []string{}},
		{"service_read_only", "audit_admin", "service", checker, true, []string{"ticket_read"}},
		{"entitlement_denied", "reader", "human", checker, false, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := mcp.NewService(mcp.Options{
				Catalog:     &fakeCatalog{tools: []dto.ToolDefinition{readOnlyTool("ticket_read", "ticket"), writeTool("ticket_write", "ticket")}},
				Executor:    &fakeExecutor{},
				Visibility:  tc.visibility,
				Entitlement: &fakeEntitlement{allowed: map[string]bool{"ticket_read": tc.entitled, "ticket_write": tc.entitled}},
			})
			require.NoError(t, err)
			resp := svc.Handle(context.Background(), &mcp.Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: mcp.MethodToolsList},
				mcp.Caller{TenantID: 42, Role: tc.role, Principal: tc.principal})
			require.Nil(t, resp.Error)
			names := []string{}
			for _, tool := range resp.Result.(mcp.ListToolsResult).Tools {
				names = append(names, tool.Name)
			}
			require.Equal(t, tc.want, names, "无权限工具不得出现在 tools/list 工具名列表")
		})
	}
}

func TestToolsCall_VisibilityDoesNotAuthorizeExecution(t *testing.T) {
	for _, visibility := range []mcp.VisibilityChecker{nil, visibilityFunc(func(context.Context, string, string, string, int) bool {
		t.Fatal("tools/call must not consult list visibility")
		return false
	})} {
		exec := &fakeExecutor{result: "ok"}
		svc, err := mcp.NewService(mcp.Options{Catalog: &fakeCatalog{}, Executor: exec, Visibility: visibility})
		require.NoError(t, err)
		resp := svc.Handle(context.Background(), &mcp.Request{
			JSONRPC: "2.0", ID: json.RawMessage("1"), Method: mcp.MethodToolsCall,
			Params: json.RawMessage(`{"name":"ticket_read"}`),
		}, mcp.Caller{TenantID: 42, Role: "reader", Principal: "human"})
		require.Nil(t, resp.Error)
		require.True(t, exec.called)
		require.Equal(t, "ticket_read", exec.calledWith)
		require.False(t, resp.Result.(mcp.CallToolResult).IsError)
	}
}
