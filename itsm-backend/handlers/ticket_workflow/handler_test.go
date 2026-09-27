package ticket_workflow

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/ticketapproval"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestHandler(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	dbName := "file:tw_test_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })

	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	workflowService := service.NewTicketWorkflowService(client, logger)
	h := NewHandler(workflowService, db, logger)

	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tenantID = v
			}
		}
		userID := 1
		if h := c.GetHeader("X-Test-User"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				userID = v
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})

	// 注册路由 - mirror router.go 契约 /tickets/workflow/*
	r.POST("/api/v1/tickets/workflow/accept", h.AcceptTicket)
	r.POST("/api/v1/tickets/workflow/approve", h.ApproveTicket)
	r.POST("/api/v1/tickets/workflow/resolve", h.ResolveTicket)
	r.GET("/api/v1/tickets/cc/my", h.ListMyCCRecords)
	r.GET("/api/v1/tickets/:id/cc", h.ListTicketCCRecords)
	r.GET("/api/v1/tickets/:id/workflow/state", h.GetTicketWorkflowState)
	r.GET("/api/v1/tickets/:id/workflow-history", h.GetTicketWorkflowHistory)

	return r
}

func TestHandler_AcceptTicket_EmptyBody(t *testing.T) {
	r := setupTestHandler(t)

	// AcceptTicketRequest 无 binding required → 空对象绑定成功，
	// TicketID=0 传给 service 报错 → 500（与旧 controller 契约一致）
	body := []byte(`{}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

func TestHandler_AcceptTicket_TicketNotFound(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"ticketId":99999,"comment":"接单"}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// service 层对不存在工单返回错误 → 500（与旧契约一致）
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

func TestHandler_AcceptTicket_MissingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbName := "file:tw_notenant_" + t.Name() + "?mode=memory&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })
	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)
	logger := zaptest.NewLogger(t).Sugar()
	workflowService := service.NewTicketWorkflowService(client, logger)
	h := NewHandler(workflowService, db, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	// 不注入 tenant_id/user_id → getAuthContext 应 401
	r.POST("/api/v1/tickets/workflow/accept", h.AcceptTicket)

	body := []byte(`{"ticketId":1,"comment":"x"}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "缺 tenant 上下文应 401")
}

func TestHandler_ListTicketCCRecords_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/tickets/invalid/cc", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicketWorkflowState_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/tickets/invalid/workflow/state", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicketWorkflowHistory_EmptyRecords(t *testing.T) {
	r := setupTestHandler(t)

	// ticket_workflow_records 表在 enttest 自动迁移中创建；查询不存在的工单 → 空列表
	req, err := http.NewRequest("GET", "/api/v1/tickets/99999/workflow-history", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 空结果应 200 + 空数组（与旧契约一致：records == nil → []）
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
}

func TestHandler_ApproveTicket_RequiresBPMNTask(t *testing.T) {
	for _, action := range []string{"approve", "reject", "delegate"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_fk=1")
			t.Cleanup(func() { client.Close() })
			tenantA, err := client.Tenant.Create().SetName("A").SetCode("a").SetStatus("active").Save(ctx)
			require.NoError(t, err)
			tenantB, err := client.Tenant.Create().SetName("B").SetCode("b").SetStatus("active").Save(ctx)
			require.NoError(t, err)
			actor, err := client.User.Create().SetUsername("approver").SetEmail("approver@example.com").
				SetName("Approver").SetPasswordHash("test").SetRole("agent").SetTenantID(tenantA.ID).Save(ctx)
			require.NoError(t, err)
			ticket, err := client.Ticket.Create().SetTitle("Approval").SetTicketNumber("APP-1").
				SetStatus("pending").SetPriority("medium").SetRequesterID(actor.ID).SetTenantID(tenantA.ID).Save(ctx)
			require.NoError(t, err)
			approval, err := client.TicketApproval.Create().SetTicketID(ticket.ID).SetLevel(1).SetLevelName("审批").
				SetApproverID(actor.ID).SetStatus("pending").SetTenantID(tenantA.ID).Save(ctx)
			require.NoError(t, err)
			logger := zaptest.NewLogger(t).Sugar()
			h := NewHandler(service.NewTicketWorkflowService(client, logger), nil, logger)
			for _, scope := range []struct{ tenantID, status, code int }{
				{tenantA.ID, http.StatusConflict, 4090},
				{tenantB.ID, http.StatusNotFound, 4004},
			} {
				r := gin.New()
				r.Use(func(c *gin.Context) { c.Set("tenant_id", scope.tenantID); c.Set("user_id", actor.ID) })
				r.POST("/api/v1/tickets/workflow/approve", h.ApproveTicket)
				for attempt := 0; attempt < 2; attempt++ {
					body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":%q,"delegateToUserId":%d}`, ticket.ID, approval.ID, action, actor.ID)
					req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/workflow/approve", bytes.NewBufferString(body))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					require.Equal(t, scope.status, w.Code, w.Body.String())
					var response struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
					assert.Equal(t, scope.code, response.Code)
					assert.NotEmpty(t, response.Message)
					assert.NotContains(t, w.Body.String(), "password")
				}
			}
			updated, err := client.Ticket.Get(ctx, ticket.ID)
			require.NoError(t, err)
			assert.Equal(t, "pending", updated.Status)
			updatedApproval, err := client.TicketApproval.Get(ctx, approval.ID)
			require.NoError(t, err)
			assert.Equal(t, "pending", updatedApproval.Status)
			count, err := client.TicketApproval.Query().Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
			count, err = client.TicketWorkflowRecord.Query().Count(ctx)
			require.NoError(t, err)
			assert.Zero(t, count)
		})
	}
}

