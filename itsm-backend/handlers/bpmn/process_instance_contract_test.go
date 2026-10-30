package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 流程实例列表与生命周期契约测试。
//
// 列表面：查询串里的 camelCase 过滤/分页参数必须真的落到
// service.ListProcessInstancesRequest，否则 handler 默认值会静默丢掉调用方的过滤条件；
// items[].instanceId 必须等于 PI-* 业务键（实例动作唯一的寻址键）。
// 生命周期面：非法 source 状态与 not-found 不得兜底成 500/5001。

type recordingInstanceService struct {
	service.ProcessInstanceService
	listRequest *service.ListProcessInstancesRequest
	instances   []*ent.ProcessInstance
	total       int
}

func (s *recordingInstanceService) ListProcessInstances(ctx context.Context, req *service.ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error) {
	s.listRequest = req
	return s.instances, s.total, nil
}

func (s *recordingInstanceService) GetProcessInstance(ctx context.Context, id string) (*ent.ProcessInstance, error) {
	return &ent.ProcessInstance{ID: 24, ProcessInstanceID: id, Status: "running"}, nil
}

func (s *recordingInstanceService) SetProcessInstanceVariables(ctx context.Context, id string, vars map[string]interface{}) error {
	return nil
}

type instanceContractEngine struct {
	*fakeProcessEngine
	instanceSvc  *recordingInstanceService
	suspendErr   error
	resumeErr    error
	terminateErr error
}

func (e *instanceContractEngine) ProcessInstanceService() service.ProcessInstanceService {
	return e.instanceSvc
}

func (e *instanceContractEngine) SuspendProcess(ctx context.Context, id, reason string) error {
	return e.suspendErr
}

func (e *instanceContractEngine) ResumeProcess(ctx context.Context, id string) error {
	return e.resumeErr
}

func (e *instanceContractEngine) TerminateProcess(ctx context.Context, id, reason string) error {
	return e.terminateErr
}

func newInstanceContractRouter(t *testing.T) (*gin.Engine, *instanceContractEngine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := &instanceContractEngine{
		fakeProcessEngine: &fakeProcessEngine{},
		instanceSvc:       &recordingInstanceService{},
	}
	ctrl := NewWorkflowHandler(engine, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 11)
		c.Set("user_id", 7)
		c.Next()
	})
	g := r.Group("/api/v1/bpmn")
	g.GET("/process-instances", ctrl.ListProcessInstances)
	g.PUT("/process-instances/:id/suspend", ctrl.SuspendProcess)
	g.PUT("/process-instances/:id/resume", ctrl.ResumeProcess)
	g.PUT("/process-instances/:id/terminate", ctrl.TerminateProcess)
	return r, engine
}

func decodeResponseEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var env map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

// 过滤与分页参数必须绑定；TenantID 只认认证上下文，忽略请求自报值。
func TestListProcessInstancesBindsCamelCaseFiltersAndPaging(t *testing.T) {
	r, engine := newInstanceContractRouter(t)
	w := doRequest(t, r, http.MethodGet,
		"/api/v1/bpmn/process-instances?processDefinitionKey=leave_flow&businessKey=ticket%3A42&status=running&page=3&pageSize=50&tenantId=999", nil)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	req := engine.instanceSvc.listRequest
	require.NotNil(t, req)
	assert.Equal(t, "leave_flow", req.ProcessDefinitionKey)
	assert.Equal(t, "ticket:42", req.BusinessKey)
	assert.Equal(t, "running", req.Status)
	assert.Equal(t, 3, req.Page)
	assert.Equal(t, 50, req.PageSize)
	assert.Equal(t, 11, req.TenantID, "租户必须来自认证上下文，不能信查询参数")
}

// 无过滤参数时 handler 仍要给默认分页，且响应信封固定为 items+分页四元组。
func TestListProcessInstancesReturnsItemsEnvelopeWithAddressableKey(t *testing.T) {
	r, engine := newInstanceContractRouter(t)
	engine.instanceSvc.instances = []*ent.ProcessInstance{{
		ID: 24, ProcessInstanceID: "PI-leave_flow-24", Status: "running",
	}}
	engine.instanceSvc.total = 1

	w := doRequest(t, r, http.MethodGet, "/api/v1/bpmn/process-instances", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	env := decodeResponseEnvelope(t, w)
	data, ok := env["data"].(map[string]interface{})
	require.True(t, ok, "body=%s", w.Body.String())
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "body=%s", w.Body.String())
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "PI-leave_flow-24", item["instanceId"], "列表项必须透出可寻址的 PI-* 键")
	assert.Equal(t, "24", item["id"], "数字 Ent ID 只作为字段透出，不是寻址键")
}

