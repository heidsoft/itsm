package ticket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// mockRepository
// -----------------------------------------------------------------------------

type mockRepository struct {
	mu          sync.Mutex
	tickets     map[int]*Ticket
	events      []string
	nextID      int
	statsCalled bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{tickets: make(map[int]*Ticket)}
}

func (m *mockRepository) Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	t := &Ticket{
		ID:           m.nextID,
		TicketNumber: "TKT-" + time.Now().Format("20060102") + "-001",
		Title:        params.Title,
		Description:  params.Description,
		Status:       "new",
		Priority:     params.Priority,
		Type:         params.Type,
		RequesterID:  params.RequesterID,
		AssigneeID:   params.AssigneeID,
		TenantID:     tenantID,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.tickets[t.ID] = t
	return t, nil
}

func (m *mockRepository) GetByID(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok || t.TenantID != tenantID {
		return nil, errors.New("ticket not found")
	}
	return t, nil
}

func (m *mockRepository) GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tickets {
		if t.TicketNumber == ticketNumber && t.TenantID == tenantID {
			return t, nil
		}
	}
	return nil, errors.New("ticket not found")
}

func (m *mockRepository) Update(ctx context.Context, id int, params *UpdateParams, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok || t.TenantID != tenantID {
		return nil, errors.New("ticket not found")
	}
	if params.Title != nil {
		t.Title = *params.Title
	}
	if params.Status != nil {
		t.Status = *params.Status
	}
	if params.Priority != nil {
		t.Priority = *params.Priority
	}
	t.Version++
	t.UpdatedAt = time.Now()
	return t, nil
}

func (m *mockRepository) Delete(ctx context.Context, id int, tenantID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok || t.TenantID != tenantID {
		return errors.New("ticket not found")
	}
	delete(m.tickets, id)
	return nil
}

func (m *mockRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, ds datascope.DataScope, currentUserID int) ([]*Ticket, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Ticket
	for _, t := range m.tickets {
		if t.TenantID == tenantID {
			out = append(out, t)
		}
	}
	return out, len(out), nil
}

func (m *mockRepository) BatchDelete(ctx context.Context, ids []int, tenantID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.tickets, id)
	}
	return nil
}

func (m *mockRepository) Exists(ctx context.Context, id int, tenantID int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	return ok && t.TenantID == tenantID, nil
}

func (m *mockRepository) FindByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Ticket
	for _, t := range m.tickets {
		if t.TenantID == tenantID && t.AssigneeID != nil && *t.AssigneeID == assigneeID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *mockRepository) FindOverdue(ctx context.Context, tenantID int) ([]*Ticket, error) {
	return []*Ticket{}, nil
}

func (m *mockRepository) Search(ctx context.Context, keyword string, tenantID int) ([]*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Ticket
	for _, t := range m.tickets {
		if t.TenantID == tenantID {
			if contains(t.Title, keyword) || contains(t.Description, keyword) {
				out = append(out, t)
			}
		}
	}
	return out, nil
}

func (m *mockRepository) GetStats(ctx context.Context, tenantID int) (*TicketStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statsCalled = true
	stats := &TicketStats{}
	for _, t := range m.tickets {
		if t.TenantID == tenantID {
			stats.TotalTickets++
		}
	}
	return stats, nil
}

func (m *mockRepository) GenerateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	return "TKT-" + time.Now().Format("20060102") + "-001", nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, id int, status string, tenantID int) (*Ticket, error) {
	return m.Update(ctx, id, &UpdateParams{Status: &status}, tenantID)
}

func (m *mockRepository) AssignTicket(ctx context.Context, id int, assigneeID int, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok || t.TenantID != tenantID {
		return nil, errors.New("ticket not found")
	}
	t.AssigneeID = &assigneeID
	if t.Status == "new" {
		t.Status = "open"
	}
	t.Version++
	return t, nil
}

