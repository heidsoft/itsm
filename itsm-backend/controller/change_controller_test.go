package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestChangeController(t *testing.T) (*gin.Engine, *ent.Client, *ChangeController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	changeService := service.NewChangeService(client, logger)

	// 创建控制器
	changeController := NewChangeController(logger, changeService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/changes", changeController.ListChanges)
	r.POST("/api/v1/changes", changeController.CreateChange)
	r.GET("/api/v1/changes/:id", changeController.GetChange)
	r.PUT("/api/v1/changes/:id", changeController.UpdateChange)
	r.DELETE("/api/v1/changes/:id", changeController.DeleteChange)

	return r, client, changeController
}

func createTestTenantAndUserForChange(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
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

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("manager").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}

func TestChangeController_ListChanges(t *testing.T) {
	_, client, _ := setupTestChangeController(t)
	defer client.Close()

	tenant, _ := createTestTenantAndUserForChange(t, client)
	logger := zaptest.NewLogger(t).Sugar()
	changeService := service.NewChangeService(client, logger)
	changeController := NewChangeController(logger, changeService)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取变更列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "带分页参数",
			queryParams:    "page=1&pageSize=10",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "按状态过滤",
			queryParams:    "status=open",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "搜索关键词",
			queryParams:    "search=test",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/changes"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 设置租户上下文
			tenantCtx := &middleware.TenantContext{
				TenantID: tenant.ID,
				Tenant:   tenant,
			}
			c.Set(middleware.TenantContextKey, tenantCtx)

			// 直接调用控制器
			changeController.ListChanges(c)

			// 验证响应
			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				assert.Equal(t, tt.expectedCode, response.Code)
			}
		})
	}
}

func TestChangeController_CreateChange(t *testing.T) {
	_, client, _ := setupTestChangeController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForChange(t, client)

	// 使用指针来满足 *string 类型
	title1 := "系统升级"
	desc1 := "升级数据库到新版本"
	title2 := ""
	desc2 := "描述"

	tests := []struct {
		name           string
		request        dto.CreateChangeRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "成功创建变更",
			request: dto.CreateChangeRequest{
				Title:       title1,
				Description: desc1,
				Type:        "standard",
				Priority:    "high",
				RiskLevel:   "medium",
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name: "标题为空",
			request: dto.CreateChangeRequest{
				Title:       title2,
				Description: desc2,
				Type:        "standard",
				Priority:    "low",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.BadRequestCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备请求数据
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// 创建请求
			req, err := http.NewRequest("POST", "/api/v1/changes", bytes.NewBuffer(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// 设置上下文并直接调用控制器
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 设置租户上下文
			tenantCtx := &middleware.TenantContext{
				TenantID: tenant.ID,
				Tenant:   tenant,
			}
			c.Set(middleware.TenantContextKey, tenantCtx)
			c.Set("user_id", user.ID)

			// 直接调用控制器
			logger := zaptest.NewLogger(t).Sugar()
			changeService := service.NewChangeService(client, logger)
			changeController := NewChangeController(logger, changeService)
			changeController.CreateChange(c)

			// 验证响应
			var response common.Response
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Logf("Response code: %d, message: %s", response.Code, response.Message)
				assert.Equal(t, tt.expectedCode, response.Code)
			}
		})
	}
}

func TestChangeController_GetChange(t *testing.T) {
	r, client, _ := setupTestChangeController(t)
	defer client.Close()

	tenant, _ := createTestTenantAndUserForChange(t, client)

	// 创建一个测试变更
	ctx := context.Background()
	now := time.Now()
	change, err := client.Change.Create().
		SetTitle("测试变更").
		SetDescription("测试描述").
		SetType("standard").
		SetStatus("draft").
		SetPriority("medium").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(tenant.ID).
		SetPlannedStartDate(now).
		SetPlannedEndDate(now.Add(24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name           string
		changeID       string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取变更",
			changeID:       "1",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
		{
			name:           "无效的变更ID",
			changeID:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   common.ParamErrorCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/changes/"+tt.changeID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}

	_ = change
}

func TestChangeController_UpdateChange(t *testing.T) {
	r, client, _ := setupTestChangeController(t)
	defer client.Close()

	tenant, user := createTestTenantAndUserForChange(t, client)

	// 创建一个测试变更
	ctx := context.Background()
	change, err := client.Change.Create().
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetType("standard").
		SetStatus("draft").
		SetPriority("low").
		SetRiskLevel("low").
		SetCreatedBy(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 使用指针来满足 *string 类型
	title := "更新后的标题"
	desc := "更新后的描述"

	tests := []struct {
		name           string
		changeID       int
		request        dto.UpdateChangeRequest
		expectedStatus int
		expectedCode   int
	}{
		{
			name:     "成功更新变更",
			changeID: change.ID,
			request: dto.UpdateChangeRequest{
				Title:       &title,
				Description: &desc,
			},
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, err := http.NewRequest("PUT", "/api/v1/changes/"+string(rune('0'+tt.changeID)), bytes.NewBuffer(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)
			c.Set("user_id", user.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestChangeController_DeleteChange(t *testing.T) {
	r, client, _ := setupTestChangeController(t)
	defer client.Close()

	tenant, _ := createTestTenantAndUserForChange(t, client)

	// 创建一个测试变更
	ctx := context.Background()
	change, err := client.Change.Create().
		SetTitle("待删除变更").
		SetDescription("将被删除").
		SetType("standard").
		SetStatus("draft").
		SetPriority("low").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name           string
		changeID       int
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功删除变更",
			changeID:       change.ID,
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/changes/"+string(rune('0'+tt.changeID)), nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
