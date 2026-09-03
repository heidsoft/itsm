package change

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/handlers/common/datascope"
)

// setupTestHandler creates a test handler with in-memory repository
func setupTestHandler(t *testing.T) (*gin.Engine, *Handler, *mockRepository) {
	gin.SetMode(gin.TestMode)

	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	svc := NewService(repo, nil, logger, nil)
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
	// mu 保护所有 map 字段。状态转换的并发回归测试（-race）会并行调用
	// Get/UpdateStatusCAS/Update，无锁会触发 data race 并掩盖真实竞态。
	mu            sync.Mutex
	changes       map[int]*Change
	approvals     map[int]*ApprovalRecord
	riskAssess    map[int]*RiskAssessment
	chains        map[int][]*ApprovalChain
	nextID        int
	approverValid bool
	submitErr     error
	replaceErr    error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		changes:       make(map[int]*Change),
		approvals:     make(map[int]*ApprovalRecord),
		riskAssess:    make(map[int]*RiskAssessment),
		chains:        make(map[int][]*ApprovalChain),
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
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.changes[id]
	if !ok || c.TenantID != tenantID {
		return nil, http.ErrMissingFile
	}
	// P1-2：返回深拷贝，确保调用方对返回实体的修改不会污染仓储当前持久化状态。
	// 这样 Service.UpdateChange 中"读取当前状态 vs 传入请求状态"的差异对比才有意义。
	return cloneChange(c), nil
}

// cloneChange 返回 Change 实体的浅结构深拷贝（对指针字段与切片复制底层数据）。
func cloneChange(c *Change) *Change {
	if c == nil {
		return nil
	}
	cc := *c
	if c.AssigneeID != nil {
		v := *c.AssigneeID
		cc.AssigneeID = &v
	}
	if c.PlannedStartDate != nil {
		v := *c.PlannedStartDate
		cc.PlannedStartDate = &v
	}
	if c.PlannedEndDate != nil {
		v := *c.PlannedEndDate
		cc.PlannedEndDate = &v
	}
	if c.ActualStartDate != nil {
		v := *c.ActualStartDate
		cc.ActualStartDate = &v
	}
	if c.ActualEndDate != nil {
		v := *c.ActualEndDate
		cc.ActualEndDate = &v
	}
	if c.AffectedCIs != nil {
		cc.AffectedCIs = append([]string(nil), c.AffectedCIs...)
	}
	if c.RelatedTickets != nil {
		cc.RelatedTickets = append([]string(nil), c.RelatedTickets...)
	}
	return &cc
}

func (m *mockRepository) List(ctx context.Context, tenantID int, page, size int, status, search, riskLevel string, dataScope datascope.DataScope, currentUserID int) ([]*Change, int, error) {
	var result []*Change
	for _, c := range m.changes {
		if c.TenantID != tenantID {
			continue
		}
		if status != "" && c.Status != status {
			continue
		}
		if riskLevel != "" && c.RiskLevel != riskLevel {
			continue
		}
		result = append(result, c)
	}
	return result, len(result), nil
}

func (m *mockRepository) Update(ctx context.Context, c *Change) (*Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.UpdatedAt = time.Now()
	m.changes[c.ID] = c
	return c, nil
}

func (m *mockRepository) UpdateStatusCAS(ctx context.Context, id, tenantID int, expectedStatus, targetStatus string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.changes[id]
	if !ok || c.TenantID != tenantID || c.Status != expectedStatus {
		return false, nil
	}
	c.Status = targetStatus
	c.UpdatedAt = time.Now()
	return true, nil
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

func (m *mockRepository) SubmitForApproval(ctx context.Context, changeID, tenantID int, plan []ApprovalLevelPlan, comment string) error {
	if m.submitErr != nil {
		return m.submitErr
	}
	c, ok := m.changes[changeID]
	if !ok || c.TenantID != tenantID || c.Status != "draft" {
		return fmt.Errorf("change is not an editable draft")
	}
	c.Status = "pending"
	for _, lvl := range plan {
		for _, approverID := range lvl.ApproverIDs {
			record := &ApprovalRecord{
				ID:         m.nextID,
				ChangeID:   changeID,
				TenantID:   tenantID,
				ApproverID: approverID,
				Status:     "pending",
				CreatedAt:  time.Now(),
			}
			m.nextID++
			m.approvals[record.ID] = record
		}
	}
	return nil
}

func (m *mockRepository) CreateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.ID = m.nextID
	m.nextID++
	r.CreatedAt = time.Now()
	m.approvals[r.ID] = r
	return r, nil
}

