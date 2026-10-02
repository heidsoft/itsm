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

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// mockRepository
// -----------------------------------------------------------------------------

type mockRepository struct {
	mu          sync.Mutex
	tickets     map[int]*Ticket
	nextID      int
	statsCalled bool
	// 记录 List 收到的分页与筛选参数，用于断言 handler 在调用仓储前已归一化并完整下传。
	lastListPage    int
	lastListSize    int
	lastListFilters map[string]interface{}
	// 行级数据权限同样要在 mock 里生效：mock 忽略 ds/currentUserID 时，
	// 非管理角色读到租户全量也不会被任何测试发现。
	lastListDataScope datascope.DataScope
	lastListUserID    int
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
	m.lastListPage = page
	m.lastListSize = size
	m.lastListFilters = filters
	m.lastListDataScope = ds
	m.lastListUserID = currentUserID

	// 与 EntRepository.List 消费同一套 filters 键位：handler 漏传一个键时
	// 这里必须返回「未筛选」的结果，测试才能区分「筛选生效」和「静默失效」。
	// 行级谓词也与仓储一致：非全量角色只能看到本人创建或受理的单据，身份缺失 fail closed。
	if ds != datascope.DataScopeAll && currentUserID <= 0 {
		return nil, 0, nil
	}
	var out []*Ticket
	for _, t := range m.tickets {
		if t.TenantID != tenantID {
			continue
		}
		if ds != datascope.DataScopeAll {
			assigned := t.AssigneeID != nil && *t.AssigneeID == currentUserID
			if t.RequesterID != currentUserID && !assigned {
				continue
			}
		}
		if v, ok := filters["status"].(string); ok && v != "" && t.Status != v {
			continue
		}
		if v, ok := filters["priority"].(string); ok && v != "" && t.Priority != v {
			continue
		}
		if v, ok := filters["type"].(string); ok && v != "" && t.Type != v {
			continue
		}
		if v, ok := filters["assignee_id"].(int); ok && v > 0 && (t.AssigneeID == nil || *t.AssigneeID != v) {
			continue
		}
		if v, ok := filters["requester_id"].(int); ok && v > 0 && t.RequesterID != v {
			continue
		}
		out = append(out, t)
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
	svc := NewService(repo, nil, nil, zap.NewNop().Sugar())
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

	// owner（user 7）先走 new → open（合法迁移）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "open"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())

	// owner（user 7）再走 open → resolved（合法迁移）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "resolved"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())

	// owner（user 7）最后走 resolved → close（合法迁移）
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

	// 先由 owner 把工单从 new 转到 open（合法迁移），再测 admin-like 角色 resolve
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "open"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())

	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/"+idStr+"/resolve",
		dto.ResolveTicketRequest{Resolution: "fixed by admin"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "9", "X-Test-Role": "manager"},
	)
	assert.Equal(t, 200, w.Code, w.Body.String())
}

// TestHandler_StateMachineWhitelist 验证工单状态机白名单：非法迁移返回 4090，合法迁移放行。
func TestHandler_StateMachineWhitelist(t *testing.T) {
	r, _ := newTestHarness(t)

	// 创建工单（status=new）
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "State machine test", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	// 从响应解析 ID
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	id := int(data["id"].(float64))
	idStr := strconv.Itoa(id)

	// 非法迁移：new → closed（不允许）
	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/"+idStr+"/close", dto.CloseTicketRequest{},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 409, w.Code, "new → closed 应返回 409")
	assert.Contains(t, w.Body.String(), "4090", "业务错误码必须是 4090")
	assert.Contains(t, w.Body.String(), "当前工单状态不允许此操作")

	// 非法迁移：new → resolved（不允许）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "resolved"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 409, w.Code, "new → resolved 应返回 409")
	assert.Contains(t, w.Body.String(), "4090")

	// 合法迁移：new → open（允许）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "open"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, "new → open 应放行", w.Body.String())

	// 合法迁移：open → resolved（允许）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "resolved"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, "open → resolved 应放行", w.Body.String())

	// 合法迁移：resolved → closed（允许）
	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/"+idStr+"/close", dto.CloseTicketRequest{},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code, "resolved → closed 应放行", w.Body.String())

	// 终态验证：closed → 任何状态（不允许）
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/"+idStr+"/status", map[string]string{"status": "open"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 409, w.Code, "closed → open 应返回 409")
	assert.Contains(t, w.Body.String(), "4090")
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

// listEnvelope 只声明本用例要断言的信封键；items 用 json.RawMessage 以免耦合工单字段。
type listEnvelope struct {
	Code int `json:"code"`
	Data struct {
		Items      []map[string]interface{} `json:"items"`
		Total      int                      `json:"total"`
		Page       int                      `json:"page"`
		PageSize   int                      `json:"pageSize"`
		TotalPages int                      `json:"totalPages"`
	} `json:"data"`
}

func seedTickets(t *testing.T, r http.Handler, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
			dto.CreateTicketRequest{Title: "list seed", Priority: "low"},
			map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
		)
		require.Equal(t, 200, w.Code, w.Body.String())
	}
}

