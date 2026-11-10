package layering

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// dashboard 整域归零回归锁（2026-11-10 成组 DTO 专项）。
//
// 该域的共享层依赖不是「一个字段换窄接口」就完事：窄契约的签名带回
// 14 个 service 包内值类型（概览、KPI、趋势、SLA、满意度、服务质量视图），
// 必须按类型闭包成组下沉 dto，service 侧留同名别名。三个容易假绿的点：
// ① migrate.go 不 import 共享层，却调 GetDashboardData，方法集漏项会静默
//    丢掉三个兼容路由；② 两个从未被调用的构造参数被删掉后，装配层若漏改
//    仍能编译过但接线语义全变；③ 窗口上限常量另起一份时 L.3 计数与编译
//    都不变，只有值/身份断言能拦。因此本文件同时锁源码形态、装配接线、
//    类型身份与 JSON 形状。

const dashboardSharedImport = `"itsm-backend/service"`

func readDashboardRepoFile(t *testing.T, parts ...string) string {
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

// TestDashboardDomainHasZeroSharedImports：整域生产文件零共享层 import +
// 基线必须删行（留 dashboard=0 等于保留一个可悄悄放宽的槽位）。
func TestDashboardDomainHasZeroSharedImports(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "dashboard")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 handlers/dashboard 失败：%v", err)
	}
	prod := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		prod++
		body, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Fatalf("读取 handlers/dashboard/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/dashboard/%s 仅 %d 字节", fn, len(body))
		}
		if strings.Contains(string(body), dashboardSharedImport) {
			t.Errorf("handlers/dashboard/%s 重新引用共享层包，整域归零被破坏", fn)
		}
	}
	// config_store.go / contracts.go / handler.go / migrate.go
	if prod < 4 {
		t.Fatalf("扫描器零命中保护：handlers/dashboard 只剩 %d 个生产 .go 文件（期望至少 4）", prod)
	}
	base := readDashboardRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if regexp.MustCompile(`(?m)^dashboard=`).MatchString(base) {
		t.Errorf("L.3 基线仍保留 dashboard= 行（整域归零必须删除，不能留 dashboard=0 槽位）")
	}
}

// TestDashboardContractsAndAssembly：契约方法集、域内字段形态、构造签名、
// 组合根接线与测试夹具调用点，编译与 L.3 计数都看不出「漏接线/复活死字段」。
func TestDashboardContractsAndAssembly(t *testing.T) {
	contract := readDashboardRepoFile(t, "handlers", "dashboard", "contracts.go")
	for _, iface := range []string{"DashboardOverviewProvider", "ServiceQualityReportBuilder"} {
		if !strings.Contains(contract, "type "+iface+" interface") {
			t.Errorf("contracts.go 未声明窄契约 %s", iface)
		}
	}
	if strings.Contains(contract, dashboardSharedImport) {
		t.Errorf("contracts.go 含共享层包路径字面量（连注释也算 L.3 命中）")
	}
	// 方法集 = handler.go + migrate.go 的调用并集（6 + 1）。
	for _, m := range []string{
		"GetDashboardOverview(", "GetDashboardOverviewStats(", "GetDashboardData(",
		"GetSystemStats(", "GetTicketTrend(", "GetUserStats(", "Build(",
	} {
		if !strings.Contains(contract, m) {
			t.Errorf("contracts.go 方法集缺 %s（域内生产文件调用并集）", m)
		}
		if strings.Count(contract, m) < 1 {
			t.Fatalf("扫描器零命中保护：%s 计数异常", m)
		}
	}

	h := readDashboardRepoFile(t, "handlers", "dashboard", "handler.go")
	if strings.Contains(h, dashboardSharedImport) {
		t.Errorf("handler.go 重新引用共享层包")
	}
	for _, decl := range []string{
		"dashboardService DashboardOverviewProvider",
		"qualityService ServiceQualityReportBuilder",
		"SetServiceQualityService(s ServiceQualityReportBuilder)",
	} {
		if !strings.Contains(h, decl) {
			t.Errorf("handler.go 缺少域内契约形态 %q", decl)
		}
	}
	ctor := regexp.MustCompile(`func NewDashboardHandler\(\s*\n\s*dashboardService DashboardOverviewProvider,\s*\n\s*logger \*zap\.SugaredLogger,\s*\n\) \*DashboardHandler`)
	if !ctor.MatchString(h) {
		t.Errorf("NewDashboardHandler 必须恰好两参（服务 + logger），死参数不得复活")
	}
	if strings.Contains(h, "ticketService") || strings.Contains(h, "incidentService") {
		t.Errorf("handler.go 复活了从未被调用的 ticketService / incidentService 字段")
	}
	if strings.Count(h, "dto.MaxQualityWindowDays") != 2 {
		t.Errorf("parseQualityWindow 两处窗口上限必须引用 dto 常量（实测 %d 处）", strings.Count(h, "dto.MaxQualityWindowDays"))
	}

	app := readDashboardRepoFile(t, "internal", "bootstrap", "app.go")
	if !strings.Contains(app, "dashboardpkg.NewDashboardHandler(dashboardService, sugar)") {
		t.Errorf("组合根装配漏接线：NewDashboardHandler 应传 2 参")
	}
	if !strings.Contains(app, "dashboardHandler.SetServiceQualityService(qualityReportService)") {
		t.Errorf("组合根装配漏接线：服务质量聚合器必须传真实实现（nil 会静默 503）")
	}
	for _, rel := range []string{
		"router/dashboard_config_route_test.go",
		"handlers/dashboard/kpi_contract_test.go",
	} {
		body := readDashboardRepoFile(t, strings.Split(rel, "/")...)
		if !strings.Contains(body, "NewDashboardHandler(") {
			t.Fatalf("扫描器零命中保护：%s 未找到构造调用点", rel)
		}
		if regexp.MustCompile(`NewDashboardHandler\([^)]*,[^)]*,[^)]*,[^)]*\)`).MatchString(body) {
			t.Errorf("%s 仍是旧的四参构造调用点", rel)
		}
	}
}

