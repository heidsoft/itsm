package contract

// API Contract Tests - 验证前后端 API 契约
// 测试覆盖：登录、工单、变更、SLA、CMDB、知识库、智能分配
//
// 契约规范：
// 1. 请求参数使用 camelCase (page, pageSize)
// 2. 响应使用统一格式 { code, message, data }
// 3. 状态码：200成功, 400参数错误, 404未找到, 500内部错误
// 4. DTO 响应字段使用 camelCase
//
// 跑测命令：cd itsm-backend && go test ./tests/contract/... -v

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupContractTest 创建测试环境
func setupContractTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Next()
	})
	return r
}

// doRequest 执行 HTTP 请求
func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ============ 通用响应格式测试 ============

func TestContract_StandardResponseFormat(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/test/success", func(c *gin.Context) {
		common.Success(c, gin.H{"id": 1, "name": "test"})
	})

	w := doRequest(r, "GET", "/api/v1/test/success", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
	assert.Equal(t, "success", response.Message)
	assert.NotNil(t, response.Data)
}

func TestContract_ParamErrorFormat(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/test/param-error", func(c *gin.Context) {
		common.ParamError(c, "Invalid parameter: id")
	})

	w := doRequest(r, "GET", "/api/v1/test/param-error", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
	assert.Contains(t, response.Message, "Invalid parameter")
}

func TestContract_NotFoundFormat(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/test/not-found", func(c *gin.Context) {
		common.NotFound(c, "Resource not found")
	})

	w := doRequest(r, "GET", "/api/v1/test/not-found", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

func TestContract_InternalErrorFormat(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/test/internal-error", func(c *gin.Context) {
		common.InternalError(c, "Internal server error")
	})

	w := doRequest(r, "GET", "/api/v1/test/internal-error", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, common.InternalErrorCode, response.Code)
}

// ============ 登录 API 契约测试 ============

func TestContract_LoginAPI_Success(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Invalid request body: "+err.Error())
			return
		}

		// 模拟登录成功
		common.Success(c, gin.H{
			"token": "mock-jwt-token",
			"user": gin.H{
				"id":       1,
				"name":     "Test User",
				"username": req.Username,
				"email":    "test@example.com",
			},
		})
	})

	// 测试正确请求
	w := doRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "password123",
	})

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.NotEmpty(t, data["token"])
	assert.NotNil(t, data["user"])
}

func TestContract_LoginAPI_InvalidCredentials(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Invalid request body")
			return
		}
		common.Fail(c, common.AuthFailedCode, "Invalid username or password")
	})

	w := doRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// AuthFailedCode returns 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, common.AuthFailedCode, response.Code)
}

func TestContract_LoginAPI_MissingFields(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Invalid request body")
			return
		}
	})

	// 测试缺少 password
	w := doRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
	})

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
}

// ============ 工单 API 契约测试 ============

func TestContract_TicketAPI_List_Pagination(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/tickets", func(c *gin.Context) {
		// 验证分页参数使用 camelCase
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("pageSize", "10")

		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":           1,
					"ticketNumber": "INC-001",
					"title":        "Test Incident",
					"status":       "open",
					"priority":     "high",
					"assigneeId":   5,
					"createdAt":    "2024-01-15T10:00:00Z",
					"updatedAt":    "2024-01-15T10:00:00Z",
				},
			},
			"total":    1,
			"page":     page,
			"pageSize": pageSize,
		})
	})

	// 测试分页参数
	w := doRequest(r, "GET", "/api/v1/tickets?page=1&pageSize=20", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.NotNil(t, data["items"])
	assert.Equal(t, float64(1), data["total"])
	assert.Equal(t, "1", data["page"])
	assert.Equal(t, "20", data["pageSize"])
}

