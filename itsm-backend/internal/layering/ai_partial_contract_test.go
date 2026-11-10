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
	"time"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// AI 域部分归零专项回归锁（2026-11-10，LLMGateway 契约化专项 B1）。
//
// 本批把 `handler.go` / `skills.go` / `builtin_skills.go` 三文件归零，
// `service.go` 仍留待 B2 的 10 个窄契约，所以这是**部分归零域**：
// 不能进 contractDomains（那个守卫逐文件扫整个域），必须用「实测计数 +
// 基线数值」双锁。最大的假绿风险是 skill 骨架的私有钩子：BaseSkill 下沉
// dto 后，若有人把 TrackedSkill / RecordResult 改回私有名字，编译与 L.3
// 计数都不变，但 SkillRegistry 的运行时类型断言会静默失效、指标不再累计。

const aiSharedImport = `"itsm-backend/service"`

func readAIRepoFile(t *testing.T, parts ...string) string {
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

// TestAIDomainIsPartiallyConverged：三个目标文件零共享层 import，全域恰好
// 还剩 1 个（service.go），基线必须精确等于 1。
func TestAIDomainIsPartiallyConverged(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "ai")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 handlers/ai 失败：%v", err)
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
			t.Fatalf("读取 handlers/ai/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/ai/%s 仅 %d 字节", fn, len(body))
		}
		if strings.Contains(string(body), aiSharedImport) {
			importing = append(importing, fn)
		}
	}
	if prod < 8 {
		t.Fatalf("扫描器零命中保护：handlers/ai 只有 %d 个生产 .go 文件", prod)
	}
	if len(importing) != 1 || importing[0] != "service.go" {
		t.Errorf("AI 域部分归零口径漂移：仍 import 共享层的是 %v，期望仅 service.go", importing)
	}
	for _, fn := range []string{"handler.go", "skills.go", "builtin_skills.go", "contracts.go", "rag_contract.go"} {
		body := readAIRepoFile(t, "handlers", "ai", fn)
		if strings.Contains(body, aiSharedImport) {
			t.Errorf("handlers/ai/%s 重新引用共享层包，B1 归零被破坏", fn)
		}
	}
	base := readAIRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if !regexp.MustCompile(`(?m)^ai=1$`).MatchString(base) {
		t.Errorf("L.3 基线 ai 必须精确为 1（B1 后仅 service.go 依赖共享层）")
	}
	if regexp.MustCompile(`(?m)^ai=0$`).MatchString(base) {
		t.Errorf("L.3 基线 ai=0 是提前宣告整域归零的假账本")
	}
}

