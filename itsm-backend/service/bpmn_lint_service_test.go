package service

import (
	"os"
	"testing"
)

// loadTestBPMN 读取内置流程模板（真实素材，非手写精简 XML）
func loadTestBPMN(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("bpmn/" + name)
	if err != nil {
		t.Fatalf("读取 BPMN 模板失败: %v", err)
	}
	return data
}

// TestBPMNLint_ValidTemplate 内置模板应零 error（warning 可有可无）
func TestBPMNLint_ValidTemplate(t *testing.T) {
	svc := NewBPMNLintService()
	for _, name := range []string{
		"ticket_assignment_flow.bpmn",
		"ticket_general_flow.bpmn",
		"change_normal_flow.bpmn",
		"incident_emergency_flow.bpmn",
		"release_approval_flow.bpmn",
		"problem_management_flow.bpmn",
	} {
		t.Run(name, func(t *testing.T) {
			result, err := svc.LintBPMNXML(loadTestBPMN(t, name))
			if err != nil {
				t.Fatalf("合法模板不应解析失败: %v", err)
			}
			if result.HasErrors {
				for _, issue := range result.Issues {
					if issue.Severity == "error" {
						t.Errorf("内置模板出现 error: [%s/%s] %s", issue.Category, issue.ElementID, issue.Message)
					}
				}
			}
		})
	}
}

// TestBPMNLint_MissingStartAndEnd 缺起止事件被 parser 流程级校验直接拒绝（LintBPMNXML 返回 error）
// parser 的 ValidateBPMNProcess 已覆盖"至少一个开始/结束事件"，lint 端的该规则为双保险，
// 正常路径下无法构造"绕过 parser 又触发 lint 规则"的输入，故此处直接验证 parser 行为。
func TestBPMNLint_MissingStartAndEnd(t *testing.T) {
	svc := NewBPMNLintService()

	// 缺开始事件
	xmlNoStart := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="broken" name="坏流程" isExecutable="true">
    <bpmn:userTask id="Task_1" name="孤立任务"/>
  </bpmn:process>
</bpmn:definitions>`
	if _, err := svc.LintBPMNXML([]byte(xmlNoStart)); err == nil {
		t.Fatal("缺开始事件应被 parser 校验拒绝（返回 error）")
	}

	// 缺结束事件
	xmlNoEnd := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="broken3" name="无终点" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:userTask id="Task_1" name="任务" assignee="bob"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Start_1" targetRef="Task_1"/>
  </bpmn:process>
</bpmn:definitions>`
	if _, err := svc.LintBPMNXML([]byte(xmlNoEnd)); err == nil {
		t.Fatal("缺结束事件应被 parser 校验拒绝（返回 error）")
	}
}

// TestBPMNLint_UnreachableAndDeadEnd 不可达节点与死路应报 warning
func TestBPMNLint_UnreachableAndDeadEnd(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="broken2" name="断链流程" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:userTask id="Task_Lonely" name="孤岛任务" assignee="alice"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Start_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(xml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if result.WarningCount < 2 {
		t.Fatalf("孤岛任务应报 不可达+死路 2 个 warning，实际 %d: %+v", result.WarningCount, result.Issues)
	}
	if result.HasErrors {
		t.Fatal("拓扑问题为 warning 级，不应 HasErrors")
	}
}

// TestBPMNLint_UserTaskNoAssignee 用户任务缺执行人应报 warning
func TestBPMNLint_UserTaskNoAssignee(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="warn1" name="无执行人" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:userTask id="Task_NoOwner" name="没人管"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Start_1" targetRef="Task_NoOwner"/>
    <bpmn:sequenceFlow id="F_2" sourceRef="Task_NoOwner" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(xml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && issue.ElementID == "Task_NoOwner" {
			found = true
		}
	}
	if !found {
		t.Fatalf("未配置执行人的用户任务应报 warning，issues=%+v", result.Issues)
	}
	if result.HasErrors {
		t.Fatal("仅缺执行人不应 error")
	}
}

// TestBPMNLint_GatewaySingleOut 网关单出边应报 warning
func TestBPMNLint_GatewaySingleOut(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="warn2" name="单出边网关" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:parallelGateway id="GW_1" name="假分叉"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Start_1" targetRef="GW_1"/>
    <bpmn:sequenceFlow id="F_2" sourceRef="GW_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(xml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == "warning" && issue.ElementID == "GW_1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("单出边并行网关应报 warning，issues=%+v", result.Issues)
	}
}

// TestBPMNLint_InvalidXML 非法 XML 应返回 error 而非 panic
func TestBPMNLint_InvalidXML(t *testing.T) {
	svc := NewBPMNLintService()
	if _, err := svc.LintBPMNXML([]byte("not xml at all <<<")); err == nil {
		t.Fatal("非法 XML 必须返回 error")
	}
}

// TestBPMNLint_MissingNamespace 缺 BPMN 命名空间应报 error
func TestBPMNLint_MissingNamespace(t *testing.T) {
	xml := `<?xml version="1.0"?><root><process id="p"/></root>`
	svc := NewBPMNLintService()
	if _, err := svc.LintBPMNXML([]byte(xml)); err == nil {
		t.Fatal("缺命名空间必须返回 error")
	}
}
