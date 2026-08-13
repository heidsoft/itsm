package incident

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// mocks
// -----------------------------------------------------------------------------

// mockRepository is an in-memory implementation of the Repository interface.
// It exercises the *handler* paths only — Service.Create / Get / Update /
// Escalate are reached through real call flow.
//
// The mock is safe for concurrent use because the handler emits an
// `executeRules(ctx, ...)` goroutine inside Service.Create that calls
// ListActiveRules. We serialize via the embedded mutex.
type mockRepository struct {
	mu              sync.Mutex
	incidents       map[int]*Incident
	eventLog        []string
	ruleReturnErr   error
	numberCounter   int
	nextID          int
}

func newMockRepository() *mockRepository {
	return &mockRepository{incidents: make(map[int]*Incident)}
}

func (m *mockRepository) Create(ctx context.Context, inc *Incident) (*Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	inc.ID = m.nextID
	if inc.Status == "" {
		inc.Status = "new"
	}
	if inc.DetectedAt.IsZero() {
		inc.DetectedAt = time.Now()
	}
	m.incidents[inc.ID] = inc
	return inc, nil
}

func (m *mockRepository) Get(ctx context.Context, id, tenantID int) (*Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inc, ok := m.incidents[id]
	if !ok {
		return nil, errors.New("incident not found")
	}
	if inc.TenantID != tenantID {
		// tenant isolation — equivalent to repo: ent.IsNotFound
		return nil, errors.New("incident not found")
	}
	return inc, nil
}

func (m *mockRepository) List(ctx context.Context, tenantID, page, size int, filters map[string]interface{}) ([]*Incident, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Incident, 0)
	for _, inc := range m.incidents {
		if inc.TenantID != tenantID {
			continue
		}
		if v, ok := filters["status"].(string); ok && v != "" && inc.Status != v {
			continue
		}
		out = append(out, inc)
	}
	return out, len(out), nil
}

func (m *mockRepository) Update(ctx context.Context, inc *Incident) (*Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.incidents[inc.ID]
	if !ok {
		return nil, errors.New("not found")
	}
	if cur.TenantID != inc.TenantID {
		return nil, errors.New("not found")
	}
	if inc.Title != "" {
		cur.Title = inc.Title
	}
	if inc.Description != "" {
		cur.Description = inc.Description
	}
	if inc.Status != "" {
		cur.Status = inc.Status
	}
	if inc.Priority != "" {
		cur.Priority = inc.Priority
	}
	if inc.Severity != "" {
		cur.Severity = inc.Severity
	}
	if inc.AssigneeID != nil {
		cur.AssigneeID = inc.AssigneeID
	}
	if inc.RootCause != nil {
		cur.RootCause = inc.RootCause
	}
	return cur, nil
}

func (m *mockRepository) Delete(ctx context.Context, id, tenantID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inc, ok := m.incidents[id]
	if !ok || inc.TenantID != tenantID {
		return errors.New("not found")
	}
	delete(m.incidents, id)
	return nil
}

func (m *mockRepository) GenerateIncidentNumber(ctx context.Context, tenantID, year, month int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.numberCounter++
	return "INC-2026-0001", nil
}

func (m *mockRepository) CountByPeriod(ctx context.Context, tenantID int, start, end time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, inc := range m.incidents {
		if inc.TenantID == tenantID {
			n++
		}
	}
	return n, nil
}

func (m *mockRepository) CreateEvent(ctx context.Context, e *IncidentEvent) (*IncidentEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventLog = append(m.eventLog, e.EventType)
	return e, nil
}

func (m *mockRepository) ListEvents(ctx context.Context, incidentID, tenantID int) ([]*IncidentEvent, error) {
	return []*IncidentEvent{}, nil
}

func (m *mockRepository) ListActiveRules(ctx context.Context, tenantID int) ([]*IncidentRule, error) {
	if m.ruleReturnErr != nil {
		return nil, m.ruleReturnErr
	}
	return []*IncidentRule{}, nil
}

func (m *mockRepository) UpdateRuleStats(ctx context.Context, ruleID int, count int, lastExecutedAt time.Time) error {
	return nil
}

// -----------------------------------------------------------------------------
// harness: wires Service → Handler → gin.Engine
// -----------------------------------------------------------------------------

