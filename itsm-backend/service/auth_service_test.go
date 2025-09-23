package service

import (
	"context"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login(t *testing.T) {
	// 创建测试数据库客户端
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 创建测试logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建AuthService实例
	authService := &AuthService{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
	}

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name           string
		request        *dto.LoginRequest
		expectedError  bool
		expectedUserID int
	}{
		{
			name: "成功登录",
			request: &dto.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectedError:  false,
			expectedUserID: testUser.ID,
		},
		{
			name: "用户名不存在",
			request: &dto.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			expectedError: true,
		},
		{
			name: "密码错误",
			request: &dto.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectedError: true,
		},
		{
			name: "空用户名",
			request: &dto.LoginRequest{
				Username: "",
				Password: "password123",
			},
			expectedError: true,
		},
		{
			name: "空密码",
			request: &dto.LoginRequest{
				Username: "testuser",
				Password: "",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := authService.Login(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, tt.expectedUserID, response.User.ID)
				assert.Equal(t, "testuser", response.User.Username)
				assert.Equal(t, "test@example.com", response.User.Email)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	authService := &AuthService{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 先登录获取refresh token
	loginReq := &dto.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	loginResp, err := authService.Login(ctx, loginReq)
	require.NoError(t, err)

	tests := []struct {
		name          string
		refreshToken  string
		expectedError bool
	}{
		{
			name:          "有效的refresh token",
			refreshToken:  loginResp.RefreshToken,
			expectedError: false,
		},
		{
			name:          "无效的refresh token",
			refreshToken:  "invalid-token",
			expectedError: true,
		},
		{
			name:          "空refresh token",
			refreshToken:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &dto.RefreshTokenRequest{
				RefreshToken: tt.refreshToken,
			}

			response, err := authService.RefreshToken(ctx, request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
			}
		})
	}
}

func TestAuthService_GetUserTenants(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	authService := &AuthService{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
	}

	ctx := context.Background()

	// 创建多个测试租户
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole("user").
		SetStatus("active").
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name           string
		userID         int
		expectedError  bool
		expectedCount  int
	}{
		{
			name:          "获取用户租户列表",
			userID:        testUser.ID,
			expectedError: false,
			expectedCount: 1, // 用户只属于一个租户
		},
		{
			name:          "不存在的用户",
			userID:        99999,
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := authService.GetUserTenants(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Tenants, tt.expectedCount)
				if tt.expectedCount > 0 {
					assert.Equal(t, tenant1.ID, response.Tenants[0].ID)
					assert.Equal(t, "Tenant 1", response.Tenants[0].Name)
				}
			}
		})
	}
}

func TestAuthService_GetUserInfo(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	authService := &AuthService{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
	}

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole("admin").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		SetDisplayName("Test User").
		SetPhone("13800138000").
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		expectedError bool
	}{
		{
			name:          "获取用户信息成功",
			userID:        testUser.ID,
			expectedError: false,
		},
		{
			name:          "用户不存在",
			userID:        99999,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := authService.GetUserInfo(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, testUser.ID, response.ID)
				assert.Equal(t, "testuser", response.Username)
				assert.Equal(t, "test@example.com", response.Email)
				assert.Equal(t, "admin", response.Role)
				assert.Equal(t, "Test User", response.DisplayName)
				assert.Equal(t, "13800138000", response.Phone)
			}
		})
	}
}

// 基准测试
func BenchmarkAuthService_Login(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	authService := &AuthService{
		client:    client,
		jwtSecret: "test-secret-key",
		logger:    logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, _ := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)

	request := &dto.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authService.Login(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}