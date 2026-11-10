package layering

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// skill 域整域归零专项回归锁（2026-11-10）。
//
// 本批把 `handlers/skill/handler.go` + `custom_skill.go` 经
// SkillRegistryCatalog 窄契约归零；SkillEntry / SkillInvokeResult 值类型与
// 4 个技能 sentinel 下沉 dto，service 侧同名别名。最大假绿风险是 sentinel
// 双实例：service 侧若保留（或改回）本地 errors.New，域内 errors.Is 永远
// 判 false，注册重复返回 500 而不是 409——编译与 L.3 计数全绿。

const skillSharedImport = `"itsm-backend/service"`

// TestSkillDomainIsFullyConverged：全域生产文件零共享层 import，基线不得
// 出现 skill=N。
func TestSkillDomainIsFullyConverged(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "skill")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 handlers/skill 失败：%v", err)
	}
	importing := []string{}
	prod := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		prod++
		body, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Fatalf("读取 handlers/skill/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/skill/%s 仅 %d 字节", fn, len(body))
		}
		if strings.Contains(string(body), skillSharedImport) {
			importing = append(importing, fn)
		}
	}
	if prod < 3 {
		t.Fatalf("扫描器零命中保护：handlers/skill 只有 %d 个生产 .go 文件", prod)
	}
	if len(importing) != 0 {
		t.Errorf("skill 域整域归零被破坏：仍 import 共享层的是 %v", importing)
	}
	for _, fn := range []string{"handler.go", "custom_skill.go", "contracts.go"} {
		body, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Fatalf("读取 handlers/skill/%s 失败：%v", fn, err)
		}
		if strings.Contains(string(body), skillSharedImport) {
			t.Errorf("handlers/skill/%s 重新引用共享层包", fn)
		}
	}
	base := readSkillRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if regexp.MustCompile(`(?m)^skill=\d+$`).MatchString(base) {
		t.Errorf("L.3 基线不得残留 skill=N 行（整域归零后应删除该行，含 skill=0 假账）")
	}
}

func readSkillRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// TestSkillSharedSentinelsAreSameInstance：sentinel 必须是同一 error 实例
// （service 别名 ↔ dto 唯一实现），否则域内 errors.Is 静默失效。
func TestSkillSharedSentinelsAreSameInstance(t *testing.T) {
	for _, s := range []struct {
		name           string
		shared, bottom error
	}{
		{"ErrSkillAlreadyRegistered", service.ErrSkillAlreadyRegistered, dto.ErrSkillAlreadyRegistered},
		{"ErrSkillNotFound", service.ErrSkillNotFound, dto.ErrSkillNotFound},
		{"ErrSkillValidation", service.ErrSkillValidation, dto.ErrSkillValidation},
		{"ErrSkillInvoke", service.ErrSkillInvoke, dto.ErrSkillInvoke},
	} {
		if !errors.Is(fmt.Errorf("wrapped: %w", s.bottom), s.shared) {
			t.Errorf("service.%s 必须与 dto sentinel 是同一实例（双实例会让 409/404 退化成 500）", s.name)
		}
	}
	// service 侧不得再出现本地 errors.New 的技能 sentinel（双实例源头）。
	regSrc := readSkillRepoFile(t, "service", "skill_registry.go")
	if strings.Contains(regSrc, `errors.New("skill `) {
		t.Errorf("service/skill_registry.go 出现本地 errors.New 技能 sentinel：唯一实例必须在 dto")
	}
}

