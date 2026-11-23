package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
)

// 规则组 10（审计 P1-06 的部署期缓解）：拒绝引擎令牌语义无法正确执行的图形。
// 引擎的汇聚用「节点完成过」、结束用「到达即整实例 completed」，没有 token/occurrence。

// lintTokenSemanticsR1 夹具：并行分叉 → 两分支 → 收敛型包容网关（入度 2）→ end。
// 包容分叉只激活命中分支，包容汇聚却等全部静态入边 → 未激活分支永远不完成 → 死锁。
const lintInclusiveJoinBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_IJ" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_IJ" isExecutable="true">
    <bpmn:startEvent id="Start_IJ"/>
    <bpmn:parallelGateway id="Fork_IJ"/>
    <bpmn:userTask id="Task_A" name="分支A"/>
    <bpmn:userTask id="Task_B" name="分支B"/>
    <bpmn:inclusiveGateway id="Join_IJ" name="包容汇聚"/>
    <bpmn:endEvent id="End_IJ"/>
    <bpmn:sequenceFlow id="IJ_0" sourceRef="Start_IJ" targetRef="Fork_IJ"/>
    <bpmn:sequenceFlow id="IJ_1" sourceRef="Fork_IJ" targetRef="Task_A"/>
    <bpmn:sequenceFlow id="IJ_2" sourceRef="Fork_IJ" targetRef="Task_B"/>
    <bpmn:sequenceFlow id="IJ_3" sourceRef="Task_A" targetRef="Join_IJ"/>
    <bpmn:sequenceFlow id="IJ_4" sourceRef="Task_B" targetRef="Join_IJ"/>
    <bpmn:sequenceFlow id="IJ_5" sourceRef="Join_IJ" targetRef="End_IJ"/>
  </bpmn:process>
</bpmn:definitions>`

// lintEarlyEndBPMN 夹具：并行分叉的两条分支各自直达 endEvent，没有汇聚。
// 先到的分支把整个实例置 completed，另一分支的任务与推进被丢弃。
const lintEarlyEndBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_EE" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_EE" isExecutable="true">
    <bpmn:startEvent id="Start_EE"/>
    <bpmn:parallelGateway id="Fork_EE"/>
    <bpmn:userTask id="Task_A" name="分支A"/>
    <bpmn:userTask id="Task_B" name="分支B"/>
    <bpmn:endEvent id="End_A"/>
    <bpmn:endEvent id="End_B"/>
    <bpmn:sequenceFlow id="EE_0" sourceRef="Start_EE" targetRef="Fork_EE"/>
    <bpmn:sequenceFlow id="EE_1" sourceRef="Fork_EE" targetRef="Task_A"/>
    <bpmn:sequenceFlow id="EE_2" sourceRef="Fork_EE" targetRef="Task_B"/>
    <bpmn:sequenceFlow id="EE_3" sourceRef="Task_A" targetRef="End_A"/>
    <bpmn:sequenceFlow id="EE_4" sourceRef="Task_B" targetRef="End_B"/>
  </bpmn:process>
</bpmn:definitions>`

// lintParallelForkJoinBPMN 合法形态：分叉 → 两分支 → 汇聚 → end。
const lintParallelForkJoinBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_PF" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_PF" isExecutable="true">
    <bpmn:startEvent id="Start_PF"/>
    <bpmn:parallelGateway id="Fork_PF"/>
    <bpmn:userTask id="Task_A" name="分支A"/>
    <bpmn:userTask id="Task_B" name="分支B"/>
    <bpmn:parallelGateway id="Join_PF"/>
    <bpmn:endEvent id="End_PF"/>
    <bpmn:sequenceFlow id="PF_0" sourceRef="Start_PF" targetRef="Fork_PF"/>
    <bpmn:sequenceFlow id="PF_1" sourceRef="Fork_PF" targetRef="Task_A"/>
    <bpmn:sequenceFlow id="PF_2" sourceRef="Fork_PF" targetRef="Task_B"/>
    <bpmn:sequenceFlow id="PF_3" sourceRef="Task_A" targetRef="Join_PF"/>
    <bpmn:sequenceFlow id="PF_4" sourceRef="Task_B" targetRef="Join_PF"/>
    <bpmn:sequenceFlow id="PF_5" sourceRef="Join_PF" targetRef="End_PF"/>
  </bpmn:process>
</bpmn:definitions>`

// lintExclusiveMultiEndBPMN 排他分叉 + 多 endEvent：每次只激活一条路径，安全。
const lintExclusiveMultiEndBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_XE" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_XE" isExecutable="true">
    <bpmn:startEvent id="Start_XE"/>
    <bpmn:exclusiveGateway id="Fork_XE"/>
    <bpmn:userTask id="Task_A" name="分支A"/>
    <bpmn:userTask id="Task_B" name="分支B"/>
    <bpmn:endEvent id="End_A"/>
    <bpmn:endEvent id="End_B"/>
    <bpmn:sequenceFlow id="XE_0" sourceRef="Start_XE" targetRef="Fork_XE"/>
    <bpmn:sequenceFlow id="XE_1" sourceRef="Fork_XE" targetRef="Task_A"/>
    <bpmn:sequenceFlow id="XE_2" sourceRef="Fork_XE" targetRef="Task_B"/>
    <bpmn:sequenceFlow id="XE_3" sourceRef="Task_A" targetRef="End_A"/>
    <bpmn:sequenceFlow id="XE_4" sourceRef="Task_B" targetRef="End_B"/>
  </bpmn:process>
</bpmn:definitions>`

