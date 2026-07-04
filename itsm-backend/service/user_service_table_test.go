package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// 表驱动测试：CreateUser
// 验证用户创建的各种场景
// ============================================================

func TestCreateUser_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		req           *dto.CreateUserRequest
		wantErr       bool
		errContains   string
		setup         func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int)
		checkResponse func(t *testing.T, user *ent.User)
	}{
		{
			name: "create user with all fields",
			req: &dto.CreateUserRequest{
				Username:   "john.doe",
				Email:      "john@example.com",
				Name:       "John Doe",
				Department: "Engineering",
				Phone:      "+1234567890",
				Password:   "secure123",
				Role:       "agent",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "john.doe", user.Username)
				assert.Equal(t, "john@example.com", user.Email)
				assert.Equal(t, "John Doe", user.Name)
				assert.Equal(t, "Engineering", user.Department)
				assert.Equal(t, "+1234567890", user.Phone)
				assert.True(t, user.Active)
				assert.Equal(t, "agent", string(user.Role))
			},
		},
		{
			name: "create user with default role",
			req: &dto.CreateUserRequest{
				Username: "jane.doe",
				Email:    "jane@example.com",
				Name:     "Jane Doe",
				Password: "secure123",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "end_user", string(user.Role)) // 默认角色
			},
		},
		{
			name: "create user with MSP role",
			req: &dto.CreateUserRequest{
				Username: "msp.agent",
				Email:    "msp@example.com",
				Name:     "MSP Agent",
				Password: "secure123",
				MSPRole:  "provider_agent",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "provider_agent", string(user.MspRole))
			},
		},
		{
			name: "duplicate username",
			req: &dto.CreateUserRequest{
				Username: "duplicate.user",
				Email:    "unique1@example.com",
				Name:     "Duplicate User",
				Password: "secure123",
			},
			wantErr:     true,
			errContains: "用户名已存在",
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				_, err := client.User.Create().
					SetUsername("duplicate.user").
					SetEmail("unique2@example.com").
					SetName("Existing User").
					SetPasswordHash("hash").
					SetActive(true).
					SetTenantID(tenantID).
					Save(ctx)
				require.NoError(t, err)
			},
		},
		{
			name: "duplicate email",
			req: &dto.CreateUserRequest{
				Username: "unique.user",
				Email:    "duplicate@example.com",
				Name:     "Duplicate Email User",
				Password: "secure123",
			},
			wantErr:     true,
			errContains: "邮箱已存在",
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				_, err := client.User.Create().
					SetUsername("existing.user").
					SetEmail("duplicate@example.com").
					SetName("Existing Email User").
					SetPasswordHash("hash").
					SetActive(true).
					SetTenantID(tenantID).
					Save(ctx)
				require.NoError(t, err)
			},
		},
		{
			name: "convert user role to end_user",
			req: &dto.CreateUserRequest{
				Username: "converted.user",
				Email:    "converted@example.com",
				Name:     "Converted User",
				Password: "secure123",
				Role:     "user", // 应转换为 end_user
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "end_user", string(user.Role))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			if tt.setup != nil {
				tt.setup(t, ctx, client, tenant.ID)
			}

			user, err := service.CreateUser(ctx, tt.req, tenant.ID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)

			if tt.checkResponse != nil {
				tt.checkResponse(t, user)
			}
		})
	}
}

// ============================================================
// 表驱动测试：ListUsers with filters
// 验证用户列表的各种过滤条件
// ============================================================

