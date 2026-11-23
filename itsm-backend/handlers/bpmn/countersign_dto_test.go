package bpmn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedCounterSignActors 补上 P1-03 对象级授权所需的最小世界：
// 首个创建的租户拿到 ID 1（与 X-Test-Tenant=1 的口径一致），随后连续创建
// count 个活跃用户（ID 1..count），让路由上下文里的 user_id=7 与审批人 8/9
// 都真实存在于租户 1——AuthorizeTaskAccess 与 validateTargetUser 都要求
// 账号真实、同租户且活跃，缺任何一环会签创建都会按设计拒绝。
func seedCounterSignActors(t *testing.T, client *ent.Client, count int) {
	t.Helper()
	ctx := context.Background()
	_, err := client.Tenant.Create().
		SetName("会签测试租户").SetCode("cs-actors").SetDomain("cs-actors.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("cs-actor-%d", i)
		_, err := client.User.Create().
			SetUsername(name).SetEmail(name + "@test.com").SetName(name).
			SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(1).
			Save(ctx)
		require.NoError(t, err)
	}
}

// POST /api/v1/bpmn/tasks/:id/counter-sign 曾直接 common.Success(tasks) 序列化
// []*ent.ProcessTask，把 Ent 的 snake_case 字段和内部列暴露到接口上。这里锁住返回
// dto.BPMNTaskResponse 的 camelCase 契约，并证明跨租户父任务无法被会签。

type counterSignEnvelope struct {
	status  int
	code    int
	message string
	raw     string
	items   []map[string]interface{}
}

func doCounterSign(t *testing.T, r *gin.Engine, parentTaskID, body, tenant string) counterSignEnvelope {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bpmn/tasks/"+parentTaskID+"/counter-sign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var parsed struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed), w.Body.String())

	env := counterSignEnvelope{status: w.Code, code: parsed.Code, message: parsed.Message, raw: w.Body.String()}
	if len(parsed.Data) > 0 && string(parsed.Data) != "null" {
		require.NoError(t, json.Unmarshal(parsed.Data, &env.items), "data 非空时必须是任务数组: %s", w.Body.String())
	}
	return env
}

func TestCreateCounterSignTasks_ReturnsDTO(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_countersign_dto?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	// user 7 = 路由操作人；8/9 = 会签审批人（P1-03 要求账号真实存在且活跃）。
	seedCounterSignActors(t, client, 9)
	definition := seedProcessDefinition(t, client, 1)
	instance := seedInstance(t, client, definition, "PI-cs")
	seedAssignedTask(t, client, instance, "task-cs-parent")

	r := newTaskListTestRouter(t, client)
	env := doCounterSign(t, r, "task-cs-parent",
		`{"approvalType":"parallel","approvers":["8","9"],"threshold":2}`, "1")

	require.Equal(t, http.StatusOK, env.status, env.raw)
	require.Equal(t, 0, env.code, env.message)
	require.Len(t, env.items, 2)

	for i, item := range env.items {
		// 后端 DTO 必须出现的 camelCase 键。
		assert.Equal(t, "task-cs-parent_countersign_"+strconv.Itoa(i), item["taskId"])
		assert.Equal(t, "approval_counter", item["taskDefinitionKey"])
		assert.Contains(t, []string{"8", "9"}, item["assignee"])
		assert.Equal(t, "assigned", item["status"])
		assert.Equal(t, float64(instance.ID), item["processInstanceId"], "实例关联用 DTO 的 processInstanceId")
		assert.Contains(t, item, "createdTime", "Ent 的 created_time 必须以 createdTime 返回")

		// Ent 模型字段一律不得出现在响应里。
		for _, leaked := range []string{
			"task_id", "process_instance_id", "task_definition_key", "created_time",
			"parent_task_id", "root_task_id", "tenant_id", "completed_time",
		} {
			assert.NotContains(t, item, leaked, "响应泄漏了 Ent 字段 %s: %s", leaked, env.raw)
		}
	}

	// 父任务属于租户 1：租户 2 用同一个 task_id 无法创建会签子任务。
	blocked := doCounterSign(t, r, "task-cs-parent",
		`{"approvalType":"parallel","approvers":["10"],"threshold":1}`, "2")
	assert.NotEqual(t, 0, blocked.code, "跨租户父任务必须失败: %s", blocked.raw)

	count, err := client.ProcessTask.Query().
		Where(processtask.ParentTaskID("task-cs-parent")).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count, "跨租户尝试不得额外创建会签子任务")
}

// TestCreateCounterSignTasks_ErrorSemantics 会签的失败语义必须可区分：
// 修复前 service 用裸 fmt.Errorf 报「会签审批人不能为空」，被兜底成 500/5001，
// 让调用方以为服务故障；父任务不存在/跨租户则只有通用的「资源不存在或已被删除」。
func TestCreateCounterSignTasks_ErrorSemantics(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_countersign_errors?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	instance := seedInstance(t, client, definition, "PI-cs-err")
	seedAssignedTask(t, client, instance, "task-cs-empty")

	r := newTaskListTestRouter(t, client)

	empty := doCounterSign(t, r, "task-cs-empty",
		`{"approvalType":"parallel","approvers":[],"threshold":1}`, "1")
	assert.Equal(t, http.StatusBadRequest, empty.status, empty.raw)
	assert.Equal(t, 4000, empty.code, "空审批人属于请求语义错误")

	missing := doCounterSign(t, r, "task-cs-not-exist",
		`{"approvalType":"parallel","approvers":["8"],"threshold":1}`, "1")
	assert.Equal(t, http.StatusNotFound, missing.status, missing.raw)
	assert.Equal(t, 4004, missing.code, "父任务不存在必须是 404，不得兜底成 5001")

	malformed := doCounterSign(t, r, "task-cs-empty", `{"approvers":`, "1")
	assert.Equal(t, 1001, malformed.code, "JSON 绑定失败仍走参数错误码")

	// 失败路径不得留下子任务。
	count, err := client.ProcessTask.Query().
		Where(processtask.ParentTaskID("task-cs-empty")).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, count, "校验/寻址失败不得创建会签子任务")
}