func (m *mockRepository) UpdateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 与生产实现保持一致：只更新调用方传入的字段，其它列（如 ApproverID）
	// 必须保留。直接覆盖 m.approvals[r.ID] = r 会丢失 ApproverID，导致
	// 并发审批场景下后续请求读取记录时退化为 "user is not an approver"。
	existing, ok := m.approvals[r.ID]
	if !ok {
		m.approvals[r.ID] = r
		return r, nil
	}
	if r.Status != "" {
		existing.Status = r.Status
	}
	if r.Comment != nil {
		existing.Comment = r.Comment
	}
	if r.ApprovedAt != nil {
		existing.ApprovedAt = r.ApprovedAt
	}
	if r.Levels != nil {
		existing.Levels = r.Levels
	}
	if r.ApproverName != "" {
		existing.ApproverName = r.ApproverName
	}
	return existing, nil
}

func (m *mockRepository) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 阻断-1：必须返回深拷贝，否则调用方在 service.go:824 读取
	// h.Status / h.ApproverID 时与 UpdateApprovalRecord 中的
	// existing.Status = r.Status 写入存在数据竞争。生产实现的
	// EntRepository.GetApprovalHistory 走的是 Ent 的 SELECT 路径，
	// 读到的也是 *ent.ApprovalRecord，但 Ent 自身的查询结果有独立副本，
	// 不会与正在被 Update 修改的同一行共享可变状态。
	var result []*ApprovalRecord
	for _, a := range m.approvals {
		if a.ChangeID == changeID {
			result = append(result, cloneApprovalRecord(a))
		}
	}
	return result, nil
}

func cloneApprovalRecord(a *ApprovalRecord) *ApprovalRecord {
	if a == nil {
		return nil
	}
	cp := *a
	if a.ApproverID != 0 {
		// int 为值类型，不需要再深拷贝
	}
	if a.Comment != nil {
		c := *a.Comment
		cp.Comment = &c
	}
	if a.ApprovedAt != nil {
		t := *a.ApprovedAt
		cp.ApprovedAt = &t
	}
	if a.Levels != nil {
		cp.Levels = append([]int(nil), a.Levels...)
	}
	return &cp
}

func (m *mockRepository) GetApprovalChain(ctx context.Context, changeID int, tenantID int) ([]*ApprovalChain, error) {
	return m.chains[changeID], nil
}

func (m *mockRepository) DeleteApprovalChain(ctx context.Context, changeID int, tenantID int) error {
	return nil
}

