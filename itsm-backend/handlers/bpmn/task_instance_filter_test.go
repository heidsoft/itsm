package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 实例详情「节点任务」页签的唯一数据源是 GET /api/v1/bpmn/tasks?processInstanceKey=。
// 该查询参数曾绑定为 form:"-"，前端传了也被后端忽略，于是详情页把「整个租户的待办」
// 当成当前实例的节点展示。这里锁住：过滤真实生效、跨租户实例键查不到、不带键时行为不变。

func newTaskListTestRouter(t *testing.T, client *ent.Client) *gin.Engine {
	t.Helper()
	return newTaskActorTestRouter(t, client, 7)
}

// newTaskActorTestRouter 与 newTaskListTestRouter 相同，只把认证上下文里的 user_id
// 换成用例自己的审批人，供需要真实归属校验的写路由（投票/认领）使用。
func newTaskActorTestRouter(t *testing.T, client *ent.Client, userID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewWorkflowHandler(service.NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()), nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if c.GetHeader("X-Test-Tenant") == "2" {
			tenantID = 2
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "admin")
		c.Set("client", client)
	})
	handler.RegisterRoutes(r.Group("/api/v1"))
	return r
}

// listTaskEnvelope 请求任务列表并返回 data 信封 + 每条任务的 processInstanceKey。
func listTaskEnvelope(t *testing.T, r *gin.Engine, query, tenant string) (map[string]interface{}, []string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bpmn/tasks"+query, nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, 0, body.Code, body.Message)

	rawItems, ok := body.Data["items"].([]interface{})
	require.True(t, ok, "items 必须是数组: %s", w.Body.String())
	keys := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		task, ok := item.(map[string]interface{})
		require.True(t, ok, "任务项必须是对象: %s", w.Body.String())
		key, _ := task["processInstanceKey"].(string)
		keys = append(keys, key)
	}
	return body.Data, keys
}

func seedInstance(t *testing.T, client *ent.Client, definition *ent.ProcessDefinition, key string) *ent.ProcessInstance {
	t.Helper()
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID(key).
		SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).
		SetTenantID(definition.TenantID).
		SetStatus("running").
		SetStartTime(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	return instance
}

func seedAssignedTask(t *testing.T, client *ent.Client, instance *ent.ProcessInstance, taskID string) {
	t.Helper()
	_, err := client.ProcessTask.Create().
		SetTaskID(taskID).
		SetTaskDefinitionKey("approval").
		SetTaskName("审批-" + taskID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessInstanceID(instance.ID).
		SetAssignee("7").
		SetStatus("assigned").
		SetTenantID(instance.TenantID).
		SetCreatedTime(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
}

func TestListUserTasks_ProcessInstanceKeyFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_task_instance_filter?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	otherTenantDef := client.ProcessDefinition.Create().
		SetKey("other-flow").
		SetName("other").
		SetVersion("1").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(definition.DeploymentID).
		SetDeployedAt(time.Now()).
		SetTenantID(2).
		SaveX(context.Background())

	instanceA := seedInstance(t, client, definition, "PI-own-a")
	instanceB := seedInstance(t, client, definition, "PI-own-b")
	// process_instance_id 全局唯一，故跨租户探测的形态是「本租户查不到该键」，
	// 而不是同名键被隔离；仍要证明用别人已存在的键无法读到别人的任务。
	instanceT2 := seedInstance(t, client, otherTenantDef, "PI-tenant-2")

	seedAssignedTask(t, client, instanceA, "task-a1")
	seedAssignedTask(t, client, instanceA, "task-a2")
	seedAssignedTask(t, client, instanceB, "task-b1")
	seedAssignedTask(t, client, instanceT2, "task-t2")

	r := newTaskListTestRouter(t, client)

	// 过滤生效：只返回该实例的 2 条任务，total 是过滤后的计数。
	data, keys := listTaskEnvelope(t, r, "?processInstanceKey=PI-own-a", "1")
	assert.ElementsMatch(t, []string{"PI-own-a", "PI-own-a"}, keys)
	assert.Equal(t, float64(2), data["total"], "total 必须是过滤后的计数")

	// 不带该键时保持原「我的待办」语义：本租户 3 条，租户 2 的任务不可见。
	_, keys = listTaskEnvelope(t, r, "", "1")
	assert.ElementsMatch(t, []string{"PI-own-a", "PI-own-a", "PI-own-b"}, keys)

	// 其他租户已存在的实例键：当前租户查不到任何任务，而不是越权返回租户 2 的任务。
	data, keys = listTaskEnvelope(t, r, "?processInstanceKey=PI-tenant-2", "1")
	assert.Empty(t, keys, "跨租户实例键不得返回任务")
	assert.Equal(t, float64(0), data["total"])

	// 反向：租户 2 用自己的键能查到自己的任务。
	_, keys = listTaskEnvelope(t, r, "?processInstanceKey=PI-tenant-2", "2")
	assert.Equal(t, []string{"PI-tenant-2"}, keys)
}
