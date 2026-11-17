package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lintSubProcessBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:subProcess id="Sub_1" name="审批子流程">
      <bpmn:userTask id="SubTask_1" name="内嵌审批"/>
    </bpmn:subProcess>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="F_0" sourceRef="Start_1" targetRef="Sub_1"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Sub_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

func TestLintRule9_SubProcessRejected(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintSubProcessBPMN))
	require.NoError(t, err)

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "structure" && strings.Contains(issue.Message, "子流程") && strings.Contains(issue.Message, "静默跳过") {
			found = true
			assert.Equal(t, "error", issue.Severity)
			assert.Equal(t, "Sub_1", issue.ElementID)
		}
	}
	assert.True(t, found, "子流程必须被 error 级拦截")
	assert.True(t, result.HasErrors)
}

const lintCallActivityBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_2" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_2" isExecutable="true">
    <bpmn:startEvent id="Start_2" name="开始"/>
    <bpmn:callActivity id="Call_1" name="复用审批流" calledElement="approval_flow"/>
    <bpmn:endEvent id="End_2" name="结束"/>
    <bpmn:sequenceFlow id="F2_0" sourceRef="Start_2" targetRef="Call_1"/>
    <bpmn:sequenceFlow id="F2_1" sourceRef="Call_1" targetRef="End_2"/>
  </bpmn:process>
</bpmn:definitions>`

func TestLintRule9_CallActivityRejected(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintCallActivityBPMN))
	require.NoError(t, err)

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "structure" && strings.Contains(issue.Message, "调用活动") && issue.ElementID == "Call_1" {
			found = true
			assert.Equal(t, "error", issue.Severity)
		}
	}
	assert.True(t, found, "调用活动必须被 error 级拦截")
}

const lintInclusiveDefaultBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_3" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_3" isExecutable="true">
    <bpmn:startEvent id="Start_3" name="开始"/>
    <bpmn:inclusiveGateway id="IG_1" name="包容网关"/>
    <bpmn:userTask id="UT_A" name="分支A"/>
    <bpmn:userTask id="UT_B" name="分支B"/>
    <bpmn:endEvent id="End_3" name="结束"/>
    <bpmn:sequenceFlow id="F3_0" sourceRef="Start_3" targetRef="IG_1"/>
    <bpmn:sequenceFlow id="F3_1" sourceRef="IG_1" targetRef="UT_A"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">amount &gt; 100</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="F3_2" sourceRef="IG_1" targetRef="UT_B"><bpmn:conditionExpression xsi:type="bpmn:tFormalExpression" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">amount &gt; 1000</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="F3_3" sourceRef="UT_A" targetRef="End_3"/>
    <bpmn:sequenceFlow id="F3_4" sourceRef="UT_B" targetRef="End_3"/>
  </bpmn:process>
</bpmn:definitions>`

func TestLintInclusiveGateway_NoDefault_Warns(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintInclusiveDefaultBPMN))
	require.NoError(t, err)

	found := false
	for _, issue := range result.Issues {
		if issue.Category == "gateways" && issue.ElementID == "IG_1" && strings.Contains(issue.Message, "default 分支") {
			found = true
			assert.Equal(t, "warning", issue.Severity)
		}
	}
	assert.True(t, found, "多出边包容网关缺 default 应产生 warning")
}

func TestLintInclusiveGateway_WithDefault_NoWarn(t *testing.T) {
	xml := strings.Replace(lintInclusiveDefaultBPMN,
		`<bpmn:inclusiveGateway id="IG_1" name="包容网关"/>`,
		`<bpmn:inclusiveGateway id="IG_1" name="包容网关" default="F3_2"/>`, 1)
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(xml))
	require.NoError(t, err)

	for _, issue := range result.Issues {
		if issue.Category == "gateways" && issue.ElementID == "IG_1" {
			t.Fatalf("声明了 default 的包容网关不应再告警: %+v", issue)
		}
	}
}
