package bpmn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"
	"itsm-backend/service/bpmn"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===================== Fake engine + TaskService for controller tests =====================

type fakeTaskService struct {
	completeByIDCalls  int
	completeByIDVars   []map[string]interface{}
	completeByIDErr    error
	completeCalls      int
	completeVars       []map[string]interface{}
	completeErr        error
	historyCalls       int
	historyInstanceKey string
	historyDecisions   []*ent.ProcessApprovalDecision
	historyErr         error
	listRequest        *service.ListUserTasksRequest

	// taskVariables 模拟 BPMN 节点写进 ProcessTask.TaskVariables 的审批配置，
	// SubmitTaskDecision 的前置校验（commentRequiredOnReject）依赖它。
	taskVariables map[string]interface{}
	// getTaskErr / getTaskByIDErr 用于模拟任务不存在或 tenant 不可见。
	getTaskErr     error
	getTaskByIDErr error
	// 查询调用记录，用于断言 ID 路由与 fallback。
	getTaskIDs     []string
	getTaskByIDIDs []int
}

func (f *fakeTaskService) GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error) {
	f.getTaskIDs = append(f.getTaskIDs, taskID)
	if f.getTaskErr != nil {
		return nil, f.getTaskErr
	}
	return &ent.ProcessTask{TaskID: taskID, TaskVariables: f.taskVariables}, nil
}

func (f *fakeTaskService) GetTaskByID(ctx context.Context, id int) (*ent.ProcessTask, error) {
	f.getTaskByIDIDs = append(f.getTaskByIDIDs, id)
	if f.getTaskByIDErr != nil {
		return nil, f.getTaskByIDErr
	}
	return &ent.ProcessTask{ID: id, TaskVariables: f.taskVariables}, nil
}

func (f *fakeTaskService) CompleteTaskByID(ctx context.Context, id int, variables map[string]interface{}) error {
	f.completeByIDCalls++
	f.completeByIDVars = append(f.completeByIDVars, variables)
	return f.completeByIDErr
}

func (f *fakeTaskService) ClaimTask(ctx context.Context, taskID, userID string) error {
	return nil
}

func (f *fakeTaskService) ClaimTaskByID(ctx context.Context, id, userID int) error {
	return nil
}

func (f *fakeTaskService) ListUserTasks(ctx context.Context, req *service.ListUserTasksRequest) ([]*ent.ProcessTask, int, error) {
	return nil, 0, nil
}

func (f *fakeTaskService) ListUserTaskViews(ctx context.Context, req *service.ListUserTasksRequest) ([]*dto.BPMNTaskResponse, int, error) {
	copyReq := *req
	f.listRequest = &copyReq
	return nil, 0, nil
}

func (f *fakeTaskService) AssignTask(ctx context.Context, taskID, assignee string) error {
	return nil
}

func (f *fakeTaskService) ReassignTask(ctx context.Context, taskID string, newAssigneeID int, reason string) error {
	return nil
}

func (f *fakeTaskService) TerminateTask(ctx context.Context, taskID, reason string) error {
	return nil
}

func (f *fakeTaskService) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	f.completeCalls++
	f.completeVars = append(f.completeVars, variables)
	return f.completeErr
}

func (f *fakeTaskService) CancelTask(ctx context.Context, taskID, reason string) error {
	return nil
}

func (f *fakeTaskService) GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeTaskService) SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error {
	return nil
}

func (f *fakeTaskService) HandleTaskTimeout(ctx context.Context, taskID string) error {
	return nil
}

func (f *fakeTaskService) RetryTask(ctx context.Context, taskID string, maxRetries int) error {
	return nil
}

func (f *fakeTaskService) DelegateTask(ctx context.Context, taskID, newAssignee string) error {
	return nil
}

func (f *fakeTaskService) EscalateTask(ctx context.Context, taskID, reason string) error {
	return nil
}

func (f *fakeTaskService) BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string, tenantID int) error {
	return nil
}

func (f *fakeTaskService) GetTaskStatistics(ctx context.Context, req *service.TaskStatisticsRequest) (*service.TaskStatistics, error) {
	return nil, nil
}