func TestHandler_ApproveTicket_BadRequest(t *testing.T) {
	r := setupTestHandler(t)

	// ApproveTicketRequest 的 Action 带 binding required,oneof
	body := []byte(`{"ticketId":1}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/approve", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_ApproveTicket_InvalidAction(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"ticketId":1,"approvalId":1,"action":"drop"}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/approve", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "非法 action 应被 binding oneof 拦截")
}

type e2eEnv struct {
	router *gin.Engine
	client *ent.Client
}

func setupE2E(t *testing.T) *e2eEnv {
	t.Helper()
	dbName := "file:e2e_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewTicketWorkflowService(client, logger)
	h := NewHandler(svc, nil, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if v := c.GetHeader("X-Test-Tenant"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				tenantID = n
			}
		}
		userID := 1
		if v := c.GetHeader("X-Test-User"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				userID = n
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})
	r.POST("/api/v1/tickets/workflow/approve", h.ApproveTicket)

	return &e2eEnv{router: r, client: client}
}

func (e *e2eEnv) createTenant(t *testing.T, code string) int {
	t.Helper()
	tenant, err := e.client.Tenant.Create().
		SetName("E2E-"+code).SetCode("e2e-"+code).SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func (e *e2eEnv) createUser(t *testing.T, tenantID int, username string) int {
	t.Helper()
	user, err := e.client.User.Create().
		SetUsername(username).SetEmail(username+"@e2e.test").SetName(username).
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return user.ID
}

func (e *e2eEnv) createTicket(t *testing.T, tenantID, requesterID int, status string) int {
	t.Helper()
	ticket, err := e.client.Ticket.Create().
		SetTitle("E2E Approval").SetTicketNumber(fmt.Sprintf("E2E-%d", time.Now().UnixNano())).
		SetStatus(status).SetPriority("medium").SetRequesterID(requesterID).SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return ticket.ID
}

func (e *e2eEnv) createApproval(t *testing.T, tenantID, ticketID, approverID int, status string) int {
	t.Helper()
	approval, err := e.client.TicketApproval.Create().
		SetTicketID(ticketID).SetLevel(1).SetLevelName("审批").
		SetApproverID(approverID).SetStatus(status).SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return approval.ID
}

// createBPMNFixture 创建最小 BPMN 流程夹具，与 service 层 bridge 测试使用相同结构。
// businessKey 格式为 "ticket:{ticketID}"，匹配 findPendingApprovalTask 的拼接逻辑。
func (e *e2eEnv) createBPMNFixture(t *testing.T, tenantID, ticketID, assigneeID int, allowDelegate bool) (taskID int) {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d_%d", ticketID, assigneeID)
	defKey := "e2e_approval_" + suffix

	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>`+
		`<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"`+
		` xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn"`+
		` id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm">`+
		`<bpmn:process id="%s" name="E2E Approval %s" isExecutable="true">`+
		`<bpmn:startEvent id="StartEvent_1"/>`+
		`<bpmn:userTask id="Approval_1" name="审批" itsm:taskPurpose="approval"`+
		` itsm:approvalMode="single" itsm:allowDelegate="%t" itsm:assignee="%d"/>`+
		`<bpmn:endEvent id="EndEvent_1"/>`+
		`<bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Approval_1"/>`+
		`<bpmn:sequenceFlow id="Flow_2" sourceRef="Approval_1" targetRef="EndEvent_1"/>`+
		`</bpmn:process></bpmn:definitions>`, defKey, defKey, suffix, allowDelegate, assigneeID)

	dep, err := e.client.ProcessDeployment.Create().
		SetDeploymentID("DEP-E2E-"+suffix).SetDeploymentName("E2E "+suffix).
		SetDeploymentTime(time.Now()).SetDeployedBy("e2e").SetIsActive(true).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := e.client.ProcessDefinition.Create().
		SetKey(defKey).SetName("E2E Approval "+suffix).SetVersion("1").SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).SetDeploymentID(dep.ID).SetDeployedAt(time.Now()).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	businessKey := fmt.Sprintf("ticket:%d", ticketID)
	instance, err := e.client.ProcessInstance.Create().
		SetProcessInstanceID("PI-E2E-"+suffix).SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).SetBusinessKey(businessKey).SetStatus("running").
		SetVariables(map[string]interface{}{
			"business_type": "ticket",
			"business_id":   strconv.Itoa(ticketID),
			"business_key":  businessKey,
		}).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	task, err := e.client.ProcessTask.Create().
		SetTaskID("TASK-E2E-"+suffix).SetTaskDefinitionKey("Approval_1").SetTaskName("审批").
		SetTaskType("user_task").SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).SetAssignee(strconv.Itoa(assigneeID)).SetStatus("assigned").
		SetTaskVariables(map[string]interface{}{
			"taskPurpose":   "approval",
			"approvalMode":  "single",
			"allowDelegate": allowDelegate,
		}).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return task.ID
}

