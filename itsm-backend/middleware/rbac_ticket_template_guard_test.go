package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 工单模板配置面的路由口径与角色可达性守卫。
//
// 实测缺陷（2026-10-21，prod v1.6.21）：模板页对没有读码的角色返回 403/2003，页面却渲染成
// 「暂无模板」+ 通用错误提示，把"你没权限"说成"系统里没有模板"；同时 it_admin 的职责包含
// 配置模板，基线却只有 ticket_template:read，保存必然 403。

func TestTicketTemplateRoutesPermissionCodes(t *testing.T) {
	cases := []struct {
		method   string
		path     string
		resource string
		action   string
	}{
		{http.MethodGet, "/api/v1/tickets/templates", "ticket_template", "read"},
		{http.MethodGet, "/api/v1/tickets/templates/categories", "ticket_template", "read"},
		{http.MethodGet, "/api/v1/tickets/templates/7", "ticket_template", "read"},
		{http.MethodPost, "/api/v1/tickets/templates", "ticket_template", "create"},
		{http.MethodPost, "/api/v1/tickets/templates/7/copy", "ticket_template", "create"},
		{http.MethodPut, "/api/v1/tickets/templates/7", "ticket_template", "update"},
		{http.MethodPatch, "/api/v1/tickets/templates/7/status", "ticket_template", "update"},
		{http.MethodDelete, "/api/v1/tickets/templates/7", "ticket_template", "delete"},
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

// 角色可达性：路由声明对了但角色没授权，仍然是同一条 403 链路。
// 每张表都是产品口径——
//   - 配置面（admin/it_admin/service_catalog_admin）：看、建、改；admin 的 create/update
//     由 write⇒同义细分动作补齐，这里锁住补齐确实生效而不是只在注释里；
//   - 删除是破坏性动作，只留给租户管理员与目录管理员，it_admin 不持；
//   - 一线（agent）只读；报障人与主管不持读码，菜单与接口同一口径。
func TestTicketTemplateRolesCoverRouteActions(t *testing.T) {
	cases := []struct {
		role       string
		allowed    []string
		notAllowed []string
	}{
		{"admin", []string{"read", "create", "update", "delete"}, nil},
		{"it_admin", []string{"read", "create", "update"}, []string{"delete"}},
		{"service_catalog_admin", []string{"read", "create", "update", "delete"}, nil},
		{"agent", []string{"read"}, []string{"create", "update", "delete"}},
		{"end_user", nil, []string{"read", "create", "update", "delete"}},
		{"manager", nil, []string{"read", "create", "update", "delete"}},
	}

	for _, tc := range cases {
		perms := rolePermissionsFor(t, tc.role)
		for _, action := range tc.allowed {
			assert.True(t, checkPermissionMatch(perms, "ticket_template", action),
				"%s 缺少 ticket_template:%s，模板配置会 403", tc.role, action)
		}
		for _, action := range tc.notAllowed {
			assert.False(t, checkPermissionMatch(perms, "ticket_template", action),
				"%s 不应持 ticket_template:%s", tc.role, action)
		}
	}
}