func TestContract_TicketAPI_List_ResponseDTOCamelCase(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/tickets", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":               1,
					"ticketNumber":     "INC-001",
					"title":            "Test Incident",
					"status":           "open",
					"priority":         "high",
					"assigneeId":       5,
					"requesterId":      10,
					"serviceCatalogId": 1,
					"createdAt":        "2024-01-15T10:00:00Z",
					"updatedAt":        "2024-01-15T10:00:00Z",
				},
			},
			"total": 1,
		})
	})

	w := doRequest(r, "GET", "/api/v1/tickets", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	item := items[0].(map[string]interface{})

	// 验证 DTO 字段使用 camelCase
	assert.Contains(t, item, "ticketNumber")
	assert.Contains(t, item, "assigneeId")
	assert.Contains(t, item, "requesterId")
	assert.Contains(t, item, "serviceCatalogId")
	assert.Contains(t, item, "createdAt")
	assert.Contains(t, item, "updatedAt")
}

func TestContract_TicketAPI_GetByID_InvalidID(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/tickets/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "invalid" {
			common.ParamError(c, "id must be a positive integer")
			return
		}
		common.NotFound(c, "Ticket not found")
	})

	// 测试无效 ID
	w := doRequest(r, "GET", "/api/v1/tickets/invalid", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
}

func TestContract_TicketAPI_GetByID_NotFound(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/tickets/:id", func(c *gin.Context) {
		common.NotFound(c, "Ticket not found")
	})

	w := doRequest(r, "GET", "/api/v1/tickets/999", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

// ============ 变更 API 契约测试 ============

func TestContract_ChangeAPI_List_Pagination(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/changes", func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("pageSize", "10")

		common.Success(c, gin.H{
			"changes": []interface{}{
				gin.H{
					"id":        1,
					"title":     "Test Change",
					"status":    "draft",
					"priority":  "medium",
					"createdAt": "2024-01-15T10:00:00Z",
				},
			},
			"total":    1,
			"page":     page,
			"pageSize": pageSize,
		})
	})

	w := doRequest(r, "GET", "/api/v1/changes?page=1&pageSize=10", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
}

func TestContract_ChangeAPI_GetByID_InvalidID(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/changes/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "invalid" {
			common.ParamError(c, "id must be a positive integer")
			return
		}
		common.NotFound(c, "Change not found")
	})

	w := doRequest(r, "GET", "/api/v1/changes/invalid", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
}

func TestContract_ChangeAPI_GetByID_NotFound(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/changes/:id", func(c *gin.Context) {
		common.NotFound(c, "Change not found")
	})

	w := doRequest(r, "GET", "/api/v1/changes/999", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

func TestContract_ChangeAPI_Delete_Success(t *testing.T) {
	r := setupContractTest()

	r.DELETE("/api/v1/changes/:id", func(c *gin.Context) {
		id, ok := common.ParsePositiveID(c, "id")
		if !ok {
			return
		}
		if id == 999 {
			common.NotFound(c, "Change not found")
			return
		}
		common.Success(c, nil)
	})

	w := doRequest(r, "DELETE", "/api/v1/changes/1", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
}

func TestContract_ChangeAPI_Delete_NotFound(t *testing.T) {
	r := setupContractTest()

	r.DELETE("/api/v1/changes/:id", func(c *gin.Context) {
		common.NotFound(c, "Change not found")
	})

	w := doRequest(r, "DELETE", "/api/v1/changes/999", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

// ============ SLA API 契约测试 ============

func TestContract_SLAAPI_List(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/sla", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":          1,
					"name":        "P1 Response SLA",
					"type":        "response_time",
					"priority":    "critical",
					"targetTime":  30,
					"warningTime": 20,
					"status":      "active",
					"isDefault":   true,
				},
			},
			"total": 1,
		})
	})

	w := doRequest(r, "GET", "/api/v1/sla", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// ============ CMDB API 契约测试 ============

func TestContract_CMDBAPI_List_Pagination(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/cmdb/cis", func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("pageSize", "10")

		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":          1,
					"name":        "Server-001",
					"type":        "server",
					"status":      "active",
					"environment": "production",
					"createdAt":   "2024-01-15T10:00:00Z",
				},
			},
			"total":    1,
			"page":     page,
			"pageSize": pageSize,
		})
	})

	w := doRequest(r, "GET", "/api/v1/cmdb/cis?page=1&pageSize=10", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.Equal(t, "1", data["page"])
	assert.Equal(t, "10", data["pageSize"])
}

