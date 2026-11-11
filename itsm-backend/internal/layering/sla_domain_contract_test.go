package layering

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/handlers/sla"
)

// sla 预测能力归位专项回归锁（2026-11-11）。
//
// 本批把 `service/sla_forecast_skill.go`（AI-Native SLA 预测 Skill，530 行）与
// `service/sla_breach_predictor.go`（违约风险排序器，156 行）连同两份专属测试
// git mv 进已有域包 `handlers/sla`。与 skill/escalation 两批不同，本批的假绿
// 风险不是 sentinel 双实例，而是「域包重新依赖共享层」：
//
//   - 构造器原本接收 `*service.LLMGateway`，若迁域时保留具体指针，域包会
//     import "itsm-backend/service"，L.3 立刻回退；本批改为 dto.LLMGatewayPort。
//   - `ticketScope` / `maxReportTickets` 原本是 service 包私有符号，若在域内
//     各复制一份，权限口径与 50000 行内存上限会悄悄漂移。本批下沉到
//     internal/datascope 单一源，两个读面共用。
//
// 因此本锁同时检查「编译可过的语义回退」：匿名接口替换端口、字面量替换
// 单一源常量、旧文件名/基线行复活——这些都不会让 go build 变红。

func readSLABackendFile(t *testing.T, parts ...string) string {
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

func readSLARepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// slaHasLine 逐行精确锚定声明：不用 squashSpace 全文模糊匹配，
// 否则同名的 setter 参数/注释会形成假绿通道（2026-11-11 escalation R2 教训）。
func slaHasLine(src, decl string) bool {
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) == strings.TrimSpace(decl) {
			return true
		}
	}
	return false
}

