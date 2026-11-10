package layering

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// bpmn 是「部分归零」域：第 1 步归零 workflow_template.go，第 2 步归零
// dashboard.go + monitoring.go，第 3 步归零 process_trigger.go，其余 4 个
// 生产文件仍 import 共享层。因此它不能进 contractDomains（那个守卫逐文件
// 扫整个域），改由本文件按「单文件 + 基线」精确锁定。后续每组归零时，把
// bpmnPartialZeroExpectedImports 与基线期望值同步下调即可。

const bpmnSharedImport = `"itsm-backend/service"`

const bpmnPartialZeroExpectedImports = 4

func readBPMNFile(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "handlers", "bpmn", rel))
	if err != nil {
		t.Fatalf("读取 handlers/bpmn/%s 失败：%v", rel, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：handlers/bpmn/%s 仅 %d 字节", rel, len(body))
	}
	return string(body)
}

// TestBPMNPartialZeroWorkflowTemplate 锁住「归零是换成了域内窄契约」而不是
// 把文件删掉或把共享层限定符换成别的写法。改回旧实现必须在这里变红。
func TestBPMNPartialZeroWorkflowTemplate(t *testing.T) {
	wf := readBPMNFile(t, "workflow_template.go")
	if strings.Contains(wf, bpmnSharedImport) {
		t.Errorf("handlers/bpmn/workflow_template.go 重新引用共享层包，L.3 归零被破坏")
	}
	code := stripLineComments(wf)
	if !regexp.MustCompile(`\bcatalog\s+WorkflowTemplateCatalog\b`).MatchString(code) {
		t.Errorf("模板目录字段未使用域内窄接口 WorkflowTemplateCatalog")
	}
	if !regexp.MustCompile(`\bNewWorkflowTemplateHandler\(catalog\s+WorkflowTemplateCatalog\)`).MatchString(code) {
		t.Errorf("构造函数未接收域内窄接口 WorkflowTemplateCatalog")
	}
	if strings.Contains(code, "*service.BPMNWorkflowTemplateCatalog") {
		t.Errorf("模板目录字段回退到共享层具体类型（旧实现）")
	}
	if strings.Contains(code, "service.Err") {
		t.Errorf("错误映射回退到共享层 sentinel 限定符（旧实现）")
	}

	contract := readBPMNFile(t, "contracts.go")
	if len(contract) < 100 {
		t.Fatalf("扫描器零命中保护：contracts.go 仅 %d 字节", len(contract))
	}
	if !strings.Contains(contract, "type WorkflowTemplateCatalog interface") {
		t.Errorf("contracts.go 未声明域内接口 WorkflowTemplateCatalog")
	}
	if strings.Contains(contract, bpmnSharedImport) {
		t.Errorf("contracts.go 出现共享层包路径字面量（连注释也会让 L.3 计数 +1）")
	}

	// 零命中不能由「目录被清空 / 文件被删光」造成。
	entries, err := os.ReadDir(filepath.Join("..", "..", "handlers", "bpmn"))
	if err != nil {
		t.Fatalf("列举 handlers/bpmn 失败：%v", err)
	}
	prodFiles := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		prodFiles++
	}
	if prodFiles < 9 {
		t.Fatalf("扫描器零命中保护：handlers/bpmn 只剩 %d 个生产 .go 文件", prodFiles)
	}
}

// TestBPMNPartialZeroMatchesL3Baseline 同时锁住代码实测计数与棘轮基线：
// 只改代码不收紧基线（或只改基线不改代码）都要变红。
func TestBPMNPartialZeroMatchesL3Baseline(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "bpmn")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("列举 handlers/bpmn 失败：%v", err)
	}
	count := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Fatalf("读取 handlers/bpmn/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/bpmn/%s 仅 %d 字节", fn, len(body))
		}
		// 与 Gate C.8 / L.3 一致：全文 grep，注释里的包路径同样算命中。
		if strings.Contains(string(body), bpmnSharedImport) {
			count++
		}
	}
	if count != bpmnPartialZeroExpectedImports {
		t.Errorf("handlers/bpmn 共享层依赖实测 %d，期望 %d（批次 3 第 1、2、3 步归零 4 个文件）", count, bpmnPartialZeroExpectedImports)
	}

	base, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt"))
	if err != nil {
		t.Fatalf("读取 L.3 基线失败：%v", err)
	}
	if len(base) < 100 {
		t.Fatalf("扫描器零命中保护：L.3 基线仅 %d 字节", len(base))
	}
	values := []string{}
	for _, line := range strings.Split(string(base), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "bpmn=") {
			values = append(values, strings.TrimPrefix(line, "bpmn="))
		}
	}
	want := "4"
	if len(values) != 1 || values[0] != want {
		t.Errorf("L.3 基线 bpmn 期望恰好一条 bpmn=%s，实际 %v（真下降必须同 commit 精确收紧）", want, values)
	}
}

