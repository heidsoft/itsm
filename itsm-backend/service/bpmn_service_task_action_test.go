package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 本文件守护「serviceTask 的 action 从 BPMN 定义走到 handler」这条链路。
//
// 背景（2026-10-24 生产栈实测）：
//   operational_commands 里 command_type='workflow.service_task.execute' 共 36 条，
//   36 条全是 dead_letter、0 条 succeeded。失败原因全部是
//   「服务任务缺少 action 配置（handler=incident_task, task=<nil>）」。
//
//   根因不在模板：所有内置模板都老老实实写了
//     <bpmn:metaData name="action">assign_incident</bpmn:metaData>
//   但这条链路两处断点——
//     1. enrichProcessElements 只解析 service_task_type，从不解析 action；
//     2. mergeServiceTaskVariables 只把 type/operationRef/cc* 抄进 variables，
//        从不写 variables["action"]。
//   而 incident/ticket/change/service_request/notification/webhook 六个 handler
//   全部只认 variables["action"]，于是永远取到空串 → default 分支 → 8 次重试 → 死信。
//   单元测试之所以全绿，是因为测试自己手工往 variables 里塞了 "action"，
//   恰好绕开了生产链路。

// 注：依赖 action 的 handler 集合直接复用生产代码的
// actionRequiringServiceTaskTypes（bpmn_lint_service.go），不在此处另立一份——
// 两份清单一旦分叉，测试就会为错误的集合背书。

// TestServiceTaskAction_ParsedFromBuiltinTemplates 是链路第 1 段：
// 内置模板里每个「依赖 action」的 serviceTask 必须把声明的动作解析进
// BPMNServiceTask.Action。两种合法写法都要认：
//   - <bpmn:metaData name="action">xxx</bpmn:metaData>（内建模板主约定）
//   - <camunda:inputParameter name="action">xxx</camunda:inputParameter>（Camunda 标准）
func TestServiceTaskAction_ParsedFromBuiltinTemplates(t *testing.T) {
	entries, err := os.ReadDir("bpmn")
	if err != nil {
		t.Fatalf("读取 bpmn 目录失败: %v", err)
	}

	parser := NewBPMNParser()
	serviceTasks := 0
	requiring := 0
	checkedFiles := 0

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".bpmn" {
			continue
		}
		name := e.Name()
		data, err := os.ReadFile(filepath.Join("bpmn", name))
		if err != nil {
			t.Fatalf("%s: 读取失败: %v", name, err)
		}
		defs, err := parser.ParseXML(data)
		if err != nil {
			t.Fatalf("%s: 解析失败: %v", name, err)
		}
		checkedFiles++
		text := string(data)

		for _, proc := range defs.Processes {
			for _, st := range proc.ServiceTasks {
				serviceTasks++
				if !actionRequiringServiceTaskTypes[st.ServiceTaskType] {
					continue
				}
				requiring++
				if strings.TrimSpace(st.Action) == "" {
					t.Errorf("%s: serviceTask %s（%s）handler=%s 依赖 variables[\"action\"]，"+
						"但模板里没有可解析的 action 声明——运行时必然 8 次重试后进 dead_letter",
						name, st.ID, st.Name, st.ServiceTaskType)
					continue
				}
				// action 必须与模板原文一致，防止解析串位
				if !strings.Contains(text, "name=\"action\">"+st.Action+"<") {
					t.Errorf("%s: serviceTask %s 的 action=%q 在模板中找不到对应声明（解析串位）",
						name, st.ID, st.Action)
				}
			}
		}
	}

	if checkedFiles == 0 {
		t.Fatal("未读到任何内置模板")
	}
	if requiring == 0 {
		t.Fatal("内置模板中没有任何依赖 action 的 serviceTask，链路断言失效")
	}
	t.Logf("serviceTask action 解析校验通过: %d 个模板 / %d 个服务任务（其中 %d 个依赖 action）",
		checkedFiles, serviceTasks, requiring)
}

// TestServiceTaskAction_MergedIntoVariables 是链路第 2 段：
// 解析出的 Action 必须落进 handler 实际读取的 variables["action"]。
func TestServiceTaskAction_MergedIntoVariables(t *testing.T) {
	task := &BPMNServiceTask{
		ID:              "Activity_AutoAssign",
		Name:            "自动分配",
		ServiceTaskType: "incident_task",
		Action:          "assign_incident",
	}

	variables := mergeServiceTaskVariables(map[string]interface{}{"incidentId": 15}, task)
	if got, _ := variables["action"].(string); got != "assign_incident" {
		t.Fatalf("variables[\"action\"] = %q，期望 assign_incident（handler 会取到空串并死信）", got)
	}
	// 实例变量必须原样保留，不能被 action 注入挤掉
	if variables["incidentId"] != 15 {
		t.Fatalf("实例变量 incidentId 丢失: %v", variables["incidentId"])
	}
}