func (f *fakeTaskService) ListApprovalDecisions(ctx context.Context, processInstanceKey string) ([]*ent.ProcessApprovalDecision, error) {
	f.historyCalls++
	f.historyInstanceKey = processInstanceKey
	return f.historyDecisions, f.historyErr
}

func (f *fakeTaskService) CreateCounterSignTasks(ctx context.Context, parentTaskID string, req *service.CounterSignRequest) ([]*ent.ProcessTask, error) {
	return nil, nil
}

func (f *fakeTaskService) GetCounterSignStatus(ctx context.Context, parentTaskID string) (*service.CounterSignStatus, error) {
	return nil, nil
}

func (f *fakeTaskService) Vote(ctx context.Context, taskID string, req *service.VoteRequest) error {
	return nil
}

type fakeProcessEngine struct {
	taskSvc *fakeTaskService
}

func (e *fakeProcessEngine) ProcessDefinitionService() service.ProcessDefinitionService { return nil }

func (e *fakeProcessEngine) ProcessInstanceService() service.ProcessInstanceService { return nil }
func (e *fakeProcessEngine) TaskService() service.TaskService                       { return e.taskSvc }
func (e *fakeProcessEngine) SetApprovalService(svc bpmn.ApprovalServiceInterface)   {}

func (e *fakeProcessEngine) StartProcess(ctx context.Context, key, biz string, vars map[string]interface{}) (*ent.ProcessInstance, error) {
	return nil, nil
}

func (e *fakeProcessEngine) CompleteTask(ctx context.Context, taskID string, vars map[string]interface{}) error {
	return e.taskSvc.CompleteTask(ctx, taskID, vars)
}

func (e *fakeProcessEngine) SuspendProcess(ctx context.Context, id, reason string) error {
	return nil
}
func (e *fakeProcessEngine) ResumeProcess(ctx context.Context, id string) error { return nil }
func (e *fakeProcessEngine) TerminateProcess(ctx context.Context, id, reason string) error {
	return nil
}

func newBPMNWorkflowTestRouter(t *testing.T) (*gin.Engine, *fakeTaskService) {
	gin.SetMode(gin.TestMode)
	fakeTask := &fakeTaskService{}
	engine := &fakeProcessEngine{taskSvc: fakeTask}
	ctrl := NewWorkflowHandler(engine, nil)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// Mirror the authenticated tenant context installed by production
		// middleware. Security tests exercise missing/zero tenants directly.
		tenantID, userID := 1, 7
		if value := c.GetHeader("X-Test-Tenant"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				tenantID = parsed
			}
		}
		if value := c.GetHeader("X-Test-User"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				userID = parsed
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})
	g := r.Group("/api/v1")
	ctrl.RegisterRoutes(g)
	return r, fakeTask
}

func doRequest(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newTaskListHandlerTestRouter(t *testing.T) (*gin.Engine, *fakeTaskService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fakeTask := &fakeTaskService{}
	ctrl := NewWorkflowHandler(&fakeProcessEngine{taskSvc: fakeTask}, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 11)
		c.Set("user_id", 7)
		c.Next()
	})
	r.GET("/tasks", ctrl.ListUserTasks)
	r.GET("/tasks/all", ctrl.ListAllTasks)
	return r, fakeTask
}

func TestListUserTasksUsesAuthenticatedIdentity(t *testing.T) {
	r, taskService := newTaskListHandlerTestRouter(t)
	w := doRequest(t, r, http.MethodGet, "/tasks?userId=99&assignee=other&candidateUsers=other&candidateGroups=other", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, taskService.listRequest)
	assert.Equal(t, 11, taskService.listRequest.TenantID)
	assert.Equal(t, 7, taskService.listRequest.UserID)
	assert.False(t, taskService.listRequest.AllTasks)
	assert.Empty(t, taskService.listRequest.Assignee)
	assert.Empty(t, taskService.listRequest.CandidateUsers)
	assert.Empty(t, taskService.listRequest.CandidateGroups)
}