func (e *e2eEnv) postApprove(t *testing.T, tenantID, userID int, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/workflow/approve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Tenant", strconv.Itoa(tenantID))
	req.Header.Set("X-Test-User", strconv.Itoa(userID))
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func TestHandler_ApproveTicket_E2E_Approve(t *testing.T) {
	env := setupE2E(t)
	tenantID := env.createTenant(t, "approve")
	actorID := env.createUser(t, tenantID, "approver")
	ticketID := env.createTicket(t, tenantID, actorID, "pending")
	approvalID := env.createApproval(t, tenantID, ticketID, actorID, "pending")
	taskID := env.createBPMNFixture(t, tenantID, ticketID, actorID, true)

	body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":"approve","comment":"同意"}`, ticketID, approvalID)
	w := env.postApprove(t, tenantID, actorID, body)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "审批通过", resp.Data["message"])

	ctx := context.Background()
	task, err := env.client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status, "BPMN 任务应已完成")

	ticket, err := env.client.Ticket.Get(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, "approved", ticket.Status, "工单状态应为 approved")

	approval, err := env.client.TicketApproval.Get(ctx, approvalID)
	require.NoError(t, err)
	assert.Equal(t, "approved", approval.Status)
	assert.Equal(t, "approve", approval.Action)
	assert.Equal(t, "同意", approval.Comment)

	decisions, err := env.client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "approve", decisions[0].Action)
	assert.Equal(t, "approved", decisions[0].Decision)
	assert.Equal(t, "同意", decisions[0].Comment)

	records, err := env.client.TicketWorkflowRecord.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "approve", records[0].Action)
}

