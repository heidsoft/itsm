package layering

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/internal/bpmncontracts"
	"itsm-backend/service"
)

// BPMN 纯契约归位专项回归锁（G0，2026-11-11）。
//
// 本批把 service/ 顶层两个生产实现（bpmn_types.go 425 行纯 XML 值类型与
// bpmn_process_trigger_interface.go 4 个流程服务接口）git mv 进
// internal/bpmncontracts，service/bpmn_types.go 保留为 28 个类型别名的过渡层；
// 同时退役零消费方 workflow_helper.go。
//
// 假绿风险不在编译，而在「编译全绿但语义已经分叉」：
//   - 别名被换成同形匿名 struct：类型身份分裂，xml 标签仍在，go build 全绿；
//   - 契约包里的 xml tag 被改动：解析结果静默变空，编译期毫无察觉；
//   - 旧文件名 / 退役符号 / 基线行复活，L.1 与 C.6.4 棘轮留免检空间；
//   - 组合根被"顺手"改成直连契约包，绕过 service 侧过渡 API。
//
// 因此锁同时覆盖：编译期双向断言、reflect.Type 身份、XML 往返行为、
// 逐行精确别名比对、退役符号全目录扫描、两份基线上界棘轮（正向不误报/回退必红）。

var bcAliasNames = []string{
	"BPMNElement", "BPMNProcess", "BPMNOutgoingRef", "BPMNStartEvent", "BPMNEndEvent",
	"BPMNCompensateEventDefinition", "BPMNTimerEventDefinition", "BPMNUserTask", "BPMNServiceTask",
	"BPMNScriptTask", "BPMNExclusiveGateway", "BPMNParallelGateway", "BPMNInclusiveGateway",
	"BPMNSequenceFlow", "BPMNConditionExpression", "BPMNSubProcess", "BPMNBusinessRuleTask",
	"BPMNManualTask", "BPMNCallActivity", "BPMNDataObject", "BPMNDataStore", "BPMNBoundaryEvent",
	"BPMNIntermediateEvent", "BPMNDefinitions",
	"ProcessTriggerServiceInterface", "ProcessBindingServiceInterface",
	"ServiceTaskHandlerInterface", "ProcessCallbackServiceInterface",
}

// 编译期断言：service 侧必须与契约包是同一类型，而不是同形副本；接口也双向可赋值。
var (
	_ *service.BPMNProcess       = (*bpmncontracts.BPMNProcess)(nil)
	_ *bpmncontracts.BPMNProcess = (*service.BPMNProcess)(nil)
	_ service.BPMNElement        = (*bpmncontracts.BPMNUserTask)(nil)
	_ bpmncontracts.BPMNElement  = (*service.BPMNUserTask)(nil)

	_ service.ProcessTriggerServiceInterface        = (bpmncontracts.ProcessTriggerServiceInterface)(nil)
	_ bpmncontracts.ProcessTriggerServiceInterface  = (service.ProcessTriggerServiceInterface)(nil)
	_ service.ServiceTaskHandlerInterface           = (bpmncontracts.ServiceTaskHandlerInterface)(nil)
	_ bpmncontracts.ProcessCallbackServiceInterface = (service.ProcessCallbackServiceInterface)(nil)
)

func bcReadBackend(t *testing.T, parts ...string) string {
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

func bcReadRepo(t *testing.T, parts ...string) string {
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

// bcHasAlias 逐行比对 type 块里的别名：lhs 必须是声明名本身，rhs 必须精确等于
// 目标限定符；gofmt 对齐空格不参与比较。别名退化成同形 struct 时立即失配。
func bcHasAlias(src, name, target string) bool {
	for _, line := range strings.Split(src, "\n") {
		l := strings.TrimSpace(line)
		if l != name && !strings.HasPrefix(l, name+" ") {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == name && strings.TrimSpace(parts[1]) == target {
			return true
		}
	}
	return false
}

// bcRetiredSymbolCount 只扫 service/ 顶层（含测试），不递归子包（子包各自的
// contains 辅助函数不属本批退役范围）。返回命中文件数与扫描文件数，供零命中保护。
func bcRetiredSymbolCount(t *testing.T, symbols []string) (hits []string, scanned int) {
	t.Helper()
	dir := filepath.Join("..", "..", "service")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 service/ 失败：%v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取 service/%s 失败：%v", e.Name(), err)
		}
		scanned++
		for _, line := range strings.Split(string(body), "\n") {
			for _, sym := range symbols {
				if strings.HasPrefix(strings.TrimSpace(line), sym) {
					hits = append(hits, e.Name()+": "+strings.TrimSpace(line))
				}
			}
		}
	}
	return hits, scanned
}

// bcIntCeilingOK 解析 "<key>=<n>" 的上界棘轮：只允许继续下降，回退必红。
func bcIntCeilingOK(body, key string, ceiling int) bool {
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, key+"=") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, key+"="))
		if len(fields) == 0 {
			return false
		}
		v, err := strconv.Atoi(fields[0])
		if err != nil {
			return false
		}
		return v >= 0 && v <= ceiling
	}
	return false
}