// TestAISharedValueTypesAreDTOAliases：整组闭包的类型身份与字段/tag 形状。
func TestAISharedValueTypesAreDTOAliases(t *testing.T) {
	pairs := []struct {
		name   string
		shared reflect.Type
		bottom reflect.Type
	}{
		{"AIEvaluationReport", reflect.TypeOf(service.AIEvaluationReport{}), reflect.TypeOf(dto.AIEvaluationReport{})},
		{"AIScenarioEval", reflect.TypeOf(service.AIScenarioEval{}), reflect.TypeOf(dto.AIScenarioEval{})},
		{"AISkillEval", reflect.TypeOf(service.AISkillEval{}), reflect.TypeOf(dto.AISkillEval{})},
		{"AICalibrationBucket", reflect.TypeOf(service.AICalibrationBucket{}), reflect.TypeOf(dto.AICalibrationBucket{})},
		{"LLMPlatformStats", reflect.TypeOf(service.LLMPlatformStats{}), reflect.TypeOf(dto.LLMPlatformStats{})},
		{"AIAuditEntry", reflect.TypeOf(service.AIAuditEntry{}), reflect.TypeOf(dto.AIAuditEntry{})},
		{"ForecastInput", reflect.TypeOf(service.ForecastInput{}), reflect.TypeOf(dto.ForecastInput{})},
		{"PredictionPoint", reflect.TypeOf(service.PredictionPoint{}), reflect.TypeOf(dto.PredictionPoint{})},
		{"ForecastOutput", reflect.TypeOf(service.ForecastOutput{}), reflect.TypeOf(dto.ForecastOutput{})},
		{"ForecastData", reflect.TypeOf(service.ForecastData{}), reflect.TypeOf(dto.ForecastData{})},
		{"SkillManifest", reflect.TypeOf(service.SkillManifest{}), reflect.TypeOf(dto.SkillManifest{})},
		{"SkillMetrics", reflect.TypeOf(service.SkillMetrics{}), reflect.TypeOf(dto.SkillMetrics{})},
		{"SkillIdentity", reflect.TypeOf(service.SkillIdentity{}), reflect.TypeOf(dto.SkillIdentity{})},
		{"BaseSkill", reflect.TypeOf(service.BaseSkill{}), reflect.TypeOf(dto.BaseSkill{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 与 dto.%s 不再是同一类型（别名被换成独立副本）", p.name, p.name)
		}
	}
	if reflect.TypeOf((*service.Skill)(nil)).Elem() != reflect.TypeOf((*dto.Skill)(nil)).Elem() {
		t.Errorf("service.Skill 与 dto.Skill 接口身份漂移")
	}

	shapes := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"AIEvaluationReport", reflect.TypeOf(dto.AIEvaluationReport{}), "GeneratedAt,LookbackDays,TotalFeedback,UsefulRate,AcceptedRate,AvgConfidence,HealthScore,HasData,ByScenario,BySkill,ConfidenceCalibration,Platform"},
		{"AIAuditEntry", reflect.TypeOf(dto.AIAuditEntry{}), "ID,CreatedAt,TenantID,UserID,RequestID,Scenario,InputRef,PromptVersion,Model,Confidence,Accepted,Suggestion,Notes"},
		{"ForecastOutput", reflect.TypeOf(dto.ForecastOutput{}), "Predictions,Confidence,Model,Insights,Seasonality,Trend,AnomalyDates"},
		{"SkillManifest", reflect.TypeOf(dto.SkillManifest{}), "Name,Version,Title,Provider,Description,LongDescription,IconURL,Screenshots,Tags,Category,Capabilities,InputSchema,OutputSchema,RequiredPermissions,MinSystemVersion,Author,Rating,InstallCount,IsOfficial,Checksum"},
		{"SkillMetrics", reflect.TypeOf(dto.SkillMetrics{}), "TotalCalls,SuccessRate,AvgLatencyMs,ErrorCount,LastUsedAt"},
	}
	for _, s := range shapes {
		if got := fieldNames(s.typ); got != s.want {
			t.Errorf("dto.%s 字段形状漂移：%s", s.name, got)
		}
	}
	tagOf := func(v reflect.Type, field string) string { f, _ := v.FieldByName(field); return string(f.Tag) }
	if got := tagOf(reflect.TypeOf(dto.AIEvaluationReport{}), "ByScenario"); got != `json:"byScenario"` {
		t.Errorf("AIEvaluationReport.ByScenario tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.PredictionPoint{}), "Predicted"); got != `json:"predicted"` {
		t.Errorf("PredictionPoint.Predicted tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.SkillManifest{}), "RequiredPermissions"); got != `json:"requiredPermissions,omitempty"` {
		t.Errorf("SkillManifest.RequiredPermissions tag 漂移：%s", got)
	}

	// 逐处别名源码守卫（不取并集）。
	aliasDecls := []struct{ file, decl string }{
		{"ai_evaluator.go", "type AIEvaluationReport = dto.AIEvaluationReport"},
		{"ai_evaluator.go", "type AIScenarioEval = dto.AIScenarioEval"},
		{"ai_evaluator.go", "type AISkillEval = dto.AISkillEval"},
		{"ai_evaluator.go", "type AICalibrationBucket = dto.AICalibrationBucket"},
		{"ai_evaluator.go", "type LLMPlatformStats = dto.LLMPlatformStats"},
		{"ai_evaluator.go", "type AIAuditEntry = dto.AIAuditEntry"},
		{"sla_forecast_skill.go", "type ForecastInput = dto.ForecastInput"},
		{"sla_forecast_skill.go", "type PredictionPoint = dto.PredictionPoint"},
		{"sla_forecast_skill.go", "type ForecastOutput = dto.ForecastOutput"},
		{"sla_forecast_skill.go", "type ForecastData = dto.ForecastData"},
		{"skill_manifest.go", "type SkillManifest = dto.SkillManifest"},
		{"skill_manifest.go", "type SkillMetrics = dto.SkillMetrics"},
		{"skill_manifest.go", "type Skill = dto.Skill"},
		{"skill_base.go", "type BaseSkill = dto.BaseSkill"},
		{"skill_base.go", "type SkillIdentity = dto.SkillIdentity"},
		{"root_cause_service.go", "var ErrAIAnalysisUnavailable = dto.ErrAIAnalysisUnavailable"},
		{"ai_telemetry_repository.go", "var ErrFeedbackNotFound = dto.ErrFeedbackNotFound"},
	}
	cache := map[string]string{}
	for _, a := range aliasDecls {
		if _, ok := cache[a.file]; !ok {
			cache[a.file] = readAIRepoFile(t, "service", a.file)
		}
		if !strings.Contains(squashSpace(cache[a.file]), squashSpace(a.decl)) {
			t.Errorf("service/%s 缺少别名声明 %q", a.file, a.decl)
		}
	}

	for _, s := range []struct {
		name           string
		shared, bottom error
	}{
		{"ErrAIAnalysisUnavailable", service.ErrAIAnalysisUnavailable, dto.ErrAIAnalysisUnavailable},
		{"ErrFeedbackNotFound", service.ErrFeedbackNotFound, dto.ErrFeedbackNotFound},
	} {
		if !errors.Is(fmt.Errorf("wrapped: %w", s.bottom), s.shared) {
			t.Errorf("service.%s 必须与 dto sentinel 是同一实例", s.name)
		}
	}
}

// TestSkillBaseHooksAreExported：私有钩子改回私人名字时编译/计数都不变，
// 但 Registry 的类型断言会静默失效——这是本批最关键的语义锁。
func TestSkillBaseHooksAreExported(t *testing.T) {
	skillDTO := readAIRepoFile(t, "dto", "skill_dto.go")
	for _, decl := range []string{
		"func (b *BaseSkill) TrackedSkill() {}",
		"func (b *BaseSkill) RecordResult(elapsed time.Duration, err error)",
		"func (m SkillManifest) ComputeChecksum() string",
		"func (m SkillManifest) ValidateForRegistration() error",
	} {
		if !strings.Contains(squashSpace(skillDTO), squashSpace(decl)) {
			t.Errorf("dto/skill_dto.go 缺少 %q", decl)
		}
	}
	registry := readAIRepoFile(t, "service", "skill_registry.go")
	for _, decl := range []string{
		"TrackedSkill()", "RecordResult(elapsed time.Duration, err error)",
		"bs.RecordResult(elapsed, execErr)",
	} {
		if !strings.Contains(squashSpace(registry), squashSpace(decl)) {
			t.Errorf("service/skill_registry.go 缺少公有两种子调用 %q", decl)
		}
	}

	// 行为锁：方法必须真的挂在下沉后的 dto 类型上，且语义未变。
	m := dto.SkillManifest{Name: "ai.x", Version: "v1", Provider: "p",
		Capabilities: []string{"b", "a"}, RequiredPermissions: []string{"z", "y"}, MinSystemVersion: "1.0"}
	m2 := dto.SkillManifest{Name: "ai.x", Version: "v1", Provider: "p",
		Capabilities: []string{"a", "b"}, RequiredPermissions: []string{"y", "z"}, MinSystemVersion: "1.0"}
	if m.ComputeChecksum() != m2.ComputeChecksum() || !strings.HasPrefix(m.ComputeChecksum(), "sha256:") {
		t.Errorf("ComputeChecksum 排序无关性/格式漂移：%s vs %s", m.ComputeChecksum(), m2.ComputeChecksum())
	}
	m3 := m
	m3.MinSystemVersion = "2.0"
	if m3.ComputeChecksum() == m.ComputeChecksum() {
		t.Errorf("ComputeChecksum 丢失 MinSystemVersion 覆盖")
	}
	if err := (dto.SkillManifest{}).ValidateForRegistration(); err == nil {
		t.Errorf("ValidateForRegistration 未保持 fail-closed")
	}

	b := dto.NewBaseSkill("ai.triage", "分诊", "v1", "ga", []string{"t"}, []string{"ticket:read"}, []string{"c"})
	manifest := b.BuildManifest()
	if manifest.Name != "ai.triage" || manifest.Checksum == "" || len(manifest.RequiredPermissions) != 1 {
		t.Errorf("BaseSkill.BuildManifest 行为漂移：%+v", manifest)
	}
	b.RecordResult(10*time.Millisecond, nil)
	b.RecordResult(20*time.Millisecond, errors.New("boom"))
	metrics := b.GetMetrics()
	if metrics.TotalCalls != 2 || metrics.ErrorCount != 1 {
		t.Errorf("BaseSkill.RecordResult 指标累计漂移：%+v", metrics)
	}

	skillType := reflect.TypeOf((*dto.Skill)(nil)).Elem()
	want := []string{"Code", "Execute", "GetMetrics", "Manifest", "Name", "Tags", "Validate"}
	got := make([]string, 0, skillType.NumMethod())
	for i := 0; i < skillType.NumMethod(); i++ {
		got = append(got, skillType.Method(i).Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("dto.Skill 方法集漂移：got %v want %v", got, want)
	}
}

// TestAISkillRegistrarWiring：窄契约形态与组合根接线。
func TestAISkillRegistrarWiring(t *testing.T) {
	contract := readAIRepoFile(t, "handlers", "ai", "contracts.go")
	if !strings.Contains(contract, "type SkillRegistrar interface") ||
		!strings.Contains(squashSpace(contract), "Register(skill dto.Skill) error") {
		t.Errorf("contracts.go 的 SkillRegistrar 契约签名漂移")
	}
	if strings.Contains(contract, aiSharedImport) {
		t.Errorf("contracts.go 不得引用共享层包")
	}
	builtins := readAIRepoFile(t, "handlers", "ai", "builtin_skills.go")
	for _, decl := range []string{"reg SkillRegistrar", "[]dto.Skill{"} {
		if !strings.Contains(builtins, decl) {
			t.Errorf("builtin_skills.go 缺少 %q", decl)
		}
	}
	app := readAIRepoFile(t, "internal", "bootstrap", "app.go")
	if !regexp.MustCompile(`RegisterBuiltinSkills\(`).MatchString(app) {
		t.Errorf("组合根缺少 RegisterBuiltinSkills 装配接线")
	}
}
