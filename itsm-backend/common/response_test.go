package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ==================== Response 结构测试 ====================

func TestResponse_Structure(t *testing.T) {
	resp := Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    map[string]interface{}{"id": 1},
	}

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestResponse_JSONSerialization(t *testing.T) {
	resp := Response{
		Code:    SuccessCode,
		Message: "success",
		Data:    map[string]interface{}{"id": 1, "name": "test"},
	}

	jsonBytes, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded Response
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Code, decoded.Code)
	assert.Equal(t, resp.Message, decoded.Message)
}

func TestResponse_EmptyData(t *testing.T) {
	resp := Response{
		Code:    SuccessCode,
		Message: "success",
	}

	jsonBytes, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded Response
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.NoError(t, err)
	assert.Nil(t, decoded.Data)
}

// ==================== Response Code 常量测试 ====================

func TestResponseCodes(t *testing.T) {
	// 验证响应码定义
	assert.Equal(t, 0, SuccessCode)
	assert.Equal(t, 1001, ParamErrorCode)
	assert.Equal(t, 1002, ValidationError)
	assert.Equal(t, 2001, AuthFailedCode)
	assert.Equal(t, 2002, UnauthorizedCode)
	assert.Equal(t, 2003, ForbiddenCode)
	assert.Equal(t, 4004, NotFoundCode)
	assert.Equal(t, 4000, BadRequestCode)
	assert.Equal(t, 4090, ConflictCode)
	assert.Equal(t, 5001, InternalErrorCode)
}

func TestResponseCodeAliases(t *testing.T) {
	assert.Equal(t, NotFoundCode, NotFoundErrorCode)
	assert.Equal(t, AuthFailedCode, AuthErrorCode)
	assert.Equal(t, ForbiddenCode, ForbiddenErrorCode)
}

// ==================== Success 函数测试 ====================

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{"id": 1, "name": "test"}
	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, SuccessCode, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

func TestSuccess_WithNilData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, SuccessCode, resp.Code)
	assert.Nil(t, resp.Data)
}

func TestSuccess_WithListData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []int{1, 2, 3, 4, 5}
	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, SuccessCode, resp.Code)
}

// ==================== Fail 函数测试 ====================

func TestFail_ParamError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ParamErrorCode, "参数错误")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ParamErrorCode, resp.Code)
	assert.Equal(t, "参数错误", resp.Message)
}

func TestFail_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ValidationError, "验证失败")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFail_AuthFailed(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, AuthFailedCode, "认证失败")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFail_Forbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ForbiddenCode, "禁止访问")

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFail_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, NotFoundCode, "资源不存在")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFail_BadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, BadRequestCode, "请求错误")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFail_Conflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ConflictCode, "版本冲突")

	assert.Equal(t, http.StatusConflict, w.Code)
}

// 对齐审计 P0 #3:2002 (未授权)必须映射到 401,不能再走 200 通道。
// 这是 Stage 1.6 (common/response 2002/2004/2005 HTTP 状态断言) 的核心契约。
func TestFail_Unauthorized2002(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, UnauthorizedCode, "未授权")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, UnauthorizedCode, resp.Code)
}

// 对齐审计 P0 #3:2004 (工具权限不足)必须映射到 403。
func TestFail_ToolPermissionDenied2004(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ToolPermissionDeniedCode, "工具权限不足")

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ToolPermissionDeniedCode, resp.Code)
}

// 对齐审计 P0 #3:2005 (未知工具)必须映射到 404。
func TestFail_UnknownTool2005(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, UnknownToolCode, "未知工具")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, UnknownToolCode, resp.Code)
}