// TestBPMNSharedValuesComeFromDTO 锁住纯值的单一来源：常量与 sentinel 定义在
// dto，service 只做别名，handler 只读 dto。任何一层另起一份都会让 errors.Is
// 或默认阈值漂移，而 L.3 计数门禁看不出来。
func TestBPMNSharedValuesComeFromDTO(t *testing.T) {
	if service.DefaultStaleThresholdHours != dto.DefaultStaleThresholdHours {
		t.Errorf("默认陈旧阈值在共享层与 dto 之间漂移：%d vs %d", service.DefaultStaleThresholdHours, dto.DefaultStaleThresholdHours)
	}

	inv := readBPMNFile(t, "inventory.go")
	invCode := stripLineComments(inv)
	if !strings.Contains(invCode, "dto.DefaultStaleThresholdHours") {
		t.Errorf("inventory.go 未从 dto 读取默认陈旧阈值")
	}
	if strings.Contains(invCode, "service.DefaultStaleThresholdHours") {
		t.Errorf("inventory.go 回退到共享层常量限定符（旧实现）")
	}

	bottom, err := os.ReadFile(filepath.Join("..", "..", "dto", "bpmn_workflow_template_sentinels.go"))
	if err != nil {
		t.Fatalf("读取 dto/bpmn_workflow_template_sentinels.go 失败：%v", err)
	}
	if len(bottom) < 100 {
		t.Fatalf("扫描器零命中保护：dto 纯值文件仅 %d 字节", len(bottom))
	}
	flat := squashSpace(string(bottom))
	for _, decl := range []string{
		"ErrWorkflowTemplateNotFound = errors.New(",
		"ErrWorkflowTemplateConflict = errors.New(",
		"ErrWorkflowTemplateInvalid = errors.New(",
		"const DefaultStaleThresholdHours = 24",
	} {
		if !strings.Contains(flat, squashSpace(decl)) {
			t.Errorf("dto 纯值文件缺少定义 %q（sentinel / 阈值必须只有一份）", decl)
		}
	}
}

// 批次 3 第 2 步需要下沉的值类型：3 个请求 DTO + 18 个响应/嵌套类型。
// service 侧必须保留同名别名，否则接口方法签名里的值类型身份会分裂成两份。
var bpmnMonitoringDTONames = []string{
	"QueryAuditLogsRequest", "ProcessMetricsRequest", "ListProcessInstanceStatusQuery",
	"DashboardMetrics", "ProcessHealth", "ProcessStat", "TaskStat", "TrendPoint",
	"BPMNProcessMetrics", "BottleneckInfo", "ProcessMetrics", "PerformanceMetrics",
	"TaskMetrics", "BottleneckAnalysis", "BottleneckTask", "SlowestPath",
	"ResourceConstraint", "ProcessInstanceStatus", "ProcessTimelineEntry",
	"SLAViolation", "TenantBPMNStats",
}

