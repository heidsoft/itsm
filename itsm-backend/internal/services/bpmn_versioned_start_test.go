package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P2-07（codex 审计）：版本选择被过度承诺 ——
//   - TriggerProcess 查到路由指定的版本，但 StartProcess 只传 key，
//     引擎重新选 active+latest，路由选中的版本被覆盖；
//   - dto.ProcessTriggerRequest.ProcessVersion（客户端显式版本）从未被读取。
// 本文件验证「按指定版本启动」真实生效。

// seedVersionedDefinition 造出同一 key 的两个版本：v1（latest+active）与 v2（active、非 latest）。
func seedVersionedDefinition(t *testing.T, ctx context.Context, client *ent.Client, key string, tenantID int) {
	t.Helper()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).
		SetDeploymentName(key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key + " v1").
		SetVersion("1").
		SetBpmnXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="F_0" sourceRef="Start_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key + " v2").
		SetVersion("2").
		SetBpmnXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_2" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_2" isExecutable="true">
    <bpmn:startEvent id="Start_2" name="开始"/>
    <bpmn:endEvent id="End_2" name="结束"/>
    <bpmn:sequenceFlow id="F_1" sourceRef="Start_2" targetRef="End_2"/>
  </bpmn:process>
</bpmn:definitions>`)).
		SetIsActive(true).
		SetIsLatest(false).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
}

// TestStartProcessWithVersion_UsesRequestedVersion 验证引擎按指定版本启动，
// 而不是像旧实现那样重新选 active+latest。
func TestStartProcessWithVersion_UsesRequestedVersion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()
	tenantID := 1

	seedVersionedDefinition(t, ctx, client, "versionedFlow", tenantID)

	// 指定版本 "2"（它是 active 但不是 latest）
	starter, ok := interface{}(engine).(VersionedProcessStarter)
	require.True(t, ok, "CustomProcessEngine 必须实现 VersionedProcessStarter")

	triggerCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	instance, err := starter.StartProcessWithVersion(triggerCtx, "versionedFlow", "2", "BK-VER-1", nil)
	require.NoError(t, err)
	require.NotNil(t, instance)

	def, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("versionedFlow"), processdefinition.Version("2")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, def.ID, instance.ProcessDefinitionID,
		"实例必须挂在被指定的版本上，而不是被 latest 覆盖")

	// 反向：latest 版本是 "1"，若实现退化为 active+latest，实例会落到 v1
	latest, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("versionedFlow"), processdefinition.IsLatest(true)).
		Only(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, latest.ID, instance.ProcessDefinitionID,
		"指定版本若被 latest 覆盖，本断言会失败（这正是修复前的行为）")
}

// TestStartProcessWithVersion_EmptyVersionFallsBackToLatest 验证未指定版本时
// 语义与 StartProcess 一致（取 active + latest）。
func TestStartProcessWithVersion_EmptyVersionFallsBackToLatest(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()
	tenantID := 1

	seedVersionedDefinition(t, ctx, client, "fallbackFlow", tenantID)

	triggerCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	instance, err := engine.StartProcessWithVersion(triggerCtx, "fallbackFlow", "", "BK-VER-2", nil)
	require.NoError(t, err)
	require.NotNil(t, instance)

	latest, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("fallbackFlow"), processdefinition.IsLatest(true)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, latest.ID, instance.ProcessDefinitionID,
		"未指定版本时应落到 latest 版本")
}

// TestStartProcessWithVersion_UnknownVersionRejected 验证指定了不存在的版本时
// 明确报错，而不是静默回落到 latest。
func TestStartProcessWithVersion_UnknownVersionRejected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()
	tenantID := 1

	seedVersionedDefinition(t, ctx, client, "missingVersionFlow", tenantID)

	triggerCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	_, err := engine.StartProcessWithVersion(triggerCtx, "missingVersionFlow", "99", "BK-VER-3", nil)
	require.Error(t, err, "指定不存在的版本必须报错，不能静默回落")

	// 被拒绝的启动不得留下实例行
	count, err := client.ProcessInstance.Query().
		Where(processinstance.TenantIDEQ(tenantID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "失败的启动不得留下实例行")
}

// TestTriggerProcess_HonorsExplicitRequestVersion 验证客户端显式传入的
// ProcessVersion 真正生效（旧实现完全没读这个字段）。
func TestTriggerProcess_HonorsExplicitRequestVersion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()
	tenantID := 1

	seedVersionedDefinition(t, ctx, client, "explicitVersionFlow", tenantID)

	trigger := NewProcessTriggerService(client, engine)
	resp, err := trigger.TriggerProcess(ctx, &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessType("ticket"),
		BusinessID:           4242,
		ProcessDefinitionKey: "explicitVersionFlow",
		ProcessVersion:       2,
		TenantID:             tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	inst, err := client.ProcessInstance.Get(ctx, resp.ProcessInstanceID)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key("explicitVersionFlow"), processdefinition.Version("2")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, def.ID, inst.ProcessDefinitionID,
		"客户端显式指定的版本必须生效，而不是被 latest 覆盖")
}
