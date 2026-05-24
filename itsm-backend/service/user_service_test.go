package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

func TestUserService_CreateUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		request       *dto.CreateUserRequest
		expectedError bool
	}{
		{
			name: "成功创建用户",
			request: &dto.CreateUserRequest{
				Username:   "testuser",
				Email:      "test@example.com",
				Name:       "Test User",
				Department: "IT",
				Phone:      "1234567890",
				Password:   "password123",
				TenantID:   testTenant.ID,
			},
			expectedError: false,
		},
		{
			name: "用户名重复",
			request: &dto.CreateUserRequest{
				Username:   "testuser",
				Email:      "test2@example.com",
				Name:       "Test User 2",
				Department: "HR",
				Phone:      "0987654321",
				Password:   "password456",
				TenantID:   testTenant.ID,
			},
			expectedError: true,
		},
		{
			name: "邮箱重复",
			request: &dto.CreateUserRequest{
				Username:   "testuser2",
				Email:      "test@example.com",
				Name:       "Test User 3",
				Department: "Finance",
				Phone:      "1111111111",
				Password:   "password789",
				TenantID:   testTenant.ID,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := userService.CreateUser(ctx, tt.request, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Username, user.Username)
				assert.Equal(t, tt.request.Email, user.Email)
				assert.Equal(t, tt.request.Name, user.Name)
				assert.Equal(t, tt.request.Department, user.Department)
				assert.Equal(t, tt.request.Phone, user.Phone)
				assert.True(t, user.Active)
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		expectedError bool
	}{
		{
			name:          "成功获取用户",
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
			user, err := userService.GetUserByID(ctx, tt.userID, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.ID, user.ID)
				assert.Equal(t, testUser.Username, user.Username)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.Name, user.Name)
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Original Name").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		request       *dto.UpdateUserRequest
		expectedError bool
	}{
		{
			name:   "成功更新用户",
			userID: testUser.ID,
			request: &dto.UpdateUserRequest{
				Email:      "updated@example.com",
				Name:       "Updated Name",
				Department: "HR",
				Phone:      "0987654321",
			},
			expectedError: false,
		},
		{
			name:   "部分更新",
			userID: testUser.ID,
			request: &dto.UpdateUserRequest{
				Name: "Partially Updated",
			},
			expectedError: false,
		},
		{
			name:   "用户不存在",
			userID: 99999,
			request: &dto.UpdateUserRequest{
				Name: "Non-existent User",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := userService.UpdateUser(ctx, tt.userID, tt.request, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
				if tt.request.Email != "" {
					assert.Equal(t, tt.request.Email, user.Email)
				}
				if tt.request.Name != "" {
					assert.Equal(t, tt.request.Name, user.Name)
				}
				if tt.request.Department != "" {
					assert.Equal(t, tt.request.Department, user.Department)
				}
				if tt.request.Phone != "" {
					assert.Equal(t, tt.request.Phone, user.Phone)
				}
			}
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		expectedError bool
	}{
		{
			name:          "成功删除用户",
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
			err := userService.DeleteUser(ctx, tt.userID, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 验证用户已被软删除（设置为非激活状态）
				user, err := client.User.Get(ctx, tt.userID)
				assert.NoError(t, err)
				assert.False(t, user.Active)
			}
		})
	}
}

func TestUserService_SearchUsers(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	users := []struct {
		username string
		email    string
		name     string
	}{
		{"john_doe", "john@example.com", "John Doe"},
		{"jane_smith", "jane@example.com", "Jane Smith"},
		{"admin_user", "admin@example.com", "Admin User"},
		{"test_user", "test@example.com", "Test User"},
	}

	for _, u := range users {
		_, err := client.User.Create().
			SetUsername(u.username).
			SetEmail(u.email).
			SetPasswordHash("hashedpassword").
			SetName(u.name).
			SetDepartment("IT").
			SetPhone("1234567890").
			SetActive(true).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		request       *dto.SearchUsersRequest
		expectedCount int
		expectedError bool
	}{
		{
			name: "搜索用户名",
			request: &dto.SearchUsersRequest{
				Keyword:  "john",
				TenantID: testTenant.ID,
				Limit:    10,
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "搜索邮箱",
			request: &dto.SearchUsersRequest{
				Keyword:  "admin@example.com",
				TenantID: testTenant.ID,
				Limit:    10,
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "搜索姓名",
			request: &dto.SearchUsersRequest{
				Keyword:  "Smith",
				TenantID: testTenant.ID,
				Limit:    10,
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "无结果搜索",
			request: &dto.SearchUsersRequest{
				Keyword:  "nonexistent",
				TenantID: testTenant.ID,
				Limit:    10,
			},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, err := userService.SearchUsers(ctx, tt.request, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, users)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.expectedCount)
			}
		})
	}
}

func TestUserService_ResetPassword(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		newPassword   string
		expectedError bool
	}{
		{
			name:          "成功重置密码",
			userID:        testUser.ID,
			newPassword:   "newpassword123",
			expectedError: false,
		},
		{
			name:          "用户不存在",
			userID:        99999,
			newPassword:   "newpassword456",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := userService.ResetPassword(ctx, tt.userID, tt.newPassword, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 验证密码已更新
				user, err := client.User.Get(ctx, tt.userID)
				assert.NoError(t, err)
				// 验证新密码可以正确验证
				err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.newPassword))
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_ChangeUserStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		active        bool
		currentUserID int
		expectedError bool
	}{
		{
			name:          "禁用用户",
			userID:        testUser.ID,
			active:        false,
			currentUserID: 0, // 使用不同的用户ID
			expectedError: false,
		},
		{
			name:          "启用用户",
			userID:        testUser.ID,
			active:        true,
			currentUserID: 0,
			expectedError: false,
		},
		{
			name:          "用户不存在",
			userID:        99999,
			active:        false,
			currentUserID: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := userService.ChangeUserStatus(ctx, tt.userID, tt.active, tt.currentUserID, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 验证状态已更新
				user, err := client.User.Get(ctx, tt.userID)
				assert.NoError(t, err)
				assert.Equal(t, tt.active, user.Active)
			}
		})
	}
}