// FailWithData 同样要遵循 2002/2004/2005 的 HTTP 状态映射。
func TestFailWithData_Unauthorized2002(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithData(c, UnauthorizedCode, "未授权", map[string]string{"hint": "login"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFailWithData_ToolPermissionDenied2004(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithData(c, ToolPermissionDeniedCode, "工具权限不足", map[string]string{"tool": "rbac_query"})

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFailWithData_UnknownTool2005(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithData(c, UnknownToolCode, "未知工具", map[string]string{"tool": "no_such_tool"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFail_InternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, InternalErrorCode, "内部错误")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFail_UnknownCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 使用未定义的错误码，应该返回 200
	Fail(c, 9999, "未知错误")

	assert.Equal(t, http.StatusOK, w.Code)
}

// ==================== FailWithData 函数测试 ====================

func TestFailWithData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{
		"errors": []string{"field1 is required", "field2 is invalid"},
	}
	FailWithData(c, ValidationError, "验证失败", data)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ValidationError, resp.Code)
	assert.Equal(t, "验证失败", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestFailWithData_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{
		"id": 123,
	}
	FailWithData(c, NotFoundCode, "记录不存在", data)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ==================== Response JSON 格式测试 ====================

func TestResponse_JSONFormat(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   123,
		Name: "测试",
	}
	Success(c, data)

	// 验证 JSON 格式
	jsonStr := w.Body.String()
	assert.Contains(t, jsonStr, `"code":0`)
	assert.Contains(t, jsonStr, `"message":"success"`)
	assert.Contains(t, jsonStr, `"id":123`)
	assert.Contains(t, jsonStr, `"name":"测试"`)
}

// ==================== 分页响应测试 ====================

func TestSuccess_WithPaginationResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := &PaginationResponse{
		Page:       1,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
		HasNext:    true,
		HasPrev:    false,
	}
	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

// ==================== 错误响应边界测试 ====================

func TestFail_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, ParamErrorCode, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "", resp.Message)
}

func TestFail_LongMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	longMessage := ""
	for i := 0; i < 100; i++ {
		longMessage += "这是一条很长的错误消息"
	}
	Fail(c, InternalErrorCode, longMessage)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== Context 状态测试 ====================

func TestSuccess_ContextModified(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 设置一些 context 值
	c.Set("tenant_id", 123)

	Success(c, nil)

	// 验证响应后 context 仍可访问
	assert.Equal(t, 123, c.GetInt("tenant_id"))
}

func TestFail_ContextModified(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set("request_id", "req-123")

	Fail(c, NotFoundCode, "Not found")

	assert.Equal(t, "req-123", c.GetString("request_id"))
}

// ==================== FailWithErr / ParamErrorWithErr (Phase 4 安全错误映射) ====================

func TestFailWithErr_DoesNotLeakRawError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal", nil)

	// 模拟驱动层错误：包含 pq / ent 等不应暴露给客户端的字符串
	rawErr := errors.New("pq: constraint violations: pq: duplicate key value violates unique constraint")
	FailWithErr(c, rawErr, "operation failed")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, InternalErrorCode, resp.Code)
	assert.Equal(t, "operation failed", resp.Message)
	// 核心断言：原始 err 中的 pq: 字符串绝不能出现在响应体中
	assert.NotContains(t, w.Body.String(), "pq:")
	assert.NotContains(t, w.Body.String(), "duplicate key")
}

func TestFailWithErr_NilErrSafe(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/x", nil)

	// err 为 nil 时不应该 panic
	FailWithErr(c, nil, "operation failed")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "operation failed", resp.Message)
}

func TestParamErrorWithErr_DoesNotLeakRawError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal", nil)

	rawErr := errors.New("ent: validation failed: field title required")
	ParamErrorWithErr(c, rawErr, "invalid request body")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ParamErrorCode, resp.Code)
	assert.Equal(t, "invalid request body", resp.Message)
	assert.NotContains(t, w.Body.String(), "ent:")
	assert.NotContains(t, w.Body.String(), "validation failed")
}

func TestInternalErrorf_FormatsMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalErrorf(c, "user %s not allowed to perform %s", "alice", "delete")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "user alice not allowed to perform delete", resp.Message)
}