func TestListUsers_WithFilters_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		req       *dto.ListUsersRequest
		wantTotal int
		wantFirst string // 期望第一个用户的 username
		setup     func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int)
	}{
		{
			name: "no filters returns all",
			req: &dto.ListUsersRequest{
				Page:     1,
				PageSize: 10,
			},
			wantTotal: 5,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 5, false)
			},
		},
		{
			name: "filter by active status",
			req: &dto.ListUsersRequest{
				Status:   "active",
				Page:     1,
				PageSize: 10,
			},
			wantTotal: 3,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 3, false) // active
				createTestUsers(t, ctx, client, tenantID, 2, true)  // inactive
			},
		},
		{
			name: "filter by inactive status",
			req: &dto.ListUsersRequest{
				Status:   "inactive",
				Page:     1,
				PageSize: 10,
			},
			wantTotal: 2,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 3, false)
				createTestUsers(t, ctx, client, tenantID, 2, true)
			},
		},
		{
			name: "filter by department",
			req: &dto.ListUsersRequest{
				Department: "Engineering",
				Page:       1,
				PageSize:   10,
			},
			wantTotal: 2,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				// Engineering department
				createTestUsersWithDept(t, ctx, client, tenantID, 2, "Engineering")
				// Sales department
				createTestUsersWithDept(t, ctx, client, tenantID, 3, "Sales")
			},
		},
		{
			name: "search by username",
			req: &dto.ListUsersRequest{
				Search:   "searchable",
				Page:     1,
				PageSize: 10,
			},
			wantTotal: 2,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsersWithPrefix(t, ctx, client, tenantID, 2, "searchable")
				createTestUsersWithPrefix(t, ctx, client, tenantID, 3, "other")
			},
		},
		{
			name: "pagination page 1 size 2",
			req: &dto.ListUsersRequest{
				Page:     1,
				PageSize: 2,
			},
			wantTotal: 5,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 5, false)
			},
		},
		{
			name: "pagination page 2 size 2",
			req: &dto.ListUsersRequest{
				Page:     2,
				PageSize: 2,
			},
			wantTotal: 5,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 5, false)
			},
		},
		{
			name: "combined filters",
			req: &dto.ListUsersRequest{
				Status:     "active",
				Department: "Engineering",
				Page:       1,
				PageSize:   10,
			},
			wantTotal: 1,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsersWithDept(t, ctx, client, tenantID, 1, "Engineering")
				createTestUsersWithDept(t, ctx, client, tenantID, 1, "Sales")
				createTestUsersWithDeptAndActive(t, ctx, client, tenantID, 1, "Engineering", false)
			},
		},
		{
			name: "no matching results",
			req: &dto.ListUsersRequest{
				Status:     "active",
				Department: "NonExistent",
				Page:       1,
				PageSize:   10,
			},
			wantTotal: 0,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 3, false)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			if tt.setup != nil {
				tt.setup(t, ctx, client, tenant.ID)
			}

			resp, err := service.ListUsers(ctx, tt.req, tenant.ID)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantTotal, resp.Pagination.Total)

			if tt.wantTotal > 0 {
				assert.Len(t, resp.Users, min(tt.req.PageSize, tt.wantTotal))
			}
		})
	}
}

// ============================================================
// 表驱动测试：UpdateUser
// 验证用户更新场景
// ============================================================