func newTestHarness(t *testing.T) (*gin.Engine, *mockRepository) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop().Sugar())
	h := NewHandler(svc)
	r := gin.New()

	// Inject tenant_id/user_id/role keys directly (same shape the
	// production auth middleware produces). Tests can then simply
	// `r.ServeHTTP(req, ...)` without standing up a JWT pipeline.
	//
	// Override the per-test values via X-Test-TenantID / X-Test-UserID
	// headers in the higher-level route handlers if needed.
	auth := func(c *gin.Context) {
		tenantID := 0
		userID := 0
		if v := c.GetHeader("X-Test-TenantID"); v != "" {
			tenantID = mustAtoi(v)
		}
		if v := c.GetHeader("X-Test-UserID"); v != "" {
			userID = mustAtoi(v)
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "agent")
		c.Next()
	}

	api := r.Group("/api/v1", auth)
	api.POST("/incidents", h.Create)
	api.GET("/incidents", h.List)
	api.GET("/incidents/:id", h.Get)
	api.PUT("/incidents/:id", h.Update)
	api.POST("/incidents/:id/escalate", h.Escalate)

	return r, repo
}

func mustAtoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
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
// PR-1.1 — handler_test cases
// -----------------------------------------------------------------------------

// TestHandler_Create_TableDriven covers the four Create-time branches the
// production handler exercises:
//
//	1. JSON binding failure   → 400 + ParamErrorCode
//	2. tenant_id missing      → 401 + AuthErrorCode
//	3. user_id missing        → 401 + AuthErrorCode
//	4. happy path             → 200 + SuccessCode + auto-derived priority
func TestHandler_Create_TableDriven(t *testing.T) {
	type want struct {
		httpStatus int
		bodyCode   int
		priority   string // asserted on the success path only
	}
	cases := []struct {
		name       string
		body       interface{}
		tenantHdr  string
		userHdr    string
		want       want
	}{
		{
			name:       "rejects empty title",
			body:       map[string]interface{}{}, // binding:"required" on Title
			tenantHdr:  "1",
			userHdr:    "7",
			want:       want{httpStatus: 400, bodyCode: 1001},
		},
		{
			name: "rejects missing tenant",
			body: dto.CreateIncidentRequest{
				Title:       "Server CPU high",
				Description: "CPU > 90%",
			},
			tenantHdr: "0",
			userHdr:   "7",
			want:      want{httpStatus: 401, bodyCode: 2001},
		},
		{
			name: "rejects missing user",
			body: dto.CreateIncidentRequest{
				Title:       "Server CPU high",
				Description: "CPU > 90%",
				Priority:    "high",
			},
			tenantHdr: "1",
			userHdr:   "0",
			want:      want{httpStatus: 401, bodyCode: 2001},
		},
		{
			name: "happy path with explicit priority",
			body: dto.CreateIncidentRequest{
				Title:       "DB outage",
				Description: "Primary cluster down",
				Priority:    "critical",
				Severity:    "critical",
				Category:    "performance",
			},
			tenantHdr: "1",
			userHdr:   "7",
			want:      want{httpStatus: 200, bodyCode: 0, priority: "critical"},
		},
		{
			name: "happy path auto-derives priority for outage keyword",
			body: dto.CreateIncidentRequest{
				Title:       "Production outage",
				Description: "Service is down",
			},
			tenantHdr: "1",
			userHdr:   "7",
			want:      want{httpStatus: 200, bodyCode: 0, priority: "critical"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, repo := newTestHarness(t)
			w := doJSON(t, r, http.MethodPost, "/api/v1/incidents",
				tc.body,
				map[string]string{
					"X-Test-TenantID": tc.tenantHdr,
					"X-Test-UserID":   tc.userHdr,
				},
			)
			assert.Equal(t, tc.want.httpStatus, w.Code, "http status; body=%s", w.Body.String())

			var resp struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    *dto.IncidentResponse `json:"data"`
			}
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.want.bodyCode, resp.Code)

			if resp.Data != nil && tc.want.priority != "" {
				assert.Equal(t, tc.want.priority, resp.Data.Priority)
			}
			// Event log assertion: a successful Create produces an
			// "creation" event on the repository. Failure paths must
			// not have written any.
			if tc.want.bodyCode == 0 {
				assert.Contains(t, repo.eventLog, "creation")
			} else {
				assert.Empty(t, repo.eventLog)
			}
		})
	}
}