func TestListAllTasksSetsAuthorizedAllTasksScope(t *testing.T) {
	r, taskService := newTaskListHandlerTestRouter(t)
	w := doRequest(t, r, http.MethodGet, "/tasks/all?userId=99", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, taskService.listRequest)
	assert.Equal(t, 11, taskService.listRequest.TenantID)
	assert.Zero(t, taskService.listRequest.UserID)
	assert.True(t, taskService.listRequest.AllTasks)
}

// doAuthedRequest overrides the router's default authenticated identity.
func doAuthedRequest(t *testing.T, r *gin.Engine, method, path string, body interface{}, tenantID, userID int) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", strconv.Itoa(tenantID))
	req.Header.Set("X-Test-User", strconv.Itoa(userID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ==================== Tests ====================

func TestSubmitTaskDecision_ApprovePath(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	taskID := "42"

	body := map[string]interface{}{"action": "approve"}
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/"+taskID+"/decisions", body)
	require.NotNil(t, w)

	assert.Equal(t, 200, w.Code, "approve path should return 200, body=%s", w.Body.String())
	assert.Equal(t, 1, fakeTask.completeByIDCalls, "approve path should call CompleteTaskByID once")
	assert.Equal(t, 0, fakeTask.completeCalls, "approve path should NOT call string CompleteTask")
	require.Len(t, fakeTask.completeByIDVars, 1)
	assert.Equal(t, "approve", fakeTask.completeByIDVars[0]["approvalAction"])
	assert.Equal(t, "approved", fakeTask.completeByIDVars[0]["approvalResult"])
}

func TestSubmitTaskDecision_RejectRequiresComment(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	// 意见是否必填是节点级配置（BPMN commentRequiredOnReject），
	// 只有节点声明了才构成 1001 参数错误。
	fakeTask.taskVariables = map[string]interface{}{"commentRequiredOnReject": true}
	body := map[string]interface{}{"action": "reject"}
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/1/decisions", body)
	require.NotNil(t, w)

	assert.Equal(t, http.StatusBadRequest, w.Code, "reject without comment yields HTTP 400, body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"], "reject without comment must yield ParamErrorCode 1001")
	assert.Contains(t, resp["message"], "拒绝时填写意见")
	assert.Equal(t, 0, fakeTask.completeByIDCalls, "engine must not be invoked when comment is missing")
	assert.Equal(t, 0, fakeTask.completeCalls, "engine must not be invoked when comment is missing")
}

// 节点未声明 commentRequiredOnReject 时，拒绝不强制意见，但必须推进流程。
func TestSubmitTaskDecision_RejectWithoutNodeConfigIsAllowed(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	fakeTask.taskVariables = map[string]interface{}{"approvalMode": "single"}
	body := map[string]interface{}{"action": "reject"}
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/8/decisions", body)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, 1, fakeTask.completeByIDCalls)
	require.Len(t, fakeTask.completeByIDVars, 1)
	assert.Equal(t, "reject", fakeTask.completeByIDVars[0]["approvalAction"])
	assert.Equal(t, "rejected", fakeTask.completeByIDVars[0]["approvalResult"])
}

// 回归：任务不存在（含跨租户不可见）时必须 404，且不得调用引擎。
func TestSubmitTaskDecision_UnknownTaskFailsClosed(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	fakeTask.getTaskErr = errors.New("ent: not found")
	fakeTask.getTaskByIDErr = errors.New("ent: not found")
	body := map[string]interface{}{"action": "approve"}

	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/42/decisions", body)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(4004), resp["code"])
	assert.Equal(t, 0, fakeTask.completeByIDCalls, "不存在的任务不得推进")
	assert.Equal(t, 0, fakeTask.completeCalls)
	// 数字 ID 必须先走 string 查询，再 fallback 到主键查询
	assert.Equal(t, []string{"42"}, fakeTask.getTaskIDs)
	assert.Equal(t, []int{42}, fakeTask.getTaskByIDIDs)
}

func TestSubmitTaskDecision_RoutesNumericAndStringIDs(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	body := map[string]interface{}{"action": "approve"}

	// numeric id → CompleteTaskByID
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/123/decisions", body)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, 1, fakeTask.completeByIDCalls, "numeric id should hit CompleteTaskByID")
	assert.Equal(t, 0, fakeTask.completeCalls, "numeric id should NOT hit string CompleteTask")

	// non-numeric id → CompleteTask(string)
	w2 := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/TASK-abc/decisions", body)
	require.Equal(t, 200, w2.Code, w2.Body.String())
	assert.Equal(t, 1, fakeTask.completeByIDCalls, "string id must NOT trigger CompleteTaskByID again")
	assert.Equal(t, 1, fakeTask.completeCalls, "string id must trigger string CompleteTask")
}

func TestSubmitTaskDecision_RejectsBadAction(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	body := map[string]interface{}{"action": "MAYBE"}
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/1/decisions", body)
	require.NotNil(t, w)

	assert.Equal(t, http.StatusBadRequest, w.Code, "non-allowlisted action yields HTTP 400, body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"], "non-allowlisted action must yield ParamErrorCode")
	assert.Equal(t, 0, fakeTask.completeByIDCalls)
}

