// Package role 角色词表跨层契约测试。
//
// 背景（2026-09-15 GitHub issue 复盘）：角色词表曾漂移成三套互不相交的
// 词汇——domain 常量、ent users.role 枚举、roles 表种子——交集仅 end_user，
// 导致「按角色查审批人恒空」「新建用户无法选角色」「candidateGroups 空转」。
//
// 本文件锁死三条契约，任何一条破坏即 CI 失败：
//  1. domain.All ⊆ users.role 枚举值（ent/schema/user.go）——所有合法角色必须可落库；
//  2. domain.All ⊆ 内置角色种子（pkg/seeder.builtinRoles）——所有合法角色必须在
//     roles 表有实体，MigrateUserRolesBackfill 与 M2M 角色解析才能命中；
//  3. domain.All 与内置种子一一对应（无重复、无遗漏、无额外值）。
package role_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"itsm-backend/domain/role"
	"itsm-backend/pkg/seeder"
)

// entEnumValues 从 ent/schema/user.go 的源码中解析 role 枚举的 Values(...) 列表。
// 直接读源码而非依赖生成代码，保证「schema 是契约源头」这一语义。
func entEnumValues(t *testing.T) map[string]bool {
	t.Helper()
	const schemaPath = "../../ent/schema/user.go"
	f, err := os.Open(schemaPath)
	if err != nil {
		t.Fatalf("打开 schema 失败: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inRoleEnum := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `field.Enum("role")`) {
			inRoleEnum = true
			continue
		}
		if inRoleEnum {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Values(") {
				values := parseValues(trimmed)
				if len(values) == 0 {
					t.Fatalf("解析到 role 枚举行但未提取到值: %q", line)
				}
				return toSet(values)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("扫描 schema 失败: %v", err)
	}
	t.Fatal("未在 ent/schema/user.go 中找到 role 枚举的 Values 定义")
	return nil
}

func parseValues(line string) []string {
	open := strings.Index(line, "(")
	closeIdx := strings.LastIndex(line, ")")
	if open < 0 || closeIdx <= open {
		return nil
	}
	raw := line[open+1 : closeIdx]
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

// TestContract_DomainAllStorableInUserEnum 契约 1：domain.All 每个角色都必须
// 是 users.role 枚举的合法值（可落库）。legacy 值（如 security）允许存在于
// 枚举但不在 domain.All 中。
func TestContract_DomainAllStorableInUserEnum(t *testing.T) {
	enumSet := entEnumValues(t)
	for _, r := range role.All {
		if !enumSet[r] {
			t.Errorf("domain 角色 %q 不在 users.role 枚举中——落库会被拒绝。请在 ent/schema/user.go 的 role 枚举补齐", r)
		}
	}
}

// TestContract_BuiltinSeedCoversDomainAll 契约 2：内置种子角色的 code 集合
// 必须与 domain.All 完全一致（不多不少）。roles 表是 M2M 角色解析的数据源，
// 缺项会让按角色查人退化为空集。
func TestContract_BuiltinSeedCoversDomainAll(t *testing.T) {
	builtin := seeder.BuiltinRoles()
	if len(builtin) != len(role.All) {
		t.Fatalf("builtinRoles 数量=%d 与 domain.All 数量=%d 不一致", len(builtin), len(role.All))
	}
	seen := map[string]bool{}
	for _, r := range builtin {
		if r.Code == "" || r.Name == "" {
			t.Errorf("内置角色种子缺少 code 或 name: %+v", r)
			continue
		}
		if seen[r.Code] {
			t.Errorf("内置角色种子 code 重复: %q", r.Code)
		}
		seen[r.Code] = true
		if !isInAll(r.Code) {
			t.Errorf("内置角色种子 %q 不在 domain.All 中——两处必须同步", r.Code)
		}
	}
	for _, r := range role.All {
		if !seen[r] {
			t.Errorf("domain 角色 %q 缺少内置种子——MigrateUserRolesBackfill 将无法为该角色回填 user_roles 边", r)
		}
	}
}

func isInAll(code string) bool {
	for _, r := range role.All {
		if r == code {
			return true
		}
	}
	return false
}

// TestContract_DTOOneofCoversDomainAll 契约 3：用户 DTO 的 role oneof 词表必须
// 覆盖 domain.All（user 为前端别名，security 为 legacy，允许额外项）。
func TestContract_DTOOneofCoversDomainAll(t *testing.T) {
	f, err := os.Open("../../dto/user_dto.go")
	if err != nil {
		t.Fatalf("打开 DTO 失败: %v", err)
	}
	defer f.Close()

	var oneofs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// 仅校验 json:"role" 字段的 oneof（排除 mspRole 等其他词表）
		if !strings.Contains(line, `json:"role`) || strings.Contains(line, `json:"roleIds`) {
			continue
		}
		for i := strings.Index(line, "oneof="); i >= 0; {
			rest := line[i+len("oneof="):]
			end := strings.Index(rest, `"`)
			if end < 0 {
				break // tag 未闭合，跳过
			}
			oneofs = append(oneofs, rest[:end])
			next := strings.Index(rest[end:], "oneof=")
			if next < 0 {
				break
			}
			i = next
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("扫描 DTO 失败: %v", err)
	}
	if len(oneofs) == 0 {
		t.Fatal("未在 dto/user_dto.go 中找到 role oneof 定义")
	}
	allowed := map[string]bool{"user": true, "security": true} // 别名/legacy
	for _, oneof := range oneofs {
		set := toSet(strings.Fields(oneof))
		for _, r := range role.All {
			if !set[r] && !allowed[r] {
				t.Errorf("DTO oneof 词表缺少 domain 角色 %q（oneof=%s）", r, oneof)
			}
		}
	}
}
