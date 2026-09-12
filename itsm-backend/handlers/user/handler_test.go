package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// mockUserService 实现 user.Service，仅测试用。
type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest, tenantID int) (*ent.User, error) {
	args := m.Called(ctx, req, tenantID)
	if u, ok := args.Get(0).(*ent.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserService) ListUsers(ctx context.Context, req *dto.ListUsersRequest, tenantID int) (*dto.PagedUsersResponse, error) {
	args := m.Called(ctx, req, tenantID)
	if r, ok := args.Get(0).(*dto.PagedUsersResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserService) GetUserByID(ctx context.Context, id int, tenantID int) (*ent.User, error) {
	args := m.Called(ctx, id, tenantID)
	if u, ok := args.Get(0).(*ent.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserService) UpdateUser(ctx context.Context, id int, req *dto.UpdateUserRequest, tenantID int) (*ent.User, error) {
	args := m.Called(ctx, id, req, tenantID)
	if u, ok := args.Get(0).(*ent.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserService) DeleteUser(ctx context.Context, id int, tenantID int) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *mockUserService) ChangeUserStatus(ctx context.Context, id int, active bool, currentUserID int, tenantID int) error {
	args := m.Called(ctx, id, active, currentUserID, tenantID)
	return args.Error(0)
}

func (m *mockUserService) ResetPassword(ctx context.Context, id int, newPassword string, tenantID int) error {
	args := m.Called(ctx, id, newPassword, tenantID)
	return args.Error(0)
}

func (m *mockUserService) GetUserStats(ctx context.Context, tenantID int) (*dto.UserStatsResponse, error) {
	args := m.Called(ctx, tenantID)
	if r, ok := args.Get(0).(*dto.UserStatsResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserService) BatchUpdateUsers(ctx context.Context, req *dto.BatchUpdateUsersRequest, tenantID int) error {
	args := m.Called(ctx, req, tenantID)
	return args.Error(0)
}

func (m *mockUserService) SearchUsers(ctx context.Context, req *dto.SearchUsersRequest, tenantID int) ([]*dto.UserDetailResponse, error) {
	args := m.Called(ctx, req, tenantID)
	if r, ok := args.Get(0).([]*dto.UserDetailResponse); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func newTestHandler(m *mockUserService) (*UserHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(m, zap.NewNop().Sugar())
	r := gin.New()
	r.POST("/api/v1/users", h.CreateUser)
	r.PUT("/api/v1/users/:id/password", h.ResetPassword)
	return h, r
}

func TestCreateUser_MissingTenant(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)

	// 复用 handler 方法直接构造 context（避免中间件差异）
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","name":"Alice","password":"password123456","role":"agent"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	// tenant_id 未设置 = 0
	h.CreateUser(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "租户信息缺失", resp.Message)
}

func TestCreateUser_RoleEscalationForbidden(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","name":"Alice","password":"password123456","role":"super_admin"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)
	c.Set("role", "agent") // 调用者只是 agent，不能分配 super_admin

	h.CreateUser(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Message, "无权限分配高于自身角色")
	m.AssertNotCalled(t, "CreateUser")
}

func TestCreateUser_Success(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)
	m.On("CreateUser", mock.Anything, mock.Anything, 1).Return(&ent.User{
		ID: 1, Username: "alice", Email: "a@b.com", Name: "Alice",
		TenantID: 1, Role: "agent", Active: true, CreatedAt: time.Now(),
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","name":"Alice","password":"password123456","role":"agent"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)
	c.Set("role", "admin")

	h.CreateUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "alice", data["username"])
	m.AssertExpectations(t)
}

func TestCreateUser_DuplicateBusinessError(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)
	m.On("CreateUser", mock.Anything, mock.Anything, 1).Return(nil, errors.New("用户名已存在"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","name":"Alice","password":"password123456","role":"agent"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)
	c.Set("role", "admin")

	h.CreateUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "请求参数错误", resp.Message)
}

func TestResetPassword_Success(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)
	m.On("ResetPassword", mock.Anything, 1, "newpass12345678", 1).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/1/password", bytes.NewBufferString(`{"newPassword":"newpass12345678"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)
	c.Set("user_id", 2)
	c.Set("role", "admin")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.ResetPassword(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

func TestResetPassword_InvalidID(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/abc/password", bytes.NewBufferString(`{"new_password":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	h.ResetPassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "ResetPassword")
}

func TestListUsers_AcceptsPageSize1000(t *testing.T) {
	// Fix for P1-5: groups 页面 Member Transfer 此前传 pageSize:500 被后端 max=200 拒，
	// 错误被 .catch 吞掉，Transfer 显示 0 用户。PageSize 改 max=1000 后应能通过。
	m := &mockUserService{}
	m.On("ListUsers", mock.Anything, mock.Anything, 1).Return(&dto.PagedUsersResponse{Users: []*dto.UserDetailResponse{}}, nil)
	h, _ := newTestHandler(m)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=1000", nil)
	c.Set("tenant_id", 1)

	h.ListUsers(c)

	assert.Equal(t, http.StatusOK, w.Code, "pageSize=1000 应被允许，body=%s", w.Body.String())
}

func TestListUsers_RejectsPageSize1001(t *testing.T) {
	m := &mockUserService{}
	h, _ := newTestHandler(m)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&pageSize=1001", nil)
	c.Set("tenant_id", 1)

	h.ListUsers(c)

	assert.NotEqual(t, http.StatusOK, w.Code, "pageSize=1001 必须被拒（max=1000）")
	m.AssertNotCalled(t, "ListUsers")
}
