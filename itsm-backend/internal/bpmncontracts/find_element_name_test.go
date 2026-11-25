package bpmncontracts

import (
	"encoding/xml"
	"testing"
)

// FindElementName 的回归锁。
//
// 事故背景：引擎写 ProcessInstance.CurrentActivityName 时曾经只认 userTask 与 endEvent，
// 其余节点把原始元素 ID 直接写进字段。prod 实测 incident_emergency_flow 停在
//   <bpmn:serviceTask id="Activity_AutoAssign" name="自动分配">
// XML 里有中文名，但实例列表/详情、工单详情、监控看板展示的都是 "Activity_AutoAssign"。
// 前端把这些界面直接呈现给非技术用户，等于让他们去读 XML 标识符。
//
// 这组用例锁两件事：
//  1. 每一种会成为 CurrentActivity 的节点类型都必须能解析出 name；
//  2. 找不到 / 无 name 时才回退空串（由调用方决定是否退回 elementID）。

// TestFindElementNameCoversEveryNodeType 逐一验证 14 类节点的 name 解析。
// 关键价值：新增节点类型若忘记接入 flowElements，这里会立刻红。
func TestFindElementNameCoversEveryNodeType(t *testing.T) {
	// 每类节点用真实 prod 流程里出现过的形态（id 是 camel/下划线混写，name 是中文）
	process := &BPMNProcess{
		ID:   "Process_1",
		Name: "流程",
		StartEvents: []*BPMNStartEvent{
			{ID: "StartEvent_1", Name: "事件报告"},
		},
		EndEvents: []*BPMNEndEvent{
			{ID: "EndEvent_1", Name: "事件完成"},
		},
		UserTasks: []*BPMNUserTask{
			{ID: "Activity_ManagerApproval", Name: "主管审批"},
		},
		ServiceTasks: []*BPMNServiceTask{
			{ID: "Activity_AutoAssign", Name: "自动分配"},
		},
		ScriptTasks: []*BPMNScriptTask{
			{ID: "Activity_Script", Name: "脚本执行"},
		},
		BusinessRuleTasks: []*BPMNBusinessRuleTask{
			{ID: "Activity_Rule", Name: "规则判定"},
		},
		ManualTasks: []*BPMNManualTask{
			{ID: "Activity_Manual", Name: "人工处理"},
		},
		CallActivities: []*BPMNCallActivity{
			{ID: "Activity_Call", Name: "调用子流程"},
		},
		ExclusiveGateways: []*BPMNExclusiveGateway{
			{ID: "Gateway_Resolved", Name: "问题已定位?"},
		},
		ParallelGateways: []*BPMNParallelGateway{
			{ID: "Gateway_Parallel", Name: "并行分支"},
		},
		InclusiveGateways: []*BPMNInclusiveGateway{
			{ID: "Gateway_Inclusive", Name: "包容分支"},
		},
		SubProcesses: []*BPMNSubProcess{
			{ID: "Activity_SubProcess", Name: "子流程"},
		},
		BoundaryEvents: []*BPMNBoundaryEvent{
			{ID: "Boundary_Timer", Name: "超时边界"},
		},
		IntermediateEvents: []*BPMNIntermediateEvent{
			{ID: "Intermediate_Notify", Name: "中间通知"},
		},
	}

	cases := []struct {
		nodeType string
		id       string
		want     string
	}{
		{"startEvent", "StartEvent_1", "事件报告"},
		{"endEvent", "EndEvent_1", "事件完成"},
		{"userTask", "Activity_ManagerApproval", "主管审批"},
		{"serviceTask", "Activity_AutoAssign", "自动分配"},
		{"scriptTask", "Activity_Script", "脚本执行"},
		{"businessRuleTask", "Activity_Rule", "规则判定"},
		{"manualTask", "Activity_Manual", "人工处理"},
		{"callActivity", "Activity_Call", "调用子流程"},
		{"exclusiveGateway", "Gateway_Resolved", "问题已定位?"},
		{"parallelGateway", "Gateway_Parallel", "并行分支"},
		{"inclusiveGateway", "Gateway_Inclusive", "包容分支"},
		{"subProcess", "Activity_SubProcess", "子流程"},
		{"boundaryEvent", "Boundary_Timer", "超时边界"},
		{"intermediateCatchEvent", "Intermediate_Notify", "中间通知"},
	}

	if len(cases) != 14 {
		t.Fatalf("节点类型清单必须覆盖全部 14 类，当前 %d 类", len(cases))
	}
	for _, c := range cases {
		t.Run(c.nodeType, func(t *testing.T) {
			if got := process.FindElementName(c.id); got != c.want {
				t.Fatalf("%s 的 name 解析错误：want %q, got %q（回退成原始 ID 就是本用例要拦的回归）",
					c.nodeType, c.want, got)
			}
		})
	}
}