func TestSubmitTaskDecision_RejectWithCommentPasses(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	body := map[string]interface{}{"action": "reject", "comment": "not acceptable"}
	w := doRequest(t, r, "POST", "/api/v1/bpmn/tasks/7/decisions", body)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, 1, fakeTask.completeByIDCalls)
	require.Len(t, fakeTask.completeByIDVars, 1)
	assert.Equal(t, "reject", fakeTask.completeByIDVars[0]["approvalAction"])
	assert.Equal(t, "rejected", fakeTask.completeByIDVars[0]["approvalResult"])
	assert.Equal(t, "not acceptable", fakeTask.completeByIDVars[0]["approvalComment"])
}

func TestGetApprovalHistory_TenantScoped(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)

	fakeTask.historyDecisions = []*ent.ProcessApprovalDecision{
		{ID: 1, TenantID: 1},
	}

	// emulate auth middleware that sets tenant_id and user_id on the gin context
	w := doAuthedRequest(t, r, "GET", "/api/v1/bpmn/process-instances/PI-1/approval-history", nil, 1, 7)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, 1, fakeTask.historyCalls, "approval-history must call ListApprovalDecisions")
	assert.Equal(t, "PI-1", fakeTask.historyInstanceKey)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
}

func TestGetApprovalHistory_PropagatesError(t *testing.T) {
	r, fakeTask := newBPMNWorkflowTestRouter(t)
	fakeTask.historyErr = errors.New("db boom")

	w := doAuthedRequest(t, r, "GET", "/api/v1/bpmn/process-instances/PI-2/approval-history", nil, 1, 7)
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(5001), resp["code"], "service error must surface as InternalErrorCode 5001")
	assert.Contains(t, resp["message"], "查询审批历史失败")
}

// silence unused-import lint on strconv when feature is dropped
var _ = strconv.Itoa

// ===================== F-5: tenant_id=0 透传绕过租户过滤 =====================

// TestGetBPMNTenantContext_RejectsZeroTenant 验证 F-5：tenant_id=0 必须被拒绝，
// 否则会透传到引擎导致 `if tenantID > 0` 租户过滤被跳过，造成跨租户读取。
func TestGetBPMNTenantContext_RejectsZeroTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("tenant_id", 0)

	_, _, ok := getBPMNTenantContext(c)
	assert.False(t, ok, "tenant_id=0 应通过 F-5 被拒绝，避免跨租户读取")
}

// TestGetBPMNTenantContext_AcceptsPositiveTenant 验证正常租户上下文透传。
func TestGetBPMNTenantContext_AcceptsPositiveTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("tenant_id", 7)
	c.Set("user_id", 3)

	ctx, tenantID, ok := getBPMNTenantContext(c)
	assert.True(t, ok)
	assert.Equal(t, 7, tenantID)
	assert.NotNil(t, ctx)
}