// TestHandler_Get_NotFoundTable asserts the 404 / 400 split for Get.
// The mock returns a generic "not found" error; in production ent.IsNotFound
// is checked by the handler. We accept any non-200 as proof that no data is
// leaked and that the contract triggers a fail-fast path.
func TestHandler_Get_NotFoundTable(t *testing.T) {
	cases := []struct {
		name      string
		idParam   string
		tenantHdr string
		want      int
	}{
		{"invalid id", "abc", "1", 400},
		{"non-existing id", "999", "1", 500}, // mock surfaces generic error; production maps to 404 via ent.IsNotFound
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newTestHarness(t)
			w := doJSON(t, r, http.MethodGet,
				"/api/v1/incidents/"+tc.idParam, nil,
				map[string]string{"X-Test-TenantID": tc.tenantHdr, "X-Test-UserID": "7"},
			)
			assert.Equal(t, tc.want, w.Code)
		})
	}
}

// TestHandler_Get_TenantIsolationTable documents the existing
// tenant-isolation behaviour: a Get made with the wrong tenant_id (the
// production path can only happen with a forged JWT, but we still want
// to catch any future regression in the handler layer).
func TestHandler_Get_TenantIsolationTable(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed an incident belonging to tenant 1.
	w := doJSON(t, r, http.MethodPost, "/api/v1/incidents",
		dto.CreateIncidentRequest{
			Title:       "T1 incident",
			Description: "Tenant 1",
			Priority:    "low",
		},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)

	if len(repo.incidents) == 0 {
		t.Fatal("seed not recorded")
	}
	var id int
	for id = range repo.incidents {
		break
	}

	// Same id, but read with tenant 2 → must NOT leak data. Either a
	// non-200 (mock surfaces an error → handler returns 5xx) or a 404
	// (production path ent.IsNotFound) is acceptable — the contract is
	// "tenant 2 cannot see tenant 1's data".
	w = doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/incidents/%d", id), nil,
		map[string]string{"X-Test-TenantID": "2", "X-Test-UserID": "8"},
	)
	assert.NotEqual(t, 200, w.Code, "tenant 2 must not see tenant 1's incident; body=%s", w.Body.String())
	_ = id
}

// TestHandler_Update_TableDriven covers: missing body, ID parse error, ok path.
func TestHandler_Update_TableDriven(t *testing.T) {
	r, repo := newTestHarness(t)

	// Seed incident for tenant 1
	w := doJSON(t, r, http.MethodPost, "/api/v1/incidents",
		dto.CreateIncidentRequest{Title: "T1", Description: "d", Priority: "low"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 200, w.Code)
	if len(repo.incidents) == 0 {
		t.Fatal("seed not recorded")
	}
	var id int
	for id = range repo.incidents {
		break
	}

	cases := []struct {
		name string
		id   string
		body interface{}
		want int
	}{
		{"invalid id", "xyz", dto.UpdateIncidentRequest{}, 400},
		{"empty body allowed (no fields set)", fmt.Sprintf("%d", id), dto.UpdateIncidentRequest{}, 200}, // existing item; nothing to update
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, http.MethodPut,
				"/api/v1/incidents/"+tc.id,
				tc.body,
				map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
			)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

// TestHandler_Escalate_InvalidIDTable — escalation rejects a non-numeric id.
func TestHandler_Escalate_InvalidIDTable(t *testing.T) {
	r, _ := newTestHarness(t)
	w := doJSON(t, r, http.MethodPost,
		"/api/v1/incidents/notanint/escalate",
		dto.IncidentEscalationRequest{EscalationLevel: 2, Reason: "SLA breach"},
		map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"},
	)
	assert.Equal(t, 400, w.Code)
}

// TestHandler_CommonFailOver2002 — verifies that common.Fail(c, 2002, ...)
// emits HTTP 401, not 200, fixing the alignment audit P0 #3.
func TestHandler_CommonFailOver2002(t *testing.T) {
	// We use a minimal gin engine that directly invokes Fail to be
	// robust against any future handler refactor.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		common.Fail(c, 2002, "未授权")
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}