// bcL1Count 统计 L.1 基线里的文件名行（排除注释与空行）。
func bcL1Count(body string) int {
	n := 0
	for _, line := range strings.Split(body, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		n++
	}
	return n
}

func TestBPMNContractsAreInfraOwned(t *testing.T) {
	// 1) 旧文件名零复活；别名过渡文件仍在（不新增生产文件）。
	for _, fn := range []string{"bpmn_process_trigger_interface.go", "workflow_helper.go"} {
		if _, err := os.Stat(filepath.Join("..", "..", "service", fn)); !os.IsNotExist(err) {
			t.Errorf("service/%s 复活：实现本体必须只在 internal/bpmncontracts", fn)
		}
	}
	alias := bcReadBackend(t, "service", "bpmn_types.go")
	if !strings.HasPrefix(alias, "package service\n") {
		t.Error("service/bpmn_types.go 包名漂移")
	}

	// 2) 契约包两份实现归位、包名正确、不得反向依赖上层。
	for _, fn := range []string{"types.go", "interfaces.go"} {
		src := bcReadBackend(t, "internal", "bpmncontracts", fn)
		if !strings.HasPrefix(src, "package bpmncontracts\n") {
			t.Errorf("internal/bpmncontracts/%s 包名漂移", fn)
		}
		if strings.Contains(src, `"itsm-backend/service"`) {
			t.Errorf("internal/bpmncontracts/%s 反向依赖 service 包", fn)
		}
		if strings.Contains(src, `"itsm-backend/handlers/`) {
			t.Errorf("internal/bpmncontracts/%s 反向依赖 handlers 域包", fn)
		}
	}
	impl := bcReadBackend(t, "internal", "bpmncontracts", "types.go")
	for _, decl := range []string{"type BPMNProcess struct {", "type BPMNUserTask struct {", "type BPMNDefinitions struct {"} {
		if !strings.Contains(impl, decl) {
			t.Errorf("契约包缺少实现声明 %q", decl)
		}
	}
	if !strings.Contains(impl, "func (e *BPMNUserTask) GetType() string") {
		t.Error("契约包缺少 BPMNUserTask.GetType 方法（方法必须留在类型定义包内）")
	}
	ifaces := bcReadBackend(t, "internal", "bpmncontracts", "interfaces.go")
	for _, name := range []string{"ProcessTriggerServiceInterface", "ProcessBindingServiceInterface", "ServiceTaskHandlerInterface", "ProcessCallbackServiceInterface"} {
		if !strings.Contains(ifaces, "type "+name+" interface {") {
			t.Errorf("契约包缺少接口声明 %s", name)
		}
	}

	// 3) service 侧只保留 28 个别名，禁止任何实现复制（逐行精确，不取并集）。
	for _, name := range bcAliasNames {
		if !bcHasAlias(alias, name, "bpmncontracts."+name) {
			t.Errorf("service/bpmn_types.go 缺少类型别名 %s = bpmncontracts.%s", name, name)
		}
	}
	if strings.Contains(alias, "struct {") || strings.Contains(alias, "func (") || strings.Contains(alias, "interface {") {
		t.Error("service 侧复制了契约实现，类型身份已分裂")
	}
	if !strings.Contains(alias, `"itsm-backend/internal/bpmncontracts"`) {
		t.Error("别名过渡层未 import 契约包")
	}

	// 4) 运行时类型身份：24 个值类型 + 4 个接口全部同一身份。
	concrete := []struct {
		name      string
		got, want reflect.Type
	}{
		{"BPMNProcess", reflect.TypeOf(service.BPMNProcess{}), reflect.TypeOf(bpmncontracts.BPMNProcess{})},
		{"BPMNOutgoingRef", reflect.TypeOf(service.BPMNOutgoingRef{}), reflect.TypeOf(bpmncontracts.BPMNOutgoingRef{})},
		{"BPMNStartEvent", reflect.TypeOf(service.BPMNStartEvent{}), reflect.TypeOf(bpmncontracts.BPMNStartEvent{})},
		{"BPMNEndEvent", reflect.TypeOf(service.BPMNEndEvent{}), reflect.TypeOf(bpmncontracts.BPMNEndEvent{})},
		{"BPMNCompensateEventDefinition", reflect.TypeOf(service.BPMNCompensateEventDefinition{}), reflect.TypeOf(bpmncontracts.BPMNCompensateEventDefinition{})},
		{"BPMNTimerEventDefinition", reflect.TypeOf(service.BPMNTimerEventDefinition{}), reflect.TypeOf(bpmncontracts.BPMNTimerEventDefinition{})},
		{"BPMNUserTask", reflect.TypeOf(service.BPMNUserTask{}), reflect.TypeOf(bpmncontracts.BPMNUserTask{})},
		{"BPMNServiceTask", reflect.TypeOf(service.BPMNServiceTask{}), reflect.TypeOf(bpmncontracts.BPMNServiceTask{})},
		{"BPMNScriptTask", reflect.TypeOf(service.BPMNScriptTask{}), reflect.TypeOf(bpmncontracts.BPMNScriptTask{})},
		{"BPMNExclusiveGateway", reflect.TypeOf(service.BPMNExclusiveGateway{}), reflect.TypeOf(bpmncontracts.BPMNExclusiveGateway{})},
		{"BPMNParallelGateway", reflect.TypeOf(service.BPMNParallelGateway{}), reflect.TypeOf(bpmncontracts.BPMNParallelGateway{})},
		{"BPMNInclusiveGateway", reflect.TypeOf(service.BPMNInclusiveGateway{}), reflect.TypeOf(bpmncontracts.BPMNInclusiveGateway{})},
		{"BPMNSequenceFlow", reflect.TypeOf(service.BPMNSequenceFlow{}), reflect.TypeOf(bpmncontracts.BPMNSequenceFlow{})},
		{"BPMNConditionExpression", reflect.TypeOf(service.BPMNConditionExpression{}), reflect.TypeOf(bpmncontracts.BPMNConditionExpression{})},
		{"BPMNSubProcess", reflect.TypeOf(service.BPMNSubProcess{}), reflect.TypeOf(bpmncontracts.BPMNSubProcess{})},
		{"BPMNBusinessRuleTask", reflect.TypeOf(service.BPMNBusinessRuleTask{}), reflect.TypeOf(bpmncontracts.BPMNBusinessRuleTask{})},
		{"BPMNManualTask", reflect.TypeOf(service.BPMNManualTask{}), reflect.TypeOf(bpmncontracts.BPMNManualTask{})},
		{"BPMNCallActivity", reflect.TypeOf(service.BPMNCallActivity{}), reflect.TypeOf(bpmncontracts.BPMNCallActivity{})},
		{"BPMNDataObject", reflect.TypeOf(service.BPMNDataObject{}), reflect.TypeOf(bpmncontracts.BPMNDataObject{})},
		{"BPMNDataStore", reflect.TypeOf(service.BPMNDataStore{}), reflect.TypeOf(bpmncontracts.BPMNDataStore{})},
		{"BPMNBoundaryEvent", reflect.TypeOf(service.BPMNBoundaryEvent{}), reflect.TypeOf(bpmncontracts.BPMNBoundaryEvent{})},
		{"BPMNIntermediateEvent", reflect.TypeOf(service.BPMNIntermediateEvent{}), reflect.TypeOf(bpmncontracts.BPMNIntermediateEvent{})},
		{"BPMNDefinitions", reflect.TypeOf(service.BPMNDefinitions{}), reflect.TypeOf(bpmncontracts.BPMNDefinitions{})},
	}
	for _, p := range concrete {
		if p.got != p.want {
			t.Errorf("service.%s 与 bpmncontracts.%s 不再是同一类型", p.name, p.name)
		}
	}
	abstract := []struct {
		name      string
		got, want reflect.Type
	}{
		{"BPMNElement", reflect.TypeOf((*service.BPMNElement)(nil)).Elem(), reflect.TypeOf((*bpmncontracts.BPMNElement)(nil)).Elem()},
		{"ProcessTriggerServiceInterface", reflect.TypeOf((*service.ProcessTriggerServiceInterface)(nil)).Elem(), reflect.TypeOf((*bpmncontracts.ProcessTriggerServiceInterface)(nil)).Elem()},
		{"ProcessBindingServiceInterface", reflect.TypeOf((*service.ProcessBindingServiceInterface)(nil)).Elem(), reflect.TypeOf((*bpmncontracts.ProcessBindingServiceInterface)(nil)).Elem()},
		{"ServiceTaskHandlerInterface", reflect.TypeOf((*service.ServiceTaskHandlerInterface)(nil)).Elem(), reflect.TypeOf((*bpmncontracts.ServiceTaskHandlerInterface)(nil)).Elem()},
		{"ProcessCallbackServiceInterface", reflect.TypeOf((*service.ProcessCallbackServiceInterface)(nil)).Elem(), reflect.TypeOf((*bpmncontracts.ProcessCallbackServiceInterface)(nil)).Elem()},
	}
	for _, p := range abstract {
		if p.got != p.want {
			t.Errorf("service.%s 与 bpmncontracts.%s 不再是同一接口", p.name, p.name)
		}
	}

	// 5) XML 往返行为锁：xml tag 归位后必须与 HEAD 语义一致（改 tag 会静默变空）。
	const sample = `<definitions id="def1" name="demo" targetNamespace="http://example.com">` +
		`<process id="p1" name="流程"><userTask id="task1" name="审批" assignee="u1" approvalLevel="2"/>` +
		`<serviceTask id="svc1" name="通知"/></process></definitions>`
	var defs service.BPMNDefinitions
	if err := xml.Unmarshal([]byte(sample), &defs); err != nil {
		t.Fatalf("BPMNDefinitions XML 解析失败：%v", err)
	}
	if len(defs.Processes) != 1 || defs.Processes[0].ID != "p1" {
		t.Fatalf("BPMNDefinitions/Process XML tag 漂移：%+v", defs.Processes)
	}
	if len(defs.Processes[0].UserTasks) != 1 {
		t.Fatalf("BPMNUserTask XML tag 漂移：%d 个", len(defs.Processes[0].UserTasks))
	}
	ut := defs.Processes[0].UserTasks[0]
	if ut.ID != "task1" || ut.Name != "审批" || ut.ApprovalLevel != 2 || ut.GetID() != "task1" || ut.GetType() != "UserTask" {
		t.Errorf("BPMNUserTask 字段/方法语义漂移：%+v", ut)
	}
	if len(defs.Processes[0].ServiceTasks) != 1 || defs.Processes[0].ServiceTasks[0].GetType() != "ServiceTask" {
		t.Error("BPMNServiceTask XML tag / GetType 语义漂移")
	}

	// 6) 退役符号零复活（扫描器零命中直接失败）。
	retired := []string{
		"func MapProcessStatus(", "func MapProcessStatusToTicketStatus(", "func BuildBusinessKey(",
		"func contains(", "func containsSubstring(",
	}
	hits, scanned := bcRetiredSymbolCount(t, retired)
	if scanned < 100 {
		t.Fatalf("退役符号扫描器零命中保护：只扫到 %d 个 service/ 顶层 .go", scanned)
	}
	if len(hits) > 0 {
		t.Errorf("workflow_helper.go 退役符号复活：%s", strings.Join(hits, " | "))
	}

	// 7) 组合根不得绕开 service 侧过渡 API（app.go 本批零改动）。
	app := bcReadBackend(t, "internal", "bootstrap", "app.go")
	if strings.Contains(app, `"itsm-backend/internal/bpmncontracts"`) {
		t.Error("app.go 直连 internal/bpmncontracts，绕过 service 侧过渡 API")
	}

	// 8) L.1 旧行零复活 + 过渡行仍在；两份棘轮上界（正向不误报 / 反向必红）。
	baseline := bcReadRepo(t, "scripts", "docs-gate", "service-prod-baseline.txt")
	for _, line := range strings.Split(baseline, "\n") {
		l := strings.TrimSpace(line)
		if l == "bpmn_process_trigger_interface.go" || l == "workflow_helper.go" {
			t.Errorf("L.1 基线复活已迁出/退役文件 %q", l)
		}
	}
	hasTransitionRow := false
	for _, line := range strings.Split(baseline, "\n") {
		if strings.TrimSpace(line) == "bpmn_types.go" {
			hasTransitionRow = true
		}
	}
	if !hasTransitionRow {
		t.Error("L.1 基线缺少 bpmn_types.go 过渡行")
	}
	if n := bcL1Count(baseline); n > 126 || n < 100 {
		t.Errorf("L.1 上界棘轮：登记 %d 行，必须 ≤126（128 → 126）且扫描器有效", n)
	}
	if !bcIntCeilingOK("service_go_files=332 # 继续下降", "service_go_files", 333) {
		t.Error("C.6.4 棘轮正向验证失败：继续下降 333→332 不应误报")
	}
	if bcIntCeilingOK("service_go_files=335 # 回退", "service_go_files", 333) {
		t.Error("C.6.4 棘轮反向验证失败：回退到 335 必须报红")
	}
	drift := bcReadRepo(t, "scripts", "docs-gate", "product-surface-baseline.txt")
	if !bcIntCeilingOK(drift, "service_go_files", 333) {
		t.Error("C.6.4 service_go_files 上界必须 ≤333（335 → 333，后续只许继续下降）")
	}
}