// seedTicketWith 按指定字段创建一张工单，用于验证筛选键位真的能区分工单。
func seedTicketWith(t *testing.T, r http.Handler, req dto.CreateTicketRequest) {
	t.Helper()
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets", req,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestHandler_ListTickets_UnpagedRequest 锁死 2026-10-02 e2e 实测的缺陷：
// GET /api/v1/tickets 省略 page/pageSize 时，DTO 零值被同时下传给仓储（负偏移）
// 和信封（除零溢出），响应里出现 totalPages: -9223372036854775808。
func TestHandler_ListTickets_UnpagedRequest(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 3)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	var got listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 0, got.Code)
	assert.Len(t, got.Data.Items, 3)
	assert.Equal(t, 3, got.Data.Total)
	assert.Equal(t, 1, got.Data.Page)
	assert.Equal(t, common.DefaultPageSize, got.Data.PageSize)
	assert.Equal(t, 1, got.Data.TotalPages)

	// 信封与真实查询必须说同一件事：归一化后的值要下传到仓储，而不是零值。
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, 1, repo.lastListPage, "page 必须在调用仓储前归一化")
	assert.Equal(t, common.DefaultPageSize, repo.lastListSize, "pageSize 必须在调用仓储前归一化")
}

// TestHandler_ListTickets_NegativePagination 覆盖 ?page=-3&pageSize=0：
// 数字但越界的分页参数能过绑定（DTO 未设 min），归一化后必须回到默认页，
// 禁止把负偏移下传给仓储或让 totalPages 除零溢出。
func TestHandler_ListTickets_NegativePagination(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 2)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?page=-3&pageSize=0", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	var got listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 1, got.Data.Page)
	assert.Equal(t, common.DefaultPageSize, got.Data.PageSize)
	assert.Equal(t, 1, got.Data.TotalPages)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, 1, repo.lastListPage)
	assert.Equal(t, common.DefaultPageSize, repo.lastListSize)
}