// 基准测试
func TestUserService_UpdateUser_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	userService := NewUserService(client, logger)
	ctx := context.Background()

	// Create two tenants
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("T1").
		SetDomain("t1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("T2").
		SetDomain("t2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// Create user for tenant 1
	user1, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@t1.com").
		SetPasswordHash("hashedpassword").
		SetName("Tenant 1 User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create user for tenant 2
	_, err = client.User.Create().
		SetUsername("testuser2").
		SetEmail("test@t2.com").
		SetPasswordHash("hashedpassword").
		SetName("Tenant 2 User").
		SetDepartment("IT").
		SetPhone("0987654321").
		SetActive(true).
		SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	// Tenant 2 tries to update tenant 1's user - should fail
	_, err = userService.UpdateUser(ctx, user1.ID, &dto.UpdateUserRequest{Name: "Hacked"}, tenant2.ID)

	assert.Error(t, err)
	// User doesn't exist from tenant 2's perspective (cross-tenant isolation works)
	assert.Contains(t, err.Error(), "用户不存在")
}

func BenchmarkUserService_CreateUser(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &dto.CreateUserRequest{
			Username:   "testuser" + string(rune(i)),
			Email:      "test" + string(rune(i)) + "@example.com",
			Name:       "Test User",
			Department: "IT",
			Phone:      "1234567890",
			Password:   "password123",
			TenantID:   testTenant.ID,
		}
		_, _ = userService.CreateUser(ctx, req, testTenant.ID)
	}
}

func BenchmarkUserService_GetUserByID(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	userService := NewUserService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = userService.GetUserByID(ctx, testUser.ID, testTenant.ID)
	}
}