func TestUpdateUser_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		updateReq     *dto.UpdateUserRequest
		wantErr       bool
		errContains   string
		checkResponse func(t *testing.T, user *ent.User)
	}{
		{
			name: "update name",
			updateReq: &dto.UpdateUserRequest{
				Name: "New Name",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "New Name", user.Name)
			},
		},
		{
			name: "update email",
			updateReq: &dto.UpdateUserRequest{
				Email: "newemail@example.com",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "newemail@example.com", user.Email)
			},
		},
		{
			name: "update department",
			updateReq: &dto.UpdateUserRequest{
				Department: "New Department",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "New Department", user.Department)
			},
		},
		{
			name: "update role to agent",
			updateReq: &dto.UpdateUserRequest{
				Role: "agent",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "agent", string(user.Role))
			},
		},
		{
			name: "update multiple fields",
			updateReq: &dto.UpdateUserRequest{
				Name:       "Multi Update",
				Email:      "multi@example.com",
				Department: "QA",
				Phone:      "+9999999999",
			},
			wantErr: false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.Equal(t, "Multi Update", user.Name)
				assert.Equal(t, "multi@example.com", user.Email)
				assert.Equal(t, "QA", user.Department)
				assert.Equal(t, "+9999999999", user.Phone)
			},
		},
		{
			name:      "empty update does nothing",
			updateReq: &dto.UpdateUserRequest{},
			wantErr:   false,
			checkResponse: func(t *testing.T, user *ent.User) {
				// 保持原样
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			// 创建测试用户
			testUser, err := client.User.Create().
				SetUsername("test.user").
				SetEmail("test@example.com").
				SetName("Test User").
				SetPasswordHash("hash").
				SetActive(true).
				SetTenantID(tenant.ID).
				Save(ctx)
			require.NoError(t, err)

			updatedUser, err := service.UpdateUser(ctx, testUser.ID, tt.updateReq, tenant.ID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, updatedUser)

			if tt.checkResponse != nil {
				tt.checkResponse(t, updatedUser)
			}
		})
	}
}

// ============================================================
// 表驱动测试：ChangeUserStatus
// 验证用户状态变更
// ============================================================

func TestChangeUserStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		initialActive bool
		newStatus     bool
		wantErr       bool
		checkResponse func(t *testing.T, user *ent.User)
	}{
		{
			name:          "activate user",
			initialActive: false,
			newStatus:     true,
			wantErr:       false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.True(t, user.Active)
			},
		},
		{
			name:          "deactivate user",
			initialActive: true,
			newStatus:     false,
			wantErr:       false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.False(t, user.Active)
			},
		},
		{
			name:          "no change same status",
			initialActive: true,
			newStatus:     true,
			wantErr:       false,
			checkResponse: func(t *testing.T, user *ent.User) {
				assert.True(t, user.Active)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			// 创建测试用户
			testUser, err := client.User.Create().
				SetUsername("status.test").
				SetEmail("status@example.com").
				SetName("Status Test").
				SetPasswordHash("hash").
				SetActive(tt.initialActive).
				SetTenantID(tenant.ID).
				Save(ctx)
			require.NoError(t, err)

			err = service.ChangeUserStatus(ctx, testUser.ID, tt.newStatus, 0, tenant.ID)

			require.NoError(t, err)

			// 验证状态已更新
			updatedUser, err := client.User.Get(ctx, testUser.ID)
			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, updatedUser)
			}
		})
	}
}

// ============================================================
// 表驱动测试：GetUserStats
// 验证用户统计功能
// ============================================================

func TestGetUserStats_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		wantTotal  int
		wantActive int
		setup      func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int)
	}{
		{
			name:       "empty tenant",
			wantTotal:  0,
			wantActive: 0,
			setup:      nil,
		},
		{
			name:       "all active users",
			wantTotal:  5,
			wantActive: 5,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 5, false)
			},
		},
		{
			name:       "mixed active/inactive",
			wantTotal:  5,
			wantActive: 3,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsers(t, ctx, client, tenantID, 3, false)
				createTestUsers(t, ctx, client, tenantID, 2, true)
			},
		},
		{
			name:       "users by department",
			wantTotal:  4,
			wantActive: 4,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestUsersWithDept(t, ctx, client, tenantID, 2, "Engineering")
				createTestUsersWithDept(t, ctx, client, tenantID, 2, "Sales")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			if tt.setup != nil {
				tt.setup(t, ctx, client, tenant.ID)
			}

			stats, err := service.GetUserStats(ctx, tenant.ID)

			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tt.wantTotal, stats.Total)
			assert.Equal(t, tt.wantActive, stats.Active)
		})
	}
}

// ============================================================
// 表驱动测试：DeleteUser
// 验证用户删除（软删除）
// ============================================================

