package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtimer"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func versionTimerStartXML(cycle string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="version_timer_proc" name="VersionTimer" isExecutable="true">
    <bpmn:startEvent id="start_timer">
      <bpmn:timerEventDefinition>
        <bpmn:timeCycle>%s</bpmn:timeCycle>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:userTask id="task1" name="Task"/>
    <bpmn:endEvent id="end1"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start_timer" targetRef="task1"/>
    <bpmn:sequenceFlow id="f2" sourceRef="task1" targetRef="end1"/>
  </bpmn:process>
</bpmn:definitions>`, cycle)
}

// TestActivateVersion_SyncsStartTimers 版本激活/回滚必须同步 start timer 时间表：
// 激活新版本后旧 pending 时间表整体替换为目标版本的；回滚经 ActivateVersion 继承。
// 此前激活只切换定义行，时间表仍按旧版本（或空）运行。
func TestActivateVersion_SyncsStartTimers(t *testing.T) {
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:version_timer_sync_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("VersionTimer Tenant").SetCode("vtimer").SetDomain("vtimer.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	store := NewDBTimerStore(client, logger)
	svc := NewBPMNVersionService(client, logger)
	svc.SetTimerStore(store)

	const key = "version_timer_proc"
	// 直接播种两个版本的定义行（首个版本走部署路径，CreateVersion 只支持增量版本）。
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-VTIMER").SetDeploymentName("VersionTimer Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey(key).SetName("V1").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte(versionTimerStartXML("R/PT10M"))).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	v2Def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName("V2").SetVersion("2").
		SetIsLatest(false).SetIsActive(false).
		SetBpmnXML([]byte(versionTimerStartXML("R/PT20M"))).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	v2 := v2Def.Version
	v2Num, err := ParseVersionNumber(v2)
	require.NoError(t, err)

	// v1 曾处于激活态：先注册它的时间表。
	_, err = SyncStartTimers(ctx, client, store, []byte(versionTimerStartXML("R/PT10M")), key, tenant.ID)
	require.NoError(t, err)

	// 激活 v2：旧时间表必须被 v2 的时间表整体替换。
	require.NoError(t, svc.ActivateVersion(ctx, key, v2Num, tenant.ID))

	pending, err := client.ProcessTimer.Query().
		Where(
			processtimer.TenantID(tenant.ID),
			processtimer.ProcessDefinitionKey(key),
			processtimer.TimerTypeEQ("start"),
			processtimer.StatusEQ("pending"),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, pending, 1, "激活后必须恰好有一份 v2 时间表")
	assert.Equal(t, "R/PT20M", pending[0].TimerExpression, "时间表必须来自新激活版本的 XML")

	cancelled, err := client.ProcessTimer.Query().
		Where(
			processtimer.TenantID(tenant.ID),
			processtimer.ProcessDefinitionKey(key),
			processtimer.TimerTypeEQ("start"),
			processtimer.StatusEQ("cancelled"),
		).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cancelled, "旧版本 pending 时间表必须被取消而非残留")

	// 回滚到 v1：经 CreateVersion + ActivateVersion，时间表必须回到 v1 表达式。
	require.NoError(t, svc.RollbackToVersion(ctx, key, v2Num-1, tenant.ID, "回归验证"))

	pendingAfterRollback, err := client.ProcessTimer.Query().
		Where(
			processtimer.TenantID(tenant.ID),
			processtimer.ProcessDefinitionKey(key),
			processtimer.TimerTypeEQ("start"),
			processtimer.StatusEQ("pending"),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, pendingAfterRollback, 1, "回滚后必须恰好有一份回滚版本时间表")
	assert.Equal(t, "R/PT10M", pendingAfterRollback[0].TimerExpression, "回滚后时间表必须来自回滚目标的 XML")
}