func TestContract_CMDBAPI_GetByID(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/cmdb/cis/:id", func(c *gin.Context) {
		id, ok := common.ParsePositiveID(c, "id")
		if !ok {
			return
		}
		if id == 999 {
			common.NotFound(c, "CI not found")
			return
		}
		common.Success(c, gin.H{
			"id":            id,
			"name":          "Server-001",
			"type":          "server",
			"ciTypeId":      1,
			"status":        "active",
			"environment":   "production",
			"criticality":   "high",
			"cloudProvider": "aws",
			"cloudRegion":   "us-east-1",
			"createdAt":     "2024-01-15T10:00:00Z",
		})
	})

	// 测试有效 ID
	w := doRequest(r, "GET", "/api/v1/cmdb/cis/1", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	// 测试不存在的 ID
	w = doRequest(r, "GET", "/api/v1/cmdb/cis/999", nil)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

// ============ 知识库 API 契约测试 ============

func TestContract_KnowledgeAPI_List_Pagination(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/knowledge-articles", func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("pageSize", "10")

		common.Success(c, gin.H{
			"articles": []interface{}{
				gin.H{
					"id":        1,
					"title":     "How to reset password",
					"category":  "IT Support",
					"status":    "published",
					"createdAt": "2024-01-15T10:00:00Z",
					"updatedAt": "2024-01-15T10:00:00Z",
				},
			},
			"total":    1,
			"page":     page,
			"pageSize": pageSize,
		})
	})

	w := doRequest(r, "GET", "/api/v1/knowledge-articles?page=1&pageSize=10", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.NotNil(t, data["articles"])
	assert.Equal(t, "1", data["page"])
	assert.Equal(t, "10", data["pageSize"])
}

// ============ 智能分配 API 契约测试 ============

func TestContract_SmartAssignmentAPI_GetRecommendations(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/tickets/assign-recommendations/:ticketId", func(c *gin.Context) {
		ticketIdStr := c.Param("ticketId")
		if ticketIdStr == "invalid" {
			common.ParamError(c, "ticketId must be a positive integer")
			return
		}

		common.Success(c, gin.H{
			"suggestions": []interface{}{
				gin.H{
					"assigneeId":   5,
					"assigneeName": "John Doe",
					"score":        0.95,
					"reason":       "Based on skills and workload",
				},
			},
		})
	})

	// 测试有效请求
	w := doRequest(r, "POST", "/api/v1/tickets/assign-recommendations/1", map[string]interface{}{
		"ticketId":   1,
		"ticketType": "incident",
		"category":   "network",
	})

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.NotNil(t, data["suggestions"])
}

func TestContract_SmartAssignmentAPI_GetRecommendations_InvalidID(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/tickets/assign-recommendations/:ticketId", func(c *gin.Context) {
		common.ParamError(c, "ticketId must be a positive integer")
	})

	w := doRequest(r, "POST", "/api/v1/tickets/assign-recommendations/invalid", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
}

