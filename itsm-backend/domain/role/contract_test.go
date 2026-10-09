// Package role 角色词表跨层契约测试。
//
// 背景（2026-09-15 GitHub issue 复盘）：角色词表曾漂移成三套互不相交的
// 词汇——domain 常量、ent users.role 枚举、roles 表种子——交集仅 end_user，
// 导致「按角色查审批人恒空」「新建用户无法选角色」「candidateGroups 空转」。
//
// 2026-10-08 收紧（角色模型重设收尾）：17 个 legacy/岗位型角色已从头到尾删除，
// 所以守卫从「domain.All ⊆ 各层」升级为集合相等，并增加退役码零出现扫描：
//  1. users.role 枚举值 == domain.All（封闭词表，无额外值可落库）；
//     1b. Tier 2/3 叠加角色不得进入 users.role 枚举；
//  2. 内置主角色种子 == domain.All；叠加角色种子 == Practice ∪ MSP；
//  3. internal/authz 权限绑定键 == All \ {super_admin} ∪ Practice，
//     兜底默认集 == All ∪ Practice ∪ MSP；
//  4. DTO role oneof == domain.All ∪ {user}（user 是前端别名）；
//  5. domain.Retired 的码在第 1~4 层、内置 BPMN 模板的角色指派位、内置种子 JSON
//     与前端主角色词表中零出现——重新引入即 CI 失败。
package role_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"itsm-backend/domain/role"
	"itsm-backend/internal/authz"
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

// dtoRoleOneofValues 解析 dto/user_dto.go 中 json:"role" 字段的 oneof 词表。
// 返回每个 oneof 的值集合（多个 DTO 结构体各自声明一份，全部都要校验）。
func dtoRoleOneofValues(t *testing.T) []map[string]bool {
	t.Helper()
	f, err := os.Open("../../dto/user_dto.go")
	if err != nil {
		t.Fatalf("打开 DTO 失败: %v", err)
	}
	defer f.Close()

	var sets []map[string]bool
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
			sets = append(sets, toSet(strings.Fields(rest[:end])))
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
	if len(sets) == 0 {
		t.Fatal("未在 dto/user_dto.go 中找到 role oneof 定义")
	}
	return sets
}