// TestFindElementNameReturnsEmptyForUnknownOrUnnamed 验证回退口径。
func TestFindElementNameReturnsEmptyForUnknownOrUnnamed(t *testing.T) {
	process := &BPMNProcess{
		ID:            "Process_1",
		ServiceTasks:  []*BPMNServiceTask{{ID: "Activity_NoName"}},
		SequenceFlows: []*BPMNSequenceFlow{{ID: "Flow_1"}},
		DataObjects:   []*BPMNDataObject{{ID: "Data_1", Name: "数据"}},
	}

	// 元素存在但没有 name：返回空串，调用方才会退回 elementID
	if got := process.FindElementName("Activity_NoName"); got != "" {
		t.Fatalf("无 name 的元素应返回空串，got %q", got)
	}
	// 元素不存在：空串
	if got := process.FindElementName("Not_Exist"); got != "" {
		t.Fatalf("不存在的 ID 应返回空串，got %q", got)
	}
	// SequenceFlow / DataObject 不会成为 CurrentActivity，不进解析集合
	if got := process.FindElementName("Flow_1"); got != "" {
		t.Fatalf("SequenceFlow 不参与 CurrentActivity 解析，got %q", got)
	}
	if got := process.FindElementName("Data_1"); got != "" {
		t.Fatalf("DataObject 不参与 CurrentActivity 解析，got %q", got)
	}
	// 空 ID / nil 流程不得 panic
	if got := process.FindElementName(""); got != "" {
		t.Fatalf("空 ID 应返回空串，got %q", got)
	}
	var nilProcess *BPMNProcess
	if got := nilProcess.FindElementName("Activity_NoName"); got != "" {
		t.Fatalf("nil Process 应返回空串，got %q", got)
	}
}

// TestFindElementNameFromRealProdXML 用 prod 流程的真实 XML 端到端验证。
//
// 直接内联 incident_emergency_flow 的关键片段（prod 实测 2026-11-15）：
// 它是当前唯一停在 serviceTask 上的流程，也是本 bug 被发现的现场。
// 断言刻意锚在 "自动分配" 这个值上——旧实现只会返回 "Activity_AutoAssign"。
func TestFindElementNameFromRealProdXML(t *testing.T) {
	const xmlSrc = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1">
  <bpmn:process id="incident_emergency_flow" name="紧急事件流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="事件报告"/>
    <bpmn:serviceTask id="Activity_AutoAssign" name="自动分配"/>
    <bpmn:userTask id="Activity_ManagerApproval" name="主管审批"/>
    <bpmn:exclusiveGateway id="Gateway_Resolved" name="问题已定位?"/>
    <bpmn:endEvent id="EndEvent_1" name="事件完成"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Activity_AutoAssign"/>
  </bpmn:process>
</bpmn:definitions>`

	var defs BPMNDefinitions
	if err := xml.Unmarshal([]byte(xmlSrc), &defs); err != nil {
		t.Fatalf("解析真实 XML 失败: %v", err)
	}
	if len(defs.Processes) != 1 {
		t.Fatalf("应解析出 1 个 Process，实际 %d", len(defs.Processes))
	}
	process := defs.Processes[0]

	// 事故现场：serviceTask 必须有中文名，不能是英文 ID
	if got := process.FindElementName("Activity_AutoAssign"); got != "自动分配" {
		t.Fatalf("prod 事故节点必须是「自动分配」，got %q（返回 Activity_AutoAssign 即 bug 复现）", got)
	}
	// 其余节点类型同样可解析
	for id, want := range map[string]string{
		"StartEvent_1":             "事件报告",
		"Activity_ManagerApproval": "主管审批",
		"Gateway_Resolved":         "问题已定位?",
		"EndEvent_1":               "事件完成",
	} {
		if got := process.FindElementName(id); got != want {
			t.Fatalf("%s 应为 %q，got %q", id, want, got)
		}
	}
}
