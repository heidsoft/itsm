package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 纯函数单元测试（无数据库依赖） ====================

func TestNormalizeBpmnStatus(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"running", "running"},
		{"RUNNING", "running"},
		{"  Running  ", "running"},
		{"suspended", "suspended"},
		{"Suspended", "suspended"},
		{"completed", "completed"},
		{"terminated", "terminated"},
		{"unknown", "not_started"},
		{"", "not_started"},
		{"pending", "not_started"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := normalizeBpmnStatus(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestNullableTimePtr(t *testing.T) {
	zero := time.Time{}
	got := nullableTimePtr(zero)
	assert.Nil(t, got, "zero time should produce nil pointer")

	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	got2 := nullableTimePtr(now)
	require.NotNil(t, got2, "non-zero time should produce non-nil pointer")
	assert.True(t, got2.Equal(now))
}

func TestUserMapToSortedSlice(t *testing.T) {
	// 空 map → nil
	assert.Nil(t, userMapToSortedSlice(map[int]dto.WorkflowUserInfo{}))

	// 非空 map → 按 ID 升序
	m := map[int]dto.WorkflowUserInfo{
		3: {ID: 3, Username: "c"},
		1: {ID: 1, Username: "a"},
		2: {ID: 2, Username: "b"},
	}
	out := userMapToSortedSlice(m)
	require.Len(t, out, 3)
	assert.Equal(t, 1, out[0].ID)
	assert.Equal(t, 2, out[1].ID)
	assert.Equal(t, 3, out[2].ID)
}

func TestMapBpmnTaskOutcome(t *testing.T) {
	// nil task → 空串
	assert.Equal(t, "", mapBpmnTaskOutcome(nil))

	cases := []struct {
		status string
		want   string
	}{
		{"completed", "completed"},
		{"cancelled", "cancelled"},
		{"started", "started"}, // 透传
		{"created", "created"},
		{"", ""},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			task := &ent.ProcessTask{Status: c.status}
			assert.Equal(t, c.want, mapBpmnTaskOutcome(task))
		})
	}
}

// ==================== BPMN XML 解析单元测试 ====================