// lintParallelForkExclusiveJoinBPMN 并行分叉 → 排他汇聚 → end：
// 排他汇聚不等待其他分支，不能当 barrier，仍属提前完成。
const lintParallelForkExclusiveJoinBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_PX" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_PX" isExecutable="true">
    <bpmn:startEvent id="Start_PX"/>
    <bpmn:parallelGateway id="Fork_PX"/>
    <bpmn:userTask id="Task_A" name="分支A"/>
    <bpmn:userTask id="Task_B" name="分支B"/>
    <bpmn:exclusiveGateway id="Join_PX"/>
    <bpmn:endEvent id="End_PX"/>
    <bpmn:sequenceFlow id="PX_0" sourceRef="Start_PX" targetRef="Fork_PX"/>
    <bpmn:sequenceFlow id="PX_1" sourceRef="Fork_PX" targetRef="Task_A"/>
    <bpmn:sequenceFlow id="PX_2" sourceRef="Fork_PX" targetRef="Task_B"/>
    <bpmn:sequenceFlow id="PX_3" sourceRef="Task_A" targetRef="Join_PX"/>
    <bpmn:sequenceFlow id="PX_4" sourceRef="Task_B" targetRef="Join_PX"/>
    <bpmn:sequenceFlow id="PX_5" sourceRef="Join_PX" targetRef="End_PX"/>
  </bpmn:process>
</bpmn:definitions>`

// findLintIssue 返回第一条消息包含 substr 的 issue，找不到返回 nil。
func findLintIssue(result *dto.BPMNLintResult, substr string) *dto.BPMNLintIssue {
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, substr) {
			return issue
		}
	}
	return nil
}

// TestLintTokenSemantics_ConvergingInclusiveJoinRejected 覆盖 R1：
// 收敛型包容网关（入度 ≥2）必须被 error 拒绝——包容分叉按需激活、包容汇聚却等全部入边。
func TestLintTokenSemantics_ConvergingInclusiveJoinRejected(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintInclusiveJoinBPMN))
	require.NoError(t, err)

	// 用消息特征定位，避免撞上 lintGateway 对「1 条出边」的既有 warning。
	issue := findLintIssue(result, "永久停在该网关上")
	require.NotNil(t, issue, "收敛型包容网关必须被拦截: %+v", result.Issues)
	assert.Equal(t, "error", issue.Severity)
	assert.Equal(t, "Join_IJ", issue.ElementID)
	assert.Contains(t, issue.Message, "永久停在该网关上")
	assert.True(t, result.HasErrors)
}

// TestLintTokenSemantics_ParallelForkEarlyEndRejected 覆盖 R2：
// 并行分叉存在不经汇聚直达 endEvent 的路径必须被 error 拒绝。
func TestLintTokenSemantics_ParallelForkEarlyEndRejected(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintEarlyEndBPMN))
	require.NoError(t, err)

	issue := findLintIssue(result, "直达结束事件")
	require.NotNil(t, issue, "分叉分支直连 endEvent 必须被拦截: %+v", result.Issues)
	assert.Equal(t, "error", issue.Severity)
	assert.Equal(t, "Fork_EE", issue.ElementID)
	assert.Contains(t, issue.Message, "另一分支的任务与后续推进会被丢弃")
	assert.True(t, result.HasErrors)
}

// TestLintTokenSemantics_ParallelForkWithJoinAllowed 合法形态不得被误伤：
// 分叉 → 汇聚 → endEvent 是引擎可正确执行的形状。
func TestLintTokenSemantics_ParallelForkWithJoinAllowed(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintParallelForkJoinBPMN))
	require.NoError(t, err)

	assert.Nil(t, findLintIssue(result, "直达结束事件"),
		"经汇聚网关后再进 endEvent 不应报令牌语义错误: %+v", result.Issues)
	assert.Nil(t, findLintIssue(result, "永久停在该网关上"))
}

// TestLintTokenSemantics_ExclusiveMultiEndAllowed 排他分叉的多 endEvent 是安全的
// （每次只激活一条路径），不得被 R2 误伤——这是与「字面拒绝所有 multi-end」的关键差异。
func TestLintTokenSemantics_ExclusiveMultiEndAllowed(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintExclusiveMultiEndBPMN))
	require.NoError(t, err)

	assert.Nil(t, findLintIssue(result, "直达结束事件"),
		"排他分叉的多 endEvent 不应报令牌语义错误: %+v", result.Issues)
	assert.Nil(t, findLintIssue(result, "永久停在该网关上"))
}

// TestLintTokenSemantics_ExclusiveJoinIsNotBarrier 排他汇聚不等待其他分支，
// 不能充当 R2 的 barrier：并行分叉经排他汇聚直达 endEvent 仍属提前完成。
func TestLintTokenSemantics_ExclusiveJoinIsNotBarrier(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintParallelForkExclusiveJoinBPMN))
	require.NoError(t, err)

	issue := findLintIssue(result, "直达结束事件")
	require.NotNil(t, issue, "排他汇聚不构成屏障，必须仍被拦截: %+v", result.Issues)
	assert.Equal(t, "Fork_PX", issue.ElementID)
}
