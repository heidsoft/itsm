package seeder

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// TestSeedCarriers_NoOffCatalogPermissionCodes 种子载体权限码守卫（R3）。
//
// 背景：config/seed/seed_data.sql 曾携带 catalog 外的权限码
// （admin:write / change:manage / tenant:manage），一条 psql -f 即可向
// role_permissions 写入清单外码，在 DBOnly 模式下形成双权威表漂移。
// 该文件已删除；本守卫把「种子载体内不得出现清单外权限码」锁成 CI 契约。
//
// 规则：扫描 config/seed/ 下全部文件（不限扩展名——未来任何 seed_data.sql 式
// 再引入都会被命中），凡形如 resource:action 的 token 必须落在
// permissionDefinitions() 清单内（按权限码或 resource:action 对匹配）。
// 基线（2026-10-03）：default.json/demo.json 均不携带权限码，命中数 0；
// 授权一律由内置 Go 清单 authz.BuiltinRolePermissionCodes() 提供。
func TestSeedCarriers_NoOffCatalogPermissionCodes(t *testing.T) {
	// 测试进程 cwd 不保证是仓库根（pkg/seeder 测试历史上吃过的亏）；
	// 用本文件位置锚定 config/seed，不依赖 cwd。
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) 失败，无法定位 config/seed 目录")
	}
	seedDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "seed")

	entries, err := os.ReadDir(seedDir)
	if err != nil {
		t.Fatalf("读取种子目录 %s 失败: %v", seedDir, err)
	}
	if len(entries) == 0 {
		t.Fatalf("种子目录 %s 为空：default.json 是生产 seeder 的必需载体，缺失即部署破坏", seedDir)
	}

	defs := permissionDefinitions()
	allowed := make(map[string]bool, len(defs)*2)
	for _, d := range defs {
		allowed[d.Code] = true
		allowed[d.Resource+":"+d.Action] = true
	}

	// 权限码全小写下划线；大写、数字开头、引号包裹键（"status": "active"）
	// 与时间戳（08:00）均不满足形状，天然不误报。
	tokenRe := regexp.MustCompile(`\b[a-z][a-z0-9_]*:[a-z][a-z0-9_]*\b`)

	var violations int
	err = filepath.WalkDir(seedDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, tok := range tokenRe.FindAllString(string(content), -1) {
			if allowed[tok] {
				continue
			}
			violations++
			t.Errorf("种子载体 %s 含清单外权限码 %q：不在 permissionDefinitions() 清单内，一旦被 psql -f 或播种加载即成双权威表漂移", path, tok)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫描种子目录失败: %v", err)
	}
	if violations == 0 {
		t.Log("基线：config/seed 全部载体 0 个 resource:action token（种子不携带权限码，授权一律走内置 Go 清单）")
	}
}
