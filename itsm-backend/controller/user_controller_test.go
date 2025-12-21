package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

func setupTestUserController(t *testing.T) (*gin.Engine, *ent.Client, *UserController) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	userService := service.NewUserService(client, logger)

	// 创建控制器
	userController := NewUserController(userService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 添加认证中间件模拟
	r.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Set("tenant_id", 1)
		c.Next()
	})

	// 注册路由
	r.GET("/api/v1/users", userController.ListUsers)
	r.POST("/api/v1/users", userController.CreateUser)
	r.GET("/api/v1/users/:id", userController.GetUser)
	r.PUT("/api/v1/users/:id", userController.UpdateUser)
	r.DELETE("/api/v1/users/:id", userController.DeleteUser)
	r.POST("/api/v1/users/search", userController.SearchUsers)
	r.PUT("/api/v1/users/:id/status", userController.ChangeUserStatus)
	r.PUT("/api/v1/users/:id/password", userController.ResetPassword)

	return r, client, userController
}

func createTestUserData(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}

func TestUserController_CreateUser(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	tenant, _ := createTestUserData(t, client)

	tests := []struct {
		name           string
		request        dto.CreateUserRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "成功创建用户",
			request: dto.CreateUserRequest{
				Username:   "newuser",
				Email:      "newuser@example.com",
				Name:       "New User",
				Department: "HR",
				Phone:      "0987654321",
				Password:   "password123",
				TenantID:   tenant.ID,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "用户名已存在",
			request: dto.CreateUserRequest{
				Username:   "testuser", // 已存在的用户名
				Email:      "another@example.com",
				Name:       "Another User",
				Department: "Finance",
				Phone:      "1111111111",
				Password:   "password123",
				TenantID:   tenant.ID,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name: "邮箱格式错误",
			request: dto.CreateUserRequest{
				Username:   "invaliduser",
				Email:      "invalid-email",
				Name:       "Invalid User",
				Department: "IT",
				Phone:      "2222222222",
				Password:   "password123",
				TenantID:   tenant.ID,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name: "密码为空",
			request: dto.CreateUserRequest{
				Username:   "nopassuser",
				Email:      "nopass@example.com",
				Name:       "No Pass User",
				Department: "IT",
				Phone:      "3333333333",
				Password:   "", // 空密码
				TenantID:   tenant.ID,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestUserController_ListUsers(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	createTestUserData(t, client)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "获取用户列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "分页查询",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按部门筛选",
			queryParams:    "?department=IT",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按状态筛选",
			queryParams:    "?active=true",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)

			if response.Code == common.SuccessCode {
				data := response.Data.(map[string]interface{})
				assert.Contains(t, data, "users")
				// 统一分页结构：data.pagination.total
				p, ok := data["pagination"].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, p, "total")
			}
		})
	}
}

func TestUserController_GetUser(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestUserData(t, client)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "获取存在的用户",
			userID:         strconv.Itoa(user.ID),
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "获取不存在的用户",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
		{
			name:           "无效的用户ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/users/"+tt.userID, nil)

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

func TestUserController_UpdateUser(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestUserData(t, client)

	tests := []struct {
		name           string
		userID         string
		request        dto.UpdateUserRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name:   "成功更新用户",
			userID: strconv.Itoa(user.ID),
			request: dto.UpdateUserRequest{
				Name:       "Updated User",
				Department: "Finance",
				Phone:      "9999999999",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:   "更新不存在的用户",
			userID: "999",
			request: dto.UpdateUserRequest{
				Name: "Non-existent User",
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
		{
			name:   "无效的用户ID",
			userID: "invalid",
			request: dto.UpdateUserRequest{
				Name: "Invalid ID User",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("PUT", "/api/v1/users/"+tt.userID, bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestUserController_DeleteUser(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestUserData(t, client)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功删除用户",
			userID:         strconv.Itoa(user.ID),
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "删除不存在的用户",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
		{
			name:           "无效的用户ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", "/api/v1/users/"+tt.userID, nil)

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

func TestUserController_SearchUsers(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	createTestUserData(t, client)

	tests := []struct {
		name           string
		request        dto.SearchUsersRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "按关键词搜索",
			request: dto.SearchUsersRequest{
				Keyword:  "test",
				TenantID: 0,
				Limit:    10,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "按租户搜索",
			request: dto.SearchUsersRequest{
				Keyword:  "test",
				TenantID: 1,
				Limit:    10,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "空搜索条件",
			request: dto.SearchUsersRequest{
				Keyword:  "",
				TenantID: 0,
				Limit:    10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("POST", "/api/v1/users/search", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestUserController_ChangeUserStatus(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestUserData(t, client)

	tests := []struct {
		name           string
		userID         string
		request        dto.ChangeUserStatusRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name:   "激活用户",
			userID: strconv.Itoa(user.ID),
			request: dto.ChangeUserStatusRequest{
				Active: true,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:   "禁用用户",
			userID: strconv.Itoa(user.ID),
			request: dto.ChangeUserStatusRequest{
				Active: false,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:   "更改不存在用户的状态",
			userID: "999",
			request: dto.ChangeUserStatusRequest{
				Active: true,
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("PUT", "/api/v1/users/"+tt.userID+"/status", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

func TestUserController_ResetPassword(t *testing.T) {
	r, client, _ := setupTestUserController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestUserData(t, client)

	tests := []struct {
		name           string
		userID         string
		request        dto.ResetPasswordRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name:   "成功重置密码",
			userID: strconv.Itoa(user.ID),
			request: dto.ResetPasswordRequest{
				NewPassword: "newpassword123",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:   "密码太短",
			userID: strconv.Itoa(user.ID),
			request: dto.ResetPasswordRequest{
				NewPassword: "123", // 太短
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:   "重置不存在用户的密码",
			userID: "999",
			request: dto.ResetPasswordRequest{
				NewPassword: "newpassword123",
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   common.NotFoundCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, _ := http.NewRequest("PUT", "/api/v1/users/"+tt.userID+"/password", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, response.Code)
		})
	}
}

// 基准测试
func BenchmarkUserController_CreateUser(b *testing.B) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 创建logger
	logger := zaptest.NewLogger(b).Sugar()

	// 创建服务
	userService := service.NewUserService(client, logger)

	// 复用基准测试 logger

	// 创建控制器
	userController := NewUserController(userService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Set("tenant_id", 1)
		c.Next()
	})
	r.POST("/api/v1/users", userController.CreateUser)

	// 创建测试数据
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := dto.CreateUserRequest{
			Username:   fmt.Sprintf("user%d", i),
			Email:      fmt.Sprintf("user%d@example.com", i),
			Name:       fmt.Sprintf("User %d", i),
			Department: "IT",
			Phone:      "1234567890",
			Password:   "password123",
			TenantID:   tenant.ID,
		}

		requestBody, _ := json.Marshal(request)
		req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUserController_ListUsers(b *testing.B) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 创建logger
	logger := zaptest.NewLogger(b).Sugar()

	// 创建服务
	userService := service.NewUserService(client, logger)

	// 复用基准测试 logger

	// 创建控制器
	userController := NewUserController(userService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Set("tenant_id", 1)
		c.Next()
	})
	r.GET("/api/v1/users", userController.ListUsers)

	// 创建测试数据
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建一些测试用户
	for i := 0; i < 10; i++ {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		client.User.Create().
			SetUsername(fmt.Sprintf("user%d", i)).
			SetEmail(fmt.Sprintf("user%d@example.com", i)).
			SetPasswordHash(string(hashedPassword)).
			SetName(fmt.Sprintf("User %d", i)).
			SetDepartment("IT").
			SetPhone("1234567890").
			SetActive(true).
			SetTenantID(tenant.ID).
			Save(ctx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/users", nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
