package middleware

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// precheckMismatchAllowlist 已知「路由声明 与 预检解析」口径错配的台账。
//
// 键格式：`<相对路径> <METHOD> <完整路径>`。每条必须写明理由与去向。
//
// 台账当前为空，但机制保留：新错配必须先登记才能合入，条目一旦不再错配
// 就会被本测试强制清理，防止台账变成永久豁免清单。
//
// ResourceActionMap 的路由条目由 cmd/authz-gen 从路由声明生成
// （rbac_precheck_gen.go），生成即对齐；本测试因此校验生成物与显式回退策略
// 合并后的整体结果。
var precheckMismatchAllowlist = map[string]string{}

// TestRoutePrecheckAlignment 守卫：路由声明的 (resource, action) 必须与路径预检解析结果一致。
//
// 背景
// ----------------------------
// 授权判定是两道闸串联，且顺序固定：
//
//	RBACMiddleware 路径预检(ResourceActionMap)  →  路由级 RequirePermission
//
// 预检先跑且失败即拒。因此只要预检口径与路由声明不一致，就会出现
// 「角色明明有声明的权限、却拿不到路由」——最隐蔽的形态是：
//
//	POST /bpmn/lint  路由声明 bpmn:read，预检却因 /api/v1/bpmn/* 解析成 bpmn:write
//	→ 只有 bpmn:read 的角色被 403
//
// 路由条目现在由 cmd/authz-gen 从路由声明生成，这类错配在生成层面就不会出现；
// 本测试因此校验生成条目与 precheckFallbackPolicies 合并后的结果，
// 防止显式回退策略引入新的口径漂移。
//
// 作用轴说明：预检无匹配（nil）→ 预检空操作，判定完全由路由级决定，不算冲突。
func TestRoutePrecheckAlignment(t *testing.T) {
	scanned, err := ScanDeclaredPermissionRoutes()
	if err != nil {
		t.Fatalf("扫描路由声明失败: %v", err)
	}
	routes := scanned

	// 按 (file, method, fullPath) 聚合声明集合，支持 RequirePermissionAny 多动作
	type key struct {
		file, method, path string
	}
	declared := map[key]map[[2]string]bool{}
	order := []key{}
	for _, r := range routes {
		k := key{r.File, r.Method, r.FullPath}
		if declared[k] == nil {
			declared[k] = map[[2]string]bool{}
			order = append(order, k)
		}
		declared[k][[2]string{r.Resource, r.Action}] = true
	}

	passed := map[string]bool{}
	failed := map[string]bool{}
	var violations []string
	for _, k := range order {
		resolved := getPermissionFromPath(k.method, k.path)
		if resolved == nil {
			continue
		}
		pair := [2]string{resolved.Resource, resolved.Action}
		id := k.file + " " + k.method + " " + k.path
		if declared[k][pair] {
			passed[id] = true
			continue
		}
		failed[id] = true
		if _, ok := precheckMismatchAllowlist[id]; ok {
			continue
		}
		want := make([]string, 0, len(declared[k]))
		for p := range declared[k] {
			want = append(want, p[0]+":"+p[1])
		}
		sort.Strings(want)
		violations = append(violations, fmt.Sprintf(
			"%s  %s %s\n      路由声明 %s，预检解析为 %s:%s",
			k.file, k.method, k.path, strings.Join(want, " | "), resolved.Resource, resolved.Action))
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("以下路由的「声明权限」与「路径预检解析」不一致，"+
			"低权角色会因预检先拒而拿不到已授权的路由：\n  %s\n\n"+
			"路由条目由声明生成（cmd/authz-gen），此失败通常来自：\n"+
			"  ① 生成物过期 → go run ./cmd/authz-gen 重新生成；\n"+
			"  ② precheckFallbackPolicies 显式回退条目与声明冲突 → 修正或删除回退条目。\n"+
			"确属待处理欠账的，加入 precheckMismatchAllowlist 并写明去向。",
			strings.Join(violations, "\n  "))
	}

	// 台账防腐化：条目要么仍有理由（仍错配），要么应被删除
	var stale []string
	for id, reason := range precheckMismatchAllowlist {
		if strings.TrimSpace(reason) == "" {
			stale = append(stale, id+" —— 缺少理由")
		}
		if !failed[id] {
			state := "已不再错配"
			if passed[id] {
				state = "预检已与声明一致"
			} else {
				state = "未命中任何已注册路由（改名/删除后请同步台账）"
			}
			stale = append(stale, id+" —— "+state)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("precheckMismatchAllowlist 存在失效条目：\n  %s", strings.Join(stale, "\n  "))
	}
}
