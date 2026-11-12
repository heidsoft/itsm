package layering

import (
	"context"
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
	"itsm-backend/handlers/skill"
	"itsm-backend/handlers/sla"
	"itsm-backend/internal/services"
)

// AI 域整域归零专项回归锁（2026-11-10，LLMGateway 契约化专项 B1+B2）。
//
// B1 把 `handler.go` / `skills.go` / `builtin_skills.go` 三文件归零，
// B2 把最后一个 `service.go` 经 8 个域内窄契约归零，AI 域已整域零共享层
// import。本文件保留三重锁：全域零引用扫描、基线行已删（不得复活 ai=N）、
// 值类型身份/形状锁。最大的假绿风险仍是 skill 骨架的私有钩子：BaseSkill 下沉
// dto 后，若有人把 TrackedSkill / RecordResult 改回私有名字，编译与 L.3
// 计数都不变，但 SkillRegistry 的运行时类型断言会静默失效、指标不再累计。

const aiSharedImport = `"itsm-backend/internal/services"`

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

// TestAIDomainIsFullyConverged：全域生产文件零共享层 import（整域归零），
// 基线不得出现 ai=N（含 ai=0 的假账）。
func TestAIDomainIsFullyConverged(t *testing.T) {
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
	if len(importing) != 0 {
		t.Errorf("AI 域整域归零被破坏：仍 import 共享层的是 %v", importing)
	}
	for _, fn := range []string{"handler.go", "skills.go", "builtin_skills.go", "contracts.go", "rag_contract.go"} {
		body := readAIRepoFile(t, "handlers", "ai", fn)
		if strings.Contains(body, aiSharedImport) {
			t.Errorf("handlers/ai/%s 重新引用共享层包，B1 归零被破坏", fn)
		}
	}
	base := readAIRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if regexp.MustCompile(`(?m)^ai=\d+$`).MatchString(base) {
		t.Errorf("L.3 基线不得残留 ai=N 行（整域归零后应删除该行，含 ai=0 假账）")
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
		{"ForecastInput", reflect.TypeOf(sla.ForecastInput{}), reflect.TypeOf(dto.ForecastInput{})},
		{"PredictionPoint", reflect.TypeOf(sla.PredictionPoint{}), reflect.TypeOf(dto.PredictionPoint{})},
		{"ForecastOutput", reflect.TypeOf(sla.ForecastOutput{}), reflect.TypeOf(dto.ForecastOutput{})},
		{"ForecastData", reflect.TypeOf(sla.ForecastData{}), reflect.TypeOf(dto.ForecastData{})},
		{"TriageResult", reflect.TypeOf(service.TriageResult{}), reflect.TypeOf(dto.TriageResult{})},
		{"AIMetrics", reflect.TypeOf(service.AIMetrics{}), reflect.TypeOf(dto.AIMetrics{})},
		{"QuotaUsage", reflect.TypeOf(service.QuotaUsage{}), reflect.TypeOf(dto.QuotaUsage{})},
		{"CostEstimate", reflect.TypeOf(service.CostEstimate{}), reflect.TypeOf(dto.CostEstimate{})},
		{"SummaryWithMetadata", reflect.TypeOf(service.SummaryWithMetadata{}), reflect.TypeOf(dto.SummaryWithMetadata{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 与 dto.%s 不再是同一类型（别名被换成独立副本）", p.name, p.name)
		}
	}
	if reflect.TypeOf((*skill.Skill)(nil)).Elem() != reflect.TypeOf((*dto.Skill)(nil)).Elem() {
		t.Errorf("skill.Skill 与 dto.Skill 接口身份漂移（域包别名被换成独立副本）")
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
		{"TriageResult", reflect.TypeOf(dto.TriageResult{}), "Category,Priority,AssigneeID,Confidence,Explanation,SuggestedFix,Model,ModelName,PromptVersion,Degraded,DegradedReason"},
		{"AIMetrics", reflect.TypeOf(dto.AIMetrics{}), "TotalRequests,TotalFeedback,UsefulFeedback,UsefulRate,ByKind,AvgResponseTimeSeconds,LLMCallCount,TotalTokens,ResponseTimeAvailable,Quota,Cost"},
		{"QuotaUsage", reflect.TypeOf(dto.QuotaUsage{}), "QuotaConfigured,MonthlyQuota,TokensUsed,TokensRemaining,UsedRatio,Alert,AlertMessage"},
		{"CostEstimate", reflect.TypeOf(dto.CostEstimate{}), "CostConfigured,TotalTokens,EstimatedCost,Currency,PricePer1K"},
		{"SummaryWithMetadata", reflect.TypeOf(dto.SummaryWithMetadata{}), "Summary,Method,Latency,CharCount,PromptVersion,Model,Confidence"},
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
	if got := tagOf(reflect.TypeOf(dto.AIMetrics{}), "TotalRequests"); got != `json:"totalRequests"` {
		t.Errorf("AIMetrics.TotalRequests tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.AIMetrics{}), "Quota"); got != `json:"quota,omitempty"` {
		t.Errorf("AIMetrics.Quota tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.TriageResult{}), "Degraded"); got != `json:"degraded,omitempty"` {
		t.Errorf("TriageResult.Degraded tag 漂移：%s", got)
	}
	if got := tagOf(reflect.TypeOf(dto.SummaryWithMetadata{}), "PromptVersion"); got != `json:"promptVersion"` {
		t.Errorf("SummaryWithMetadata.PromptVersion tag 漂移：%s", got)
	}

	// 逐处别名源码守卫（不取并集）。
	aliasDecls := []struct{ file, decl string }{
		{"ai_evaluator.go", "type AIEvaluationReport = dto.AIEvaluationReport"},
		{"ai_evaluator.go", "type AIScenarioEval = dto.AIScenarioEval"},
		{"ai_evaluator.go", "type AISkillEval = dto.AISkillEval"},
		{"ai_evaluator.go", "type AICalibrationBucket = dto.AICalibrationBucket"},
		{"ai_evaluator.go", "type LLMPlatformStats = dto.LLMPlatformStats"},
		{"ai_evaluator.go", "type AIAuditEntry = dto.AIAuditEntry"},
		{"root_cause_service.go", "var ErrAIAnalysisUnavailable = dto.ErrAIAnalysisUnavailable"},
		{"ai_telemetry_repository.go", "var ErrFeedbackNotFound = dto.ErrFeedbackNotFound"},
		{"triage_service.go", "type TriageResult = dto.TriageResult"},
		{"ai_telemetry.go", "type AIMetrics = dto.AIMetrics"},
		{"ai_quota.go", "type QuotaUsage = dto.QuotaUsage"},
		{"ai_quota.go", "type CostEstimate = dto.CostEstimate"},
		{"summarize_service.go", "type SummaryWithMetadata = dto.SummaryWithMetadata"},
	}
	cache := map[string]string{}
	for _, a := range aliasDecls {
		if _, ok := cache[a.file]; !ok {
			cache[a.file] = readAIRepoFile(t, "internal", "services", a.file)
		}
		if !strings.Contains(squashSpace(cache[a.file]), squashSpace(a.decl)) {
			t.Errorf("internal/services/%s 缺少别名声明 %q", a.file, a.decl)
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
	registry := readAIRepoFile(t, "handlers", "skill", "registry.go")
	for _, decl := range []string{
		"TrackedSkill()", "RecordResult(elapsed time.Duration, err error)",
		"bs.RecordResult(elapsed, execErr)",
	} {
		if !strings.Contains(squashSpace(registry), squashSpace(decl)) {
			t.Errorf("handlers/skill/registry.go 缺少公有两种子调用 %q", decl)
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

// TestAIServiceContractShapes：B2 的 8 个窄契约源码级形态锁（逐处比对，
// 不取并集）。签名若漂移（例如 viewer 参数换回共享层类型），生产具体指针
// 会静默不满足接口，运行时装配报错才暴露——这里在编译期就锁死。
func TestAIServiceContractShapes(t *testing.T) {
	contract := readAIRepoFile(t, "handlers", "ai", "contracts.go")
	for _, decl := range []string{
		"Execute(ctx context.Context, tenantID int, name string, args map[string]interface{}) (interface{}, error)",
		"GetTool(name string) *dto.ToolDefinition",
		"ListTools() []dto.ToolDefinition",
		"ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition",
		"GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int, viewer dto.DashboardViewer) (*dto.DeepAnalyticsResponse, error)",
		"GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, viewer dto.DashboardViewer) (*dto.TrendPredictionResponse, error)",
		"Execute(ctx context.Context, input *dto.ForecastInput) (*dto.ForecastOutput, error)",
		"SuggestForTenant(ctx context.Context, title, description string, tenantID int) dto.TriageResult",
		"AnalyzeIncident(ctx context.Context, incidentID, tenantID int) (*dto.IncidentAIAnalysisResponse, error)",
		"AnalyzeTicket(ctx context.Context, ticketID, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error)",
		"GetAnalysisReport(ctx context.Context, ticketID, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error)",
		"SummarizeTicket(ctx context.Context, ticketID, tenantID int) (map[string]interface{}, error)",
		"Evaluate(ctx context.Context, tenantID int, lookbackDays int) (*dto.AIEvaluationReport, error)",
		"GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (*dto.AIMetrics, error)",
		"ListAuditLogs(ctx context.Context, tenantID int, page, pageSize int, kind string, lookbackDays int) ([]dto.AIAuditEntry, int, error)",
		"UpdateAuditAccepted(ctx context.Context, tenantID int, id int64, accepted bool) error",
		"RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)",
		"SummarizeWithMetadata(ctx context.Context, text string, maxLen int) (*dto.SummaryWithMetadata, error)",
	} {
		if !strings.Contains(squashSpace(contract), squashSpace(decl)) {
			t.Errorf("handlers/ai/contracts.go 契约签名漂移：%q", decl)
		}
	}
	for _, iface := range []string{
		"type ToolCatalog interface", "type DeepAnalyticsProvider interface",
		"type TrendPredictor interface", "type SLAForecastRunner interface",
		"type TriageAdvisor interface", "type RootCauseAnalyzer interface",
		"type AITelemetryFacade interface", "type Summarizer interface",
	} {
		if !strings.Contains(contract, iface) {
			t.Errorf("contracts.go 缺少 %q", iface)
		}
	}
	// service.go 的字段/构造必须使用域内窄接口而非共享层具体类型。
	holder := readAIRepoFile(t, "handlers", "ai", "service.go")
	for _, decl := range []string{
		"tools ToolCatalog", "analytics DeepAnalyticsProvider", "prediction TrendPredictor",
		"slaForecastSkill SLAForecastRunner", "triageService TriageAdvisor",
		"rca RootCauseAnalyzer", "aiTelemetryService AITelemetryFacade",
		"summarizeService Summarizer",
	} {
		if !strings.Contains(squashSpace(holder), squashSpace(decl)) {
			t.Errorf("handlers/ai/service.go 字段类型漂移（应使用域内窄接口）：%q", decl)
		}
	}
	// auditLedger 默认装配依赖 AITelemetryFacade ⊇ AuditLedgerWriter：
	// 编译期断言从域内上移到本集中锁。
	var _ AuditLedgerAssert = (*service.AITelemetryService)(nil)
}

// AuditLedgerAssert 是域内 AuditLedgerWriter 的镜像签名（供集中锁做
// 编译期满足检查；生产实现是共享层 AI 遥测服务）。
type AuditLedgerAssert interface {
	RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)
}