// TestServiceTaskAction_TemplateWinsOverInstanceVariables 固定优先级：
// 节点上声明的 action 是该节点的配置，必须压过流程变量里的同名键，
// 否则一条带 action 的历史流程变量能把后续所有节点带偏。
func TestServiceTaskAction_TemplateWinsOverInstanceVariables(t *testing.T) {
	task := &BPMNServiceTask{ID: "Activity_AutoEscalate", Action: "escalate_incident"}
	variables := mergeServiceTaskVariables(
		map[string]interface{}{"action": "close_incident"}, task)
	if got, _ := variables["action"].(string); got != "escalate_incident" {
		t.Fatalf("variables[\"action\"] = %q，期望节点声明值 escalate_incident", got)
	}
}

// TestServiceTaskAction_NoActionKeepsInstanceVariables 兜底：
// 节点没声明 action 时不得抹掉流程变量里已有的 action（老流程兼容）。
func TestServiceTaskAction_NoActionKeepsInstanceVariables(t *testing.T) {
	task := &BPMNServiceTask{ID: "Activity_Legacy"}
	variables := mergeServiceTaskVariables(
		map[string]interface{}{"action": "update_incident"}, task)
	if got, _ := variables["action"].(string); got != "update_incident" {
		t.Fatalf("variables[\"action\"] = %q，期望保留实例变量 update_incident", got)
	}
}

