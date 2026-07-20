package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupTestTicketController wires the ticket controller with a middleware that
// injects tenant_id and user_id from headers (X-Test-Tenant, X-Test-User),
// defaulting to 1. To simulate a missing tenant or user, send 0.
func setupTestTicketController(t *testing.T) (*gin.Engine, *ent.Client, *TicketController) {
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	logger := zaptest.NewLogger(t).Sugar()

	ticketService := service.NewTicketServiceForTest(client, logger)
	var ticketDependencyService *service.TicketDependencyService

	ticketController := NewTicketController(ticketService, ticketDependencyService, nil, logger)

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

	r.POST("/api/v1/tickets", ticketController.CreateTicket)
	r.GET("/api/v1/tickets", ticketController.ListTickets)
	r.GET("/api/v1/tickets/:id", ticketController.GetTicket)
	r.PUT("/api/v1/tickets/:id", ticketController.UpdateTicket)
	r.DELETE("/api/v1/tickets/:id", ticketController.DeleteTicket)

	return r, client, ticketController
}

func createTestTenantAndUserForTicket(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
	ctx := context.Background()
	uniqueID := uniqueTestID()

	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("end_user").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}

// apiResp decodes the standard {code,message,data} envelope without coupling
// to a specific success DTO shape.
type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// doJSONRequest runs a request through the router and decodes the envelope. It
// fails the test if the body cannot be parsed. The HTTP status is returned so
// individual cases can assert both code + status if needed. (Renamed from
// doRequest to avoid collision with bpmn_workflow_controller_test.go.)
func doJSONRequest(t *testing.T, r http.Handler, req *http.Request) (apiResp, int) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp apiResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "body=%s", w.Body.String())
	return resp, w.Code
}

func TestTicketController_CreateTicket(t *testing.T) {
	r, client, _ := setupTestTicketController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForTicket(t, client)

	tests := []struct {
		name         string
		request      dto.CreateTicketRequest
		tenantHeader string // "" uses middleware default (1); "0" simulates missing
		userHeader   string
		expectedCode int
	}{
		{
			name: "成功创建工单",
			request: dto.CreateTicketRequest{
				Title:       "测试工单",
				Description: "这是一个测试工单的详细描述",
				Priority:    "medium",
				Category:    "incident",
				FormFields: map[string]interface{}{
					"category": "hardware",
				},
			},
			tenantHeader: strconv.Itoa(tenant.ID),
			userHeader:   strconv.Itoa(user.ID),
			expectedCode: common.SuccessCode,
		},
		{
			name: "标题为空",
			request: dto.CreateTicketRequest{
				Title:       "",
				Description: "描述",
				Priority:    "medium",
				Category:    "incident",
			},
			tenantHeader: strconv.Itoa(tenant.ID),
			userHeader:   strconv.Itoa(user.ID),
			expectedCode: common.ParamErrorCode,
		},
		{
			name: "描述为空",
			request: dto.CreateTicketRequest{
				Title:       "标题",
				Description: "",
				Priority:    "medium",
				Category:    "incident",
			},
			tenantHeader: strconv.Itoa(tenant.ID),
			userHeader:   strconv.Itoa(user.ID),
			expectedCode: common.ParamErrorCode,
		},
		{
			name: "缺少租户ID",
			request: dto.CreateTicketRequest{
				Title:       "测试工单",
				Description: "描述",
				Priority:    "medium",
				Category:    "incident",
			},
			tenantHeader: "0",
			userHeader:   strconv.Itoa(user.ID),
			expectedCode: common.ParamErrorCode,
		},
		{
			name: "缺少用户ID",
			request: dto.CreateTicketRequest{
				Title:       "测试工单",
				Description: "描述",
				Priority:    "medium",
				Category:    "incident",
			},
			tenantHeader: strconv.Itoa(tenant.ID),
			userHeader:   "0",
			expectedCode: common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/tickets", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-Tenant", tt.tenantHeader)
			req.Header.Set("X-Test-User", tt.userHeader)

			resp, _ := doJSONRequest(t, r, req)
			assert.Equal(t, tt.expectedCode, resp.Code, "message=%s", resp.Message)

			// For success cases, the data envelope must contain a created ticket.
			if tt.expectedCode == common.SuccessCode {
				var created dto.TicketResponse
				require.NoError(t, json.Unmarshal(resp.Data, &created), "data=%s", string(resp.Data))
				assert.NotEmpty(t, created.ID, "created ticket should have an ID")
				assert.Equal(t, tt.request.Title, created.Title)
			}
		})
	}
}

func TestTicketController_GetTicket(t *testing.T) {
	r, client, _ := setupTestTicketController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForTicket(t, client)

	ctx := context.Background()
	ticket, err := client.Ticket.Create().
		SetTicketNumber("TKT-001").
		SetTitle("测试工单").
		SetDescription("描述").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ticketID     string
		expectedCode int
	}{
		{
			name:         "成功获取工单",
			ticketID:     strconv.Itoa(ticket.ID),
			expectedCode: common.SuccessCode,
		},
		{
			name:         "无效的工单ID",
			ticketID:     "invalid",
			expectedCode: common.ParamErrorCode,
		},
		{
			name:         "不存在的工单ID",
			ticketID:     "999999",
			expectedCode: common.NotFoundCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/tickets/"+tt.ticketID, nil)
			require.NoError(t, err)
			req.Header.Set("X-Test-Tenant", strconv.Itoa(tenant.ID))

			resp, _ := doJSONRequest(t, r, req)
			assert.Equal(t, tt.expectedCode, resp.Code, "message=%s", resp.Message)
		})
	}
}

