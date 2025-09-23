package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
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
	authService := service.NewAuthService(client, "test-secret", logger)

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
			expectedStatus: http.StatusOK,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "密码错误",
			request: dto.LoginRequest{
				Username: user.Username,
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "用户名为空",
			request: dto.LoginRequest{
				Username: "",
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name: "密码为空",
			request: dto.LoginRequest{
				Username: user.Username,
				Password: "",
			},
			expectedStatus: http.StatusOK,
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
				assert.Equal(t, float64(user.ID), data["user_id"])
				assert.Equal(t, user.Username, data["username"])
				assert.Equal(t, user.Name, data["name"])
				assert.Equal(t, float64(tenant.ID), data["tenant_id"])
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
			expectedStatus: http.StatusOK,
			expectedCode:   common.AuthFailedCode,
		},
		{
			name: "空的刷新令牌",
			request: dto.RefreshTokenRequest{
				RefreshToken: "",
			},
			expectedStatus: http.StatusOK,
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
				assert.NotEmpty(t, data["refresh_token"])
				assert.Equal(t, float64(user.ID), data["user_id"])
				assert.Equal(t, user.Username, data["username"])
			}
		})
	}
}

func TestAuthController_LoginWithInactiveUser(t *testing.T) {
	r, client, _ := setupTestAuthController(t)
	defer client.Close()

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

	// 创建非活跃用户
	inactiveUser, err := client.User.Create().
		SetUsername("inactiveuser").
		SetEmail("inactive@example.com").
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
	assert.Equal(t, http.StatusOK, w.Code)

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

	// 创建非活跃租户
	inactiveTenant, err := client.Tenant.Create().
		SetName("Inactive Tenant").
		SetCode("INACTIVE").
		SetDomain("inactive.com").
		SetStatus("inactive"). // 设置为非活跃
		Save(ctx)
	require.NoError(t, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// 在非活跃租户下创建用户
	user, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@inactive.com").
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
	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 非活跃租户下的用户应该登录失败
	assert.Equal(t, common.AuthFailedCode, response.Code)
	assert.Contains(t, response.Message, "租户已被禁用")
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
			expectedStatus: http.StatusOK,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "刷新令牌接口无效JSON",
			requestBody:    `{"refresh_token":}`, // 无效JSON
			endpoint:       "/api/v1/refresh-token",
			expectedStatus: http.StatusOK,
			expectedCode:   common.ParamErrorCode,
		},
		{
			name:           "登录接口空请求体",
			requestBody:    "",
			endpoint:       "/api/v1/login",
			expectedStatus: http.StatusOK,
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
	authService := service.NewAuthService(client, "test-secret", logger)

	// 创建控制器
	authController := NewAuthController(authService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/login", authController.Login)

	// 创建测试数据
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(b, err)

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
	authService := service.NewAuthService(client, "test-secret", logger)

	// 创建控制器
	authController := NewAuthController(authService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/api/v1/login", authController.Login)
	r.POST("/api/v1/refresh-token", authController.RefreshToken)

	// 创建测试数据
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(b, err)

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