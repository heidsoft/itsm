package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/internal/authz"
)

// 工单流转路由的权限归属守卫。
//
// 实测缺陷（2026-10-20 部署 v1.6.21）：这 16 条路由曾挂在 workflow:read/write 上，
// 而 workflow:* 在权限目录里的语义是「创建、编辑工作流」（含 /bpmn/ai/templates 发布）。
// 后果两头都错——一线坐席与主管不持 workflow:*，工单详情流转面板和批准/接单/拒绝/解决
// 全部 403/2003；反之要给坐席开结单能力，就必须连带授予流程模板发布权。
// 本守卫锁住归位后的口径：路由的动作对象是工单，权限码就必须是 ticket:*。

func TestTicketWorkflowRoutesAreTicketScoped(t *testing.T) {
	cases := []struct {
		method   string
		path     string
		resource string
		action   string
	}{
		{http.MethodGet, "/api/v1/tickets/cc/my", "ticket", "read"},
		{http.MethodGet, "/api/v1/tickets/7/cc", "ticket", "read"},
		{http.MethodGet, "/api/v1/tickets/7/workflow/state", "ticket", "read"},
		{http.MethodGet, "/api/v1/tickets/7/workflow/state-v2", "ticket", "read"},
		{http.MethodGet, "/api/v1/tickets/7/workflow-history", "ticket", "read"},
		{http.MethodGet, "/api/v1/tickets/7/workflow_records", "ticket", "read"},
		{http.MethodPost, "/api/v1/tickets/workflow/accept", "ticket", "assign"},
		{http.MethodPost, "/api/v1/tickets/workflow/forward", "ticket", "assign"},
		{http.MethodPost, "/api/v1/tickets/workflow/cc", "ticket", "assign"},
		{http.MethodPost, "/api/v1/tickets/workflow/approve", "ticket", "approve"},
		{http.MethodPost, "/api/v1/tickets/workflow/reject", "ticket", "approve"},
		{http.MethodPost, "/api/v1/tickets/workflow/resolve", "ticket", "resolve"},
		{http.MethodPost, "/api/v1/tickets/workflow/close", "ticket", "close"},
		{http.MethodPost, "/api/v1/tickets/workflow/reopen", "ticket", "close"},
		{http.MethodPost, "/api/v1/tickets/workflow/withdraw", "ticket", "update"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			perm := getPermissionFromPath(tc.method, tc.path)
			require.NotNil(t, perm, "路由必须被预检表覆盖，否则回退到族级默认策略")
			assert.Equal(t, tc.resource, perm.Resource)
			assert.Equal(t, tc.action, perm.Action)
		})
	}
}

// 处理者角色必须真的拿得到这些码：路由声明对了但角色没授权，仍然是同一条 403 链路。
// 每张表都是一个产品口径，而不是"角色权限快照"：
//   - 一线（agent/manager/it_admin）：接单、审批、解决、关闭全链路；
//   - 二线技术员：处理指派给自己的单据，能接单/解决/关闭，但审批门不在二线（approve 缺席是刻意的）；
//   - 报障人：只能看进度和撤回自己提交的单。
func TestTicketFlowRolesCoverTicketWorkflowActions(t *testing.T) {
	cases := []struct {
		role       string
		allowed    []string
		notAllowed []string
	}{
		{"agent", []string{"read", "assign", "approve", "resolve", "close", "update"}, nil},
		{"manager", []string{"read", "assign", "approve", "resolve", "close", "update"}, nil},
		{"it_admin", []string{"read", "assign", "approve", "resolve", "close", "update"}, nil},
		{"technician", []string{"read", "assign", "resolve", "close", "update"}, []string{"approve"}},
		{"end_user", []string{"read", "update"}, []string{"assign", "approve", "resolve", "close"}},
	}

	for _, tc := range cases {
		perms := rolePermissionsFor(t, tc.role)
		for _, action := range tc.allowed {
			assert.True(t, checkPermissionMatch(perms, "ticket", action),
				"%s 缺少 ticket:%s，工单流转链路会 403", tc.role, action)
		}
		for _, action := range tc.notAllowed {
			assert.False(t, checkPermissionMatch(perms, "ticket", action),
				"%s 不应持 ticket:%s", tc.role, action)
		}
	}
}

func rolePermissionsFor(t *testing.T, role string) []Permission {
	t.Helper()

	codes, ok := authz.RolePermissionDefaults()[role]
	require.True(t, ok, "内置角色 %s 无权限默认集", role)

	perms := make([]Permission, 0, len(codes))
	for _, code := range codes {
		resource, action, ok := authz.SplitPermissionCode(code)
		require.True(t, ok, "权限码 %q 格式非法", code)
		perms = append(perms, Permission{Resource: resource, Action: action})
	}
	return perms
}
