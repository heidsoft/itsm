package scenarios

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"

	"github.com/stretchr/testify/require"
)

var scenarioDBCounter int64

func scenarioDSN(prefix string) string {
	return fmt.Sprintf("file:scenario_%s_%d?mode=memory&cache=shared&_fk=1", prefix, atomic.AddInt64(&scenarioDBCounter, 1))
}

func mustCreateTenant(ctx context.Context, t *testing.T, client *ent.Client, name, code, domain string) *ent.Tenant {
	t.Helper()
	tn, err := client.Tenant.Create().
		SetName(name).
		SetCode(code).
		SetDomain(domain).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tn
}

func mustCreateUser(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username, email, role string) *ent.User {
	t.Helper()
	return mustCreateScopedUser(ctx, t, client, tenantID, username, email, role, "")
}

// mustCreateScopedUser 额外设置部门：服务请求一级审批（manager）要求审批人与请求人同部门，
// 空部门会让审批链兜底判定直接判为无权。
func mustCreateScopedUser(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username, email, role, department string) *ent.User {
	t.Helper()
	builder := client.User.Create().
		SetUsername(username).
		SetEmail(email).
		SetName(username).
		SetPasswordHash("hashed").
		SetRole(entuser.Role(role)).
		SetActive(true).
		SetTenantID(tenantID)
	if department != "" {
		builder.SetDepartment(department)
	}
	u, err := builder.Save(ctx)
	require.NoError(t, err)
	return u
}

// seedApprovalProcess 为业务对象播种「运行中流程实例 + 串行审批用户任务」的首个待办，
// businessKey 沿用 ProcessTriggerService 的 "<业务类型>:<id>" 约定。
//
// 只预建第一个待办：后续层级必须由真实 BPMN 引擎在完成当前待办时创建，
// 测试不得预先插入后续任务、不得在流程缺失时回退业务直批（否则覆盖的是兜底路径而非生产路径）。
func seedApprovalProcess(t *testing.T, client *ent.Client, tenantID int, businessKey, suffix string, assignees ...int) int {
	t.Helper()
	require.NotEmpty(t, assignees)
	ctx := context.Background()

	defKey := "scenario_approval_" + suffix
	var nodes strings.Builder
	for i, assignee := range assignees {
		fmt.Fprintf(&nodes, `<bpmn:userTask id="Approval_%d" name="Approval %d" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:assignee="%d"/>`, i+1, i+1, assignee)
		if i > 0 {
			fmt.Fprintf(&nodes, `<bpmn:sequenceFlow id="Next_%d" sourceRef="Approval_%d" targetRef="Approval_%d"/>`, i, i, i+1)
		}
	}
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" targetNamespace="https://github.com/heidsoft/itsm">
<bpmn:process id="%s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/>
%s<bpmn:endEvent id="EndEvent_1"/>
<bpmn:sequenceFlow id="Start" sourceRef="StartEvent_1" targetRef="Approval_1"/>
<bpmn:sequenceFlow id="End" sourceRef="Approval_%d" targetRef="EndEvent_1"/>
</bpmn:process></bpmn:definitions>`, defKey, nodes.String(), len(assignees))

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("SCN-DEP-" + suffix).
		SetDeploymentName("Scenario Deployment " + suffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("Scenario Approval " + suffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("SCN-PI-" + suffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"business_type": strings.SplitN(businessKey, ":", 2)[0],
			"business_id":   strings.SplitN(businessKey, ":", 2)[1],
			"business_key":  businessKey,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("SCN-TASK-" + suffix).
		SetTaskDefinitionKey("Approval_1").
		SetTaskName("Approval 1").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(assignees[0])).
		SetStatus("assigned").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance.ID
}
