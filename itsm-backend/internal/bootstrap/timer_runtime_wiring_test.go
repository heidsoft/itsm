package bootstrap

import (
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
	service "itsm-backend/internal/services"
)

// 审计 P1-07 的装配守卫：timer 运行时必须真的注册到进程级依赖。
//
// 这里刻意**不**用 `os.ReadFile("app.go") + strings.Contains` 判接线：
// 那类源码文本锁在语句被注释掉时照样绿（既有教训：feishu 审计装配守卫）。
// 改为行为探针——注册 store 后进程级运行时必须真的持有它，且后建的裸引擎
// （模拟业务审批桥接/工单流程）必须能拿到 store；否则中间定时事件会报
// 「定时器服务未配置」整笔回滚、边界定时事件静默不注册。
func TestNewRuntimeTimerStore_RegistersProcessLevelRuntime(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:bootstrap-timer-runtime?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	// 清场：进程级运行时是全局状态，本用例结束后必须复原，避免影响同包其他用例。
	service.SetTimerRuntime(nil, nil)
	t.Cleanup(func() { service.SetTimerRuntime(nil, nil) })

	store := newRuntimeTimerStore(client, logger)
	require.NotNil(t, store)

	snapshotStore, snapshotScheduler := service.TimerRuntimeSnapshot()
	assert.NotNil(t, snapshotStore, "建 store 后进程级运行时必须持有它（裸引擎靠它拿到定时器依赖）")
	assert.Nil(t, snapshotScheduler, "调度器尚未创建时不应凭空出现")
}

// TestWireTimerRuntime_RegistersStoreAndScheduler 验证补齐调度器后，
// 进程级运行时同时持有 store 与 scheduler（调度器负责到期触发与内存计时器取消）。
func TestWireTimerRuntime_RegistersStoreAndScheduler(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:bootstrap-timer-wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	service.SetTimerRuntime(nil, nil)
	t.Cleanup(func() { service.SetTimerRuntime(nil, nil) })

	store := newRuntimeTimerStore(client, logger)
	engine, ok := service.NewCustomProcessEngine(client, logger).(*service.CustomProcessEngine)
	require.True(t, ok, "必须构造出 CustomProcessEngine")
	require.NotNil(t, engine)

	scheduler := service.NewTimerScheduler(service.TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: logger,
	})
	wireTimerRuntime(engine, store, scheduler)

	snapshotStore, snapshotScheduler := service.TimerRuntimeSnapshot()
	assert.NotNil(t, snapshotStore, "补齐后进程级运行时必须仍持有 store")
	assert.NotNil(t, snapshotScheduler, "补齐后进程级运行时必须持有调度器（否则定时器落库但不会触发）")
}
