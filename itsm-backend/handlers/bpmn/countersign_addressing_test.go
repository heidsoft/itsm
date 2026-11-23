package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这一组测试锁住 2026-11-07 在 prod 实测到的会签三件套缺陷：
//   - POST/GET/PUT /api/v1/bpmn/tasks/:id/{counter-sign,counter-sign-status,vote}
//     只认 task_id 业务键，而任务列表 DTO 给前端的寻址字段是数字主键 id；
//     用列表里的 id 加签必然 404「父任务 60 不存在」，同族
//     GetTask/ClaimTask/CompleteTask 却早已双口径可寻址。
//   - GET counter-sign-status 对不存在或跨租户的父任务返回 200，且 total=0 时
//     threshold 也取 0，让 0>=0 成立，把「没有任何人审批」报成会签「已通过」。

func csParentTask(t *testing.T, client *ent.Client, taskID string) *ent.ProcessTask {
	t.Helper()
	task, err := client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		Only(context.Background())
	require.NoError(t, err)
	return task
}

type counterSignStatusResult struct {
	status  int
	code    int
	message string
	data    map[string]interface{}
	raw     string
}

// doCounterSignStatus 走真实注册的 GET /bpmn/tasks/:id/counter-sign-status。
func doCounterSignStatus(t *testing.T, r *gin.Engine, id, tenant string) counterSignStatusResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bpmn/tasks/"+id+"/counter-sign-status", nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return counterSignStatusResult{
		status: w.Code, code: body.Code, message: body.Message, data: body.Data, raw: w.Body.String(),
	}
}

func doVote(t *testing.T, r *gin.Engine, id, body, tenant string) (int, int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/bpmn/tasks/"+id+"/vote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed), w.Body.String())
	return w.Code, parsed.Code, parsed.Message
}

// countersignWireBPMN 是「开始 → 会签用户任务 → 结束」的最小可执行流程，
// 让末票推进在测试里走到真实结束事件。
const countersignWireBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start"/>
    <bpmn:userTask id="CounterSignTask" name="会签审批"/>
    <bpmn:endEvent id="EndEvent_1" name="End"/>
    <bpmn:sequenceFlow id="Flow_0" sourceRef="StartEvent_1" targetRef="CounterSignTask"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="CounterSignTask" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

// TestCounterSignRoutes_NumericTaskIDIsTheSameContract 数字主键与业务键必须落到同一个父任务：
// 两种寻址给出同构的子任务，且子任务的 parent_task_id 一律写父任务业务键
// （会签汇总与投票推进都按它关联）。修复前数字主键一律 404。
func TestCounterSignRoutes_NumericTaskIDIsTheSameContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_countersign_numeric?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	// P1-03：创建走对象级授权，路由操作人(7)与审批人(8/9)必须真实存在。
	seedCounterSignActors(t, client, 9)
	r := newTaskListTestRouter(t, client)

	definition := seedProcessDefinition(t, client, 1)
	instance := seedInstance(t, client, definition, "PI-cs-num")
	seedAssignedTask(t, client, instance, "cs-by-id")
	seedAssignedTask(t, client, instance, "cs-by-key")
	parent := csParentTask(t, client, "cs-by-id")

	byID := doCounterSign(t, r, strconv.Itoa(parent.ID),
		`{"approvalType":"parallel","approvers":["8","9"],"threshold":0}`, "1")
	require.Equal(t, http.StatusOK, byID.status, byID.raw)
	require.Len(t, byID.items, 2)

	byKey := doCounterSign(t, r, "cs-by-key",
		`{"approvalType":"parallel","approvers":["8","9"],"threshold":0}`, "1")
	require.Equal(t, http.StatusOK, byKey.status, byKey.raw)
	require.Len(t, byKey.items, 2)

	for i := range byID.items {
		assert.Equal(t, "cs-by-id_countersign_"+strconv.Itoa(i), byID.items[i]["taskId"],
			"子任务标识必须以父任务业务键为前缀")
		assert.Equal(t, byKey.items[i]["taskDefinitionKey"], byID.items[i]["taskDefinitionKey"])
	}

	children, err := client.ProcessTask.Query().
		Where(processtask.ParentTaskID("cs-by-id")).
		All(context.Background())
	require.NoError(t, err)
	require.Len(t, children, 2, "数字主键寻址的子任务必须按父任务业务键关联")

	// 父任务不存在/跨租户仍是 404，且文案带上调用方传的标识。
	unknown := doCounterSign(t, r, "TASK-cs-does-not-exist",
		`{"approvalType":"parallel","approvers":["8"],"threshold":1}`, "1")
	assert.Equal(t, http.StatusNotFound, unknown.status, unknown.raw)
	assert.Equal(t, 4004, unknown.code)
	assert.Contains(t, unknown.message, "TASK-cs-does-not-exist")
}

