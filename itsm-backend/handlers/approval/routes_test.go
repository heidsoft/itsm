package approval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/approval"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type approvalRouteIdentity struct {
	tenantID int
	userID   int
	role     string
}

type approvalRouteFixture struct {
	client    *ent.Client
	router    *gin.Engine
	identity  *approvalRouteIdentity
	tenantA   int
	tenantB   int
	actorA    int
	actorB    int
	recordA   *ent.ApprovalRecord
	recordB   *ent.ApprovalRecord
	mutations int
}

func newApprovalRouteFixture(t *testing.T) *approvalRouteFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	f := &approvalRouteFixture{client: client, identity: &approvalRouteIdentity{}}
	for _, suffix := range []string{"A", "B"} {
		tenant := client.Tenant.Create().SetName("Tenant " + suffix).SetCode("retired-" + suffix).SaveX(ctx)
		actor := client.User.Create().SetUsername("approver" + suffix).SetName("Approver " + suffix).
			SetEmail("approver" + suffix + "@example.com").SetPasswordHash("test-only").
			SetRole("agent").SetTenantID(tenant.ID).SaveX(ctx)
		readPermission := client.Permission.Create().SetCode("approval:read").SetName("Read history").
			SetResource("approval").SetAction("read").SetTenantID(tenant.ID).SaveX(ctx)
		writePermission := client.Permission.Create().SetCode("approval:write").SetName("Submit approval").
			SetResource("approval").SetAction("write").SetTenantID(tenant.ID).SaveX(ctx)
		approverRole := client.Role.Create().SetCode("retirement-approver").SetName("Approver").SetTenantID(tenant.ID).SaveX(ctx)
		readerRole := client.Role.Create().SetCode("retirement-reader").SetName("Reader").SetTenantID(tenant.ID).SaveX(ctx)
		for _, role := range []*ent.Role{approverRole, readerRole} {
			client.RolePermission.Create().SetRoleID(role.ID).SetPermissionID(readPermission.ID).SetTenantID(tenant.ID).SaveX(ctx)
		}
		client.RolePermission.Create().SetRoleID(approverRole.ID).SetPermissionID(writePermission.ID).SetTenantID(tenant.ID).SaveX(ctx)
		workflow := client.ApprovalWorkflow.Create().SetName("Legacy " + suffix).SetIsActive(true).
			SetTenantID(tenant.ID).SetNodes([]map[string]interface{}{{
			"level": 1, "name": "Approval", "approverType": "user", "approverIds": []int{actor.ID},
			"approvalMode": "any", "allowReject": true, "allowDelegate": true,
		}}).SaveX(ctx)
		ticket := client.Ticket.Create().SetTitle("Historical ticket " + suffix).SetDescription("private description").
			SetTicketNumber("LEGACY-" + suffix).SetStatus("open").SetPriority("medium").SetType("ticket").
			SetRequesterID(actor.ID).SetTenantID(tenant.ID).SaveX(ctx)
		record := client.ApprovalRecord.Create().SetWorkflowID(workflow.ID).SetWorkflowName(workflow.Name).
			SetTicketID(ticket.ID).SetTicketNumber(ticket.TicketNumber).SetTicketTitle(ticket.Title).
			SetApproverID(actor.ID).SetApproverName(actor.Name).SetCurrentLevel(1).SetTotalLevels(1).
			SetStepOrder(1).SetStatus("pending").SetTenantID(tenant.ID).SaveX(ctx)
		if suffix == "A" {
			f.tenantA, f.actorA, f.recordA = tenant.ID, actor.ID, record
		} else {
			f.tenantB, f.actorB, f.recordB = tenant.ID, actor.ID, record
		}
	}
	*f.identity = approvalRouteIdentity{tenantID: f.tenantA, userID: f.actorA, role: "retirement-approver"}
	f.router = gin.New()
	// Only authentication context is injected; production route registration and
	// RequirePermission still execute against tenant-scoped in-memory RBAC data.
	f.router.Use(func(c *gin.Context) {
		c.Set("client", client)
		if f.identity.tenantID != 0 {
			c.Set("tenant_id", f.identity.tenantID)
		}
		if f.identity.userID != 0 {
			c.Set("user_id", f.identity.userID)
		}
		if f.identity.role != "" {
			c.Set("role", f.identity.role)
		}
		c.Next()
	})
	router.SetupTicketRoutes(f.router.Group("/api/v1"), &router.RouterConfig{
		ApprovalHandler: approval.NewHandler(service.NewApprovalService(client, zaptest.NewLogger(t).Sugar())),
	})
	client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			f.mutations++
			return next.Mutate(ctx, mutation)
		})
	})
	return f
}

func (f *approvalRouteFixture) snapshot(t *testing.T) []byte {
	t.Helper()
	ctx := context.Background()
	// Cover legacy rows plus any accidental BPMN migration or outbox enqueue.
	value := map[string]interface{}{
		"records":     f.client.ApprovalRecord.Query().Order(ent.Asc("id")).AllX(ctx),
		"tickets":     f.client.Ticket.Query().Order(ent.Asc("id")).AllX(ctx),
		"workflows":   f.client.ApprovalWorkflow.Query().Order(ent.Asc("id")).AllX(ctx),
		"definitions": f.client.ProcessDefinition.Query().CountX(ctx),
		"instances":   f.client.ProcessInstance.Query().CountX(ctx),
		"tasks":       f.client.ProcessTask.Query().CountX(ctx),
		"commands":    f.client.OperationalCommand.Query().CountX(ctx),
	}
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}

