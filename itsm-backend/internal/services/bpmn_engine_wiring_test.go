package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
)

// 生产 handler 全部经 ProcessDefinitionService()/TaskService() 访问器操作引擎，
// 访问器必须返回 SetTimerServices 接线过的实例。返回新建副本会让发布定义时的
// start timer 同步、任务完成时的定时器取消全部静默失效。

const timerStartBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="timer_start_proc" isExecutable="true">
    <bpmn:startEvent id="timerStart">
      <bpmn:timerEventDefinition>
        <bpmn:timeCycle>R10/PT2M</bpmn:timeCycle>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:userTask id="task1" name="处理"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="timerStart" targetRef="task1"/>
    <bpmn:sequenceFlow id="f2" sourceRef="task1" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

func setupWiringEngine(t *testing.T) (*CustomProcessEngine, *memoryTimerStore, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, testDSN())
	t.Cleanup(func() { _ = client.Close() })
	tenantID := createTimerTenant(t, client, "wiring")
	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	store := newMemoryTimerStore()
	engine.SetTimerServices(store, nil)
	return engine, store, ctx, tenantID
}

func TestProcessDefinitionService_AccessorCarriesTimerStore(t *testing.T) {
	engine, store, ctx, tenantID := setupWiringEngine(t)

	dep, err := engine.client.ProcessDeployment.Create().SetDeploymentID("DEP-WIRING").SetDeploymentName("wiring").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = engine.client.ProcessDefinition.Create().SetKey("timerStartProc").SetName("timer start").SetVersion("1").
		SetBpmnXML([]byte(timerStartBPMN)).SetDeploymentID(dep.ID).SetTenantID(tenantID).
		SetIsActive(false).SetIsLatest(false).Save(ctx)
	require.NoError(t, err)

	// 与 handlers/bpmn/workflow.go:143 相同的调用形态：经接口访问器发布。
	_, err = engine.ProcessDefinitionService().PublishProcessDefinition(ctx, "timerStartProc", "1", &UpdateProcessDefinitionRequest{
		BPMNXML: timerStartBPMN,
	})
	require.NoError(t, err)

	timers, _, err := store.List(ctx, TimerListFilter{
		TenantID:             tenantID,
		TimerType:            string(TimerTypeStart),
		Status:               string(TimerStatusPending),
		ProcessDefinitionKey: "timerStartProc",
	})
	require.NoError(t, err)
	require.NotEmpty(t, timers, "发布含 timer start event 的定义必须同步时间表；访问器丢 store 时此处为空")
}

func TestTaskService_AccessorUsesWiredEngine(t *testing.T) {
	engine, _, _, _ := setupWiringEngine(t)

	taskSvc, ok := engine.TaskService().(*bpmnTaskService)
	require.True(t, ok)
	require.Same(t, engine, taskSvc.engine, "TaskService() 必须返回带所属引擎的实例，任务完成才复用定时器设施")

	defSvc, ok := engine.ProcessDefinitionService().(*bpmnProcessDefinitionService)
	require.True(t, ok)
	require.NotNil(t, defSvc.timerStore, "ProcessDefinitionService() 必须携带 SetTimerServices 注入的 store")
}