func (m *mockRepository) ResolveTicket(ctx context.Context, id int, resolution string, tenantID int) (*Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tickets[id]
	if !ok || t.TenantID != tenantID {
		return nil, errors.New("ticket not found")
	}
	t.Status = "resolved"
	t.Resolution = &resolution
	now := time.Now()
	t.ResolvedAt = &now
	t.Version++
	return t, nil
}

func (m *mockRepository) CloseTicket(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	return m.UpdateStatus(ctx, id, "closed", tenantID)
}

func (m *mockRepository) EscalateTicket(ctx context.Context, id int, reason string, tenantID int, escalatedBy int) (*Ticket, error) {
	return m.UpdateStatus(ctx, id, "in_progress", tenantID)
}

func (m *mockRepository) UpdateSLADeadlines(ctx context.Context, id int, responseDeadline, resolutionDeadline *time.Time, slaDefinitionID *int, tenantID int) error {
	return nil
}

func (m *mockRepository) CreateTemplate(ctx context.Context, tmpl *TicketTemplate, tenantID int) (*TicketTemplate, error) {
	return tmpl, nil
}

func (m *mockRepository) UpdateTemplate(ctx context.Context, id int, tmpl *TicketTemplate, tenantID int) (*TicketTemplate, error) {
	return tmpl, nil
}

func (m *mockRepository) DeleteTemplate(ctx context.Context, id int, tenantID int) error {
	return nil
}

func (m *mockRepository) GetTemplate(ctx context.Context, id int, tenantID int) (*TicketTemplate, error) {
	return nil, errors.New("template not found")
}

func (m *mockRepository) ListTemplates(ctx context.Context, tenantID int) ([]*TicketTemplate, error) {
	return []*TicketTemplate{}, nil
}

func (m *mockRepository) UpdateTemplateStatus(ctx context.Context, id int, isActive bool, tenantID int) (*TicketTemplate, error) {
	return &TicketTemplate{ID: id, IsActive: isActive}, nil
}

func (m *mockRepository) CopyTemplate(ctx context.Context, id int, newName string, tenantID int) (*TicketTemplate, error) {
	return &TicketTemplate{ID: id, Name: newName}, nil
}

func (m *mockRepository) GetTemplateCategories(ctx context.Context, tenantID int) ([]string, error) {
	return []string{}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// test harness
// -----------------------------------------------------------------------------

func newTestHarness(t *testing.T) (*gin.Engine, *mockRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newMockRepository()
	svc := NewService(repo, nil, zap.NewNop().Sugar())
	h := NewHandler(svc)
	r := gin.New()

	auth := func(c *gin.Context) {
		if v := c.GetHeader("X-Test-TenantID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: n})
			}
		}
		if v := c.GetHeader("X-Test-UserID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Set("user_id", n)
			}
		}
		if v := c.GetHeader("X-Test-Role"); v != "" {
			c.Set("role", v)
		} else {
			c.Set("role", "agent")
		}
		c.Next()
	}

	api := r.Group("/api/v1", auth)
	api.POST("/tickets", h.CreateTicket)
	api.GET("/tickets", h.ListTickets)
	api.GET("/tickets/:id", h.GetTicket)
	api.PUT("/tickets/:id", h.UpdateTicket)
	api.DELETE("/tickets/:id", h.DeleteTicket)
	api.POST("/tickets/:id/assign", h.AssignTicket)
	api.POST("/tickets/:id/escalate", h.EscalateTicket)
	api.POST("/tickets/:id/resolve", h.ResolveTicket)
	api.POST("/tickets/:id/close", h.CloseTicket)
	api.PUT("/tickets/:id/status", h.UpdateTicketStatus)
	api.GET("/tickets/stats", h.GetTicketStats)
	api.GET("/tickets/search", h.SearchTickets)

	return r, repo
}