func TestHandler_ApproveTicket_E2E_Reject(t *testing.T) {
	env := setupE2E(t)
	tenantID := env.createTenant(t, "reject")
	actorID := env.createUser(t, tenantID, "rejecter")
	ticketID := env.createTicket(t, tenantID, actorID, "pending")
	approvalID := env.createApproval(t, tenantID, ticketID, actorID, "pending")
	taskID := env.createBPMNFixture(t, tenantID, ticketID, actorID, true)

	body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":"reject","comment":"不同意"}`, ticketID, approvalID)
	w := env.postApprove(t, tenantID, actorID, body)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "审批拒绝", resp.Data["message"])

	ctx := context.Background()
	task, err := env.client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status, "BPMN 任务应已完成")

	ticket, err := env.client.Ticket.Get(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", ticket.Status, "工单状态应为 rejected")

	approval, err := env.client.TicketApproval.Get(ctx, approvalID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", approval.Status)

	decisions, err := env.client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "reject", decisions[0].Action)
	assert.Equal(t, "rejected", decisions[0].Decision)
	assert.Equal(t, "不同意", decisions[0].Comment)

	records, err := env.client.TicketWorkflowRecord.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "approve_reject", records[0].Action)
}

func TestHandler_ApproveTicket_E2E_Delegate(t *testing.T) {
	env := setupE2E(t)
	tenantID := env.createTenant(t, "delegate")
	actorID := env.createUser(t, tenantID, "delegator")
	delegateeID := env.createUser(t, tenantID, "delegatee")
	ticketID := env.createTicket(t, tenantID, actorID, "pending")
	approvalID := env.createApproval(t, tenantID, ticketID, actorID, "pending")
	taskID := env.createBPMNFixture(t, tenantID, ticketID, actorID, true)

	body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":"delegate","delegateToUserId":%d}`,
		ticketID, approvalID, delegateeID)
	w := env.postApprove(t, tenantID, actorID, body)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "已委派", resp.Data["message"])

	ctx := context.Background()
	task, err := env.client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status, "委派只换 assignee 不改状态")
	assert.Equal(t, strconv.Itoa(delegateeID), task.Assignee, "BPMN 任务应已转派给受托人")

	originalApproval, err := env.client.TicketApproval.Get(ctx, approvalID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", originalApproval.Status, "原审批记录应已取消")

	newApprovals, err := env.client.TicketApproval.Query().
		Where(
			ticketapproval.TicketID(ticketID),
			ticketapproval.ApproverID(delegateeID),
			ticketapproval.Status("pending"),
		).All(ctx)
	require.NoError(t, err)
	require.Len(t, newApprovals, 1, "应为受托人创建新的待审批记录")

	decisions, err := env.client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "delegate", decisions[0].Action)
	assert.Equal(t, "delegated", decisions[0].Decision)

	records, err := env.client.TicketWorkflowRecord.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "delegate", records[0].Action)
}

func TestHandler_ApproveTicket_E2E_CrossTenantRejected(t *testing.T) {
	env := setupE2E(t)
	tenantA := env.createTenant(t, "cta")
	tenantB := env.createTenant(t, "ctb")
	actorA := env.createUser(t, tenantA, "actor-a")
	actorB := env.createUser(t, tenantB, "actor-b")

	ticketID := env.createTicket(t, tenantA, actorA, "pending")
	approvalID := env.createApproval(t, tenantA, ticketID, actorA, "pending")
	env.createBPMNFixture(t, tenantA, ticketID, actorA, true)

	body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":"approve","comment":"cross"}`, ticketID, approvalID)
	w := env.postApprove(t, tenantB, actorB, body)

	assert.Equal(t, http.StatusNotFound, w.Code, "跨租户应 404")
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 4004, resp.Code)

	ctx := context.Background()
	ticket, err := env.client.Ticket.Get(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, "pending", ticket.Status, "跨租户操作不得改变工单状态")

	task, err := env.client.ProcessTask.Query().
		Where(processtask.Status("assigned")).All(ctx)
	require.NoError(t, err)
	require.Len(t, task, 1)
	assert.Equal(t, "assigned", task[0].Status, "BPMN 任务应保持待办")
}
