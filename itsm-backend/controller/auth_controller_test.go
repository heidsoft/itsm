package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver for tests

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // 导入 sqlite3 驱动
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

func setupTestAuthController(t *testing.T) (*gin.Engine, *ent.Client, *AuthController) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	authService := service.NewAuthService(client, "test-secret", logger, nil)

	// 创建控制器
	authController := NewAuthController(authService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.POST("/api/v1/login", authController.Login)
	r.POST("/api/v1/refresh-token", authController.RefreshToken)

	return r, client, authController
}

func createTestTenantAndUser(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
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

func TestAuthController_Login(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	tenant, user := createTestTenantAndUser(t, client)

	tests := []struct {
		name           string
		request        dto.LoginRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "成功登录",
			request: dto.LoginRequest{
				Username: user.Username,
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "用户名不存在",
			request: dto.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "密码错误",
			request: dto.LoginRequest{
				Username: user.Username,
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "用户名为空",
			request: dto.LoginRequest{
				Username: "",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name: "密码为空",
			request: dto.LoginRequest{
				Username: user.Username,
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备请求数据
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// 创建请求
			req, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// 执行请求
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tt.expectedStatus, w.Code)

			// 解析响应
			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// 验证响应码
			assert.Equal(t, tt.expectedCode, response.Code)

			// 如果登录成功，验证返回的数据
			if tt.expectedCode == common.SuccessCode {
				assert.NotNil(t, response.Data)

				// 将data转换为map以便访问字段
				data, ok := response.Data.(map[string]interface{})
				require.True(t, ok)

				// 验证返回的字段
				assert.NotEmpty(t, data["access_token"])
				assert.NotEmpty(t, data["refresh_token"])
				// user / tenant 为对象
				u, ok := data["user"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, float64(user.ID), u["id"])
				assert.Equal(t, user.Username, u["username"])
				assert.Equal(t, user.Name, u["name"])
				ten, ok := data["tenant"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, float64(tenant.ID), ten["id"])
			}
		})
	}
}

func TestAuthController_RefreshToken(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestTenantAndUser(t, client)

	// 首先登录获取refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	require.Equal(t, common.SuccessCode, loginResponse.Code)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	tests := []struct {
		name           string
		request        dto.RefreshTokenRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "成功刷新令牌",
			request: dto.RefreshTokenRequest{
				RefreshToken: refreshToken,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "无效的刷新令牌",
			request: dto.RefreshTokenRequest{
				RefreshToken: "invalid_token",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "空的刷新令牌",
			request: dto.RefreshTokenRequest{
				RefreshToken: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备请求数据
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// 创建请求
			req, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// 执行请求
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tt.expectedStatus, w.Code)

			// 解析响应
			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// 验证响应码
			assert.Equal(t, tt.expectedCode, response.Code)

			// 如果刷新成功，验证返回的数据
			if tt.expectedCode == common.SuccessCode {
				assert.NotNil(t, response.Data)

				// 将data转换为map以便访问字段
				data, ok := response.Data.(map[string]interface{})
				require.True(t, ok)

				// 验证返回的字段
				assert.NotEmpty(t, data["access_token"])
				// RefreshTokenResponse 仅返回 access_token（不做 refresh token 轮换）
			}
		})
	}
}

func TestAuthController_LoginWithInactiveUser(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 创建非活跃用户
	inactiveUser, err := client.User.Create().
		SetUsername("inactiveuser" + uniqueID).
		SetEmail("inactive" + uniqueID + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Inactive User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(false). // 设置为非活跃
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 尝试用非活跃用户登录
	loginRequest := dto.LoginRequest{
		Username: inactiveUser.Username,
		Password: "password123",
	}

	requestBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 非活跃用户应该登录失败
	assert.Equal(t, common.AuthFailedCode, response.Code)
	assert.Contains(t, response.Message, "用户已被禁用")
}

func TestAuthController_LoginWithInactiveTenant(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建非活跃租户
	inactiveTenant, err := client.Tenant.Create().
		SetName("Inactive Tenant").
		SetCode("INACTIVE" + uniqueID).
		SetDomain("inactive.com").
		SetStatus("inactive"). // 设置为非活跃
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 在非活跃租户下创建用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@inactive.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(inactiveTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 尝试登录
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	requestBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 非活跃租户下的用户应该登录失败
	assert.Equal(t, common.AuthFailedCode, response.Code)
	assert.Contains(t, response.Message, "租户已被暂停或过期")
}

func TestAuthController_InvalidJSONRequest(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	tests := []struct {
		name           string
		requestBody    string
		endpoint       string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "登录接口无效JSON",
			requestBody:    `{"username": "test", "password":}`, // 无效JSON
			endpoint:       "/api/v1/login",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "刷新令牌接口无效JSON",
			requestBody:    `{"refresh_token":}`, // 无效JSON
			endpoint:       "/api/v1/refresh-token",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "登录接口空请求体",
			requestBody:    "",
			endpoint:       "/api/v1/login",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tt.endpoint, bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)
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
func BenchmarkAuthController_Login(b *testing.B) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 创建logger
	logger := zaptest.NewLogger(b).Sugar()

	// 创建服务
	authService := service.NewAuthService(client, "test-secret", logger, nil)

	// 创建控制器
	authController := NewAuthController(authService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/login", authController.Login)

	// 创建测试数据
	ctx := context.Background()
	uniqueID := fmt.Sprintf("%d", b.N)

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(b, err)

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(b, err)

	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	requestBody, err := json.Marshal(loginRequest)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkAuthController_RefreshToken(b *testing.B) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 创建logger
	logger := zaptest.NewLogger(b).Sugar()

	// 创建服务
	authService := service.NewAuthService(client, "test-secret", logger, nil)

	// 创建控制器
	authController := NewAuthController(authService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/login", authController.Login)
	r.POST("/api/v1/refresh-token", authController.RefreshToken)

	// 创建测试数据
	ctx := context.Background()
	uniqueID := fmt.Sprintf("%d", b.N)

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(b, err)

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(b, err)

	// 先登录获取refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, _ := json.Marshal(loginRequest)
	loginReq, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	var loginResponse common.Response
	json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	loginData := loginResponse.Data.(map[string]interface{})
	refreshToken := loginData["refresh_token"].(string)

	refreshRequest := dto.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	requestBody, err := json.Marshal(refreshRequest)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// ==================== Token 刷新机制补充测试 ====================

func TestAuthController_RefreshTokenWithInactiveUser(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 创建活跃用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 首先登录获取 refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	// 将用户设置为非活跃
	_, err = client.User.UpdateOneID(user.ID).
		SetActive(false).
		Save(ctx)
	require.NoError(t, err)

	// 尝试用被禁用的用户刷新 token
	refreshRequest := dto.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	requestBody, err := json.Marshal(refreshRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 被禁用的用户应该刷新失败
	assert.Equal(t, common.AuthFailedCode, response.Code)
	assert.Contains(t, response.Message, "用户已被禁用")
}

func TestAuthController_RefreshTokenWithExpiredToken(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestTenantAndUser(t, client)

	// 首先登录获取 refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	// 使用过期的 token 格式（模拟过期 token）
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJleHAiOjB9.expired"

	refreshRequest := dto.RefreshTokenRequest{
		RefreshToken: expiredToken,
	}

	requestBody, err := json.Marshal(refreshRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 过期的 token 应该刷新失败
	assert.Equal(t, common.AuthFailedCode, response.Code)
}

func TestAuthController_RefreshTokenMultipleTimes(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestTenantAndUser(t, client)

	// 首先登录获取 refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	// 第一次刷新
	refreshRequest := dto.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	requestBody, err := json.Marshal(refreshRequest)
	require.NoError(t, err)

	req1, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var response1 common.Response
	err = json.Unmarshal(w1.Body.Bytes(), &response1)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response1.Code)

	response1Data, ok := response1.Data.(map[string]interface{})
	require.True(t, ok)
	newAccessToken1 := response1Data["access_token"].(string)
	assert.NotEmpty(t, newAccessToken1)

	// 第二次刷新（使用同一个 refresh token）
	req2, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var response2 common.Response
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response2.Code)

	response2Data, ok := response2.Data.(map[string]interface{})
	require.True(t, ok)
	newAccessToken2 := response2Data["access_token"].(string)
	assert.NotEmpty(t, newAccessToken2)

	// 两次刷新返回的 access token 可能不同（因为时间戳不同）
	assert.NotEmpty(t, newAccessToken1)
	assert.NotEmpty(t, newAccessToken2)
}

func TestAuthController_RefreshTokenConcurrent(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestTenantAndUser(t, client)

	// 首先登录获取 refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	// 并发刷新请求
	numRequests := 5
	results := make(chan struct {
		status int
		code   int
	}, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			refreshRequest := dto.RefreshTokenRequest{
				RefreshToken: refreshToken,
			}

			requestBody, _ := json.Marshal(refreshRequest)
			req, _ := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			var response common.Response
			json.Unmarshal(w.Body.Bytes(), &response)

			results <- struct {
				status int
				code   int
			}{
				status: w.Code,
				code:   response.Code,
			}
		}()
	}

	// 收集结果
	successCount := 0
	for i := 0; i < numRequests; i++ {
		result := <-results
		if result.status == http.StatusOK && result.code == common.SuccessCode {
			successCount++
		}
	}

	// 所有并发请求都应该成功
	assert.Equal(t, numRequests, successCount, "所有并发刷新请求都应该成功")
}

func TestAuthController_RefreshTokenResponseFormat(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	// 创建测试数据
	_, user := createTestTenantAndUser(t, client)

	// 首先登录获取 refresh token
	loginRequest := dto.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginBody, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)

	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse common.Response
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok)
	refreshToken := loginData["refresh_token"].(string)

	// 刷新 token
	refreshRequest := dto.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	requestBody, err := json.Marshal(refreshRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 验证响应格式
	assert.Equal(t, common.SuccessCode, response.Code)
	assert.Equal(t, "success", response.Status)
	assert.NotEmpty(t, response.Message)
	assert.NotNil(t, response.Data)

	// 验证返回的数据结构
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	// RefreshTokenResponse 仅返回 access_token
	assert.NotEmpty(t, data["access_token"])
	// 不应该返回 refresh_token（不做轮换）
	_, hasRefreshToken := data["refresh_token"]
	assert.False(t, hasRefreshToken, "RefreshTokenResponse 不应该返回新的 refresh_token")
}

func TestAuthController_RefreshTokenWithDifferentUserTokens(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

	ctx := context.Background()
	uniqueID1 := uniqueTestID()
	uniqueID2 := uniqueTestID()

	// 创建两个租户
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("T1" + uniqueID1).
		SetDomain("t1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("T2" + uniqueID2).
		SetDomain("t2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 创建两个用户
	user1, err := client.User.Create().
		SetUsername("user1" + uniqueID1).
		SetEmail("user1" + uniqueID1 + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("User 1").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	user2, err := client.User.Create().
		SetUsername("user2" + uniqueID2).
		SetEmail("user2" + uniqueID2 + "@example.com").
		SetPasswordHash(string(hashedPassword)).
		SetName("User 2").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(tenant2.ID).
		Save(ctx)
	require.NoError(t, err)

	// 用户 1 登录
	loginRequest1 := dto.LoginRequest{
		Username: user1.Username,
		Password: "password123",
	}

	loginBody1, _ := json.Marshal(loginRequest1)
	loginReq1, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody1))
	loginReq1.Header.Set("Content-Type", "application/json")

	loginW1 := httptest.NewRecorder()
	r.ServeHTTP(loginW1, loginReq1)

	var loginResponse1 common.Response
	json.Unmarshal(loginW1.Body.Bytes(), &loginResponse1)
	loginData1 := loginResponse1.Data.(map[string]interface{})
	refreshToken1 := loginData1["refresh_token"].(string)

	// 用户 2 登录
	loginRequest2 := dto.LoginRequest{
		Username: user2.Username,
		Password: "password123",
	}

	loginBody2, _ := json.Marshal(loginRequest2)
	loginReq2, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(loginBody2))
	loginReq2.Header.Set("Content-Type", "application/json")

	loginW2 := httptest.NewRecorder()
	r.ServeHTTP(loginW2, loginReq2)

	var loginResponse2 common.Response
	json.Unmarshal(loginW2.Body.Bytes(), &loginResponse2)
	loginData2 := loginResponse2.Data.(map[string]interface{})
	refreshToken2 := loginData2["refresh_token"].(string)

	// 用用户 1 的 refresh token 刷新，应该成功
	refreshRequest1 := dto.RefreshTokenRequest{
		RefreshToken: refreshToken1,
	}

	requestBody1, _ := json.Marshal(refreshRequest1)
	req1, _ := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody1))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// 用用户 2 的 refresh token 刷新，应该成功
	refreshRequest2 := dto.RefreshTokenRequest{
		RefreshToken: refreshToken2,
	}

	requestBody2, _ := json.Marshal(refreshRequest2)
	req2, _ := http.NewRequest("POST", "/api/v1/refresh-token", bytes.NewBuffer(requestBody2))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
}
