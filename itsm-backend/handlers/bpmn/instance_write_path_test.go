package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processauditlog"
	"itsm-backend/ent/processinstance"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这一组测试锁住 2026-11-07 在 prod（v1.6.46）实测到的实例写路径三件套缺陷：
//   - PUT /api/v1/bpmn/process-instances/42/suspend 返回 404「资源不存在或已被删除」，
//     同一实例改用 PI-* 业务键则 200；而实例详情早已双口径可寻址，列表 DTO 的 id
//     就是数字主键，直接回传它的 CLI/MCP 调用方在生命周期路由上必然失败。
//   - POST /bpmn/process-instances 把客户端传的 `_injected` 原样落库
//     （SQL 证据：variables = {"_injected":"x","probeMarker":"by-id"}），
//     引擎控制命名空间可被调用方污染；变量 PATCH 虽然拒绝 `_` 键却兜底成 500。
//   - 同一个 POST 响应里 currentActivityId=Start_1，库里该行已是 Task_Approve：
//     返回的是创建时的旧快照，不是 executeStep 推进后的真实状态。

func doInstanceAction(t *testing.T, r *gin.Engine, method, path, body, tenant string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var env map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return w.Code, env
}

func reloadInstance(t *testing.T, client *ent.Client, id int) *ent.ProcessInstance {
	t.Helper()
	instance, err := client.ProcessInstance.Get(context.Background(), id)
	require.NoError(t, err)
	return instance
}

// 开始 → 用户任务 → 结束：启动后实例应停在 Task_Approve（与 prod 实测的流程同形）。
const instanceStartWireBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:userTask id="Task_Approve" name="审批"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="Flow_0" sourceRef="Start_1" targetRef="Task_Approve"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Task_Approve" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

// 开始 → 结束：启动即完成，用来验证响应快照不是只对用户任务分支正确。
const instanceAutoWireBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="Flow_0" sourceRef="Start_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

func seedStartableDefinition(t *testing.T, client *ent.Client, key, bpmnXML string) *ent.ProcessDefinition {
	t.Helper()
	ctx := context.Background()
	deployment := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).SetDeploymentName(key).
		SetIsActive(true).SetTenantID(1).SaveX(ctx)
	return client.ProcessDefinition.Create().
		SetKey(key).SetName(key).SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).SetTenantID(1).SaveX(ctx)
}

