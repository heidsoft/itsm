package common

import (
	"encoding/json"
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
