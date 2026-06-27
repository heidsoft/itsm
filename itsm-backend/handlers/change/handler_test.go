package change

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
)

// setupTestHandler creates a test handler with in-memory repository
func setupTestHandler(t *testing.T) (*gin.Engine, *Handler, *mockRepository) {
	gin.SetMode(gin.TestMode)

	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	svc := NewService(repo, nil, logger)
	handler := NewHandler(svc)

	r := gin.New()
	r.Use(gin.Recovery())

	// Add auth middleware mock
	r.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Set("tenant_id", 1)
		c.Next()
	})

	// Register routes
	r.GET("/api/v1/changes", handler.ListChanges)
	r.POST("/api/v1/changes", handler.CreateChange)
	r.GET("/api/v1/changes/:id", handler.GetChange)
	r.PUT("/api/v1/changes/:id", handler.UpdateChange)
	r.DELETE("/api/v1/changes/:id", handler.DeleteChange)
	r.GET("/api/v1/changes/stats", handler.GetStats)
	r.POST("/api/v1/changes/:id/submit", handler.SubmitChange)
	r.POST("/api/v1/changes/:id/assign", handler.AssignChange)
	r.GET("/api/v1/changes/:id/approval-summary", handler.GetApprovalSummary)
	r.GET("/api/v1/changes/:id/risk-assessment", handler.GetRiskAssessment)
	r.GET("/api/v1/changes/:id/cmdb-impact", handler.GetCMDBImpactSummary)

	return r, handler, repo
}

// mockRepository implements Repository interface for testing
type mockRepository struct {
	changes       map[int]*Change
	approvals     map[int]*ApprovalRecord
	riskAssess    map[int]*RiskAssessment
	nextID        int
	approverValid bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		changes:       make(map[int]*Change),
		approvals:     make(map[int]*ApprovalRecord),
		riskAssess:    make(map[int]*RiskAssessment),
		nextID:        1,
		approverValid: true,
	}
}

func (m *mockRepository) Create(ctx context.Context, c *Change) (*Change, error) {
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	m.changes[c.ID] = c
	return c, nil
}

func (m *mockRepository) Get(ctx context.Context, id int, tenantID int) (*Change, error) {
	c, ok := m.changes[id]
	if !ok || c.TenantID != tenantID {
		return nil, http.ErrMissingFile
	}
	return c, nil
}

func (m *mockRepository) List(ctx context.Context, tenantID int, page, size int, status, search string) ([]*Change, int, error) {
	var result []*Change
	for _, c := range m.changes {
		if c.TenantID == tenantID {
			if status == "" || c.Status == status {
				result = append(result, c)
			}
		}
	}
	return result, len(result), nil
}

func (m *mockRepository) Update(ctx context.Context, c *Change) (*Change, error) {
	c.UpdatedAt = time.Now()
	m.changes[c.ID] = c
	return c, nil
}

func (m *mockRepository) Delete(ctx context.Context, id int, tenantID int) error {
	c, ok := m.changes[id]
	if !ok || c.TenantID != tenantID {
		return http.ErrMissingFile
	}
	delete(m.changes, id)
	return nil
}

func (m *mockRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	stats := &Stats{}
	for _, c := range m.changes {
		if c.TenantID == tenantID {
			stats.Total++
			switch c.Status {
			case "pending":
				stats.Pending++
			case "approved":
				stats.Approved++
			case "in_progress":
				stats.InProgress++
			case "completed":
				stats.Completed++
			case "rolled_back":
				stats.RolledBack++
			case "rejected":
				stats.Rejected++
			case "cancelled":
				stats.Cancelled++
			}
		}
	}
	return stats, nil
}

func (m *mockRepository) CreateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error) {
	r.ID = m.nextID
	m.nextID++
	r.CreatedAt = time.Now()
	m.approvals[r.ID] = r
	return r, nil
}

func (m *mockRepository) UpdateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error) {
	m.approvals[r.ID] = r
	return r, nil
}

