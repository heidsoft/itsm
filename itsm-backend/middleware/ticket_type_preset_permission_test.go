package middleware

import (
	"fmt"
	"sort"
	"testing"
)

// TestTicketTypePresetRoutePermissions 锁定工单类型预设链路的真实路由权限口径。
//
// 2026-10-24 本地 prod 栈实测：只有 ticket_type:read 的坐席点「从预设安装」时，
// GET /api/v1/ticket-type-presets 挂在 manage 上直接 403（code 2003），而 authz catalog
// 对 ticket_type:read 的定义就是「查看工单类型和预设」。预设是产品内置模板、不含租户数据，
// 因此读口径回归 read；安装仍然单独按 install_preset 授权。
func TestTicketTypePresetRoutePermissions(t *testing.T) {
	declared, err := ScanDeclaredPermissionRoutes()
	if err != nil {
		t.Fatalf("扫描路由声明失败: %v", err)
	}

	byRoute := map[string]map[[2]string]bool{}
	for _, r := range declared {
		key := fmt.Sprintf("%s %s", r.Method, r.FullPath)
		if byRoute[key] == nil {
			byRoute[key] = map[[2]string]bool{}
		}
		byRoute[key][[2]string{r.Resource, r.Action}] = true
	}

	want := map[string][2]string{
		"GET /api/v1/ticket-type-presets":            {"ticket_type", "read"},
		"POST /api/v1/ticket-type-presets/*/install": {"ticket_type", "install_preset"},
	}
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		got := byRoute[key]
		if len(got) == 0 {
			t.Errorf("%s 未声明 RequirePermission：DBOnly 下非超管角色恒 403", key)
			continue
		}
		expected := want[key]
		if !got[expected] {
			t.Errorf("%s 权限口径漂移：期望 %s:%s，实际声明 %v", key, expected[0], expected[1], got)
		}
	}
}
