package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 审计 P1-07：timer 运行时依赖必须能被「事务内另建的裸引擎」共享，
// 否则业务审批桥接/工单流程推进到 timer 节点要么整笔回滚、要么静默丢超时。

// withTimerRuntime 注册进程级 timer 运行时并在测试结束时恢复 nil，
// 防止全局状态泄漏到同包内其他用例（包内测试无 t.Parallel）。
func withTimerRuntime(t *testing.T, store TimerStore, scheduler *TimerScheduler) {
	t.Helper()
	SetTimerRuntime(store, scheduler)
	t.Cleanup(func() { SetTimerRuntime(nil, nil) })
}

// TestNewCustomProcessEngine_AdoptsRegisteredTimerRuntime 断言构造函数读取进程级运行时：
// 注册后新建引擎拿到 store/scheduler，恢复 nil 后新建引擎回到「未配置」基线。
func TestNewCustomProcessEngine_AdoptsRegisteredTimerRuntime(t *testing.T) {
	client := newPhase4TestClient(t, "timer-runtime-adopt")
	logger := zaptest.NewLogger(t).Sugar()

	store := NewDBTimerStore(client, logger)
	withTimerRuntime(t, store, nil)

	wired := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	require.NotNil(t, wired.timerStore, "注册运行时后新建引擎必须拿到 timerStore（桥接裸引擎场景）")
	assert.Nil(t, wired.timerScheduler, "未注册 scheduler 时保持 nil")

	// 显式注入永远优先于进程级运行时。
	explicit := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	other := NewDBTimerStore(client, logger)
	explicit.SetTimerServices(other, nil)
	assert.True(t, explicit.timerStore == TimerStore(other),
		"显式 SetTimerServices 必须覆盖运行时快照")

	// 清除后新建引擎回到基线（既有单测依赖「未配置」行为）。
	SetTimerRuntime(nil, nil)
	plain := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	assert.Nil(t, plain.timerStore, "未注册运行时的引擎必须保持 nil store")
}

// TestRegisteredTimerRuntime_EnablesIntermediateTimerOnBareEngine 复现审计场景：
// 业务桥接那样不调用 SetTimerServices 的裸引擎，在运行时注册后应能正常注册
// 中间定时器（旧实现报「定时器服务未配置」并把整笔业务事务回滚）。
func TestRegisteredTimerRuntime_EnablesIntermediateTimerOnBareEngine(t *testing.T) {
	client := newPhase4TestClient(t, "timer-runtime-bare")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "rtbare")
	ctx := context.Background()

	store := NewDBTimerStore(client, logger)
	withTimerRuntime(t, store, nil)

	// 关键：只构造，不调用 SetTimerServices —— 与 BPMNApprovalBridge 一致。
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithIntermediateTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "timer_wait")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithIntermediateTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	event := engine.findIntermediateEvent(process, "timer_wait")
	require.NotNil(t, event)

	require.NoError(t, engine.handleIntermediateCatchEvent(ctx, client, instance, process, event),
		"运行时已注册的裸引擎不应再报「定时器服务未配置」")

	timers, total, err := store.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeIntermediate),
		ProcessInstanceID: &instance.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total, "中间定时器必须真的落库")
	assert.Equal(t, "timer_wait", timers[0].ActivityID)
}
