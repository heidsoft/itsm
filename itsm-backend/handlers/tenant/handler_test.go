package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockTenantService 实现 tenant.Service，仅测试用。
type mockTenantService struct {
	mock.Mock
}

func (m *mockTenantService) CreateTenant(ctx context.Context, req *dto.CreateTenantRequest) (*ent.Tenant, error) {
	args := m.Called(ctx, req)
	if t, ok := args.Get(0).(*ent.Tenant); ok {
		return t, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTenantService) ListTenants(ctx context.Context, req *dto.ListTenantsRequest) ([]*ent.Tenant, int, error) {
	args := m.Called(ctx, req)
	if l, ok := args.Get(0).([]*ent.Tenant); ok {
		return l, args.Int(1), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *mockTenantService) TenantUsageStats(ctx context.Context, tenantIDs []int) (map[int]dto.TenantUsageStat, error) {
	args := m.Called(ctx, tenantIDs)
	if stats, ok := args.Get(0).(map[int]dto.TenantUsageStat); ok {
		return stats, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTenantService) GetTenant(ctx context.Context, tenantID int) (*ent.Tenant, error) {
	args := m.Called(ctx, tenantID)
	if t, ok := args.Get(0).(*ent.Tenant); ok {
		return t, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTenantService) UpdateTenant(ctx context.Context, tenantID int, req *dto.UpdateTenantRequest) (*ent.Tenant, error) {
	args := m.Called(ctx, tenantID, req)
	if t, ok := args.Get(0).(*ent.Tenant); ok {
		return t, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTenantService) UpdateTenantStatus(ctx context.Context, tenantID int, status string) error {
	args := m.Called(ctx, tenantID, status)
	return args.Error(0)
}

func (m *mockTenantService) DeleteTenant(ctx context.Context, tenantID int) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func newTenantHandler(m *mockTenantService) *Handler {
	gin.SetMode(gin.TestMode)
	return NewHandler(m, nil, zap.NewNop().Sugar())
}

func doJSON(h *Handler, method, path, body string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)
	c.Set("role", "super_admin")
	if method == http.MethodPost {
		// CreateTenant 无 path param
	} else {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
	}
	return w, c
}

func sampleTenant() *ent.Tenant {
	return &ent.Tenant{
		ID: 1, Name: "Acme", Code: "acme", Domain: "acme.com",
		Status: "active", Type: "standard", BillingEnabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestCreateTenant_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("CreateTenant", mock.Anything, mock.Anything).Return(sampleTenant(), nil)

	w, c := doJSON(h, http.MethodPost, "/api/v1/tenants", `{"name":"Acme","code":"acme","type":"standard"}`)
	h.CreateTenant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "acme", data["code"])
	m.AssertExpectations(t)
}

func TestCreateTenant_ValidationFailure(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	// 缺 name（required）→ 绑定失败 400
	w, c := doJSON(h, http.MethodPost, "/api/v1/tenants", `{"code":"acme","type":"standard"}`)
	h.CreateTenant(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "CreateTenant")
}

func TestCreateTenant_TypeInvalid(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	// type 不在 oneof 白名单 → 绑定失败 400
	w, c := doJSON(h, http.MethodPost, "/api/v1/tenants", `{"name":"Acme","code":"acme","type":"hacker"}`)
	h.CreateTenant(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "CreateTenant")
}

func TestCreateTenant_AllowsUnderscoreCode(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("CreateTenant", mock.Anything, mock.Anything).Return(sampleTenant(), nil)

	// 带下划线的合法编码不能被 alphanum 绑定拒绝，必须放行到 service 校验
	w, c := doJSON(h, http.MethodPost, "/api/v1/tenants", `{"name":"FinOps","code":"finops_001","type":"standard"}`)
	h.CreateTenant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertCalled(t, "CreateTenant", mock.Anything, mock.Anything)
}

func TestCreateTenant_RejectsInvalidCode(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	// 含空格的非法编码 → 400，且不调用 service
	w, c := doJSON(h, http.MethodPost, "/api/v1/tenants", `{"name":"Acme","code":"bad code","type":"standard"}`)
	h.CreateTenant(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "CreateTenant")
}

func TestListTenants_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("ListTenants", mock.Anything, mock.Anything).Return([]*ent.Tenant{sampleTenant()}, 1, nil)
	m.On("TenantUsageStats", mock.Anything, mock.Anything).
		Return(map[int]dto.TenantUsageStat{}, nil)

	w, c := doJSON(h, http.MethodGet, "/api/v1/tenants", "")
	h.ListTenants(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(1), data["total"])
	m.AssertExpectations(t)
}

// TestListTenants_IncludesUsageStats 锁缺陷 O8：
// 此前列表响应不含 userCount / ticketCount，前端 `|| 0` 把 undefined 显示成
// 「0 用户 0 工单」（实测 Default Tenant 有 3 用户 / 2 工单仍显示 0）。
func TestListTenants_IncludesUsageStats(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	tnt := sampleTenant()
	tnt.ID = 7
	m.On("ListTenants", mock.Anything, mock.Anything).Return([]*ent.Tenant{tnt}, 1, nil)
	m.On("TenantUsageStats", mock.Anything, []int{7}).
		Return(map[int]dto.TenantUsageStat{7: {UserCount: 3, TicketCount: 2}}, nil)

	w, c := doJSON(h, http.MethodGet, "/api/v1/tenants", "")
	h.ListTenants(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	items, ok := data["items"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, items, 1)
	item, ok := items[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(3), item["userCount"], "用户数必须是真实聚合值，不能是硬编码 0")
	assert.Equal(t, float64(2), item["ticketCount"], "工单数必须是真实聚合值，不能是硬编码 0")
	m.AssertExpectations(t)
}

// TestListTenants_UsageStatsFailureDoesNotFakeZero 锁「聚合失败不得伪造 0」：
// 统计接口报错时列表仍应成功返回，但两个字段必须缺失（前端显示「—」），
// 而不是变成 0 —— 否则又回到「0 用户 0 工单」的假读数。
func TestListTenants_UsageStatsFailureDoesNotFakeZero(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	tnt := sampleTenant()
	tnt.ID = 9
	m.On("ListTenants", mock.Anything, mock.Anything).Return([]*ent.Tenant{tnt}, 1, nil)
	m.On("TenantUsageStats", mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	w, c := doJSON(h, http.MethodGet, "/api/v1/tenants", "")
	h.ListTenants(c)

	assert.Equal(t, http.StatusOK, w.Code, "统计聚合失败不应让租户列表整体失败")
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	_, hasUserCount := item["userCount"]
	_, hasTicketCount := item["ticketCount"]
	assert.False(t, hasUserCount, "未聚合时 userCount 必须省略，不得输出 0")
	assert.False(t, hasTicketCount, "未聚合时 ticketCount 必须省略，不得输出 0")
	// 租户主体信息仍须完整，不能因统计失败而丢字段。
	assert.Equal(t, tnt.Name, item["name"])
	m.AssertExpectations(t)
}

// TestToTenantResponse_OmitsUsageByDefault 锁单实体映射不伪造统计值：
// 创建 / 详情 / 更新响应都没有做聚合，这三个端点若带 userCount=0 就是假读数。
func TestToTenantResponse_OmitsUsageByDefault(t *testing.T) {
	response := dto.ToTenantResponse(sampleTenant())
	assert.NotNil(t, response)
	assert.Nil(t, response.UserCount)
	assert.Nil(t, response.TicketCount)

	payload, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.NotContains(t, string(payload), `"userCount"`)
	assert.NotContains(t, string(payload), `"ticketCount"`)
}

func TestUpdateTenantStatus_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("GetTenant", mock.Anything, 1).Return(sampleTenant(), nil)
	m.On("UpdateTenantStatus", mock.Anything, 1, "suspended").Return(nil)

	w, c := doJSON(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"suspended"}`)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestUpdateTenantStatus_BlocksSystemTenant(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	sysTenant := sampleTenant()
	sysTenant.Code = protectedSystemTenantCode
	m.On("GetTenant", mock.Anything, 1).Return(sysTenant, nil)

	w, c := doJSON(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"suspended"}`)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.ConflictCode, resp.Code)
	m.AssertNotCalled(t, "UpdateTenantStatus")
}

func TestUpdateTenantStatus_AllowsActivateSystemTenant(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	sysTenant := sampleTenant()
	sysTenant.Code = protectedSystemTenantCode
	m.On("GetTenant", mock.Anything, 1).Return(sysTenant, nil)
	m.On("UpdateTenantStatus", mock.Anything, 1, "active").Return(nil)

	w, c := doJSON(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"active"}`)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestUpdateTenantStatus_MissingStatus(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	w, c := doJSON(h, http.MethodPut, "/api/v1/tenants/1/status", `{"foo":"bar"}`)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "UpdateTenantStatus")
}

func TestGetTenant_InvalidID(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tenants/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.GetTenant(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "GetTenant")
}

func TestDeleteTenant_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("GetTenant", mock.Anything, 1).Return(sampleTenant(), nil)
	m.On("DeleteTenant", mock.Anything, 1).Return(nil)

	w, c := doJSON(h, http.MethodDelete, "/api/v1/tenants/1", "")
	h.DeleteTenant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestDeleteTenant_BlocksSystemTenant(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	sysTenant := sampleTenant()
	sysTenant.Code = protectedSystemTenantCode
	m.On("GetTenant", mock.Anything, 1).Return(sysTenant, nil)

	w, c := doJSON(h, http.MethodDelete, "/api/v1/tenants/1", "")
	h.DeleteTenant(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	m.AssertNotCalled(t, "DeleteTenant")
}