// TestDashboardDTOsAreSingleSource：14 个值类型逐处锁别名（指定文件，不取并集），
// reflect.Type 身份 + 常量身份 + 禁止 service 侧重新定义 struct。
func TestDashboardDTOsAreSingleSource(t *testing.T) {
	type alias struct{ file, decl string }
	aFile, bFile := "dashboard_service.go", "service_quality_report_service.go"
	aNames := []string{
		"DashboardOverviewStats", "SLAComplianceData", "DashboardOverviewData",
		"KPIMetricData", "TicketTrendData", "IncidentDistributionData",
		"SatisfactionData", "QuickActionData", "RecentActivityData",
		"ResponseTimeDistributionData", "TeamWorkloadData", "PeakHourData",
	}
	bNames := []string{"ServiceQualityWindow", "ServiceQualityReport"}
	aliases := []alias{}
	for _, n := range aNames {
		aliases = append(aliases, alias{aFile, "type " + n + " = dto." + n})
	}
	for _, n := range bNames {
		aliases = append(aliases, alias{bFile, "type " + n + " = dto." + n})
	}
	aliases = append(aliases, alias{bFile, "const MaxQualityWindowDays = dto.MaxQualityWindowDays"})
	cache := map[string][]string{}
	for _, a := range aliases {
		lines, ok := cache[a.file]
		if !ok {
			body := readDashboardRepoFile(t, "service", a.file)
			lines = strings.Split(body, "\n")
			cache[a.file] = lines
		}
		found := false
		for _, line := range lines {
			if squashSpace(line) == squashSpace(a.decl) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("service/%s 缺少逐处别名 %q（跨文件搬移不算）", a.file, a.decl)
		}
	}
	if len(aliases) != 15 {
		t.Fatalf("扫描器零命中保护：别名清单应恰好 15 条，实测 %d", len(aliases))
	}
	for _, body := range cache {
		joined := strings.Join(body, "\n")
		if regexp.MustCompile(`(?m)^type (?:` + strings.Join(append(aNames, bNames...), "|") + `) struct`).MatchString(joined) {
			t.Errorf("service 侧重新定义了已下沉的值类型（应只保留别名）")
		}
	}

	pairs := []struct {
		name           string
		shared, bottom reflect.Type
	}{
		{"DashboardOverviewStats", reflect.TypeOf(service.DashboardOverviewStats{}), reflect.TypeOf(dto.DashboardOverviewStats{})},
		{"SLAComplianceData", reflect.TypeOf(service.SLAComplianceData{}), reflect.TypeOf(dto.SLAComplianceData{})},
		{"DashboardOverviewData", reflect.TypeOf(service.DashboardOverviewData{}), reflect.TypeOf(dto.DashboardOverviewData{})},
		{"KPIMetricData", reflect.TypeOf(service.KPIMetricData{}), reflect.TypeOf(dto.KPIMetricData{})},
		{"TicketTrendData", reflect.TypeOf(service.TicketTrendData{}), reflect.TypeOf(dto.TicketTrendData{})},
		{"IncidentDistributionData", reflect.TypeOf(service.IncidentDistributionData{}), reflect.TypeOf(dto.IncidentDistributionData{})},
		{"SatisfactionData", reflect.TypeOf(service.SatisfactionData{}), reflect.TypeOf(dto.SatisfactionData{})},
		{"QuickActionData", reflect.TypeOf(service.QuickActionData{}), reflect.TypeOf(dto.QuickActionData{})},
		{"RecentActivityData", reflect.TypeOf(service.RecentActivityData{}), reflect.TypeOf(dto.RecentActivityData{})},
		{"ResponseTimeDistributionData", reflect.TypeOf(service.ResponseTimeDistributionData{}), reflect.TypeOf(dto.ResponseTimeDistributionData{})},
		{"TeamWorkloadData", reflect.TypeOf(service.TeamWorkloadData{}), reflect.TypeOf(dto.TeamWorkloadData{})},
		{"PeakHourData", reflect.TypeOf(service.PeakHourData{}), reflect.TypeOf(dto.PeakHourData{})},
		{"ServiceQualityWindow", reflect.TypeOf(service.ServiceQualityWindow{}), reflect.TypeOf(dto.ServiceQualityWindow{})},
		{"ServiceQualityReport", reflect.TypeOf(service.ServiceQualityReport{}), reflect.TypeOf(dto.ServiceQualityReport{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 必须是 dto.%s 的类型别名", p.name, p.name)
		}
	}
	if len(pairs) != 14 {
		t.Fatalf("扫描器零命中保护：类型身份应恰好 14 对，实测 %d", len(pairs))
	}
	if service.MaxQualityWindowDays != dto.MaxQualityWindowDays {
		t.Errorf("窗口上限常量身份漂移：service=%d dto=%d", service.MaxQualityWindowDays, dto.MaxQualityWindowDays)
	}
	if dto.MaxQualityWindowDays != 90 {
		t.Errorf("窗口上限值漂移：dto=%d，期望 90", dto.MaxQualityWindowDays)
	}
}

// TestDashboardDTOJSONShape：成组下沉最容易静默漂移的是 JSON tag；用实序列化
// 锁住响应契约的关键 key，tag 改错时前端拿到的是 undefined 而不是编译错误。
func TestDashboardDTOJSONShape(t *testing.T) {
	overview := dto.DashboardOverviewData{
		KPIMetrics:               []dto.KPIMetricData{{ID: "open", Comparable: true, Scope: "tenant"}},
		TicketTrend:              []dto.TicketTrendData{{Date: "2026-11-10"}},
		IncidentDistribution:     []dto.IncidentDistributionData{{Category: "network"}},
		SLAData:                  &dto.SLAComplianceData{ComplianceRate: 0.9},
		SatisfactionData:         []dto.SatisfactionData{{Month: "2026-11"}},
		QuickActions:             []dto.QuickActionData{{ID: "create"}},
		RecentActivities:         []dto.RecentActivityData{{ID: 1}},
		ResponseTimeDistribution: []dto.ResponseTimeDistributionData{{TimeRange: "0-1h"}},
		TeamWorkload:             []dto.TeamWorkloadData{{Assignee: "u1"}},
		PeakHours:                []dto.PeakHourData{{Hour: "10"}},
	}
	checkKeys(t, overview, []string{
		"kpiMetrics", "ticketTrend", "incidentDistribution", "slaData",
		"satisfactionData", "quickActions", "recentActivities",
		"responseTimeDistribution", "teamWorkload", "peakHours",
	})
	report := dto.ServiceQualityReport{
		MetricVersion:        "1.1",
		Window:               dto.ServiceQualityWindow{Days: 30},
		Scope:                "tenant",
		KPI:                  []dto.KPIMetricData{{ID: "open"}},
		SLA:                  &dto.SLAComplianceData{},
		IncidentDistribution: []dto.IncidentDistributionData{},
		Satisfaction:         []dto.SatisfactionData{},
		TicketTrend:          []dto.TicketTrendData{},
	}
	checkKeys(t, report, []string{
		"metricVersion", "metricVersions", "generatedAt", "window", "scope",
		"kpi", "sla", "incidentDistribution", "satisfaction", "ticketTrend",
	})
	checkKeys(t, dto.KPIMetricData{ID: "x"}, []string{"id", "comparable", "scope"})
	checkKeys(t, dto.ServiceQualityWindow{Days: 1}, []string{"days", "from", "to"})
	checkKeys(t, dto.SLAComplianceData{}, []string{"complianceRate", "responseTimeCompliance", "atRiskTickets", "breachedTickets"})
}

func checkKeys(t *testing.T, v any, keys []string) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化 %T 失败：%v", v, err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("反序列化 %T 失败：%v", v, err)
	}
	if len(m) == 0 {
		t.Fatalf("扫描器零命中保护：%T 序列化后为空", v)
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("%T 缺少 JSON key %q（tag 漂移或字段被删）", v, k)
		}
	}
}