func doJSON(t *testing.T, r http.Handler, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func strconvAtoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// -----------------------------------------------------------------------------
// handler tests
// -----------------------------------------------------------------------------

func TestHandler_Create_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		body      interface{}
		tenantHdr string
		userHdr   string
		wantCode  int
	}{
		{
			name:      "rejects empty title",
			body:      map[string]interface{}{},
			tenantHdr: "1",
			userHdr:   "7",
			wantCode:  400,
		},
		{
			name:      "rejects missing tenant",
			body:      dto.CreateTicketRequest{Title: "Test", Priority: "low"},
			tenantHdr: "0",
			userHdr:   "7",
			wantCode:  401,
		},
		{
			name:      "rejects missing user",
			body:      dto.CreateTicketRequest{Title: "Test", Priority: "low"},
			tenantHdr: "1",
			userHdr:   "0",
			wantCode:  401,
		},
		{
			name:      "happy path",
			body:      dto.CreateTicketRequest{Title: "Server down", Description: "Production", Priority: "high"},
			tenantHdr: "1",
			userHdr:   "7",
			wantCode:  200,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newTestHarness(t)
			w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
				tc.body,
				map[string]string{
					"X-Test-TenantID": tc.tenantHdr,
					"X-Test-UserID":   tc.userHdr,
				},
			)
			assert.Equal(t, tc.wantCode, w.Code, "body=%s", w.Body.String())
		})
	}
}

func TestHandler_Get_NotFoundTable(t *testing.T) {
	cases := []struct {
		name      string
		idParam   string
		tenantHdr string
		want      int
	}{
		// GetTicket 现支持业务工单号 fallback（非数字 ID 走 GetByNumber），
		// 无效/不存在的工单号统一返回 404 而非 400。
		{"invalid id", "abc", "1", 404},
		{"non-existing id", "999", "1", 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newTestHarness(t)
			w := doJSON(t, r, http.MethodGet,
				"/api/v1/tickets/"+tc.idParam, nil,
				map[string]string{"X-Test-TenantID": tc.tenantHdr, "X-Test-UserID": "7"},
			)
			assert.Equal(t, tc.want, w.Code)
		})
	}
}

func TestHandler_Get_TenantIsolation(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed a ticket for tenant 1
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "T1 ticket", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	if len(repo.tickets) == 0 {
		t.Fatal("seed not recorded")
	}
	var id int
	for id = range repo.tickets {
		break
	}

	// Same id, but read with tenant 2 → must NOT leak data
	w = doJSON(t, r, http.MethodGet,
		"/api/v1/tickets/"+strconv.Itoa(id), nil,
		map[string]string{"X-Test-TenantID": "2", "X-Test-UserID": "8"},
	)
	assert.NotEqual(t, 200, w.Code, "tenant 2 must not see tenant 1's ticket")
}

func TestHandler_Update_TableDriven(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed ticket
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "Original", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}

	cases := []struct {
		name string
		id   string
		body interface{}
		want int
	}{
		{"invalid id", "xyz", dto.UpdateTicketRequest{}, 400},
		{"valid update", strconv.Itoa(id), dto.UpdateTicketRequest{Title: "Updated"}, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, http.MethodPut,
				"/api/v1/tickets/"+tc.id,
				tc.body,
				map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
			)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestHandler_Delete_TableDriven(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed ticket
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "To delete", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}

	// Delete
	w = doJSON(t, r, http.MethodDelete,
		"/api/v1/tickets/"+strconv.Itoa(id), nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	// Verify gone
	w = doJSON(t, r, http.MethodGet,
		"/api/v1/tickets/"+strconv.Itoa(id), nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 404, w.Code)
}

// TestHandler_WritePath_RowLevelForbidden 锁定 P1-DataScope 行级写权限：
// 测试基建默认 role=agent；agent 仅能改/删自己创建的工单，
// 换一个非 owner 的 agent 身份访问必须 403。
func TestHandler_WritePath_RowLevelForbidden(t *testing.T) {
	r, repo := newTestHarness(t)

	// user 7 (agent) 创建工单 → requester_id=7
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "Owner only", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}

	// user 8 (agent, 非 owner 非 assignee) 更新 → 403
	w = doJSON(t, r, http.MethodPut,
		"/api/v1/tickets/"+strconv.Itoa(id),
		dto.UpdateTicketRequest{Title: "hijack"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "8"},
	)
	assert.Equal(t, 403, w.Code, w.Body.String())

	// user 8 删除 → 403
	w = doJSON(t, r, http.MethodDelete,
		"/api/v1/tickets/"+strconv.Itoa(id), nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "8"},
	)
	assert.Equal(t, 403, w.Code, w.Body.String())

	// owner (user 7) 更新仍成功
	w = doJSON(t, r, http.MethodPut,
		"/api/v1/tickets/"+strconv.Itoa(id),
		dto.UpdateTicketRequest{Title: "owner edit"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())
	assert.NotEqual(t, "hijack", repo.tickets[id].Title, "被拒绝的修改不应落库")
}