// TestHandler_ListTickets_NonNumericPagination 固定住另一侧边界：
// 非数字分页参数属于输入错误，必须 400，而不是被静默当成默认页。
func TestHandler_ListTickets_NonNumericPagination(t *testing.T) {
	r, _ := newTestHarness(t)
	seedTickets(t, r, 1)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?page=abc", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestHandler_ListTickets_ExplicitPaging 确认归一化没有把合法请求改坏：
// 显式 pageSize=1 时必须逐键回显并算出 3 页。
func TestHandler_ListTickets_ExplicitPaging(t *testing.T) {
	r, _ := newTestHarness(t)
	seedTickets(t, r, 3)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?page=2&pageSize=1", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	var got listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 2, got.Data.Page)
	assert.Equal(t, 1, got.Data.PageSize)
	assert.Equal(t, 3, got.Data.TotalPages)
}

// TestHandler_ListTickets_ForwardsEveryDeclaredFilter 锁死 2026-10-02 实测的筛选静默失效：
// ListTicketsRequest 声明了 assigneeId/requesterId/type/categoryId/parentTicketId/templateId，
// 仓储层也按这些键位实现筛选，但 handler 只下传 status/priority/keyword/排序，
// 于是 GET /api/v1/tickets?assigneeId=33 返回租户全量工单（实测 41 条、受理人全是别人），
// 而 src/app/(main)/tickets/page.tsx 正在发送 assigneeId。
func TestHandler_ListTickets_ForwardsEveryDeclaredFilter(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTicketWith(t, r, dto.CreateTicketRequest{Title: "mine", Priority: "low", Type: "incident", AssigneeID: 7})
	seedTicketWith(t, r, dto.CreateTicketRequest{Title: "theirs", Priority: "low", Type: "problem", AssigneeID: 8})

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?assigneeId=7&type=incident", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	var got listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, 1, got.Data.Total, "assigneeId/type 必须真实生效，不能回全量")
	require.Len(t, got.Data.Items, 1)
	assert.Equal(t, "mine", got.Data.Items[0]["title"])

	repo.mu.Lock()
	defer repo.mu.Unlock()
	// 键位与仓储契约一致（filters["assignee_id"]/filters["type"]）。
	assert.Equal(t, 7, repo.lastListFilters["assignee_id"])
	assert.Equal(t, "incident", repo.lastListFilters["type"])
}

// TestHandler_ListTickets_FilterOwnershipKeys 断言其余归属类筛选同样下传，
// 防止以后只修 assigneeId 又让 requesterId/categoryId 回到静默失效。
func TestHandler_ListTickets_FilterOwnershipKeys(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTicketWith(t, r, dto.CreateTicketRequest{Title: "req 9", Priority: "low", RequesterID: 9})

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?requesterId=9&categoryId=3&parentTicketId=4&templateId=5&isOverdue=true", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, 9, repo.lastListFilters["requester_id"])
	assert.Equal(t, 3, repo.lastListFilters["category_id"])
	assert.Equal(t, 4, repo.lastListFilters["parent_ticket_id"])
	assert.Equal(t, 5, repo.lastListFilters["template_id"])
	assert.Equal(t, true, repo.lastListFilters["is_overdue"])
}

// TestHandler_ListTickets_AbsentFiltersAreNotSent 保证「未传」与「传 0」都不会污染查询：
// 零值/缺省键位若下传，仓储的 >0 判断虽会忽略，但断言能挡住把 0 当成筛选条件的写法。
func TestHandler_ListTickets_AbsentFiltersAreNotSent(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTicketWith(t, r, dto.CreateTicketRequest{Title: "plain", Priority: "low", AssigneeID: 7})

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?assigneeId=0", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"})
	require.Equal(t, 200, w.Code, w.Body.String())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	_, sent := repo.lastListFilters["assignee_id"]
	assert.False(t, sent, "assigneeId=0 是「未指定处理人」，不得作为筛选条件下传")
}

// TestHandler_ListTickets_RowLevelDataScope 锁住生产路由的行级数据权限：
// 非管理角色的 GET /api/v1/tickets 必须被收窄到本人创建或受理的单据。
// 存量 service/ticket_service_test.go 里的同名场景测试走的是旧 TicketService
// 路径（直接使用仓储层枚举），因此生产路由把 DataScope 传丢时它仍然是绿的。
func TestHandler_ListTickets_RowLevelDataScope(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 2) // 创建人 7
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
		dto.CreateTicketRequest{Title: "foreign", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "8"})
	require.Equal(t, 200, w.Code, w.Body.String())

	list := func(userID, role string) (int, int, datascope.DataScope) {
		t.Helper()
		lw := doJSON(t, r, http.MethodGet, "/api/v1/tickets?pageSize=100", nil,
			map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": userID, "X-Test-Role": role})
		require.Equal(t, 200, lw.Code, lw.Body.String())
		var got listEnvelope
		require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &got))
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return got.Data.Total, len(got.Data.Items), repo.lastListDataScope
	}

	total, items, ds := list("7", "technician")
	assert.Equal(t, datascope.DataScopeOwnedOrAssigned, ds, "非管理角色必须下传收窄档")
	assert.Equal(t, 2, total, "technician 不应看到他人工单")
	assert.Equal(t, 2, items)

	total, _, ds = list("8", "end_user")
	assert.Equal(t, datascope.DataScopeOwnedOrAssigned, ds)
	assert.Equal(t, 1, total, "end_user 只应看到自己创建的那张")

	total, _, ds = list("1", "admin")
	assert.Equal(t, datascope.DataScopeAll, ds, "管理角色下传全量档")
	assert.Equal(t, 3, total, "admin 可见全租户工单")
}

// TestHandler_ListTickets_MissingUserIDFailsClosed 覆盖身份缺失：
// 非管理角色拿不到 user_id 时必须返回空集，而不是回落到租户全量。
func TestHandler_ListTickets_MissingUserIDFailsClosed(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 2)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets", nil,
		map[string]string{"X-Test-TenantID": "1", "X-Test-Role": "end_user"})
	require.Equal(t, 200, w.Code, w.Body.String())

	var got listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 0, got.Data.Total, "缺少 user_id 时行级权限必须 fail closed")

	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, datascope.DataScopeOwnedOrAssigned, repo.lastListDataScope)
	assert.Equal(t, 0, repo.lastListUserID)
}
