package bootstrap

import (
	"go.uber.org/zap"

	"itsm-backend/ent"
	service "itsm-backend/internal/services"
)

// 定时器运行时装配（codex 审计 P1-07）。
//
// 背景：业务审批桥接与工单流程在事务内另建裸引擎（`NewCustomProcessEngine(txc, logger)`），
// 构造函数不带任何 timer 依赖——于是中间定时事件命中 nil store 报「定时器服务未配置」
// 整笔回滚，边界定时事件则静默不注册（超时永远不触发且无痕迹）。
//
// TimerStore / TimerScheduler 本就是进程级单例（bootstrap 各构造一次），因此这里
// 提供一次注册、全部引擎共享的装配入口。放在独立文件而非 app.go，是为了保持组合根
// 行数不增长（产品表面棘轮 bootstrap_app_lines 只降不升），且让「timer 运行时怎么共享」
// 这件事在一处可读。
//
// 装配顺序要求：TimerStore 必须早于任何「部署/激活流程定义」的调用（部署期要注册
// start timer 时间表），因此注册分两步——建 store 时先注册 store，调度器就绪后再补齐
// 调度器（补齐后新建的引擎同时拿到到期触发与内存计时器取消能力）。
//
// 显式 SetTimerServices 始终优先于进程级快照，两者并存不冲突。

// newRuntimeTimerStore 创建 TimerStore 并注册为进程级运行时依赖。
//
// 进程内任何后建的引擎（含业务审批桥接/工单流程的裸引擎）都能拿到 store：
// 它们的构造发生在请求期，晚于本次注册。
func newRuntimeTimerStore(client *ent.Client, sugar *zap.SugaredLogger) service.TimerStore {
	store := service.NewDBTimerStore(client, sugar)
	service.SetTimerRuntime(store, nil)
	return store
}

// wireTimerRuntime 把定时器运行时接入主引擎（store + 调度器）并注册为进程级依赖。
//
// 与 newRuntimeTimerStore 的差别是补齐调度器：调度器负责到期触发与内存计时器取消，
// 缺它时定时器仍能落库但不会被触发、也不会随任务完成而取消。
func wireTimerRuntime(engine *service.CustomProcessEngine, store service.TimerStore, scheduler *service.TimerScheduler) {
	engine.SetTimerServices(store, scheduler)
	service.SetTimerRuntime(store, scheduler)
}