// TestCounterSignStatus_CanonicalKeyAndNoFakeApproval 无子任务时是 pending 而不是 approved，
// 父任务不存在或跨租户时是 404/4004 而不是空成功。
func TestCounterSignStatus_CanonicalKeyAndNoFakeApproval(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_countersign_status?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	r := newTaskListTestRouter(t, client)

	definition := seedProcessDefinition(t, client, 1)
	instance := seedInstance(t, client, definition, "PI-cs-status")
	seedAssignedTask(t, client, instance, "cs-status-parent")
	parent := csParentTask(t, client, "cs-status-parent")

	byNumber := doCounterSignStatus(t, r, strconv.Itoa(parent.ID), "1")
	require.Equal(t, http.StatusOK, byNumber.status, byNumber.raw)
	assert.Equal(t, "cs-status-parent", byNumber.data["parentTaskId"], "响应回规范化的业务键")
	assert.Equal(t, float64(0), byNumber.data["total"])
	assert.Equal(t, "pending", byNumber.data["status"], "没有会签子任务不得报 approved")

	byKey := doCounterSignStatus(t, r, "cs-status-parent", "1")
	require.Equal(t, http.StatusOK, byKey.status, byKey.raw)
	assert.Equal(t, byNumber.data, byKey.data, "数字主键与业务键必须给出完全相同的状态")

	unknown := doCounterSignStatus(t, r, "TASK-cs-not-exist", "1")
	assert.Equal(t, http.StatusNotFound, unknown.status, unknown.raw)
	assert.Equal(t, 4004, unknown.code, unknown.raw)
	assert.NotContains(t, unknown.raw, `"status"`, "父任务不存在不得返回会签统计: %s", unknown.raw)

	// 跨租户：租户 2 用租户 1 的任务标识（两种寻址都试）只能得到 404。
	crossNumber := doCounterSignStatus(t, r, strconv.Itoa(parent.ID), "2")
	assert.Equal(t, http.StatusNotFound, crossNumber.status, crossNumber.raw)
	assert.Equal(t, 4004, crossNumber.code, crossNumber.raw)
	crossKey := doCounterSignStatus(t, r, "cs-status-parent", "2")
	assert.Equal(t, http.StatusNotFound, crossKey.status, crossKey.raw)
	assert.Equal(t, 4004, crossKey.code, crossKey.raw)
}

// TestVote_AddressesCounterSignChildByNumericID 投票与同族任务路由共用一套寻址，并走完整链路：
// 末票达标后父任务由系统推进，实例进入结束事件。
// 修复前只有 TASK-* 可投票，任务列表里的数字 id 直接 404。
func TestVote_AddressesCounterSignChildByNumericID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_countersign_vote?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	// 先建租户，保证它拿到 ID 1（路由测试的 X-Test-Tenant 口径）。
	tenant := client.Tenant.Create().
		SetName("会签").SetCode("cs-wire").SetDomain("cs-wire.test").SetStatus("active").SaveX(ctx)
	approver := client.User.Create().
		SetUsername("cs-approver").SetEmail("cs-approver@test.com").SetName("会签审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)

	deployment := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CS-WIRE").SetDeploymentName("cs-wire").
		SetIsActive(true).SetTenantID(tenant.ID).SaveX(ctx)
	definition := client.ProcessDefinition.Create().
		SetKey("cs-vote-flow").SetName("会签推进").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(countersignWireBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).SaveX(ctx)
	instance := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-cs-vote").
		SetProcessDefinitionKey(definition.Key).SetProcessDefinitionID(definition.ID).
		SetStatus("running").SetCurrentActivityID("CounterSignTask").
		SetVariables(map[string]interface{}{}).SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.ProcessTask.Create().
		SetTaskID("cs-vote-parent").SetTaskDefinitionKey("CounterSignTask").SetTaskName("会签父任务").
		SetProcessDefinitionKey(definition.Key).SetProcessInstanceID(instance.ID).
		SetStatus("assigned").SetAssignee(strconv.Itoa(approver.ID)).SetTenantID(tenant.ID).
		SetCreatedTime(time.Now()).SaveX(ctx)

	r := newTaskActorTestRouter(t, client, approver.ID)
	env := doCounterSign(t, r, strconv.Itoa(parent.ID),
		`{"approvalType":"parallel","approvers":["`+strconv.Itoa(approver.ID)+`"],"threshold":1}`, "1")
	require.Equal(t, http.StatusOK, env.status, env.raw)
	require.Len(t, env.items, 1)
	childID := strconv.Itoa(int(env.items[0]["id"].(float64)))

	status, code, message := doVote(t, r, childID, `{"approved":true,"comment":"同意"}`, "1")
	require.Equal(t, http.StatusOK, status, message)
	require.Equal(t, 0, code, message)

	completed, err := client.ProcessTask.Query().
		Where(processtask.TaskID("cs-vote-parent_countersign_0"), processtask.Status("completed")).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, completed, "数字主键投票必须真的完成该会签子任务")

	finished, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", finished.Status, "末票达标后实例应由系统推进到结束")
}