// TestBPMNPartialZeroMonitoringGroup 锁住「归零是换成域内窄契约」而不是
// 删文件或改写一个等价的共享层限定符。把两个 handler 改回 HEAD 版本必须变红。
func TestBPMNPartialZeroMonitoringGroup(t *testing.T) {
	dash := readBPMNFile(t, "dashboard.go")
	mon := readBPMNFile(t, "monitoring.go")

	for name, body := range map[string]string{"dashboard.go": dash, "monitoring.go": mon} {
		if strings.Contains(body, bpmnSharedImport) {
			t.Errorf("handlers/bpmn/%s 重新引用共享层包，L.3 归零被破坏", name)
		}
		if strings.Contains(stripLineComments(body), "service.") {
			t.Errorf("handlers/bpmn/%s 残留共享层限定符（旧实现）", name)
		}
	}

	dashCode := stripLineComments(dash)
	for _, want := range []string{
		`\bmetricsService\s+DashboardMetricsProvider\b`,
		`\bauditService\s+BPMNAuditLogReader\b`,
		`\btenantService\s+BPMNTenantStatsProvider\b`,
		`\bslaService\s+BPMNSLAProvider\b`,
		`&dto\.QueryAuditLogsRequest\{`,
	} {
		if !regexp.MustCompile(want).MatchString(dashCode) {
			t.Errorf("dashboard.go 缺少窄契约/值类型写法 %q", want)
		}
	}
	monCode := stripLineComments(mon)
	for _, want := range []string{
		`\bmonitoringService\s+ProcessMonitoringProvider\b`,
		`\bNewMonitoringHandler\(monitoringService\s+ProcessMonitoringProvider\)`,
		`\bSetMonitoringService\(s\s+ProcessMonitoringProvider\)`,
		`&dto\.ProcessMetricsRequest\{`,
		`&dto\.ListProcessInstanceStatusQuery\{`,
	} {
		if !regexp.MustCompile(want).MatchString(monCode) {
			t.Errorf("monitoring.go 缺少窄契约/值类型写法 %q", want)
		}
	}

	contract := readBPMNFile(t, "contracts.go")
	if strings.Contains(contract, bpmnSharedImport) {
		t.Errorf("contracts.go 出现共享层包路径字面量（连注释也会让 L.3 计数 +1）")
	}
	for _, iface := range []string{
		"WorkflowTemplateCatalog", "DashboardMetricsProvider", "BPMNAuditLogReader",
		"BPMNTenantStatsProvider", "BPMNSLAProvider", "ProcessMonitoringProvider",
	} {
		if !strings.Contains(contract, "type "+iface+" interface") {
			t.Errorf("contracts.go 未声明域内接口 %s", iface)
		}
	}
}

// TestBPMNMonitoringDTOsHaveSingleSource 逐处比对 dto 定义与 service 别名，
// 不取并集：任何一处别名丢失都会让接口方法集在编译期与运行期同时漂移。
func TestBPMNMonitoringDTOsHaveSingleSource(t *testing.T) {
	dtoBody, err := os.ReadFile(filepath.Join("..", "..", "dto", "bpmn_monitoring_dto.go"))
	if err != nil {
		t.Fatalf("读取 dto/bpmn_monitoring_dto.go 失败：%v", err)
	}
	if len(dtoBody) < 1000 {
		t.Fatalf("扫描器零命中保护：dto 值类型文件仅 %d 字节", len(dtoBody))
	}
	if strings.Contains(string(dtoBody), bpmnSharedImport) {
		t.Errorf("dto 不得反向依赖共享层")
	}
	for _, n := range bpmnMonitoringDTONames {
		if !regexp.MustCompile(`\btype\s+` + n + `\s+struct\s*\{`).MatchString(string(dtoBody)) {
			t.Errorf("dto 缺少值类型定义 %s", n)
		}
	}

	byFile := map[string][]string{
		"bpmn_audit_service.go":      {"QueryAuditLogsRequest"},
		"bpmn_metrics_service.go":    {"DashboardMetrics", "ProcessHealth", "ProcessStat", "TaskStat", "TrendPoint", "BPMNProcessMetrics", "BottleneckInfo"},
		"bpmn_monitoring_service.go": {"ProcessMetricsRequest", "ListProcessInstanceStatusQuery", "ProcessMetrics", "PerformanceMetrics", "TaskMetrics", "BottleneckAnalysis", "BottleneckTask", "SlowestPath", "ResourceConstraint", "ProcessInstanceStatus", "ProcessTimelineEntry"},
		"bpmn_sla_service.go":        {"SLAViolation"},
		"bpmn_tenant_service.go":     {"TenantBPMNStats"},
	}
	seen := 0
	for file, list := range byFile {
		body, err := os.ReadFile(filepath.Join("..", "..", "service", file))
		if err != nil {
			t.Fatalf("读取 service/%s 失败：%v", file, err)
		}
		if len(body) < 500 {
			t.Fatalf("扫描器零命中保护：service/%s 仅 %d 字节", file, len(body))
		}
		flat := squashSpace(stripLineComments(string(body)))
		for _, n := range list {
			seen++
			want := squashSpace("type " + n + " = dto." + n)
			if !strings.Contains(flat, want) {
				t.Errorf("service/%s 丢失别名 %q（值类型不得在共享层另起一份）", file, n)
			}
		}
	}
	if seen != len(bpmnMonitoringDTONames) {
		t.Fatalf("扫描器零命中保护：别名检查覆盖 %d 处，期望 %d 处", seen, len(bpmnMonitoringDTONames))
	}
}

