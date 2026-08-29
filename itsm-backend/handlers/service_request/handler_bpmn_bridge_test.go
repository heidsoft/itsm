package service_request

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// srSetupBridge 与 srSetupRole 类似，但额外返回 ent client，用于播种 BPMN 流程数据。
// 请求人与审批人必须是不同用户（服务请求禁止本人审批自己的请求）。
func srSetupBridge(t *testing.T) (*gin.Engine, *ent.Client, int, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_bridge_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("SRTenant").
		SetCode("SR" + srUID()).
		SetDomain("sr.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := createServiceCatalogForTest(ctx, scSvc, "SRCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)
	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, client, logger, nil)
	h := NewHandler(svc)
	requester, err := client.User.Create().
		SetUsername("sr-bridge-requester-" + srUID()).
		SetEmail("sr-bridge-req-" + srUID() + "@example.com").
		SetName("SR Bridge Requester").
		SetPasswordHash("hash").
		SetRole("agent").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	approver, err := client.User.Create().
		SetUsername("sr-bridge-user-" + srUID()).
		SetEmail("sr-bridge-" + srUID() + "@example.com").
		SetName("SR Bridge User").
		SetPasswordHash("hash").
		SetRole("manager").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	r := gin.New()
	r.Use(srAuthRoleActors(tenant.ID, requester.ID, approver.ID, "manager", "IT"))
	r.POST("/api/v1/service-requests", h.Create)
	r.GET("/api/v1/service-requests/:id", h.Get)
	r.POST("/api/v1/service-requests/:id/approval", h.ApplyApproval)
	return r, client, tenant.ID, approver.ID, cat.ID
}

// srCreateBPMNFixture 播种 运行中流程实例 + 待办用户任务，
// businessKey 采用与 ProcessTriggerService 相同的 "service_request:{id}" 约定
// （与 service 包 createBridgeProcessFixture 保持一致的最小副本，跨包无法复用）。
func srCreateBPMNFixture(t *testing.T, client *ent.Client, tenantID int, keySuffix string, requestID, assigneeID int) (taskID int) {
	t.Helper()
	ctx := context.Background()
	businessKey := fmt.Sprintf("service_request:%d", requestID)

	defKey := "sr_bridge_approval_" + keySuffix
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="SR Bridge Approval %s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/><bpmn:userTask id="Approval_1" name="审批" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:assignee="%d"/><bpmn:endEvent id="EndEvent_1"/><bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Approval_1"/><bpmn:sequenceFlow id="Flow_2" sourceRef="Approval_1" targetRef="EndEvent_1"/></bpmn:process></bpmn:definitions>`, defKey, defKey, keySuffix, assigneeID)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("SR-DEP-" + keySuffix).
		SetDeploymentName("SR Deployment " + keySuffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("SR Bridge Approval " + keySuffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("SR-PI-" + keySuffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"business_type": "service_request",
			"business_id":   strconv.Itoa(requestID),
			"business_key":  businessKey,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("SR-TASK-" + keySuffix).
		SetTaskDefinitionKey("Approval_1").
		SetTaskName("审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(assigneeID)).
		SetStatus("assigned").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return task.ID
}

func TestServiceRequestApproval_BridgesBPMNTask(t *testing.T) {
	r, client, tenantID, uid, catID := srSetupBridge(t)
	id := srCreateOne(t, r, catID)
	taskID := srCreateBPMNFixture(t, client, tenantID, "ok1", id, uid)

	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve", Comment: "同意"})
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", srStr(resp))

	// BPMN 待办任务应被桥接完成
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 业务侧审批链同步推进
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "manager_approved", data["status"])
}

func TestServiceRequestApproval_BridgeFailClosedForUnauthorizedActor(t *testing.T) {
	r, client, tenantID, uid, catID := srSetupBridge(t)
	id := srCreateOne(t, r, catID)
	// BPMN 任务指派给其他人：当前操作人不是流程任务审批人，必须 fail-closed
	taskID := srCreateBPMNFixture(t, client, tenantID, "fc1", id, uid+1000)

	resp := srDoReq(t, r, "POST", "/api/v1/service-requests/"+strconv.Itoa(id)+"/approval",
		dto.ServiceRequestApprovalActionRequest{Action: "approve"})
	require.NotEqual(t, common.SuccessCode, resp.Code, "BPMN 审批人不匹配时必须中止业务侧审批, body=%s", srStr(resp))

	// 流程任务保持待办，业务状态未推进
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)

	getResp := srDoReq(t, r, "GET", "/api/v1/service-requests/"+strconv.Itoa(id), nil)
	require.Equal(t, common.SuccessCode, getResp.Code)
	getData := getResp.Data.(map[string]interface{})
	assert.Equal(t, "submitted", getData["status"], "fail-closed 时业务状态不得推进")
}