func (m *mockRepository) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	var result []*ApprovalRecord
	for _, a := range m.approvals {
		if a.ChangeID == changeID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateApprovalChain(ctx context.Context, chain []*ApprovalChain) error {
	return nil
}

func (m *mockRepository) GetApprovalChain(ctx context.Context, changeID int) ([]*ApprovalChain, error) {
	return []*ApprovalChain{}, nil
}

func (m *mockRepository) DeleteApprovalChain(ctx context.Context, changeID int) error {
	return nil
}

func (m *mockRepository) CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	ra.ID = m.nextID
	m.nextID++
	ra.CreatedAt = time.Now()
	ra.UpdatedAt = time.Now()
	m.riskAssess[ra.ChangeID] = ra
	return ra, nil
}

func (m *mockRepository) GetRiskAssessment(ctx context.Context, changeID int) (*RiskAssessment, error) {
	ra, ok := m.riskAssess[changeID]
	if !ok {
		return nil, nil
	}
	return ra, nil
}

func (m *mockRepository) ValidateApproverBelongsToTenant(ctx context.Context, approverID, tenantID int) (bool, error) {
	return m.approverValid, nil
}

func (m *mockRepository) ListByDateRange(ctx context.Context, tenantID int, startDate, endDate, status string) ([]*Change, error) {
	var result []*Change
	for _, c := range m.changes {
		if c.TenantID == tenantID {
			if status == "" || c.Status == status {
				result = append(result, c)
			}
		}
	}
	return result, nil
}

// Helper function to create test change
func createTestChange(repo *mockRepository, tenantID, userID int) *Change {
	c := &Change{
		Title:         "Test Change",
		Description:   "Test Description",
		Justification: "Test Justification",
		Type:          "normal",
		Status:        "draft",
		Priority:      "medium",
		ImpactScope:   "low",
		RiskLevel:     "low",
		CreatedBy:     userID,
		TenantID:      tenantID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.changes[1] = c
	repo.nextID = 2
	return c
}

// TestChangeController_ListChanges tests GET /api/v1/changes
func TestChangeController_ListChanges(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	createTestChange(repo, 1, 1)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取变更列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "带分页参数",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按状态筛选",
			queryParams:    "?status=draft",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/changes"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)

			if response.Code == common.SuccessCode {
				data := response.Data.(map[string]interface{})
				assert.Contains(t, data, "changes")
				assert.Contains(t, data, "total")
			}
		})
	}
}

func TestChangeService_GetCMDBImpactSummary_WithoutEntClient(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	createTestChange(repo, 1, 1)

	svc := NewService(repo, nil, logger)
	_, err := svc.GetCMDBImpactSummary(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CMDB impact summary unavailable")
}

// TestChangeController_CreateChange tests POST /api/v1/changes
func TestChangeController_CreateChange(t *testing.T) {
	r, _, _ := setupTestHandler(t)

	tests := []struct {
		name           string
		request        dto.CreateChangeRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "成功创建变更",
			request: dto.CreateChangeRequest{
				Title:         "新变更请求",
				Description:   "变更描述",
				Justification: "变更理由",
				Type:          "normal",
				Priority:      "medium",
				ImpactScope:   "low",
				RiskLevel:     "low",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "带计划时间的变更",
			request: dto.CreateChangeRequest{
				Title:              "计划变更",
				Description:        "带计划时间",
				Justification:      "理由",
				Type:               "standard",
				Priority:           "high",
				ImpactScope:        "medium",
				RiskLevel:          "medium",
				ImplementationPlan: "实施计划",
				RollbackPlan:       "回滚计划",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("POST", "/api/v1/changes", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)

			// Verify the change was created
			if response.Code == common.SuccessCode && tt.request.Title != "" {
				data := response.Data.(map[string]interface{})
				assert.Equal(t, tt.request.Title, data["title"])
				assert.Equal(t, "draft", data["status"]) // Default status
			}
		})
	}
}

// TestChangeController_GetChange tests GET /api/v1/changes/:id
func TestChangeController_GetChange(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	tests := []struct {
		name           string
		changeID       string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取变更",
			changeID:       strconv.Itoa(change.ID),
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "获取不存在的变更",
			changeID:       "999",
			expectedStatus: http.StatusOK, // Handler returns 200 with error message
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "无效的变更ID",
			changeID:       "invalid",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/changes/"+tt.changeID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
		})
	}
}

