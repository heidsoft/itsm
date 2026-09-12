package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockCloudService 实现 cloud.Service（15 方法），仅测试用到的方法有行为。
type mockCloudService struct{ mock.Mock }

func (m *mockCloudService) CreateCloudAccount(ctx context.Context, tenantID int, req *dto.CreateCloudAccountRequest) (*ent.CloudAccount, error) {
	args := m.Called(ctx, tenantID, req)
	if r, ok := args.Get(0).(*ent.CloudAccount); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) GetCloudAccount(ctx context.Context, tenantID, id int) (*ent.CloudAccount, error) {
	args := m.Called(ctx, tenantID, id)
	if r, ok := args.Get(0).(*ent.CloudAccount); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) UpdateCloudAccount(ctx context.Context, tenantID, id int, req *dto.UpdateCloudAccountRequest) (*ent.CloudAccount, error) {
	args := m.Called(ctx, tenantID, id, req)
	if r, ok := args.Get(0).(*ent.CloudAccount); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) DeleteCloudAccount(ctx context.Context, tenantID, id int) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *mockCloudService) ListCloudAccounts(ctx context.Context, tenantID int, req *dto.ListCloudAccountsRequest) ([]*ent.CloudAccount, int, error) {
	args := m.Called(ctx, tenantID, req)
	if l, ok := args.Get(0).([]*ent.CloudAccount); ok {
		return l, args.Int(1), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *mockCloudService) CreateCloudService(ctx context.Context, tenantID int, req *dto.CreateCloudServiceRequest) (*ent.CloudService, error) {
	args := m.Called(ctx, tenantID, req)
	if r, ok := args.Get(0).(*ent.CloudService); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) GetCloudService(ctx context.Context, tenantID, id int) (*ent.CloudService, error) {
	args := m.Called(ctx, tenantID, id)
	if r, ok := args.Get(0).(*ent.CloudService); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) UpdateCloudService(ctx context.Context, tenantID, id int, req *dto.UpdateCloudServiceRequest) (*ent.CloudService, error) {
	args := m.Called(ctx, tenantID, id, req)
	if r, ok := args.Get(0).(*ent.CloudService); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) DeleteCloudService(ctx context.Context, tenantID, id int) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *mockCloudService) ListCloudServices(ctx context.Context, tenantID int, req *dto.ListCloudServicesRequest) ([]*ent.CloudService, int, error) {
	args := m.Called(ctx, tenantID, req)
	if l, ok := args.Get(0).([]*ent.CloudService); ok {
		return l, args.Int(1), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *mockCloudService) CreateCloudResource(ctx context.Context, tenantID int, req *dto.CreateCloudResourceRequest) (*ent.CloudResource, error) {
	args := m.Called(ctx, tenantID, req)
	if r, ok := args.Get(0).(*ent.CloudResource); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) GetCloudResource(ctx context.Context, tenantID, id int) (*ent.CloudResource, error) {
	args := m.Called(ctx, tenantID, id)
	if r, ok := args.Get(0).(*ent.CloudResource); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) UpdateCloudResource(ctx context.Context, tenantID, id int, req *dto.UpdateCloudResourceRequest) (*ent.CloudResource, error) {
	args := m.Called(ctx, tenantID, id, req)
	if r, ok := args.Get(0).(*ent.CloudResource); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCloudService) DeleteCloudResource(ctx context.Context, tenantID, id int) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *mockCloudService) ListCloudResources(ctx context.Context, tenantID int, req *dto.ListCloudResourcesRequest) ([]*ent.CloudResource, int, error) {
	args := m.Called(ctx, tenantID, req)
	if l, ok := args.Get(0).([]*ent.CloudResource); ok {
		return l, args.Int(1), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func newCloudTestHandler(svc *mockCloudService) *Handler {
	gin.SetMode(gin.TestMode)
	return NewHandler(svc, zap.NewNop().Sugar())
}

func cloudCtx(method, path, body string, withTenant bool) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body != "" {
		c.Request = httptest.NewRequest(method, path, nil)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.Body = io.NopCloser(bytes.NewBufferString(body))
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	if withTenant {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 5})
	}
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	return w, c
}

func TestCloudCreateAccount_Success(t *testing.T) {
	m := &mockCloudService{}
	h := newCloudTestHandler(m)
	m.On("CreateCloudAccount", mock.Anything, 5, mock.Anything).
		Return(&ent.CloudAccount{ID: 1, Provider: "aliyun", AccountID: "acc-1", AccountName: "主账号"}, nil)

	w, c := cloudCtx(http.MethodPost, "/api/v1/cloud/accounts",
		`{"provider":"aliyun","accountId":"acc-1","accountName":"主账号"}`, true)
	h.CreateCloudAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	m.AssertExpectations(t)
}

// 关键回归：租户缺失必须 401 fail-closed（历史为裸 key + 0 静默继续）。
func TestCloudCreateAccount_MissingTenant_401(t *testing.T) {
	m := &mockCloudService{}
	h := newCloudTestHandler(m)

	w, c := cloudCtx(http.MethodPost, "/api/v1/cloud/accounts",
		`{"provider":"aliyun","accountId":"acc-1","accountName":"主账号"}`, false)
	h.CreateCloudAccount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.AuthFailedCode, resp.Code)
	m.AssertNotCalled(t, "CreateCloudAccount")
}

func TestCloudCreateAccount_ProviderNotInEnum_400(t *testing.T) {
	m := &mockCloudService{}
	h := newCloudTestHandler(m)

	// oneof 白名单外的 provider → binding 失败
	w, c := cloudCtx(http.MethodPost, "/api/v1/cloud/accounts",
		`{"provider":"gcp","accountId":"acc-1","accountName":"主账号"}`, true)
	h.CreateCloudAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "CreateCloudAccount")
}

func TestCloudListAccounts_Success(t *testing.T) {
	m := &mockCloudService{}
	h := newCloudTestHandler(m)
	m.On("ListCloudAccounts", mock.Anything, 5, mock.Anything).
		Return([]*ent.CloudAccount{{ID: 1, Provider: "tencent"}}, 1, nil)

	w, c := cloudCtx(http.MethodGet, "/api/v1/cloud/accounts", "", true)
	h.ListCloudAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	assert.Contains(t, w.Body.String(), "total")
	m.AssertExpectations(t)
}

func TestCloudDeleteAccount_ServiceError_500(t *testing.T) {
	m := &mockCloudService{}
	h := newCloudTestHandler(m)
	m.On("DeleteCloudAccount", mock.Anything, 5, 1).Return(assert.AnError)

	w, c := cloudCtx(http.MethodDelete, "/api/v1/cloud/accounts/1", "", true)
	h.DeleteCloudAccount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	m.AssertExpectations(t)
}