// TestInstanceLifecycle_NumericIDIsTheSameContract 数字主键与 PI-* 业务键必须落到同一行：
// 详情、暂停、恢复、终止四个入口同口径；非法迁移是 409 而不是 404；
// 不存在与跨租户一律 404/4004，文案带上调用方传的标识。
func TestInstanceLifecycle_NumericIDIsTheSameContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_instance_lifecycle_addr?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	r := newTaskListTestRouter(t, client)

	definition := seedProcessDefinition(t, client, 1)
	instance := seedInstance(t, client, definition, "PI-life-addr")
	// 审计的 activity_id 有非空校验，实例必须停在某个节点上（prod 实例均如此）。
	client.ProcessInstance.UpdateOneID(instance.ID).
		SetCurrentActivityID("approval").SetCurrentActivityName("审批").
		ExecX(context.Background())
	numeric := strconv.Itoa(instance.ID)
	base := "/api/v1/bpmn/process-instances/"

	status, env := doInstanceAction(t, r, http.MethodGet, base+numeric, "", "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	data, ok := env["data"].(map[string]interface{})
	require.True(t, ok, "%v", env)
	assert.Equal(t, "PI-life-addr", data["instanceId"], "数字主键详情必须回规范化的业务键")

	status, env = doInstanceAction(t, r, http.MethodPut, base+numeric+"/suspend", `{"reason":"实测暂停"}`, "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	assert.EqualValues(t, 0, env["code"], "%v", env)
	stored := reloadInstance(t, client, instance.ID)
	assert.Equal(t, "suspended", stored.Status)
	assert.Equal(t, "实测暂停", stored.SuspendedReason, "暂停理由必须落库")

	// 业务键指向同一行：重复暂停是状态冲突，不是「实例不存在」。
	status, env = doInstanceAction(t, r, http.MethodPut, base+instance.ProcessInstanceID+"/suspend", `{"reason":"再来一次"}`, "1")
	assert.Equal(t, http.StatusConflict, status, "%v", env)
	assert.EqualValues(t, common.ConflictCode, env["code"], "%v", env)
	assert.Contains(t, env["message"], "不能暂停")
	assert.Contains(t, env["message"], "running", "冲突文案要给出允许的 source 状态")

	status, env = doInstanceAction(t, r, http.MethodPut, base+numeric+"/resume", "", "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	assert.Equal(t, "running", reloadInstance(t, client, instance.ID).Status)

	status, env = doInstanceAction(t, r, http.MethodPut, base+instance.ProcessInstanceID+"/resume", "", "1")
	assert.Equal(t, http.StatusConflict, status, "%v", env)
	assert.EqualValues(t, common.ConflictCode, env["code"], "%v", env)

	status, env = doInstanceAction(t, r, http.MethodPut, base+numeric+"/terminate", `{"reason":"实测终止"}`, "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	stored = reloadInstance(t, client, instance.ID)
	assert.Equal(t, "terminated", stored.Status)
	assert.False(t, stored.EndTime.IsZero(), "终止必须写结束时间")

	status, env = doInstanceAction(t, r, http.MethodPut, base+instance.ProcessInstanceID+"/suspend", `{"reason":"已终止"}`, "1")
	assert.Equal(t, http.StatusConflict, status, "已终止实例用业务键寻址仍应是 409 而不是 404: %v", env)

	for _, identifier := range []string{"PI-life-missing", "999999"} {
		status, env = doInstanceAction(t, r, http.MethodPut, base+identifier+"/suspend", `{"reason":"x"}`, "1")
		assert.Equal(t, http.StatusNotFound, status, "%v", env)
		assert.EqualValues(t, common.NotFoundCode, env["code"], "%v", env)
		assert.Contains(t, env["message"], identifier, "404 文案要带上调用方传的标识")
	}

	// 跨租户：租户 2 用租户 1 的实例（两种寻址都试）只能得到 404，且状态不得被改写。
	for _, identifier := range []string{numeric, instance.ProcessInstanceID} {
		status, env = doInstanceAction(t, r, http.MethodPut, base+identifier+"/terminate", `{"reason":"越权"}`, "2")
		assert.Equal(t, http.StatusNotFound, status, "%v", env)
		assert.EqualValues(t, common.NotFoundCode, env["code"], "%v", env)
	}
	assert.Equal(t, "terminated", reloadInstance(t, client, instance.ID).Status)

	// 三个动作各留一条审计，操作人取自认证上下文而不是 0；被拒绝的重复动作不留假审计。
	for _, action := range []string{"suspended", "resumed", "terminated"} {
		rows := client.ProcessAuditLog.Query().
			Where(processauditlog.ProcessInstanceID(instance.ID), processauditlog.Action(action)).
			AllX(context.Background())
		require.Len(t, rows, 1, "%s 必须且只能落一条审计", action)
		assert.Equal(t, 7, rows[0].UserID, "审计操作人必须是认证上下文里的用户")
		assert.Equal(t, 1, rows[0].TenantID)
		assert.Equal(t, "approval", rows[0].ActivityID)
	}
}

// TestStartProcess_RejectsEngineControlVariables 启动与变量 PATCH 是同一条写入边界：
// `_` 前缀的引擎控制键必须 400/4000 拒绝（不是 200 落库，也不是 500 兜底），
// 被拒绝的启动不得留下实例，被拒绝的 PATCH 不得改动库里任何变量。
func TestStartProcess_RejectsEngineControlVariables(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_start_control_vars?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	r := newTaskListTestRouter(t, client)
	seedStartableDefinition(t, client, "start-guard", instanceStartWireBPMN)

	status, env := doInstanceAction(t, r, http.MethodPost, "/api/v1/bpmn/process-instances",
		`{"processDefinitionKey":"start-guard","businessKey":"wf-guard-rejected","variables":{"_injected":"x","probeMarker":"ok"}}`, "1")
	require.Equal(t, http.StatusBadRequest, status, "%v", env)
	assert.EqualValues(t, common.BadRequestCode, env["code"], "%v", env)
	assert.Contains(t, env["message"], "_injected")
	assert.Zero(t, client.ProcessInstance.Query().
		Where(processinstance.BusinessKey("wf-guard-rejected")).CountX(ctx), "被拒绝的启动不得留下实例")

	status, env = doInstanceAction(t, r, http.MethodPost, "/api/v1/bpmn/process-instances",
		`{"processDefinitionKey":"start-guard","businessKey":"wf-guard-ok","variables":{"probeMarker":"ok"}}`, "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	data, ok := env["data"].(map[string]interface{})
	require.True(t, ok, "%v", env)
	key, _ := data["instanceId"].(string)
	require.NotEmpty(t, key)

	stored := client.ProcessInstance.Query().Where(processinstance.ProcessInstanceID(key)).OnlyX(ctx)
	require.NotContains(t, stored.Variables, "_injected")
	assert.Equal(t, "ok", stored.Variables["probeMarker"])
	before := stored.Variables

	status, env = doInstanceAction(t, r, http.MethodPut, "/api/v1/bpmn/process-instances/"+key+"/variables",
		`{"variables":{"_done_":{"Task_Approve":true}}}`, "1")
	assert.Equal(t, http.StatusBadRequest, status, "变量 PATCH 拒绝控制键必须是 400 而不是 500: %v", env)
	assert.EqualValues(t, common.BadRequestCode, env["code"], "%v", env)
	assert.Contains(t, env["message"], "_done_")
	assert.Equal(t, before, reloadInstance(t, client, stored.ID).Variables, "被拒绝的 PATCH 不得改动引擎状态")

	status, env = doInstanceAction(t, r, http.MethodPut, "/api/v1/bpmn/process-instances/"+strconv.Itoa(stored.ID)+"/variables",
		`{"variables":{"businessNote":"按数字主键改"}}`, "1")
	require.Equal(t, http.StatusOK, status, "%v", env)
	assert.Equal(t, "按数字主键改", reloadInstance(t, client, stored.ID).Variables["businessNote"])
}

// TestStartProcess_ResponseReflectsExecutedState 启动响应必须是 executeStep 推进后的真实快照：
// 停在用户任务时 currentActivityId 是任务节点而不是 Start_1；直接走完的流程必须报 completed。
// 响应与库里同一行逐字段一致，且响应里的数字 id 能直接用于后续生命周期动作。
func TestStartProcess_ResponseReflectsExecutedState(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_start_snapshot?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	r := newTaskListTestRouter(t, client)
	seedStartableDefinition(t, client, "snapshot-user-task", instanceStartWireBPMN)
	seedStartableDefinition(t, client, "snapshot-auto-end", instanceAutoWireBPMN)

	cases := []struct {
		key            string
		wantStatus     string
		wantActivityID string
		wantEndTime    bool
	}{
		{"snapshot-user-task", "running", "Task_Approve", false},
		{"snapshot-auto-end", "completed", "End_1", true},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			status, env := doInstanceAction(t, r, http.MethodPost, "/api/v1/bpmn/process-instances",
				`{"processDefinitionKey":"`+tc.key+`","businessKey":"bk-`+tc.key+`","variables":{"probeMarker":"snapshot"}}`, "1")
			require.Equal(t, http.StatusOK, status, "%v", env)
			data, ok := env["data"].(map[string]interface{})
			require.True(t, ok, "%v", env)

			id, _ := data["id"].(string)
			require.NotEmpty(t, id)
			numericID, err := strconv.Atoi(id)
			require.NoError(t, err)
			stored := reloadInstance(t, client, numericID)

			assert.Equal(t, tc.wantStatus, data["status"])
			assert.Equal(t, tc.wantActivityID, data["currentActivityId"], "启动响应不得停留在开始事件")
			assert.Equal(t, stored.Status, data["status"], "响应状态必须与库一致")
			assert.Equal(t, stored.CurrentActivityID, data["currentActivityId"], "响应当前节点必须与库一致")
			assert.Equal(t, stored.ProcessInstanceID, data["instanceId"])
			assert.Equal(t, stored.EndTime.IsZero(), !tc.wantEndTime)
			if tc.wantEndTime {
				assert.Contains(t, data, "endTime", "已完成实例必须透出结束时间")
			} else {
				assert.NotContains(t, data, "endTime")
			}
			assert.NotContains(t, data["variables"], "_done_", "引擎控制键不属于对外契约")

			// 响应里的数字 id 就是生命周期动作的寻址键（前端实例页按钮的口径）。
			if tc.wantStatus == "running" {
				status, env = doInstanceAction(t, r, http.MethodPut,
					"/api/v1/bpmn/process-instances/"+id+"/suspend", `{"reason":"响应 id 可直接暂停"}`, "1")
				require.Equal(t, http.StatusOK, status, "%v", env)
				assert.Equal(t, "suspended", reloadInstance(t, client, numericID).Status)
			}
		})
	}
}