func (m *mockRepository) ReplaceApprovalChain(ctx context.Context, changeID, tenantID int, chain []*ApprovalChain) error {
	if m.replaceErr != nil {
		return m.replaceErr
	}
	m.chains[changeID] = append([]*ApprovalChain(nil), chain...)
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

func (m *mockRepository) GetRiskAssessment(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error) {
	ra, ok := m.riskAssess[changeID]
	if !ok {
		return nil, nil
	}
	if ra.TenantID != 0 && ra.TenantID != tenantID {
		return nil, nil
	}
	return ra, nil
}

func (m *mockRepository) UpdateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	ra.UpdatedAt = time.Now()
	m.riskAssess[ra.ChangeID] = ra
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
	// Use dynamic ID instead of hardcoded 1
	c.ID = repo.nextID
	repo.changes[c.ID] = c
	repo.nextID++
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
			queryParams:    "?page=1&pageSize=10",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按状态筛选",
			queryParams:    "?status=draft",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按风险等级筛选(snake_case)",
			queryParams:    "?risk_level=low",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按风险等级筛选(camelCase)",
			queryParams:    "?riskLevel=low",
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

	svc := NewService(repo, nil, logger, nil)
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
			name:           "无效的变更ID应返回400",
			changeID:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "非正数ID应返回400",
			changeID:       "0",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "获取不存在的变更应返回404",
			changeID:       "999",
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
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
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
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
			name:     "更新不存在的变更应返回404",
			changeID: "999",
			request: dto.UpdateChangeRequest{
				Title: strPtr("不存在的变更"),
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
		{
			name:     "P1-2 draft 允许修改治理字段（RiskLevel/Type/Justification）",
			changeID: strconv.Itoa(change.ID),
			request: dto.UpdateChangeRequest{
				RiskLevel:     ptrChangeRisk(dto.ChangeRiskHigh),
				Type:          ptrChangeType(dto.ChangeTypeEmergency),
				Justification: strPtr("紧急变更理由"),
				ImpactScope:   ptrChangeImpact(dto.ChangeImpactHigh),
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

// TestChangeController_UpdateChange_P12_GovernanceGuard 验证已提审状态（非 draft）下
// PUT /changes/:id 不再能静默修改治理字段；运营类字段仍可编辑。
func TestChangeController_UpdateChange_P12_GovernanceGuard(t *testing.T) {
	r, _, repo := setupTestHandler(t)

	// 预置 pending 状态的变更（已提审，治理字段冻结）
	pendingChange := createTestChange(repo, 1, 1)
	pendingChange.Status = "pending"
	pendingChange.Type = "normal"
	pendingChange.RiskLevel = "low"
	pendingChange.ImpactScope = "low"
	pendingChange.Justification = "原理由"
	pendingChange.ImplementationPlan = "原实施计划"
	pendingChange.RollbackPlan = "原回滚计划"
	pendingChange.AffectedCIs = []string{"CI-001"}
	repo.changes[pendingChange.ID] = pendingChange

	t.Run("P1-2 pending 下改 RiskLevel 被拒", func(t *testing.T) {
		body := dto.UpdateChangeRequest{RiskLevel: ptrChangeRisk(dto.ChangeRiskHigh)}
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", "/api/v1/changes/"+strconv.Itoa(pendingChange.ID), bytes.NewBuffer(buf))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, common.InternalErrorCode, resp.Code)
		assert.Contains(t, resp.Message, "治理字段")
		assert.Contains(t, resp.Message, "draft")
	})

	t.Run("P1-2 pending 下改 Type+ImpactScope+Justification 组合被拒并命名字段", func(t *testing.T) {
		body := dto.UpdateChangeRequest{
			Type:          ptrChangeType(dto.ChangeTypeEmergency),
			ImpactScope:   ptrChangeImpact(dto.ChangeImpactHigh),
			Justification: strPtr("新理由绕过审批"),
		}
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", "/api/v1/changes/"+strconv.Itoa(pendingChange.ID), bytes.NewBuffer(buf))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		for _, want := range []string{"Type", "ImpactScope", "Justification"} {
			assert.Contains(t, resp.Message, want)
		}
	})

	t.Run("P1-2 pending 下改 ImplementationPlan/RollbackPlan/AffectedCIs 被拒", func(t *testing.T) {
		body := dto.UpdateChangeRequest{
			ImplementationPlan: strPtr("绕过审批改实施计划"),
			RollbackPlan:       strPtr("绕过审批改回滚计划"),
			AffectedCIs:        []string{"CI-999", "CI-008"},
		}
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", "/api/v1/changes/"+strconv.Itoa(pendingChange.ID), bytes.NewBuffer(buf))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Message, "ImplementationPlan")
		assert.Contains(t, resp.Message, "RollbackPlan")
		assert.Contains(t, resp.Message, "AffectedCIs")
	})

	t.Run("P1-2 approved 下仅改运营字段（Title/Description/Planned*/RelatedTickets）放行", func(t *testing.T) {
		approvedChange := createTestChange(repo, 1, 1)
		approvedChange.Status = "approved"
		approvedChange.Title = "原始标题"
		approvedChange.Description = "原始描述"
		approvedChange.RelatedTickets = nil
		repo.changes[approvedChange.ID] = approvedChange

		newDate := time.Now().AddDate(0, 0, 7)
		body := dto.UpdateChangeRequest{
			Title:            strPtr("运营调整后的标题"),
			Description:      strPtr("运营调整后的描述"),
			PlannedStartDate: timePtr(newDate),
			PlannedEndDate:   timePtr(newDate.AddDate(0, 0, 1)),
			RelatedTickets:   []string{"T-1", "T-2"},
		}
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", "/api/v1/changes/"+strconv.Itoa(approvedChange.ID), bytes.NewBuffer(buf))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, common.SuccessCode, resp.Code)
	})

	t.Run("P1-2 scheduled 下改治理字段+运营字段混合 被拒（治理优先守卫）", func(t *testing.T) {
		scheduledChange := createTestChange(repo, 1, 1)
		scheduledChange.Status = "scheduled"
		repo.changes[scheduledChange.ID] = scheduledChange

		body := dto.UpdateChangeRequest{
			Title:     strPtr("运营允许修改"),
			RiskLevel: ptrChangeRisk(dto.ChangeRiskMedium), // 治理字段
		}
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", "/api/v1/changes/"+strconv.Itoa(scheduledChange.ID), bytes.NewBuffer(buf))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp common.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Message, "RiskLevel")
	})
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
		expectedCode   int
	}{
		{
			name:           "成功删除变更",
			changeID:       strconv.Itoa(change.ID),
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "删除不存在的变更应返回500",
			changeID:       "999",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   common.InternalErrorCode,
		},
		{
			name:           "无效ID应返回400",
			changeID:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", "/api/v1/changes/"+tt.changeID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
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
	assert.Equal(t, common.SuccessCode, response.Code)

	// Verify response data
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

	// Use camelCase field name as per API contract
	assignReq := map[string]interface{}{
		"assigneeId": 2,
	}
	requestBody, err := json.Marshal(assignReq)
	require.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/v1/changes/"+strconv.Itoa(change.ID)+"/assign", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)

	// Verify assignment was successful
	if response.Code == common.SuccessCode {
		data := response.Data.(map[string]interface{})
		assert.Equal(t, float64(2), data["assigneeId"])
	}
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

func TestSubmitChangeAtomicFailureLeavesDraftUnchanged(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	repo.changes[1] = &Change{ID: 1, TenantID: 1, CreatedBy: 1, Status: "draft"}
	repo.submitErr = errors.New("injected transaction failure")
	svc := NewService(repo, nil, logger, nil)

	_, err := svc.SubmitChange(context.Background(), 1, 1, 1, &dto.SubmitChangeRequest{
		ApproverIDs: []int{1},
		Comment:     "review",
	})

	require.ErrorContains(t, err, "提交变更审批失败")
	require.Equal(t, "draft", repo.changes[1].Status)
	require.Empty(t, repo.approvals)
	require.Empty(t, repo.chains[1])
}

func TestConfigureWorkflowAtomicFailurePreservesOldChain(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	repo.changes[1] = &Change{ID: 1, TenantID: 1, CreatedBy: 1, Status: "draft"}
	oldChain := []*ApprovalChain{{ID: 10, ChangeID: 1, TenantID: 1, Level: 1, ApproverID: 1}}
	repo.chains[1] = oldChain
	repo.replaceErr = errors.New("injected transaction failure")
	svc := NewService(repo, nil, logger, nil)

	err := svc.ConfigureWorkflow(context.Background(), 1, 1, []*ApprovalChain{
		{Level: 1, ApproverID: 2},
	})

	require.ErrorContains(t, err, "failed to replace approval chain")
	require.Equal(t, oldChain, repo.chains[1])
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

func ptrChangeType(p dto.ChangeType) *dto.ChangeType {
	return &p
}

func ptrChangeImpact(p dto.ChangeImpact) *dto.ChangeImpact {
	return &p
}

func ptrChangeRisk(p dto.ChangeRisk) *dto.ChangeRisk {
	return &p
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// ===================== CMDB Impact Summary Helper Tests =====================

func TestRecommendRiskLevel(t *testing.T) {
	cases := []struct {
		name          string
		totalCIs      int
		criticalCIs   int
		highRiskDeps  int
		openIncidents int
		changeType    string
		want          string
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