func TestLegacyApprovalSubmitRoutes_Retired(t *testing.T) {
	f := newApprovalRouteFixture(t)
	before := f.snapshot(t)
	for _, path := range []string{"/api/v1/approvals/submit", "/api/v1/tickets/approval/submit"} {
		for _, action := range []string{"approve", "reject", "delegate", "unknown"} {
			t.Run(path+"/"+action, func(t *testing.T) {
				for _, record := range []*ent.ApprovalRecord{f.recordA, f.recordB} {
					for attempt := 0; attempt < 2; attempt++ {
						body := fmt.Sprintf(`{"ticketId":%d,"approvalId":%d,"action":%q,"comment":"private comment","delegateToUserId":%d}`, record.TicketID, record.ID, action, f.actorB)
						req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
						req.Header.Set("Content-Type", "application/json")
						w := httptest.NewRecorder()
						f.router.ServeHTTP(w, req)
						assert.Equal(t, http.StatusGone, w.Code, w.Body.String())
						assert.Equal(t, `</api/v1/bpmn/tasks/:id/decisions>; rel="successor-version"`, w.Header().Get("Link"))
						var response common.Response
						require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
						assert.Equal(t, 4100, response.Code)
						assert.Contains(t, response.Message, "BPMN")
						assert.Nil(t, response.Data)
						assert.NotContains(t, w.Body.String(), "private")
						assert.NotContains(t, w.Body.String(), "approvalId")
					}
				}
			})
		}
	}
	assert.Zero(t, f.mutations, "retired HTTP submissions must not perform any Ent mutations")
	assert.JSONEq(t, string(before), string(f.snapshot(t)))
}

func TestLegacyApprovalSubmitRoutes_RequireIdentityAndPermission(t *testing.T) {
	f := newApprovalRouteFixture(t)
	for _, path := range []string{"/api/v1/approvals/submit", "/api/v1/tickets/approval/submit"} {
		for _, tc := range []struct {
			name     string
			identity approvalRouteIdentity
			status   int
			code     int
		}{
			{"no identity", approvalRouteIdentity{}, http.StatusUnauthorized, common.AuthFailedCode},
			{"no tenant", approvalRouteIdentity{userID: f.actorA, role: "retirement-approver"}, http.StatusUnauthorized, common.AuthFailedCode},
			{"no user", approvalRouteIdentity{tenantID: f.tenantA, role: "retirement-approver"}, http.StatusUnauthorized, common.AuthFailedCode},
			{"invalid user", approvalRouteIdentity{tenantID: f.tenantA, userID: -1, role: "retirement-approver"}, http.StatusUnauthorized, common.AuthFailedCode},
			{"no permission", approvalRouteIdentity{tenantID: f.tenantA, userID: f.actorA, role: "retirement-reader"}, http.StatusForbidden, common.ForbiddenCode},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				*f.identity = tc.identity
				// Caller-supplied identity must never bypass the middleware context.
				req := httptest.NewRequest(http.MethodPost, path+"?tenantId=1&userId=1", strings.NewReader(`{"tenantId":1,"userId":1}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Tenant-ID", "1")
				req.Header.Set("X-User-ID", "1")
				w := httptest.NewRecorder()
				f.router.ServeHTTP(w, req)
				assert.Equal(t, tc.status, w.Code, w.Body.String())
				var response common.Response
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				assert.Equal(t, tc.code, response.Code)
			})
		}
	}
	assert.Zero(t, f.mutations)
}

func TestLegacyApprovalHistoryRoutes_PreserveTenantScope(t *testing.T) {
	f := newApprovalRouteFixture(t)
	for _, identity := range []approvalRouteIdentity{
		{tenantID: f.tenantA, userID: f.actorA, role: "retirement-reader"},
		{tenantID: f.tenantB, userID: f.actorB, role: "retirement-reader"},
	} {
		*f.identity = identity
		for _, path := range []string{"/api/v1/tickets/approval/records", "/api/v1/approvals/records", "/api/v1/approval-records", "/api/v1/my-approvals"} {
			w := httptest.NewRecorder()
			f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path+"?page=1&pageSize=20", nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var response struct {
				Code int `json:"code"`
				Data struct {
					Items    []dto.ApprovalRecordResponse `json:"items"`
					Total    int                          `json:"total"`
					Page     int                          `json:"page"`
					PageSize int                          `json:"pageSize"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Zero(t, response.Code)
			require.Len(t, response.Data.Items, 1)
			assert.Equal(t, 1, response.Data.Total)
			assert.Equal(t, 1, response.Data.Page)
			assert.Equal(t, 20, response.Data.PageSize)
			assert.Equal(t, identity.userID, response.Data.Items[0].ApproverID)
			assert.Equal(t, "pending", response.Data.Items[0].Status)
			assert.Contains(t, w.Body.String(), `"ticketNumber"`)
			assert.NotContains(t, w.Body.String(), `"ticket_number"`)
		}
	}
	assert.Zero(t, f.mutations)
}