// assertExactSet 断言 actual 与 expected 完全相等，并把差异按方向分别报告，
// 使「漏值」与「多值」在失败信息里一眼可分。
func assertExactSet(t *testing.T, label string, actual map[string]bool, expected []string) {
	t.Helper()
	for _, e := range expected {
		if !actual[e] {
			t.Errorf("%s 缺少 %q", label, e)
		}
	}
	extra := make([]string, 0)
	for a := range actual {
		if !contains(expected, a) {
			extra = append(extra, a)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("%s 含词表外的值 %v——domain/role 是唯一词表来源，新增角色必须改 domain/role 并同步所有层", label, extra)
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// assertNoRetired 断 values 里没有退役角色码。
func assertNoRetired(t *testing.T, label string, values map[string]bool) {
	t.Helper()
	for v := range values {
		if role.IsRetired(v) {
			t.Errorf("%s 含退役角色 %q——2026-10-07 角色模型重设已删除该值，禁止重新引入（存量数据见 migrations/20261008_retire_legacy_roles.sql）", label, v)
		}
	}
}

// TestContract_DomainAllStorableInUserEnum 契约 1：users.role 枚举必须恰好等于
// domain.All。多出的值意味着存在「可落库但不被任何一层承认」的角色（legacy
// security 正是这样长期存活的）；缺失则 domain 角色无法落库。
func TestContract_DomainAllStorableInUserEnum(t *testing.T) {
	enumSet := entEnumValues(t)
	assertExactSet(t, "users.role 枚举", enumSet, role.All)
	assertNoRetired(t, "users.role 枚举", enumSet)
}

// TestContract_PracticeAndMspNotStorableInUserEnum 契约 1b：Tier 2/3 角色只能通过
// user_roles 边叠加，不得进入 users.role 单字段枚举——否则同一角色出现两条权威路径。
func TestContract_PracticeAndMspNotStorableInUserEnum(t *testing.T) {
	enumSet := entEnumValues(t)
	for _, code := range append(append([]string{}, role.Practice...), role.MSP...) {
		if enumSet[code] {
			t.Errorf("Tier 2/3 角色 %q 出现在 users.role 枚举中——叠加型角色只应存在于 roles 表与 user_roles 边", code)
		}
	}
}

// TestContract_BuiltinSeedCoversDomainAll 契约 2：内置主角色种子必须与 domain.All
// 一一对应（无重复、无遗漏、无额外值）。roles 表是 M2M 角色解析的数据源，
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
	}
	assertExactSet(t, "内置主角色种子", seen, role.All)
	assertNoRetired(t, "内置主角色种子", seen)
	for _, r := range role.All {
		if !seen[r] {
			t.Errorf("domain 角色 %q 缺少内置种子——MigrateUserRolesBackfill 将无法为该角色回填 user_roles 边", r)
		}
	}
}

// TestContract_PracticeSeedCoversOverlayRoles 契约 2b：叠加型角色种子（PracticeRoles）
// 必须恰好覆盖 domain.Practice ∪ domain.MSP，否则这些角色在 roles 表无实体、
// 管理员无法分配、按角色查审批人恒空。
func TestContract_PracticeSeedCoversOverlayRoles(t *testing.T) {
	overlay := append(append([]string{}, role.Practice...), role.MSP...)
	seen := map[string]bool{}
	for _, r := range seeder.PracticeRoles() {
		if r.Code == "" || r.Name == "" {
			t.Errorf("叠加角色种子缺少 code 或 name: %+v", r)
			continue
		}
		if seen[r.Code] {
			t.Errorf("叠加角色种子 code 重复: %q", r.Code)
		}
		seen[r.Code] = true
	}
	assertExactSet(t, "叠加角色种子", seen, overlay)
	assertNoRetired(t, "叠加角色种子", seen)
}

// TestContract_AuthzRoleKeysMatchVocabulary 契约 3：internal/authz 的角色键集合
// 必须与词表分层一致——权限绑定权威源覆盖 All \ {super_admin} ∪ Practice
// （超管走 Login ["*"] 旁路，播种侧不需要条目），兜底默认集在此基础上
// 再覆盖 MSP 与 super_admin 通配。词表外的键即「兜底表养幽灵角色」的入口。
func TestContract_AuthzRoleKeysMatchVocabulary(t *testing.T) {
	binding := authz.BuiltinRolePermissionCodes()
	bindingKeys := map[string]bool{}
	for code := range binding {
		bindingKeys[code] = true
	}
	bindingExpected := make([]string, 0, len(role.All)+len(role.Practice))
	for _, code := range role.All {
		if code == role.SuperAdmin {
			continue // 通配旁路，见 defaults.go
		}
		bindingExpected = append(bindingExpected, code)
	}
	bindingExpected = append(bindingExpected, role.Practice...)
	assertExactSet(t, "authz 权限绑定角色键", bindingKeys, bindingExpected)
	assertNoRetired(t, "authz 权限绑定角色键", bindingKeys)

	defaults := authz.RolePermissionDefaults()
	defaultKeys := map[string]bool{}
	for code := range defaults {
		defaultKeys[code] = true
	}
	assertExactSet(t, "authz 兜底默认角色键", defaultKeys,
		append(append(append([]string{}, role.All...), role.Practice...), role.MSP...))
	assertNoRetired(t, "authz 兜底默认角色键", defaultKeys)

	for code, codes := range defaults {
		if len(codes) == 0 {
			t.Errorf("authz 角色 %q 权限码为空——兜底态会让该用户任何请求都 403", code)
		}
	}
}

// TestContract_DTOOneofCoversDomainAll 契约 4：用户 DTO 的 role oneof 必须恰好等于
// domain.All ∪ {user}（user 是前端别名，服务端归一为 end_user）。
// 词表外的值等于给 legacy 角色开了写入口。
func TestContract_DTOOneofCoversDomainAll(t *testing.T) {
	allowed := map[string]bool{"user": true} // 前端别名
	primary := append([]string{}, role.All...)
	sort.Strings(primary)
	for _, set := range dtoRoleOneofValues(t) {
		for _, r := range role.All {
			if !set[r] {
				t.Errorf("DTO oneof 词表缺少 domain 角色 %q", r)
			}
		}
		extra := make([]string, 0)
		for v := range set {
			if !contains(primary, v) && !allowed[v] {
				extra = append(extra, v)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("DTO oneof 词表含词表外的值 %v", extra)
		}
		assertNoRetired(t, "DTO role oneof", set)
	}
}

// seedJSONRoleCodes 解析内置种子 JSON 的 roles 段 code 列表。
// 该段在默认清单里为空（主角色/叠加角色都由 Go 词表提供），保留守卫是为了
// 有人往里加角色时不会带进退役码。
func seedJSONRoleCodes(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("../../config/seed/default.json")
	if err != nil {
		t.Fatalf("读取内置种子 JSON 失败: %v", err)
	}
	var doc struct {
		Roles []struct {
			Code string `json:"code"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析内置种子 JSON 失败: %v", err)
	}
	set := map[string]bool{}
	for _, r := range doc.Roles {
		if r.Code != "" {
			set[r.Code] = true
		}
	}
	return set
}

// bpmnRoleValues 收集 BPMN 模板里被引擎/审批解析器当作角色使用的取值：
// camunda 的 assignee/candidateGroups/candidateUsers 属性，以及 metaData 的
// assignee_type/notify_roles/target_role/participant_roles/assignee_value。
// 其余 metaData（action、expert_type 等）是描述性文本，不属于角色词表——
// 与 migrations/20261008_retire_legacy_roles.sql 对已部署定义采用的边界一致。
var bpmnRoleValues = regexp.MustCompile(
	`camunda:(?:assignee|candidateGroups|candidateUsers)="([^"]*)"` +
		`|<bpmn:metaData name="(?:assignee_type|notify_roles|target_role|participant_roles|assignee_value)">([^<]*)</bpmn:metaData>`)

// TestContract_RetiredCodesHaveZeroHits 契约 5：退役码在任何词表承载面都不得出现。
// 覆盖 authz/DTO/枚举/种子（Go 与 JSON）在前面各契约已单独断言，这里补齐两个
// 容易被忽略的承载面：内置 BPMN 模板的角色指派位，以及前端主角色词表。
func TestContract_RetiredCodesHaveZeroHits(t *testing.T) {
	for _, pattern := range []string{
		"../../service/bpmn/*.bpmn",
		"../../pkg/seeder/templates/*.bpmn",
	} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("展开 %s 失败: %v", pattern, err)
		}
		if len(files) == 0 {
			t.Fatalf("BPMN 模板通配 %s 未匹配到文件——守卫本身失效", pattern)
		}
		for _, file := range files {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("读取 %s 失败: %v", file, err)
			}
			for _, m := range bpmnRoleValues.FindAllStringSubmatch(string(raw), -1) {
				value := m[1]
				if value == "" {
					value = m[2]
				}
				for _, token := range strings.FieldsFunc(value, func(r rune) bool {
					return r == ',' || r == ' ' || r == '\n' || r == '\t'
				}) {
					if role.IsRetired(token) {
						t.Errorf("%s 的角色指派位仍使用退役码 %q（值=%q）——流程任务会解析不到人",
							filepath.Base(file), token, value)
					}
				}
			}
		}
	}

	assertNoRetired(t, "内置种子 JSON roles 段", seedJSONRoleCodes(t))

	// 前端词表：文件不存在时（如仅构建后端）跳过，不静默通过。
	const frontendPath = "../../../itsm-frontend/src/lib/api/user-api.ts"
	raw, err := os.ReadFile(frontendPath)
	if err != nil {
		t.Skipf("前端词表文件不可读，跳过跨仓库校验: %v", err)
	}
	ts := string(raw)
	union := frontendSet(t, ts, "export type UserRole =", ";")
	options := frontendSet(t, ts, "export const PRIMARY_ROLE_OPTIONS", "];")
	assertExactSet(t, "前端 UserRole 联合类型", union, role.All)
	assertExactSet(t, "前端 PRIMARY_ROLE_OPTIONS", options, role.All)
	assertNoRetired(t, "前端 UserRole 联合类型", union)
	assertNoRetired(t, "前端 PRIMARY_ROLE_OPTIONS", options)
}

// frontendSet 从 fromMarker 到下一个 terminator 之间提取所有单引号小写标识符。
func frontendSet(t *testing.T, src, fromMarker, terminator string) map[string]bool {
	t.Helper()
	start := strings.Index(src, fromMarker)
	if start < 0 {
		t.Fatalf("前端词表未找到 %q——结构变更需同步本守卫", fromMarker)
	}
	rest := src[start:]
	end := strings.Index(rest, terminator)
	if end < 0 {
		t.Fatalf("前端词表 %q 未找到结束标记 %q", fromMarker, terminator)
	}
	set := map[string]bool{}
	for _, m := range regexp.MustCompile(`'([a-z_]+)'`).FindAllStringSubmatch(rest[:end], -1) {
		set[m[1]] = true
	}
	return set
}