// TestSkillValueTypesAreDTOAliases：值类型身份与形状/tag 锁。
func TestSkillValueTypesAreDTOAliases(t *testing.T) {
	pairs := []struct {
		name   string
		shared reflect.Type
		bottom reflect.Type
	}{
		{"SkillEntry", reflect.TypeOf(service.SkillEntry{}), reflect.TypeOf(dto.SkillEntry{})},
		{"SkillInvokeResult", reflect.TypeOf(service.SkillInvokeResult{}), reflect.TypeOf(dto.SkillInvokeResult{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 与 dto.%s 不再是同一类型（别名被换成独立副本）", p.name, p.name)
		}
	}
	if got := fieldNames(reflect.TypeOf(dto.SkillEntry{})); got != "Code,Name,Version,Title,Provider,Description,Category,Tags,Capabilities,RequiredPerms,Author,IsOfficial,Checksum,IsBuiltin,TenantID,Status,Manifest,Metrics,LongDescription" {
		t.Errorf("dto.SkillEntry 字段形状漂移：%s", got)
	}
	if got := fieldNames(reflect.TypeOf(dto.SkillInvokeResult{})); got != "Output,LatencyMs,SkippedMetrics" {
		t.Errorf("dto.SkillInvokeResult 字段形状漂移：%s", got)
	}
	tagOf := func(v reflect.Type, field string) string { f, _ := v.FieldByName(field); return string(f.Tag) }
	if got := tagOf(reflect.TypeOf(dto.SkillEntry{}), "RequiredPerms"); got != `json:"requiredPermissions"` {
		t.Errorf("SkillEntry.RequiredPerms tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.SkillEntry{}), "TenantID"); got != `json:"tenantId"` {
		t.Errorf("SkillEntry.TenantID tag 漂移：%s", got)
	}

	// 逐处别名源码守卫。
	for _, a := range []struct{ file, decl string }{
		{"skill_registry.go", "type SkillEntry = dto.SkillEntry"},
		{"skill_registry.go", "type SkillInvokeResult = dto.SkillInvokeResult"},
		{"skill_registry.go", "var ErrSkillAlreadyRegistered = dto.ErrSkillAlreadyRegistered"},
		{"skill_registry.go", "var ErrSkillNotFound = dto.ErrSkillNotFound"},
		{"skill_registry.go", "var ErrSkillValidation = dto.ErrSkillValidation"},
		{"skill_registry.go", "var ErrSkillInvoke = dto.ErrSkillInvoke"},
	} {
		src := readSkillRepoFile(t, "service", a.file)
		if !strings.Contains(squashSpace(src), squashSpace(a.decl)) {
			t.Errorf("service/%s 缺少别名声明 %q", a.file, a.decl)
		}
	}
}

// TestSkillCatalogContractShape：窄契约形态与 handler 字段/构造锁。
func TestSkillCatalogContractShape(t *testing.T) {
	contract := readSkillRepoFile(t, "handlers", "skill", "contracts.go")
	for _, decl := range []string{
		"type SkillRegistryCatalog interface",
		"ListEntriesScoped(tenantID int) []dto.SkillEntry",
		"ListEntriesByTagScoped(tenantID int, tag string) []dto.SkillEntry",
		"GetEntryScoped(tenantID int, code string) (dto.SkillEntry, error)",
		"GetScoped(tenantID int, code string) (dto.Skill, error)",
		"RegisterScoped(tenantID int, skill dto.Skill) error",
		"UnregisterScoped(tenantID int, code string) error",
		"InvokeWithMetrics(ctx context.Context, code string, input interface{}) (dto.SkillInvokeResult, error)",
	} {
		if !strings.Contains(squashSpace(contract), squashSpace(decl)) {
			t.Errorf("handlers/skill/contracts.go 契约签名漂移：%q", decl)
		}
	}
	holder := readSkillRepoFile(t, "handlers", "skill", "handler.go")
	for _, decl := range []string{
		"registry SkillRegistryCatalog",
		"func NewHandler(registry SkillRegistryCatalog, logger *zap.SugaredLogger, client *ent.Client) *Handler",
	} {
		if !strings.Contains(squashSpace(holder), squashSpace(decl)) {
			t.Errorf("handlers/skill/handler.go 字段/构造签名漂移：%q", decl)
		}
	}
	custom := readSkillRepoFile(t, "handlers", "skill", "custom_skill.go")
	for _, decl := range []string{
		"*dto.BaseSkill",
		"b := dto.NewBaseSkill(",
	} {
		if !strings.Contains(custom, decl) {
			t.Errorf("handlers/skill/custom_skill.go 缺少 %q", decl)
		}
	}
}