func TestContract_SmartAssignmentAPI_TestRule(t *testing.T) {
	r := setupContractTest()

	r.POST("/api/v1/tickets/assignment-rules/test", func(c *gin.Context) {
		var req struct {
			RuleID int `json:"ruleId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Invalid request body")
			return
		}
		if req.RuleID <= 0 {
			common.ParamError(c, "ruleId is required for rule testing")
			return
		}

		common.Success(c, gin.H{
			"suggestions": []interface{}{
				gin.H{
					"assigneeId":   5,
					"assigneeName": "John Doe",
					"score":        0.95,
					"matchedRule":  req.RuleID,
				},
			},
		})
	})

	// 测试带 ruleId 的规则测试
	w := doRequest(r, "POST", "/api/v1/tickets/assignment-rules/test", map[string]interface{}{
		"ruleId":     1,
		"ticketId":   1,
		"ticketType": "incident",
	})

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
}

func TestContract_SmartAssignmentAPI_ListRules(t *testing.T) {
	r := setupContractTest()

	r.GET("/api/v1/tickets/assignment-rules", func(c *gin.Context) {
		common.Success(c, gin.H{
			"rules": []interface{}{
				gin.H{
					"id":           1,
					"name":         "Critical Priority Rule",
					"priority":     "critical",
					"strategy":     "skill_based",
					"assigneeId":   5,
					"assigneeName": "John Doe",
					"isActive":     true,
					"createdAt":    "2024-01-15T10:00:00Z",
				},
			},
			"total": 1,
		})
	})

	w := doRequest(r, "GET", "/api/v1/tickets/assignment-rules", nil)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// ============ 分页参数规范化测试 ============

// TestContract_Pagination_ParameterNaming 校验分页契约的唯一解析入口。
//
// 此处必须委托真实的 common.GetPaginationFromQuery，而不是在测试里再抄一份解析逻辑：
// 早期版本自带 `c.DefaultQuery("pageSize", c.DefaultQuery("page_size", "10"))` 的桩 handler，
// 断言的其实是桩自己的行为（page_size 恒等于「被接受」），既无法发现生产漂移，
// 又把 snake_case 别名写成了契约文档。
func TestContract_Pagination_ParameterNaming(t *testing.T) {
	tests := []struct {
		name        string
		queryParams string
		expectPage  int
		expectSize  int
	}{
		{"camelCase pageSize 生效", "?page=1&pageSize=10", 1, 10},
		{"snake_case page_size 被忽略并回落默认值", "?page=1&page_size=10", 1, 20},
		{"只传 page 时 pageSize 取默认值", "?page=2", 2, 20},
		{"缺省参数取默认值", "", 1, 20},
		// 注意：HTTP 边界的 GetPaginationFromQuery 对越界值是「拒绝并回落默认值 20」，
		// 而 service 边界的 ValidatePagination 是「夹紧到 MaxPageSize(100)」，
		// DTO binding 的 max=100 则是「返回 400」。三者语义不一致，此处锁定 HTTP 边界的真实行为。
		{"pageSize 超上限回落默认值而非夹紧", "?pageSize=500", 1, 20},
		{"pageSize 非法值回落默认值", "?pageSize=abc", 1, 20},
		{"pageSize 非正数回落默认值", "?pageSize=0", 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupContractTest()

			r.GET("/api/v1/test/pagination", func(c *gin.Context) {
				pagination := common.GetPaginationFromQuery(c)
				common.Success(c, gin.H{
					"page":     pagination.Page,
					"pageSize": pagination.PageSize,
				})
			})

			w := doRequest(r, "GET", "/api/v1/test/pagination"+tt.queryParams, nil)

			var response common.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, common.SuccessCode, response.Code)

			data := response.Data.(map[string]interface{})
			// JSON 数字反序列化为 float64
			assert.Equal(t, float64(tt.expectPage), data["page"])
			assert.Equal(t, float64(tt.expectSize), data["pageSize"])

			// 响应字段命名契约：只允许 camelCase
			assert.NotContains(t, data, "page_size")
			assert.NotContains(t, data, "size")
		})
	}
}

// TestContract_Pagination_NoSnakeCaseAliasInProductionHandlers 静态守卫：
// 生产 handler 不得再读取 page_size / 输出 page_size 响应键。
// 契约测试若只覆盖桩路由，就无法阻止 handler 层重新引入 snake_case 别名。
func TestContract_Pagination_NoSnakeCaseAliasInProductionHandlers(t *testing.T) {
	var offenders []string
	walkErr := filepath.WalkDir("../../handlers", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(raw), "\n") {
			// 只关心真实的查询参数读取与响应键输出
			if strings.Contains(line, `Query("page_size")`) ||
				strings.Contains(line, `DefaultQuery("page_size"`) ||
				strings.Contains(line, `form:"page_size"`) ||
				strings.Contains(line, `"page_size":`) {
				offenders = append(offenders, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	require.NoError(t, walkErr)

	assert.Empty(t, offenders,
		"生产 handler 仍在使用 snake_case page_size，违反 API 契约 camelCase-only 规则：\n%s",
		strings.Join(offenders, "\n"))
}
