package router

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

	"itsm-backend/common"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketapproval"
	"itsm-backend/ent/ticketworkflowrecord"
	"itsm-backend/middleware"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketApprovalOwnershipRoutes(t *testing.T) {
	t.Setenv("ITSM_APPROVAL_BPMN_FIRST", "false")
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticket_approval_ownership?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	router := setupTicketCCListRouter(t, client, zaptest.NewLogger(t).Sugar())
	tenant := seedCCListTenant(ctx, t, client, "pq01", 0)
	otherTenant := seedCCListTenant(ctx, t, client, "pq01-other", 0)
	approver := createCCListActor(ctx, t, client, tenant.tenantID, "pq01-approver")
	taskOwner := createCCListActor(ctx, t, client, tenant.tenantID, "pq01-task-owner")

	for _, tc := range []struct {
		name       string
		endpoint   string
		actor      ccListActor
		assignee   ccListActor
		bound      bool
		wantStatus int
		wantCode   int
	}{
		{"requester_cannot_approve", "approve", tenant.admin, tenant.admin, false, http.StatusForbidden, common.ForbiddenCode},
		{"requester_cannot_reject", "reject", tenant.admin, tenant.admin, false, http.StatusForbidden, common.ForbiddenCode},
		{"independent_approver_succeeds", "approve", approver, approver, false, http.StatusOK, 0},
		{"non_approver_cannot_reject_unbound", "reject", approver, tenant.admin, false, http.StatusForbidden, common.ForbiddenCode},
		{"different_task_owner_cannot_reject", "reject", approver, approver, true, http.StatusForbidden, common.ForbiddenCode},
		{"different_tenant_cannot_approve", "approve", otherTenant.admin, approver, false, http.StatusNotFound, common.NotFoundCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ticket, err := client.Ticket.Create().SetTenantID(tenant.tenantID).
				SetRequesterID(tenant.admin.userID).SetTicketNumber(tc.name).SetTitle(tc.name).
				SetStatus("pending").SetPriority("medium").Save(ctx)
			require.NoError(t, err)
			approval, err := client.TicketApproval.Create().SetTenantID(tenant.tenantID).
				SetTicketID(ticket.ID).SetLevel(1).SetLevelName("审批").
				SetApproverID(tc.assignee.userID).SetStatus("pending").Save(ctx)
			require.NoError(t, err)
			var taskID int
			if tc.bound {
				deployment, err := client.ProcessDeployment.Create().SetTenantID(tenant.tenantID).
					SetDeploymentID(tc.name).SetDeploymentName(tc.name).SetDeployedBy("fixture").
					SetDeploymentTime(time.Now()).Save(ctx)
				require.NoError(t, err)
				definition, err := client.ProcessDefinition.Create().SetTenantID(tenant.tenantID).
					SetKey("pq01").SetName("审批").SetVersion("1").SetDeploymentID(deployment.ID).
					SetBpmnXML([]byte("<definitions/>")).SetDeployedAt(time.Now()).Save(ctx)
				require.NoError(t, err)
				instance, err := client.ProcessInstance.Create().SetTenantID(tenant.tenantID).
					SetProcessInstanceID(tc.name).SetProcessDefinitionKey(definition.Key).SetProcessDefinitionID(definition.ID).
					SetBusinessKey(fmt.Sprintf("ticket:%d", ticket.ID)).SetStatus("running").Save(ctx)
				require.NoError(t, err)
				task, err := client.ProcessTask.Create().SetTenantID(tenant.tenantID).
					SetProcessInstanceID(instance.ID).SetProcessDefinitionKey("pq01").
					SetTaskID(tc.name).SetTaskDefinitionKey("approval").SetTaskName("审批").
					SetTaskType("user_task").SetStatus("assigned").
					SetAssignee(strconv.Itoa(taskOwner.userID)).
					SetCandidateUsers(strconv.Itoa(approver.userID)).
					SetTaskVariables(map[string]interface{}{"taskPurpose": "approval"}).Save(ctx)
				require.NoError(t, err)
				taskID = task.ID
			}

			ticket, err = client.Ticket.Get(ctx, ticket.ID)
			require.NoError(t, err)
			approval, err = client.TicketApproval.Get(ctx, approval.ID)
			require.NoError(t, err)

			body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":"approve","reason":"review","comment":"review"}`, ticket.ID, approval.ID)
			token, err := middleware.GenerateAccessToken(tc.actor.userID, tc.actor.username, "super_admin", tc.actor.tenantID, ticketCCListSecret, time.Hour)
			require.NoError(t, err)
			if tc.actor.userID == tenant.admin.userID && tc.endpoint == "approve" {
				stateReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/workflow/state-v2", ticket.ID), nil)
				stateReq.Header.Set("Authorization", "Bearer "+token)
				stateResponse := httptest.NewRecorder()
				router.ServeHTTP(stateResponse, stateReq)
				require.Equal(t, http.StatusOK, stateResponse.Code, stateResponse.Body.String())
				var stateEnvelope rawEnvelope
				require.NoError(t, json.Unmarshal(stateResponse.Body.Bytes(), &stateEnvelope))
				require.Zero(t, stateEnvelope.Code)
				var state struct {
					TicketID         int      `json:"ticketId"`
					CanApprove       bool     `json:"canApprove"`
					AvailableActions []string `json:"availableActions"`
				}
				require.NoError(t, json.Unmarshal(stateEnvelope.Data, &state))
				require.Equal(t, ticket.ID, state.TicketID)
				require.False(t, state.CanApprove)
				require.NotContains(t, state.AvailableActions, "approve")
				require.NotContains(t, state.AvailableActions, "approve_reject")
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/workflow/"+tc.endpoint, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			var envelope rawEnvelope
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			require.Equal(t, tc.wantCode, envelope.Code)
			require.NotContains(t, response.Body.String(), "passwordHash")
			require.NotContains(t, response.Body.String(), "tenant_id")

			actualTicket, err := client.Ticket.Get(ctx, ticket.ID)
			require.NoError(t, err)
			approvals, err := client.TicketApproval.Query().Where(ticketapproval.TicketID(ticket.ID)).All(ctx)
			require.NoError(t, err)
			require.Len(t, approvals, 1)
			actualApproval := approvals[0]
			history, err := client.TicketWorkflowRecord.Query().Where(ticketworkflowrecord.TicketID(ticket.ID)).All(ctx)
			require.NoError(t, err)
			if tc.wantCode == 0 {
				require.JSONEq(t, `{"message":"审批通过"}`, string(envelope.Data))
				require.Equal(t, "approved", actualTicket.Status)
				require.Equal(t, "approved", actualApproval.Status)
				require.Len(t, history, 1)
				require.Equal(t, "approve", history[0].Action)
				return
			}
			// Compare persisted fields without Ent's internal driver callbacks.
			before, err := json.Marshal([]any{ticket, approval})
			require.NoError(t, err)
			after, err := json.Marshal([]any{actualTicket, actualApproval})
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
			require.Empty(t, history)
			if taskID != 0 {
				task, err := client.ProcessTask.Get(ctx, taskID)
				require.NoError(t, err)
				require.Equal(t, "assigned", task.Status)
				require.Equal(t, strconv.Itoa(taskOwner.userID), task.Assignee)
				decisions, err := client.ProcessApprovalDecision.Query().Count(ctx)
				require.NoError(t, err)
				require.Zero(t, decisions)
			}
		})
	}
}
