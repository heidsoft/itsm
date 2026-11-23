package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/processinstance"
)

// 同步推进预算（codex 审计 P1-11）：
// 既有 64 层防护只统计相邻并行/包容网关嵌套，dispatch→handleElement 与
// handleElement→网关两处会丢/重置 depth，排他网关等元素级同步环没有任何推进
// 上限，可无限递归耗尽请求资源、事务连接与调用栈。修复要求：每次同步推进共享
// 且只递增的 step budget，并检查 ctx.Err。

// 排他网关自环：出边无条件恒真，引擎会同步绕环永不自停。
// 这是审计给出的原始攻击面（exclusiveGateway 自环），修复后必须在步数上限处
// 确定性失败，而不是挂死请求。
const syncLoopBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="sync_loop_proc" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:exclusiveGateway id="loop_gw"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="s1" sourceRef="start" targetRef="loop_gw"/>
    <bpmn:sequenceFlow id="loop_back" sourceRef="loop_gw" targetRef="loop_gw"/>
    <bpmn:sequenceFlow id="s2" sourceRef="loop_gw" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

func TestStartProcess_SyncLoopHitsStepBudget(t *testing.T) {
	engine, ctx, tenantID := lifecycleTestSetup(t, "sync-loop-budget")
	createTestProcessDefinition(t, engine.client, tenantID, "sync_loop", syncLoopBPMN)

	start := time.Now()
	_, err := engine.StartProcess(ctx, "sync_loop", "BK-LOOP", map[string]interface{}{})
	require.Error(t, err, "同步环路必须在步数上限处失败，不得挂死")
	assert.Contains(t, err.Error(), "步数上限")
	assert.Contains(t, err.Error(), "同步环路")
	assert.Less(t, time.Since(start), 30*time.Second, "预算必须让环路在有限时间内确定性失败")

	// 启动事务随错误回滚：预算中止不得留下半推进的实例行。
	count, cErr := engine.client.ProcessInstance.Query().
		Where(processinstance.ProcessDefinitionKey("sync_loop")).
		Count(ctx)
	require.NoError(t, cErr)
	assert.Equal(t, 0, count, "预算中止的启动必须零残留")
}

// 预算是共享、只递增且有边界的：正好 maxSyncAdvanceSteps 步内放行，下一步拒绝；
// 未携带预算的独立入口从零计数，互不影响（不同请求的推进链彼此隔离）。
func TestAdvanceSyncStep_SharedMonotonicBudget(t *testing.T) {
	ctx := context.Background()
	var err error
	for i := 0; i < maxSyncAdvanceSteps; i++ {
		ctx, err = advanceSyncStep(ctx)
		require.NoError(t, err, "第 %d 步不应超限", i+1)
	}
	_, err = advanceSyncStep(ctx)
	require.Error(t, err, "超过上限必须报错")
	assert.Contains(t, err.Error(), "步数上限")

	fresh, err := advanceSyncStep(context.Background())
	require.NoError(t, err, "独立推进链必须持有自己的预算")
	_ = fresh
}

// ctx 取消在下一步立即生效（而不是等环路跑完）。
func TestAdvanceSyncStep_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := advanceSyncStep(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已取消")
}