func TestTicketController_ListTickets(t *testing.T) {
	r, client, _ := setupTestTicketController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForTicket(t, client)

	ctx := context.Background()
	uniqueID := uniqueTestID()
	for i := 0; i < 3; i++ {
		_, err := client.Ticket.Create().
			SetTicketNumber(fmt.Sprintf("TKT-%s-%03d", uniqueID, i+1)).
			SetTitle(fmt.Sprintf("测试工单 %c", 'A'+i)).
			SetDescription(fmt.Sprintf("描述 %c", 'A'+i)).
			SetPriority("medium").
			SetStatus("open").
			SetRequesterID(user.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name         string
		queryParams  string
		expectedCode int
	}{
		{
			name:         "成功获取工单列表",
			queryParams:  "",
			expectedCode: common.SuccessCode,
		},
		{
			name:         "带分页参数",
			queryParams:  "page=1&pageSize=10",
			expectedCode: common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/tickets"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)
			req.Header.Set("X-Test-Tenant", strconv.Itoa(tenant.ID))

			resp, _ := doJSONRequest(t, r, req)
			assert.Equal(t, tt.expectedCode, resp.Code, "message=%s", resp.Message)

			// Verify the data is a paginated list, not an error envelope.
			var listResp dto.ListTicketsResponse
			require.NoError(t, json.Unmarshal(resp.Data, &listResp), "data=%s", string(resp.Data))
			assert.GreaterOrEqual(t, listResp.Total, 3, "should list all 3 seeded tickets")
			assert.GreaterOrEqual(t, len(listResp.Tickets), 3, "items slice should match total")
		})
	}
}

func TestTicketController_UpdateTicket(t *testing.T) {
	r, client, _ := setupTestTicketController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForTicket(t, client)

	ctx := context.Background()
	ticket, err := client.Ticket.Create().
		SetTicketNumber("TKT-002").
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetPriority("low").
		SetStatus("open").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ticketID     string
		request      dto.UpdateTicketRequest
		expectedCode int
	}{
		{
			name:     "成功更新工单",
			ticketID: strconv.Itoa(ticket.ID),
			request: dto.UpdateTicketRequest{
				Title:       "更新后的标题",
				Description: "更新后的详细描述内容足够长以通过校验",
				Priority:    "high",
			},
			expectedCode: common.SuccessCode,
		},
		{
			name:     "无效的工单ID格式",
			ticketID: "abc",
			request: dto.UpdateTicketRequest{
				Title: "更新后的标题",
			},
			expectedCode: common.ParamErrorCode,
		},
		{
			name:     "工单不存在",
			ticketID: "99999",
			request: dto.UpdateTicketRequest{
				Title: "更新后的标题",
			},
			expectedCode: common.NotFoundCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, err := http.NewRequest("PUT", "/api/v1/tickets/"+tt.ticketID, bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-Tenant", strconv.Itoa(tenant.ID))
			req.Header.Set("X-Test-User", strconv.Itoa(user.ID))

			resp, _ := doJSONRequest(t, r, req)
			assert.Equal(t, tt.expectedCode, resp.Code, "message=%s", resp.Message)

			// For success cases, the returned ticket should reflect the update.
			if tt.expectedCode == common.SuccessCode {
				var updated dto.TicketResponse
				require.NoError(t, json.Unmarshal(resp.Data, &updated), "data=%s", string(resp.Data))
				assert.Equal(t, tt.request.Title, updated.Title)
				assert.Equal(t, tt.request.Priority, updated.Priority)
			}
		})
	}
}

func TestTicketController_DeleteTicket(t *testing.T) {
	r, client, _ := setupTestTicketController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForTicket(t, client)

	ctx := context.Background()
	ticket, err := client.Ticket.Create().
		SetTicketNumber("TKT-003").
		SetTitle("待删除工单").
		SetDescription("将被删除").
		SetPriority("low").
		SetStatus("open").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ticketID     string
		expectedCode int
	}{
		{
			name:         "成功删除工单",
			ticketID:     strconv.Itoa(ticket.ID),
			expectedCode: common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/tickets/"+tt.ticketID, nil)
			require.NoError(t, err)
			req.Header.Set("X-Test-Tenant", strconv.Itoa(tenant.ID))
			req.Header.Set("X-Test-User", strconv.Itoa(user.ID))

			resp, _ := doJSONRequest(t, r, req)
			assert.Equal(t, tt.expectedCode, resp.Code, "message=%s", resp.Message)

			// Verify deletion: a follow-up GET should return NotFoundCode.
			verifyReq, err := http.NewRequest("GET", "/api/v1/tickets/"+tt.ticketID, nil)
			require.NoError(t, err)
			verifyReq.Header.Set("X-Test-Tenant", strconv.Itoa(tenant.ID))
			verifyResp, _ := doJSONRequest(t, r, verifyReq)
			assert.Equal(t, common.NotFoundCode, verifyResp.Code, "deleted ticket should 404")
		})
	}
}