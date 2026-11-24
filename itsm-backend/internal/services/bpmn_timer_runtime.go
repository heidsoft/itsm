package service

import "sync"

// BPMN 定时器运行时依赖（进程级单例）。
//
// 背景（codex 审计 P1-07）：引擎在业务事务内推进，但 timer 注册依赖 TimerStore。
// 主引擎由 bootstrap 显式 SetTimerServices；而业务审批桥接
// （bpmn_approval_bridge_service）、工单流程（ticket_workflow_service）等在
// CompleteTask 时另建裸引擎 `NewCustomProcessEngine(txc, logger)`，构造函数不带
// 任何 timer 依赖——于是：
//   - 中间定时事件命中 `timerStore == nil` → 报「定时器服务未配置」整笔回滚，
//     业务审批推进到 timer 节点直接失败；
//   - 边界定时事件命中 nil store → 静默不注册，超时永远不会触发（更危险）。
//
// 桥接点分散在 6+ 个构造位置（bootstrap 与各域 service 构造函数内），逐个穿参
// 侵入面过大；TimerStore/TimerScheduler 在本就是进程级单例（bootstrap 各构造一次），
// 因此这里提供一次注册、全部引擎共享的运行时依赖：
//
//	SetTimerRuntime(store, scheduler)   // bootstrap 在 store/scheduler 创建后各调一次
//	NewCustomProcessEngine(...)         // 构造时读取当前运行时，nil 时保持原有行为
//
// 语义约束：
//   - 只在未显式 SetTimerServices 前生效；显式注入（主引擎、单测）永远优先；
//   - 未注册时快照为 nil，行为与引入本文件之前完全一致（既有单测依赖此点）；
//   - 单测可用 SetTimerRuntime 注入并在 t.Cleanup 里恢复 nil，包内测试无 t.Parallel。
var bpmnTimerRuntime struct {
	mu        sync.RWMutex
	store     TimerStore
	scheduler *TimerScheduler
}

// SetTimerRuntime 注册进程级 timer 运行时依赖。传 nil 表示清除（测试恢复用）。
// 由 bootstrap 调用；重复调用以最后一次为准。
func SetTimerRuntime(store TimerStore, scheduler *TimerScheduler) {
	bpmnTimerRuntime.mu.Lock()
	defer bpmnTimerRuntime.mu.Unlock()
	bpmnTimerRuntime.store = store
	bpmnTimerRuntime.scheduler = scheduler
}

// timerRuntimeSnapshot 返回当前注册的 timer 运行时依赖（未注册时均为 nil）。
func timerRuntimeSnapshot() (TimerStore, *TimerScheduler) {
	bpmnTimerRuntime.mu.RLock()
	defer bpmnTimerRuntime.mu.RUnlock()
	return bpmnTimerRuntime.store, bpmnTimerRuntime.scheduler
}

// TimerRuntimeSnapshot 导出当前进程级 timer 运行时依赖（只读）。
//
// 供装配层验证接线是否真的生效（bootstrap 的 timer 运行时守卫），
// 以及运维诊断「本进程的引擎是否拿得到定时器依赖」。
// 返回值均为 nil 表示尚未注册（装配缺失或已显式清场）。
func TimerRuntimeSnapshot() (TimerStore, *TimerScheduler) {
	return timerRuntimeSnapshot()
}