// 未发生的结束/挂起时间必须省略，而不是输出 Go 零值 0001-01-01T00:00:00Z。
// 运行中的实例被消费方当成「已排期/已结束」是最典型的零值时间误读。
func TestListProcessInstancesOmitsUnsetTimes(t *testing.T) {
	r, engine := newInstanceContractRouter(t)
	started := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	finished := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine.instanceSvc.instances = []*ent.ProcessInstance{
		{ID: 1, ProcessInstanceID: "PI-running", Status: "running", StartTime: started},
		{ID: 2, ProcessInstanceID: "PI-completed", Status: "completed", StartTime: started, EndTime: finished},
	}
	engine.instanceSvc.total = 2

	w := doRequest(t, r, http.MethodGet, "/api/v1/bpmn/process-instances", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	env := decodeResponseEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	require.Len(t, items, 2)

	running := items[0].(map[string]interface{})
	assert.NotContains(t, running, "endTime")
	assert.NotContains(t, running, "suspendedTime")

	completed := items[1].(map[string]interface{})
	assert.Equal(t, finished.Format(time.RFC3339), completed["endTime"])
	assert.NotContains(t, completed, "suspendedTime")
}

// 生命周期错误语义：领域冲突 409/4090、not-found 404/4004，不得兜底 500/5001。
func TestProcessLifecycleMapsConflictAndNotFound(t *testing.T) {
	conflict := common.NewAppError(common.ErrCodeConflict, "流程实例当前状态为 completed，不能暂停", http.StatusConflict, nil)
	notFound := common.NewNotFoundError("流程实例")

	cases := []struct {
		name     string
		path     string
		body     interface{}
		inject   func(*instanceContractEngine)
		wantCode int
		wantBiz  int
	}{
		{
			"suspend_conflict", "/api/v1/bpmn/process-instances/PI-x/suspend",
			map[string]string{"reason": "r"},
			func(e *instanceContractEngine) { e.suspendErr = conflict }, http.StatusConflict, common.ConflictCode,
		},
		{
			"resume_conflict", "/api/v1/bpmn/process-instances/PI-x/resume", nil,
			func(e *instanceContractEngine) { e.resumeErr = conflict }, http.StatusConflict, common.ConflictCode,
		},
		{
			"terminate_conflict", "/api/v1/bpmn/process-instances/PI-x/terminate",
			map[string]string{"reason": "r"},
			func(e *instanceContractEngine) { e.terminateErr = conflict }, http.StatusConflict, common.ConflictCode,
		},
		{
			"terminate_not_found", "/api/v1/bpmn/process-instances/PI-missing/terminate",
			map[string]string{"reason": "r"},
			func(e *instanceContractEngine) { e.terminateErr = notFound }, http.StatusNotFound, common.NotFoundCode,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, engine := newInstanceContractRouter(t)
			tc.inject(engine)
			w := doRequest(t, r, http.MethodPut, tc.path, tc.body)

			require.Equal(t, tc.wantCode, w.Code, "body=%s", w.Body.String())
			env := decodeResponseEnvelope(t, w)
			assert.Equal(t, float64(tc.wantBiz), env["code"], "body=%s", w.Body.String())
		})
	}
}

// TestProcessTriggerActionsRejectMalformedBody 流程取消/挂起（真实生产入口
// POST /api/v1/bpmn/process-trigger/{cancel,suspend}/:instance_id）此前丢弃了
// ShouldBindJSON 的返回错误：body 非法时 req 保持零值，取消/挂起理由变成空串，
// 流程照样被取消/挂起，而调用方拿到 200——理由丢失且无人知晓。
// 现在必须显式拒绝（4xx），而不是「成功但语义残缺」。
func TestProcessTriggerActionsRejectMalformedBody(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_trigger_badbody?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	gin.SetMode(gin.TestMode)
	ctrl := NewProcessTriggerHandler(service.NewProcessTriggerService(client, &fakeProcessEngine{}), nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 11})
		c.Set("user_id", 7)
		c.Next()
	})
	g := r.Group("/api/v1/bpmn/process-trigger")
	g.POST("/cancel/:instance_id", ctrl.CancelProcess)
	g.POST("/suspend/:instance_id", ctrl.SuspendProcess)

	for _, path := range []string{
		"/api/v1/bpmn/process-trigger/cancel/1",
		"/api/v1/bpmn/process-trigger/suspend/1",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{not-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code,
			"%s 收到非法 body 必须拒绝，不能以空 reason 继续执行；body=%s", path, w.Body.String())
	}
}