func TestDeleteUser_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		wantErr     bool
		errContains string
		setup       func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) int // 返回创建的 user ID
	}{
		{
			name:    "delete existing user (soft delete)",
			wantErr: false,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) int {
				u, err := client.User.Create().
					SetUsername("to.delete").
					SetEmail("delete@example.com").
					SetName("To Delete").
					SetPasswordHash("hash").
					SetActive(true).
					SetTenantID(tenantID).
					Save(ctx)
				require.NoError(t, err)
				return u.ID
			},
		},
		{
			name:        "delete non-existent user",
			wantErr:     true,
			errContains: "用户不存在",
			setup:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer client.Close()
			ctx := context.Background()
			logger := zaptest.NewLogger(t).Sugar()

			tenant, err := client.Tenant.Create().
				SetName("Test Tenant").
				SetCode("TEST").
				SetDomain("test.com").
				SetStatus("active").
				Save(ctx)
			require.NoError(t, err)

			service := NewUserService(client, logger)

			var userID int
			if tt.setup != nil {
				userID = tt.setup(t, ctx, client, tenant.ID)
			} else {
				userID = 99999 // 不存在的用户
			}

			err = service.DeleteUser(ctx, userID, tenant.ID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			// 验证用户已被软删除（Active = false）
			deletedUser, err := client.User.Get(ctx, userID)
			require.NoError(t, err)
			assert.False(t, deletedUser.Active, "User should be inactive after soft delete")
		})
	}
}

// ============================================================
// 辅助函数
// ============================================================

func createTestUsers(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, inactive bool) []int {
	t.Helper()
	ids := make([]int, 0, count)
	// Use timestamp to ensure uniqueness across test runs
	ts := time.Now().UnixNano()
	prefix := fmt.Sprintf("u%d", ts%100000)
	for i := 0; i < count; i++ {
		u, err := client.User.Create().
			SetUsername(fmt.Sprintf("%s%d", prefix, i)).
			SetEmail(fmt.Sprintf("%s%d@test.com", prefix, i)).
			SetName("Test User").
			SetPasswordHash("hash").
			SetActive(!inactive).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, u.ID)
	}
	return ids
}

func createTestUsersWithDept(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, dept string) []int {
	t.Helper()
	ids := make([]int, 0, count)
	ts := time.Now().UnixNano()
	prefix := fmt.Sprintf("d%d%s", ts%100000, dept)
	for i := 0; i < count; i++ {
		u, err := client.User.Create().
			SetUsername(fmt.Sprintf("%s%d", prefix, i)).
			SetEmail(fmt.Sprintf("%s%d@test.com", prefix, i)).
			SetName("Test User").
			SetDepartment(dept).
			SetPasswordHash("hash").
			SetActive(true).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, u.ID)
	}
	return ids
}

func createTestUsersWithDeptAndActive(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, dept string, active bool) []int {
	t.Helper()
	ids := make([]int, 0, count)
	ts := time.Now().UnixNano()
	prefix := fmt.Sprintf("da%d%s", ts%100000, dept)
	for i := 0; i < count; i++ {
		u, err := client.User.Create().
			SetUsername(fmt.Sprintf("%s%d", prefix, i)).
			SetEmail(fmt.Sprintf("%s%d@test.com", prefix, i)).
			SetName("Test User").
			SetDepartment(dept).
			SetPasswordHash("hash").
			SetActive(active).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, u.ID)
	}
	return ids
}

func createTestUsersWithPrefix(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, prefix string) []int {
	t.Helper()
	ids := make([]int, 0, count)
	ts := time.Now().UnixNano()
	uniqPrefix := fmt.Sprintf("p%d%s", ts%100000, prefix)
	for i := 0; i < count; i++ {
		u, err := client.User.Create().
			SetUsername(fmt.Sprintf("%s%d", uniqPrefix, i)).
			SetEmail(fmt.Sprintf("%s%d@test.com", uniqPrefix, i)).
			SetName("Test User").
			SetPasswordHash("hash").
			SetActive(true).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, u.ID)
	}
	return ids
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