// TestBPMNLint_ServiceTaskMissingActionIsError 验证部署期门禁：
// handler 依赖 variables["action"] 的 serviceTask 漏写 action 必须是 error，
// 把缺陷挡在部署前而不是让它到运行期重试 8 次进 dead_letter。
func TestBPMNLint_ServiceTaskMissingActionIsError(t *testing.T) {
	missing := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="p_action_missing" name="漏配action" isExecutable="true">
    <bpmn:startEvent id="S1"/>
    <bpmn:serviceTask id="ST1" name="自动分配" implementation="##WebService">
      <bpmn:extensionElements>
        <bpmn:metaData name="service_task_type">incident_task</bpmn:metaData>
      </bpmn:extensionElements>
    </bpmn:serviceTask>
    <bpmn:endEvent id="E1"/>
    <bpmn:sequenceFlow id="F1" sourceRef="S1" targetRef="ST1"/>
    <bpmn:sequenceFlow id="F2" sourceRef="ST1" targetRef="E1"/>
  </bpmn:process>
</bpmn:definitions>`

	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(missing))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !result.HasErrors {
		t.Fatal("handler=incident_task 的 serviceTask 未声明 action，必须报 error（否则运行期必然死信）")
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "tasks" && issue.Severity == "error" && issue.ElementID == "ST1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("应产生 tasks error 指向 ST1，实际 issues: %+v", result.Issues)
	}

	// 声明了 action 就不该再报这条
	declared := strings.Replace(missing,
		`<bpmn:metaData name="service_task_type">incident_task</bpmn:metaData>`,
		`<bpmn:metaData name="service_task_type">incident_task</bpmn:metaData>
        <bpmn:metaData name="action">assign_incident</bpmn:metaData>`, 1)
	result2, err := svc.LintBPMNXML([]byte(declared))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	for _, issue := range result2.Issues {
		if issue.Category == "tasks" && issue.Severity == "error" && issue.ElementID == "ST1" {
			t.Errorf("已声明 action 的 serviceTask 不应报 error: %s", issue.Message)
		}
	}

	// generic_task 不读 action，漏写也不该报这条（cloud_security_scan_flow 即此形态）
	generic := strings.Replace(missing, "incident_task", "generic_task", 1)
	result3, err := svc.LintBPMNXML([]byte(generic))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	for _, issue := range result3.Issues {
		if issue.Category == "tasks" && issue.Severity == "error" && issue.ElementID == "ST1" {
			t.Errorf("generic_task 不依赖 action，不应报 error: %s", issue.Message)
		}
	}
}

// TestServiceTaskAction_HandlerAcceptsParsedAction 是端到端断言：
// 用真实模板解析 + 真实合并，喂给真实 handler，不能再报「缺少 action 配置」。
// 这里只验证 action 被识别（走到业务分支），不关心业务结果，
// 因为业务依赖 DB，属于别的测试的职责。
func TestServiceTaskAction_HandlerAcceptsParsedAction(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("bpmn", "incident_emergency_flow.bpmn"))
	if err != nil {
		t.Fatalf("读取模板失败: %v", err)
	}
	defs, err := NewBPMNParser().ParseXML(data)
	if err != nil {
		t.Fatalf("解析模板失败: %v", err)
	}

	var target *BPMNServiceTask
	for _, proc := range defs.Processes {
		for _, st := range proc.ServiceTasks {
			if st.ID == "Activity_AutoAssign" {
				target = st
			}
		}
	}
	if target == nil {
		t.Fatal("模板中找不到 Activity_AutoAssign")
	}

	variables := mergeServiceTaskVariables(map[string]interface{}{}, target)
	action, _ := variables["action"].(string)

	// handler 的 switch 只认这套动作；空串必然落到 serviceTaskActionError
	allowed := []string{
		"create_incident", "assign_incident", "escalate_incident", "resolve_incident",
		"close_incident", "update_incident", "acknowledge_incident", "categorize_incident",
	}
	found := false
	for _, a := range allowed {
		if a == action {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Activity_AutoAssign 合并后 action=%q，不在 incident_task 支持的动作集内：%v",
			action, allowed)
	}
}

// ---------------------------------------------------------------------------
// 第二处断点：引擎注入 business_id，handler 读 <business_type>_id
//
// action 修好之后，生产栈立刻暴露下一层：同一个 serviceTask 的报错从
// 「缺少 action 配置」变成「无效的事件ID: 0」。原因是流程变量里只有
// business_id=16 / business_type=incident，而 incident_handler 读的是
// variables["incident_id"]。两边键名不同，从未有任何桥接。
// ---------------------------------------------------------------------------

// TestServiceTaskAction_BusinessIDAliasedPerType 覆盖六种业务类型的 ID 键映射。
func TestServiceTaskAction_BusinessIDAliasedPerType(t *testing.T) {
	cases := []struct {
		businessType string
		wantKeys     []string
	}{
		{"incident", []string{"incident_id"}},
		{"change", []string{"change_id"}},
		{"ticket", []string{"ticket_id"}},
		{"service_request", []string{"request_id", "service_request_id"}},
		{"problem", []string{"problem_id"}},
		{"release", []string{"release_id"}},
	}

	for _, tc := range cases {
		t.Run(tc.businessType, func(t *testing.T) {
			task := &BPMNServiceTask{ID: "ST", Action: "whatever"}
			variables := mergeServiceTaskVariables(map[string]interface{}{
				"business_id":   16,
				"business_type": tc.businessType,
				"business_key":  tc.businessType + ":16",
			}, task)

			for _, key := range tc.wantKeys {
				if got, ok := variableInt(variables[key]); !ok || got != 16 {
					t.Errorf("variables[%q] = %v，期望 16（handler 会读到 0 并报「无效的 ID: 0」）",
						key, variables[key])
				}
			}
			// 标准变量本身不能被改写
			if got, _ := variableInt(variables["business_id"]); got != 16 {
				t.Errorf("business_id 被改写为 %v", variables["business_id"])
			}
		})
	}
}

// TestServiceTaskAction_ExplicitIDWinsOverBusinessID 固定优先级：
// 调用方在 trigger.variables 里显式写的 incident_id（例如指向关联事件）
// 必须压过 business_id，否则会把「操作关联事件」变成「操作主事件」。
func TestServiceTaskAction_ExplicitIDWinsOverBusinessID(t *testing.T) {
	task := &BPMNServiceTask{ID: "ST", Action: "assign_incident"}
	variables := mergeServiceTaskVariables(map[string]interface{}{
		"business_id":   16,
		"business_type": "incident",
		"incident_id":   99,
	}, task)

	if got, _ := variableInt(variables["incident_id"]); got != 99 {
		t.Fatalf("incident_id = %d，期望保留显式值 99", got)
	}
}

// TestServiceTaskAction_NoAliasWithoutBusinessID 兜底：
// 没有 business_id（或非法值）时不得凭空造出 *_id，否则会把 0 写进库。
func TestServiceTaskAction_NoAliasWithoutBusinessID(t *testing.T) {
	task := &BPMNServiceTask{ID: "ST", Action: "assign_incident"}

	for _, businessID := range []interface{}{nil, 0, -1, "16"} {
		variables := mergeServiceTaskVariables(map[string]interface{}{
			"business_id":   businessID,
			"business_type": "incident",
		}, task)
		if _, exists := variables["incident_id"]; exists {
			t.Errorf("business_id=%v 时不应注入 incident_id（实际 %v）", businessID, variables["incident_id"])
		}
	}
}

// TestServiceTaskAction_UnknownBusinessTypeNoAlias 未知业务类型不注入任何键。
func TestServiceTaskAction_UnknownBusinessTypeNoAlias(t *testing.T) {
	task := &BPMNServiceTask{ID: "ST", Action: "x"}
	variables := mergeServiceTaskVariables(map[string]interface{}{
		"business_id":   16,
		"business_type": "unknown_domain",
	}, task)
	for key := range variables {
		if strings.HasSuffix(key, "_id") && key != "business_id" {
			t.Errorf("未知业务类型不应注入 %q", key)
		}
	}
}

// TestServiceTaskAction_BusinessTypeCaseInsensitive 大小写/空白不影响映射。
func TestServiceTaskAction_BusinessTypeCaseInsensitive(t *testing.T) {
	task := &BPMNServiceTask{ID: "ST", Action: "x"}
	variables := mergeServiceTaskVariables(map[string]interface{}{
		"business_id":   16,
		"business_type": "  Incident ",
	}, task)
	if got, _ := variableInt(variables["incident_id"]); got != 16 {
		t.Fatalf("大小写/空白未归一，incident_id = %v", variables["incident_id"])
	}
}