// TestBPMNMonitoringDTOJSONShape 锁住下沉后最容易静默漂移的 JSON tag：
// 结构体字段没变但 tag 改一个字母，前端就会读到 undefined，而编译与 L.3
// 计数都看不出来。逐字段比对只能防「搬定义」时手误，这里防后续修改。
func TestBPMNMonitoringDTOJSONShape(t *testing.T) {
	cases := []struct {
		name string
		v    interface{}
		want []string
	}{
		{
			"DashboardMetrics",
			dto.DashboardMetrics{
				ProcessHealth:    &dto.ProcessHealth{},
				TopProcesses:     []dto.ProcessStat{{}},
				TaskDistribution: []dto.TaskStat{{}},
				TrendData:        []dto.TrendPoint{{}},
			},
			[]string{`"totalProcesses"`, `"slaComplianceRate"`, `"slaCompliantSamples"`, `"slaTotalSamples"`, `"avgCompletionTimeMinutes"`, `"processHealth"`, `"trendData"`},
		},
		{
			"ProcessMetrics",
			dto.ProcessMetrics{
				PerformanceMetrics: &dto.PerformanceMetrics{},
				TaskMetrics:        map[string]*dto.TaskMetrics{},
				BottleneckAnalysis: &dto.BottleneckAnalysis{},
			},
			[]string{`"processDefinitionKey"`, `"performanceMetrics"`, `"taskMetrics"`, `"bottleneckAnalysis"`, `"tenantId"`, `"latencyPercentile95"`},
		},
		{
			"ProcessTimelineEntry",
			dto.ProcessTimelineEntry{VariablesBefore: map[string]interface{}{"k": 1}, Metadata: map[string]interface{}{"m": 2}},
			[]string{`"eventType"`, `"variablesBefore"`, `"metadata"`, `"nodeDurationMs"`},
		},
	}
	for _, c := range cases {
		raw, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("序列化 %s 失败：%v", c.name, err)
		}
		for _, key := range c.want {
			if !strings.Contains(string(raw), key) {
				t.Errorf("%s 的 JSON 契约丢失 key %s，实际 %s", c.name, key, string(raw))
			}
		}
	}
}