// TestChangeController_GetChange_NotFound tests 404 scenario
func TestChangeController_GetChange_NotFound(t *testing.T) {
	r, _, _ := setupTestHandler(t)

	req, _ := http.NewRequest("GET", "/api/v1/changes/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The handler returns 200 with error message for not found
	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	// For not found, the handler uses common.Fail with 404 status
	assert.Contains(t, response.Message, "not found")
}

// TestChangeController_UpdateChange tests PUT /api/v1/changes/:id
func TestChangeController_UpdateChange(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	tests := []struct {
		name           string
		changeID       string
		request        dto.UpdateChangeRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name:     "成功更新变更",
			changeID: strconv.Itoa(change.ID),
			request: dto.UpdateChangeRequest{
				Title:       strPtr("更新后的标题"),
				Description: strPtr("更新后的描述"),
				Priority:    ptrChangePriority(dto.ChangePriorityHigh),
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:     "更新不存在的变更",
			changeID: "999",
			request: dto.UpdateChangeRequest{
				Title: strPtr("不存在的变更"),
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("PUT", "/api/v1/changes/"+tt.changeID, bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestChangeController_DeleteChange tests DELETE /api/v1/changes/:id
func TestChangeController_DeleteChange(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	tests := []struct {
		name           string
		changeID       string
		expectedStatus int
	}{
		{
			name:           "成功删除变更",
			changeID:       strconv.Itoa(change.ID),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "删除不存在的变更",
			changeID:       "999",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", "/api/v1/changes/"+tt.changeID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestChangeController_GetStats tests GET /api/v1/changes/stats
func TestChangeController_GetStats(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data with different statuses
	for i, status := range []string{"draft", "pending", "approved", "in_progress", "completed"} {
		c := &Change{
			ID:        i + 1,
			Title:     "Change " + strconv.Itoa(i),
			Status:    status,
			TenantID:  1,
			CreatedBy: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		repo.changes[c.ID] = c
	}

	req, _ := http.NewRequest("GET", "/api/v1/changes/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	// Stats struct uses camelCase/lowercase JSON tags
	assert.Contains(t, data, "total")
}

// TestChangeController_SubmitChange tests POST /api/v1/changes/:id/submit
func TestChangeController_SubmitChange(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data - change must be in draft status
	change := createTestChange(repo, 1, 1)

	req := dto.SubmitChangeRequest{
		ApproverIDs: []int{2, 3},
		Comment:     "请审批",
	}
	requestBody, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, _ := http.NewRequest("POST", "/api/v1/changes/"+strconv.Itoa(change.ID)+"/submit", bytes.NewBuffer(requestBody))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response
	if response.Code == common.SuccessCode {
		data := response.Data.(map[string]interface{})
		assert.Equal(t, "pending", data["status"])
	}
}

// TestChangeController_AssignChange tests POST /api/v1/changes/:id/assign
func TestChangeController_AssignChange(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	assignReq := map[string]interface{}{
		"assignee_id": 2,
	}
	requestBody, err := json.Marshal(assignReq)
	require.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/v1/changes/"+strconv.Itoa(change.ID)+"/assign", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestChangeController_GetApprovalSummary tests GET /api/v1/changes/:id/approval-summary
func TestChangeController_GetApprovalSummary(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	req, _ := http.NewRequest("GET", "/api/v1/changes/"+strconv.Itoa(change.ID)+"/approval-summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// TestChangeController_GetRiskAssessment tests GET /api/v1/changes/:id/risk-assessment
func TestChangeController_GetRiskAssessment(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// Create test data
	change := createTestChange(repo, 1, 1)

	req, _ := http.NewRequest("GET", "/api/v1/changes/"+strconv.Itoa(change.ID)+"/risk-assessment", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func ptrChangePriority(p dto.ChangePriority) *dto.ChangePriority {
	return &p
}

// ===================== CMDB Impact Summary Helper Tests =====================

func TestRecommendRiskLevel(t *testing.T) {
	cases := []struct {
		name             string
		totalCIs         int
		criticalCIs      int
		highRiskDeps     int
		openIncidents    int
		changeType       string
		want             string
	}{
		{"emergency overrides everything", 1, 0, 0, 0, "emergency", "high"},
		{"critical CI wins", 1, 1, 0, 0, "normal", "high"},
		{"high risk deps >= 4", 3, 0, 4, 0, "normal", "high"},
		{"open incidents >= 2", 3, 0, 0, 2, "normal", "high"},
		{"medium: 5+ CIs", 5, 0, 0, 0, "normal", "medium"},
		{"medium: 1 high risk dep", 2, 0, 1, 0, "normal", "medium"},
		{"medium: 1 open incident", 2, 0, 0, 1, "normal", "medium"},
		{"low: nothing matches", 2, 0, 0, 0, "normal", "low"},
		{"low: 0 CI", 0, 0, 0, 0, "normal", "low"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := recommendRiskLevel(tc.totalCIs, tc.criticalCIs, tc.highRiskDeps, tc.openIncidents, tc.changeType)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRecommendImpactScope(t *testing.T) {
	cases := []struct {
		name         string
		totalCIs     int
		criticalCIs  int
		highRiskDeps int
		want         string
	}{
		{"critical CI triggers high", 1, 1, 0, "high"},
		{"5+ CIs triggers high", 5, 0, 0, "high"},
		{"3+ high risk deps triggers high", 2, 0, 3, "high"},
		{"2 CIs triggers medium", 2, 0, 0, "medium"},
		{"1 high risk dep triggers medium", 1, 0, 1, "medium"},
		{"nothing triggers low", 1, 0, 0, "low"},
		{"empty triggers low", 0, 0, 0, "low"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := recommendImpactScope(tc.totalCIs, tc.criticalCIs, tc.highRiskDeps)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildWorkflowHints(t *testing.T) {
	t.Run("no CIs, not emergency", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 0}
		hints := buildWorkflowHints(summary, "normal")
		assert.Contains(t, hints, "补充受影响 CI 后再发起审批，以便自动执行风险分流。")
		assert.NotContains(t, hints, "紧急变更建议启用快速审批路径，并在实施后自动创建 PIR 任务。")
	})

	t.Run("critical CI triggers CAB hint", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 1, CriticalCICount: 1}
		hints := buildWorkflowHints(summary, "normal")
		assert.Contains(t, hints, "命中关键 CI，建议走 CAB 审批并校验变更窗口。")
	})

	t.Run("open incidents trigger conflict check hint", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 1, OpenIncidentCount: 1}
		hints := buildWorkflowHints(summary, "normal")
		assert.Contains(t, hints, "受影响 CI 当前存在未关闭事件，建议先做冲突检查和实施前健康确认。")
	})

	t.Run("high risk deps trigger rollback drill hint", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 1, HighRiskDependencyCount: 1}
		hints := buildWorkflowHints(summary, "normal")
		assert.Contains(t, hints, "存在高风险依赖，建议在工作流中增加影响确认和回滚演练节点。")
	})

	t.Run("emergency triggers fast track hint", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 1}
		hints := buildWorkflowHints(summary, "emergency")
		assert.Contains(t, hints, "紧急变更建议启用快速审批路径，并在实施后自动创建 PIR 任务。")
	})

	t.Run("requires backout plan triggers integrity hint", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{TotalAffectedCIs: 1, RequiresBackoutPlan: true}
		hints := buildWorkflowHints(summary, "normal")
		assert.Contains(t, hints, "建议在提交流程前强制校验回滚计划与实施计划完整性。")
	})

	t.Run("combined all triggers all hints", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{
			TotalAffectedCIs:        2,
			CriticalCICount:         1,
			HighRiskDependencyCount: 2,
			OpenIncidentCount:       1,
			RequiresBackoutPlan:     true,
		}
		hints := buildWorkflowHints(summary, "emergency")
		// 5 个触发条件（除 TotalAffectedCIs==0 分支）：critical + open incident + high risk dep + emergency + backout plan
		assert.Len(t, hints, 5)
	})
}

func TestInferITILPractices(t *testing.T) {
	t.Run("all triggers all 4 practices", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{
			CriticalCICount:         1,
			HighRiskDependencyCount: 1,
			OpenIncidentCount:       1,
			RequiresCAB:             true,
		}
		got := inferITILPractices(summary)
		assert.ElementsMatch(t, []string{
			"incident_management",
			"risk_management",
			"change_enablement",
			"monitoring_and_event_management",
		}, got)
	})

	t.Run("only incident triggers 1 practice", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{OpenIncidentCount: 1}
		got := inferITILPractices(summary)
		assert.Equal(t, []string{"incident_management"}, got)
	})

	t.Run("nothing triggers empty", func(t *testing.T) {
		summary := &dto.ChangeCMDBImpactSummary{}
		got := inferITILPractices(summary)
		assert.Empty(t, got)
	})
}