// TestHandler_LifecycleOps_RowLevelForbidden 锁定 P1-DataScope #25 批次：
// resolve/close/escalate/updateStatus 四个生命周期写操作与 Update/Delete
// 同风险面——非 owner 普通角色（agent）操作他人工单必须 403，且行级
// AppError 不得被兑底吞成 500；owner 操作正常。
func TestHandler_LifecycleOps_RowLevelForbidden(t *testing.T) {
	r, repo := newTestHarness(t)

	// user 7 (agent) 创建工单 → requester_id=7
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "Lifecycle guard", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}
	idStr := strconv.Itoa(id)

	// 非 owner（user 8, agent）四操作全部 403
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"resolve 非owner拒绝", http.MethodPost, "/api/v1/tickets/" + idStr + "/resolve", dto.ResolveTicketRequest{Resolution: "fixed"}},
		{"close 非owner拒绝", http.MethodPost, "/api/v1/tickets/" + idStr + "/close", dto.CloseTicketRequest{}},
		{"escalate 非owner拒绝", http.MethodPost, "/api/v1/tickets/" + idStr + "/escalate", dto.EscalateTicketRequest{Reason: "breach"}},
		{"updateStatus 非owner拒绝", http.MethodPut, "/api/v1/tickets/" + idStr + "/status", map[string]string{"status": "closed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, tc.method, tc.path, tc.body,
				map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "8"},
			)
			assert.Equal(t, 403, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "仅创建人、受理人或管理员可操作", "必须是行级守卫文案而非通用 RBAC 拒绝")
		})
	}

	// owner（user 7）close 正常放行
	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/"+idStr+"/close", dto.CloseTicketRequest{},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// TestHandler_LifecycleOps_AdminLikeBypass 锁定管理角色全租户可写语义：
// 非 owner 的 admin-like 角色四操作放行。
func TestHandler_LifecycleOps_AdminLikeBypass(t *testing.T) {
	r, repo := newTestHarness(t)

	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "Admin bypass", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}
	idStr := strconv.Itoa(id)

	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/"+idStr+"/resolve",
		dto.ResolveTicketRequest{Resolution: "fixed by admin"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "9", "X-Test-Role": "manager"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())
}

func TestHandler_AssignTicket(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed ticket
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "Unassigned", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	var id int
	for id = range repo.tickets {
		break
	}

	w = doJSON(t, r, http.MethodPost,
		"/api/v1/tickets/"+strconv.Itoa(id)+"/assign",
		dto.AssignTicketRequest{AssigneeID: 42},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)
}

func TestHandler_SearchTickets_EmptyKeyword(t *testing.T) {
	r, _ := newTestHarness(t)
	w := doJSON(t, r, http.MethodGet,
		"/api/v1/tickets/search?q=", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 400, w.Code)
}