const (
	bpmnApprovalXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" name="Approval" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="提交"/>
    <bpmn:userTask id="Task_Approve" name="审批"/>
    <bpmn:exclusiveGateway id="Gateway_1" name="是否通过"/>
    <bpmn:userTask id="Task_Reject" name="驳回处理"/>
    <bpmn:endEvent id="EndEvent_1" name="结束"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_Approve"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_Approve" targetRef="Gateway_1"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Gateway_1" targetRef="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_4" sourceRef="Gateway_1" targetRef="Task_Reject"/>
  </bpmn:process>
</bpmn:definitions>`
)

func TestParseBpmnProcessGraph_Namespaced(t *testing.T) {
	g, err := parseBpmnProcessGraph([]byte(bpmnApprovalXML))
	require.NoError(t, err)
	require.NotNil(t, g)

	// 节点名称/类型映射
	assert.Equal(t, "提交", g.nodeNames["StartEvent_1"])
	assert.Equal(t, "审批", g.nodeNames["Task_Approve"])
	assert.Equal(t, "是否通过", g.nodeNames["Gateway_1"])
	assert.Equal(t, "驳回处理", g.nodeNames["Task_Reject"])

	assert.Equal(t, "startEvent", g.nodeTypes["StartEvent_1"])
	assert.Equal(t, "userTask", g.nodeTypes["Task_Approve"])
	assert.Equal(t, "exclusiveGateway", g.nodeTypes["Gateway_1"])
	assert.Equal(t, "userTask", g.nodeTypes["Task_Reject"])
	assert.Equal(t, "endEvent", g.nodeTypes["EndEvent_1"])

	// 网关标记
	assert.True(t, g.gatewayIDs["Gateway_1"], "exclusiveGateway should be flagged")
	assert.False(t, g.gatewayIDs["Task_Approve"])

	// 出边
	assert.Equal(t, []string{"Task_Approve"}, g.outgoing["StartEvent_1"])
	assert.Equal(t, []string{"Gateway_1"}, g.outgoing["Task_Approve"])
	require.Len(t, g.outgoing["Gateway_1"], 2, "gateway should have two outgoing flows")
	assert.ElementsMatch(t, []string{"EndEvent_1", "Task_Reject"}, g.outgoing["Gateway_1"])
	assert.Nil(t, g.outgoing["EndEvent_1"], "end event has no outgoing flows")
}

func TestParseBpmnProcessGraph_NonNamespaced(t *testing.T) {
	// 兼容无 bpmn: 命名空间的 XML
	xmlNoNS := `<?xml version="1.0" encoding="UTF-8"?>
<definitions>
  <process id="P">
    <startEvent id="S"/>
    <userTask id="U" name="审核"/>
    <sequenceFlow id="f1" sourceRef="S" targetRef="U"/>
  </process>
</definitions>`
	g, err := parseBpmnProcessGraph([]byte(xmlNoNS))
	require.NoError(t, err)
	assert.Equal(t, "startEvent", g.nodeTypes["S"])
	assert.Equal(t, "userTask", g.nodeTypes["U"])
	assert.Equal(t, "审核", g.nodeNames["U"])
	assert.Equal(t, []string{"U"}, g.outgoing["S"])
}

func TestParseBpmnProcessGraph_EmptyAndInvalid(t *testing.T) {
	// 空输入 → 空 graph，不报错
	g, err := parseBpmnProcessGraph(nil)
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Empty(t, g.outgoing)

	// 非法 XML → 报错
	_, err = parseBpmnProcessGraph([]byte("<not-xml"))
	assert.Error(t, err)
}

func TestParseBpmnProcessGraph_MalformedSequenceFlowSkipped(t *testing.T) {
	// 缺 sourceRef/targetRef 的 sequenceFlow 应被跳过，不污染 outgoing map
	xmlBad := `<?xml version="1.0" encoding="UTF-8"?>
<definitions>
  <process>
    <sequenceFlow id="bad1"/>
    <sequenceFlow id="bad2" sourceRef="A"/>
    <sequenceFlow id="bad3" targetRef="B"/>
    <sequenceFlow id="good" sourceRef="A" targetRef="B"/>
    <startEvent id="A"/>
    <endEvent id="B"/>
  </process>
</definitions>`
	g, err := parseBpmnProcessGraph([]byte(xmlBad))
	require.NoError(t, err)
	assert.Equal(t, []string{"B"}, g.outgoing["A"])
}

// ==================== enrichBpmnProcessState 集成测试（in-memory sqlite） ====================

func newBpmnStateTestClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { client.Close() })
	return client
}

func createBpmnTestTenant(t *testing.T, client *ent.Client, suffix string) *ent.Tenant {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("BPMN State Tenant " + suffix).
		SetCode("bs" + suffix).
		SetDomain("bs" + suffix + ".test").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant
}

func createBpmnTestUser(t *testing.T, client *ent.Client, tenantID int, suffix string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername("bsuser" + suffix).
		SetEmail("bs" + suffix + "@test.com").
		SetName("BPMN State User " + suffix).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func createBpmnTestTicket(t *testing.T, client *ent.Client, tenantID, requesterID int) *ent.Ticket {
	t.Helper()
	tk, err := client.Ticket.Create().
		SetTitle("BPMN State Ticket").
		SetStatus("in_progress").
		SetPriority("medium").
		SetTicketNumber(fmt.Sprintf("BS-TKT-%d-%d", tenantID, time.Now().UnixNano())).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return tk
}

func createBpmnDeploymentAndDefinition(t *testing.T, client *ent.Client, tenantID int, key, bpmnXML string) *ent.ProcessDefinition {
	t.Helper()
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).
		SetDeploymentName("Deployment " + key).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("Definition " + key).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def
}

func TestEnrichBpmnProcessState_NoInstanceReturnsNotStarted(t *testing.T) {
	client := newBpmnStateTestClient(t, "bpmn_state_none")
	tenant := createBpmnTestTenant(t, client, "none")
	user := createBpmnTestUser(t, client, tenant.ID, "none")
	tk := createBpmnTestTicket(t, client, tenant.ID, user.ID)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.enrichBpmnProcessState(context.Background(), tk, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "not_started", state.BpmnStatus)
	assert.Empty(t, state.CurrentActivityID)
	assert.Empty(t, state.NextActivities)
	assert.Empty(t, state.History)
}

func TestEnrichBpmnProcessState_NilTicketReturnsNotStarted(t *testing.T) {
	client := newBpmnStateTestClient(t, "bpmn_state_nil")
	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.enrichBpmnProcessState(context.Background(), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, "not_started", state.BpmnStatus)
}

func TestEnrichBpmnProcessState_RunningReturnsCurrentAndNext(t *testing.T) {
	client := newBpmnStateTestClient(t, "bpmn_state_running")
	tenant := createBpmnTestTenant(t, client, "run")
	requester := createBpmnTestUser(t, client, tenant.ID, "req")
	assignee := createBpmnTestUser(t, client, tenant.ID, "asg")
	tk := createBpmnTestTicket(t, client, tenant.ID, requester.ID)

	def := createBpmnDeploymentAndDefinition(t, client, tenant.ID, "approval_process", bpmnApprovalXML)
	ctx := context.Background()
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-RUN").
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(fmt.Sprintf("ticket:%d", tk.ID)).
		SetStatus("running").
		SetCurrentActivityID("Task_Approve").
		SetCurrentActivityName("审批").
		SetStartTime(time.Now()).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 一个已完成节点（历史）
	_, err = client.ProcessTask.Create().
		SetTaskID("T-DONE").
		SetTaskDefinitionKey("StartEvent_1").
		SetTaskName("提交").
		SetTaskType("startEvent").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(requester.ID)).
		SetStatus("completed").
		SetCreatedTime(time.Now().Add(-2 * time.Hour)).
		SetCompletedTime(time.Now().Add(-90 * time.Minute)).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 当前节点（任务匹配 current_activity_id）
	_, err = client.ProcessTask.Create().
		SetTaskID("T-CURRENT").
		SetTaskDefinitionKey("Task_Approve").
		SetTaskName("审批").
		SetTaskType("userTask").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(assignee.ID)).
		SetStatus("assigned").
		SetCreatedTime(time.Now().Add(-30 * time.Minute)).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.enrichBpmnProcessState(ctx, tk, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, "running", state.BpmnStatus)
	assert.Equal(t, "PI-RUN", state.ProcessInstanceID)
	assert.Equal(t, def.Key, state.ProcessDefinitionKey)
	assert.Equal(t, "Definition "+def.Key, state.ProcessDefinitionName)
	assert.Equal(t, "Task_Approve", state.CurrentActivityID)
	assert.Equal(t, "审批", state.CurrentActivityName)
	assert.Equal(t, "userTask", state.CurrentActivityType)
	require.Len(t, state.CurrentAssignees, 1)
	assert.Equal(t, assignee.ID, state.CurrentAssignees[0].ID)

	// NextActivities: 当前节点 Task_Approve → Gateway_1
	require.Len(t, state.NextActivities, 1)
	assert.Equal(t, "Gateway_1", state.NextActivities[0].ActivityID)
	assert.Equal(t, "是否通过", state.NextActivities[0].ActivityName)
	assert.Equal(t, "exclusiveGateway", state.NextActivities[0].ActivityType)
	assert.True(t, state.NextActivities[0].IsGateway)

	// History: 已完成 StartEvent_1
	require.Len(t, state.History, 1)
	assert.Equal(t, "StartEvent_1", state.History[0].ActivityID)
	assert.Equal(t, "completed", state.History[0].Outcome)
	require.NotNil(t, state.History[0].Assignee)
	assert.Equal(t, requester.ID, state.History[0].Assignee.ID)
}

func TestEnrichBpmnProcessState_TerminalReturnsNoCurrentOrNext(t *testing.T) {
	client := newBpmnStateTestClient(t, "bpmn_state_done")
	tenant := createBpmnTestTenant(t, client, "done")
	user := createBpmnTestUser(t, client, tenant.ID, "done")
	tk := createBpmnTestTicket(t, client, tenant.ID, user.ID)

	def := createBpmnDeploymentAndDefinition(t, client, tenant.ID, "approval_done", bpmnApprovalXML)
	ctx := context.Background()
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DONE").
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(fmt.Sprintf("ticket:%d", tk.ID)).
		SetStatus("completed").
		SetCurrentActivityID("EndEvent_1"). // 终态下 current 不应被填充
		SetCurrentActivityName("结束").
		SetStartTime(time.Now().Add(-time.Hour)).
		SetEndTime(time.Now()).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 一个已完成节点，贡献 history
	_, err = client.ProcessTask.Create().
		SetTaskID("T-1").
		SetTaskDefinitionKey("Task_Approve").
		SetTaskName("审批").
		SetTaskType("userTask").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(user.ID)).
		SetStatus("completed").
		SetCreatedTime(time.Now().Add(-time.Hour)).
		SetCompletedTime(time.Now()).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.enrichBpmnProcessState(ctx, tk, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, "completed", state.BpmnStatus)
	assert.Empty(t, state.CurrentActivityID, "终态不应暴露 currentActivityId")
	assert.Empty(t, state.CurrentActivityName)
	assert.Empty(t, state.NextActivities, "终态不应计算 next")
	require.Len(t, state.History, 1, "终态仍需返回 history 供详情页回溯")
	require.NotNil(t, state.EndedAt)
}

func TestEnrichBpmnProcessState_TenantIsolation(t *testing.T) {
	// 工单在租户 A，BPMN 实例在租户 B：应返回 not_started（fail closed）
	client := newBpmnStateTestClient(t, "bpmn_state_iso")
	tenantA := createBpmnTestTenant(t, client, "A")
	tenantB := createBpmnTestTenant(t, client, "B")
	userA := createBpmnTestUser(t, client, tenantA.ID, "A")
	tk := createBpmnTestTicket(t, client, tenantA.ID, userA.ID)

	def := createBpmnDeploymentAndDefinition(t, client, tenantB.ID, "approval_B", bpmnApprovalXML)
	ctx := context.Background()
	_, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-OTHER-TENANT").
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(fmt.Sprintf("ticket:%d", tk.ID)). // 业务键相同
		SetStatus("running").
		SetTenantID(tenantB.ID). // 但属于不同租户
		Save(ctx)
	require.NoError(t, err)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.enrichBpmnProcessState(ctx, tk, tenantA.ID)
	require.NoError(t, err)
	assert.Equal(t, "not_started", state.BpmnStatus, "跨租户实例不应被聚合到工单详情")
}

// ==================== GetTicketWorkflowStateV2 端到端契约测试 ====================

func TestGetTicketWorkflowStateV2_NoTicketReturnsError(t *testing.T) {
	client := newBpmnStateTestClient(t, "v2_no_ticket")
	tenant := createBpmnTestTenant(t, client, "v2")
	user := createBpmnTestUser(t, client, tenant.ID, "v2")

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	_, err := svc.GetTicketWorkflowStateV2(context.Background(), 999999, user.ID, tenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工单不存在")
}

func TestGetTicketWorkflowStateV2_TenantIsolation(t *testing.T) {
	client := newBpmnStateTestClient(t, "v2_tenant_iso")
	tenantA := createBpmnTestTenant(t, client, "A")
	tenantB := createBpmnTestTenant(t, client, "B")
	userA := createBpmnTestUser(t, client, tenantA.ID, "A")
	tk := createBpmnTestTicket(t, client, tenantA.ID, userA.ID)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	// 用 tenantB 的 userID 访问 tenantA 的工单
	_, err := svc.GetTicketWorkflowStateV2(context.Background(), tk.ID, userA.ID, tenantB.ID)
	require.Error(t, err, "跨租户访问应被拒绝 (fail closed)")
}

func TestGetTicketWorkflowStateV2_ContractCamelCase(t *testing.T) {
	client := newBpmnStateTestClient(t, "v2_contract")
	tenant := createBpmnTestTenant(t, client, "cc")
	user := createBpmnTestUser(t, client, tenant.ID, "cc")
	tk := createBpmnTestTicket(t, client, tenant.ID, user.ID)

	svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
	state, err := svc.GetTicketWorkflowStateV2(context.Background(), tk.ID, user.ID, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, state)

	// 序列化验证 JSON 字段名都是 camelCase
	raw, err := json.Marshal(state)
	require.NoError(t, err)
	js := string(raw)
	for _, expect := range []string{
		`"ticketId"`,
		`"currentStatus"`,
		`"bpmnProcessState"`,
		`"bpmnStatus"`,
		`"processInstanceId"`,
	} {
		assert.Contains(t, js, expect, "响应应包含 camelCase 字段 %s", expect)
	}
	// 不应暴露下划线字段
	assert.NotContains(t, js, `"process_instance_id"`)
	assert.NotContains(t, js, `"bpmn_status"`)

	// 顶层 BpmnProcessState 已嵌入
	require.NotNil(t, state.BpmnProcessState)
	assert.Equal(t, "not_started", state.BpmnProcessState.BpmnStatus)
}

// ==================== 端点契约 HTTP 测试 ====================

func TestGetTicketWorkflowStateV2_HTTPContract(t *testing.T) {
	client := newBpmnStateTestClient(t, "v2_http")
	tenant := createBpmnTestTenant(t, client, "http")
	user := createBpmnTestUser(t, client, tenant.ID, "http")
	tk := createBpmnTestTicket(t, client, tenant.ID, user.ID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/tickets/:id/workflow/state-v2", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("tenant_id", tenant.ID)
		tid, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "PARAM_INVALID", "message": "无效的工单ID"})
			return
		}
		svc := NewTicketWorkflowService(client, zaptest.NewLogger(t).Sugar())
		state, err := svc.GetTicketWorkflowStateV2(c.Request.Context(), tid, user.ID, tenant.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": state})
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/workflow/state-v2", tk.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP 200 期望")
	body := w.Body.String()
	assert.Contains(t, body, `"code":0`)
	assert.Contains(t, body, `"bpmnProcessState"`)
	assert.Contains(t, body, `"bpmnStatus":"not_started"`)
}

// ==================== ensure 引用的 import 不被 lint 删除 ====================

var _ = strings.TrimSpace