// 批次 3 第 3 步：层级域配置的 sentinel 与值类型只有一份定义。
// 尤其要锁住 errors.Is 的「各层级均未命中 -> data=null」映射：共享层若另起
// 一个同文案 error 实例，编译、L.3 计数与 JSON 契约都看不出退化，只会把
// 合法空结果变成 500。
func TestBPMNDomainConfigValuesComeFromDTO(t *testing.T) {
	if service.ErrConfigNotFound != dto.ErrConfigNotFound {
		t.Errorf("ErrConfigNotFound 在共享层与 dto 之间分裂，errors.Is 空结果映射会静默失效")
	}

	bottom, err := os.ReadFile(filepath.Join("..", "..", "dto", "bpmn_config_inheritance_dto.go"))
	if err != nil {
		t.Fatalf("读取 dto/bpmn_config_inheritance_dto.go 失败：%v", err)
	}
	if len(bottom) < 200 {
		t.Fatalf("扫描器零命中保护：dto 配置值文件仅 %d 字节", len(bottom))
	}
	if strings.Contains(string(bottom), bpmnSharedImport) {
		t.Errorf("dto 不得反向依赖共享层")
	}
	flat := squashSpace(stripLineComments(string(bottom)))
	for _, decl := range []string{
		"var ErrConfigNotFound = errors.New(",
		"type ResolvedConfig struct {",
	} {
		if !strings.Contains(flat, squashSpace(decl)) {
			t.Errorf("dto 配置值文件缺少定义 %q（sentinel / 值类型必须只有一份）", decl)
		}
	}

	// JSON tag 是 /domain-configs/effective 的真实响应契约，改一个字母前端就读不到。
	raw, err := json.Marshal(dto.ResolvedConfig{
		Key: "k", Value: map[string]interface{}{"a": 1},
		Source: "tenant", InheritMode: "override", Version: 3,
	})
	if err != nil {
		t.Fatalf("序列化 ResolvedConfig 失败：%v", err)
	}
	for _, key := range []string{`"key"`, `"value"`, `"source"`, `"inheritMode"`, `"version"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("ResolvedConfig 的 JSON 契约丢失 key %s，实际 %s", key, string(raw))
		}
	}

	svc, err := os.ReadFile(filepath.Join("..", "..", "service", "config_inheritance_service.go"))
	if err != nil {
		t.Fatalf("读取 service/config_inheritance_service.go 失败：%v", err)
	}
	if len(svc) < 1000 {
		t.Fatalf("扫描器零命中保护：service/config_inheritance_service.go 仅 %d 字节", len(svc))
	}
	svcFlat := squashSpace(stripLineComments(string(svc)))
	for _, decl := range []string{
		"var ErrConfigNotFound = dto.ErrConfigNotFound",
		"type ResolvedConfig = dto.ResolvedConfig",
	} {
		if !strings.Contains(svcFlat, squashSpace(decl)) {
			t.Errorf("service/config_inheritance_service.go 丢失别名 %q", decl)
		}
	}
	if strings.Contains(svcFlat, "type ResolvedConfig struct") {
		t.Errorf("共享层重新定义了 ResolvedConfig（值类型身份分裂）")
	}
}

// TestBPMNPartialZeroTriggerGroup 锁住「归零是换成域内窄契约」而不是删文件、
// 或把共享层限定符换成别的等价写法。把 process_trigger.go 改回 HEAD 版本必须变红。
func TestBPMNPartialZeroTriggerGroup(t *testing.T) {
	pt := readBPMNFile(t, "process_trigger.go")
	if strings.Contains(pt, bpmnSharedImport) {
		t.Errorf("handlers/bpmn/process_trigger.go 重新引用共享层包，L.3 归零被破坏")
	}
	code := stripLineComments(pt)
	if strings.Contains(code, "service.") {
		t.Errorf("handlers/bpmn/process_trigger.go 残留共享层限定符（旧实现）")
	}
	for _, want := range []string{
		`\btriggerService\s+ProcessTriggerRunner\b`,
		`\bbindingService\s+ProcessBindingCatalog\b`,
		`\bconfigService\s+DomainConfigResolver\b`,
		`\bNewProcessTriggerHandler\(triggerService\s+ProcessTriggerRunner,\s*bindingService\s+ProcessBindingCatalog,\s*configService\s+DomainConfigResolver\)`,
		`errors\.Is\(err,\s*dto\.ErrConfigNotFound\)`,
	} {
		if !regexp.MustCompile(want).MatchString(code) {
			t.Errorf("process_trigger.go 缺少窄契约/值类型写法 %q", want)
		}
	}
	if strings.Contains(code, "service.ErrConfigNotFound") {
		t.Errorf("process_trigger.go 错误映射回退到共享层 sentinel（旧实现）")
	}

	contract := readBPMNFile(t, "contracts.go")
	if strings.Contains(contract, bpmnSharedImport) {
		t.Errorf("contracts.go 出现共享层包路径字面量（连注释也会让 L.3 计数 +1）")
	}
	for _, iface := range []string{"ProcessTriggerRunner", "ProcessBindingCatalog", "DomainConfigResolver"} {
		if !strings.Contains(contract, "type "+iface+" interface") {
			t.Errorf("contracts.go 未声明域内接口 %s", iface)
		}
	}
	if !strings.Contains(contract, "(*dto.ResolvedConfig, error)") {
		t.Errorf("contracts.go 的 GetEffectiveConfig 未使用 dto 值类型")
	}
}
