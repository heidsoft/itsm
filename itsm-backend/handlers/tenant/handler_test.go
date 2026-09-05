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
	return NewHandler(m, zap.NewNop().Sugar())
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

func TestListTenants_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("ListTenants", mock.Anything, mock.Anything).Return([]*ent.Tenant{sampleTenant()}, 1, nil)

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

func TestUpdateTenantStatus_Success(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("UpdateTenantStatus", mock.Anything, 1, "suspended").Return(nil)

	w, c := doJSON(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"suspended"}`)
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
	m.On("DeleteTenant", mock.Anything, 1).Return(nil)

	w, c := doJSON(h, http.MethodDelete, "/api/v1/tenants/1", "")
	h.DeleteTenant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}
