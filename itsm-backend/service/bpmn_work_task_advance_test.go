package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// createWorkProcessFixture 创建 Start → Work_1 → End 的运行中实例，
// 待办任务显式标注 taskPurpose=work，模拟交付成功后需要收口的履约节点。
func createWorkProcessFixture(t *testing.T, client *ent.Client, tenantID int, keySuffix, businessKey string) (instanceID, taskID int) {
	t.Helper()
	ctx := context.Background()

	defKey := "bridge_work_" + keySuffix
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="Bridge Work %s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/><bpmn:userTask id="Work_1" name="执行服务" itsm:taskPurpose="work"/><bpmn:endEvent id="EndEvent_1"/><bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Work_1"/><bpmn:sequenceFlow id="Flow_2" sourceRef="Work_1" targetRef="EndEvent_1"/></bpmn:process></bpmn:definitions>`, defKey, defKey, keySuffix)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + keySuffix).
		SetDeploymentName("Deployment " + keySuffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("Bridge Work " + keySuffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-" + keySuffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus("running").
		SetCurrentActivityID("Work_1").
		SetVariables(map[string]interface{}{
			"business_type": "service_request",
			"business_key":  businessKey,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-" + keySuffix).
		SetTaskDefinitionKey("Work_1").
		SetTaskName("执行服务").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(0)).
		SetStatus("assigned").
		SetTaskVariables(map[string]interface{}{"taskPurpose": "work"}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance.ID, task.ID
}

func TestCompleteBusinessWorkTasks_AdvancesWorkAndCompletesInstance(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_work_advance")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "work")
	instanceID, taskID := createWorkProcessFixture(t, client, tenantID, "work1", "service_request:321")
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	completed, err := bridge.CompleteBusinessWorkTasks(
		context.Background(), tenantID, actorID, "service_request", 321,
		map[string]interface{}{"satisfied": true}, 5,
	)
	require.NoError(t, err)
	require.Equal(t, 1, completed)

	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.Equal(t, "completed", task.Status)

	instance, err := client.ProcessInstance.Get(context.Background(), instanceID)
	require.NoError(t, err)
	require.Equal(t, "completed", instance.Status, "作业节点完成后流程应正常结束")
}

func TestCompleteBusinessWorkTasks_RefusesToSignApprovalNode(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_work_guard")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "guard")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "guard1", "service_request:322", actorID)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	completed, err := bridge.CompleteBusinessWorkTasks(
		context.Background(), tenantID, actorID, "service_request", 322,
		map[string]interface{}{"satisfied": true}, 5,
	)
	require.Error(t, err)
	require.Zero(t, completed)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, common.ConflictCode, bizErr.Code)

	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	require.NotEqual(t, "completed", task.Status, "审批节点不能被交付收口代签")
}