func TestSLAPredictionImplementationIsDomainOwned(t *testing.T) {
	// 1) 共享层两个旧文件名零复活（编译可过，只有源码守卫能挡）。
	for _, fn := range []string{"sla_forecast_skill.go", "sla_breach_predictor.go"} {
		if _, err := os.Stat(filepath.Join("..", "..", "service", fn)); !os.IsNotExist(err) {
			t.Errorf("service/%s 复活：实现本体必须只在 handlers/sla 域包", fn)
		}
	}

	// 2) 两个生产实现归域包，且零共享层 import。
	for _, fn := range []string{"sla_forecast_skill.go", "sla_breach_predictor.go"} {
		src := readSLABackendFile(t, "handlers", "sla", fn)
		if !strings.HasPrefix(src, "package sla\n") {
			t.Errorf("handlers/sla/%s 包名漂移", fn)
		}
		if strings.Contains(src, `"itsm-backend/service"`) {
			t.Errorf("handlers/sla/%s 回退依赖共享层 service 包（L.3 归零被破坏）", fn)
		}
	}

	// 3) 关键声明逐处源码锚定（编译可过的语义回退也必须红）。
	forecast := readSLABackendFile(t, "handlers", "sla", "sla_forecast_skill.go")
	for _, decl := range []string{
		"gateway dto.LLMGatewayPort",
		"func NewSLAForecastSkill(client *ent.Client, gateway dto.LLMGatewayPort, logger *zap.SugaredLogger) *SLAForecastSkill {",
		"type ForecastInput = dto.ForecastInput",
		"type PredictionPoint = dto.PredictionPoint",
		"type ForecastOutput = dto.ForecastOutput",
		"type ForecastData = dto.ForecastData",
		"Where(ticketrepo.DataScopePredicates(datascope.ViewerTicketScope(input.Viewer.ScopeAll, input.Viewer.Role), input.Viewer.UserID)...).",
		"Limit(datascope.MaxReportTickets + 1).",
		"if len(tickets) > datascope.MaxReportTickets {",
		"messages := []dto.LLMMessage{",
	} {
		if !slaHasLine(forecast, decl) {
			t.Errorf("handlers/sla/sla_forecast_skill.go 缺少声明 %q", decl)
		}
	}
	predictor := readSLABackendFile(t, "handlers", "sla", "sla_breach_predictor.go")
	for _, decl := range []string{
		"type SLABreachPredictor struct{}",
		"func NewSLABreachPredictor() *SLABreachPredictor { return &SLABreachPredictor{} }",
		"func (p *SLABreachPredictor) Predict(f SLABreachFeatures) (*SLABreachRisk, error) {",
	} {
		if !slaHasLine(predictor, decl) {
			t.Errorf("handlers/sla/sla_breach_predictor.go 缺少声明 %q", decl)
		}
	}

	// 4) 值类型仍是 dto 别名（域包别名被换成独立副本会让窄契约签名漂移）。
	for _, p := range []struct {
		name string
		got  reflect.Type
		want reflect.Type
	}{
		{"ForecastInput", reflect.TypeOf(sla.ForecastInput{}), reflect.TypeOf(dto.ForecastInput{})},
		{"PredictionPoint", reflect.TypeOf(sla.PredictionPoint{}), reflect.TypeOf(dto.PredictionPoint{})},
		{"ForecastOutput", reflect.TypeOf(sla.ForecastOutput{}), reflect.TypeOf(dto.ForecastOutput{})},
		{"ForecastData", reflect.TypeOf(sla.ForecastData{}), reflect.TypeOf(dto.ForecastData{})},
	} {
		if p.got != p.want {
			t.Errorf("sla.%s 与 dto.%s 不再是同一类型", p.name, p.name)
		}
	}

	// 5) ticketScope / maxReportTickets 单一源：域包与两个共享层读面不得各自复制。
	ds := readSLABackendFile(t, "internal", "datascope", "datascope.go")
	for _, decl := range []string{
		"const MaxReportTickets = 50000",
		"func ViewerTicketScope(scopeAll bool, role string) DataScope {",
	} {
		if !slaHasLine(ds, decl) {
			t.Errorf("internal/datascope/datascope.go 缺少单一源声明 %q", decl)
		}
	}
	analytics := readSLABackendFile(t, "service", "analytics_service.go")
	if !slaHasLine(analytics, "maxReportTickets    = datascope.MaxReportTickets") {
		t.Error("service/analytics_service.go 未引用 datascope.MaxReportTickets 单一源")
	}
	dashboard := readSLABackendFile(t, "service", "dashboard_service.go")
	if !slaHasLine(dashboard, "return datascope.ViewerTicketScope(v.ScopeAll, v.Role)") {
		t.Error("service/dashboard_service.go 未引用 datascope.ViewerTicketScope 单一源")
	}

	// 6) 组合根与评测链路：不得退回共享层构造器。
	app := readSLABackendFile(t, "internal", "bootstrap", "app.go")
	if !strings.Contains(app, "sla.NewSLAForecastSkill(client, llmGateway, sugar)") {
		t.Error("app.go 缺少域包 SLAForecastSkill 装配")
	}
	if strings.Contains(app, "service.NewSLAForecastSkill(") {
		t.Error("app.go 回退到共享层 NewSLAForecastSkill")
	}
	evalMain := readSLABackendFile(t, "cmd", "ai-eval", "main.go")
	if !strings.Contains(evalMain, "sla.NewSLABreachPredictor()") {
		t.Error("cmd/ai-eval 未接域包预测器，AUC 门禁会退化成不测生产代码")
	}
	if strings.Contains(evalMain, "service.NewSLABreachPredictor()") {
		t.Error("cmd/ai-eval 回退到共享层预测器")
	}

	// 7) 基线棘轮：L.1 两行不得复活，C.6.4 必须精确等于实测值 339。
	baseline := readSLARepoFile(t, "scripts", "docs-gate", "service-prod-baseline.txt")
	for _, line := range strings.Split(baseline, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "sla_forecast_skill.go" || trimmed == "sla_breach_predictor.go" {
			t.Errorf("L.1 基线复活已迁出文件 %q", trimmed)
		}
	}
	drift := readSLARepoFile(t, "scripts", "docs-gate", "product-surface-baseline.txt")
	// 棘轮语义：本批把上界从 343 压到 339，后续批次继续下降合规；
	// 回退到 343（含）以上必红。不用「精确 339」硬编码，避免下一批误报。
	serviceFiles := -1
	for _, line := range strings.Split(drift, "\n") {
		if strings.HasPrefix(line, "service_go_files=") {
			fields := strings.Fields(strings.TrimPrefix(line, "service_go_files="))
			if len(fields) == 0 {
				t.Fatal("C.6.4 service_go_files 行无数值")
			}
			v, err := strconv.Atoi(fields[0])
			if err != nil {
				t.Fatalf("C.6.4 service_go_files 数值非法：%v", err)
			}
			serviceFiles = v
			break
		}
	}
	if serviceFiles < 0 || serviceFiles > 339 {
		t.Errorf("C.6.4 service_go_files 上界必须 ≤339（本批 343 → 339，后续只许继续下降），当前 %d", serviceFiles)
	}
}
