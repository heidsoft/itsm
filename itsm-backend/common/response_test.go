package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

func TestRespondError_AppErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantHTTP int
		wantMsg  string
	}{
		{"bad_request", NewBadRequestError("requester_id is required", nil), BadRequestCode, http.StatusBadRequest, "requester_id is required"},
		{"not_found", NewNotFoundError("ticket"), NotFoundCode, http.StatusNotFound, "ticket not found"},
		{"forbidden", NewForbiddenError("no permission"), ForbiddenCode, http.StatusForbidden, "no permission"},
		{"conflict", NewConflictError("ticket", "duplicate"), ConflictCode, http.StatusConflict, "ticket already exists"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/test", nil)
			RespondError(c, tc.err, "fallback")
			assert.Equal(t, tc.wantHTTP, w.Code)
			var resp Response
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantMsg, resp.Message)
		})
	}
}

func TestRespondError_BusinessErrorPassthrough(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/test", nil)
	RespondError(c, NewBusinessError(4090, "conflict", ""), "fallback")
	assert.Equal(t, http.StatusConflict, w.Code)
	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 4090, resp.Code)
}

func TestRespondError_WrappedAppError(t *testing.T) {
	// errors.As 兼容包装：领域层 %w 包一层后仍能识别语义。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/test", nil)
	inner := NewNotFoundError("ticket")
	RespondError(c, fmt.Errorf("load: %w", inner), "fallback")
	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, NotFoundCode, resp.Code)
}

func TestRespondError_FallbackOnInternal(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/test", nil)
	raw := errors.New("pq: connection refused")
	RespondError(c, raw, "操作失败")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, InternalErrorCode, resp.Code)
	assert.Equal(t, "操作失败", resp.Message)
	assert.NotContains(t, w.Body.String(), "connection refused")
}

// ==================== E4-5 错误分类单一所有者 ====================

// statusForCode 收敛后，Fail 与 FailWithData 必须对每个业务码给出同一个 HTTP 状态。
// 这是把两份 switch 合并成唯一所有者之后才可能成立的不变量。
func TestStatusForCode_FailAndFailWithDataAgree(t *testing.T) {
	codes := []struct {
		code     int
		wantHTTP int
	}{
		{ParamErrorCode, http.StatusBadRequest},
		{ValidationError, http.StatusBadRequest},
		{BadRequestCode, http.StatusBadRequest},
		{AuthFailedCode, http.StatusUnauthorized},
		{UnauthorizedCode, http.StatusUnauthorized},
		{ForbiddenCode, http.StatusForbidden},
		{ToolPermissionDeniedCode, http.StatusForbidden},
		{NotFoundCode, http.StatusNotFound},
		{UnknownToolCode, http.StatusNotFound},
		{ConflictCode, http.StatusConflict},
		{UnprocessableEntityCode, http.StatusUnprocessableEntity},
		{InternalErrorCode, http.StatusInternalServerError},
		{ServiceUnavailableCode, http.StatusServiceUnavailable},
		{9999, http.StatusOK},
	}
	for _, tc := range codes {
		failW := httptest.NewRecorder()
		failC, _ := gin.CreateTestContext(failW)
		Fail(failC, tc.code, "m")

		withW := httptest.NewRecorder()
		withC, _ := gin.CreateTestContext(withW)
		FailWithData(withC, tc.code, "m", gin.H{"k": "v"})

		assert.Equal(t, tc.wantHTTP, failW.Code, "code=%d via Fail", tc.code)
		assert.Equal(t, tc.wantHTTP, withW.Code, "code=%d via FailWithData", tc.code)
	}
}

// FailWithErr 过去把所有错误一律压成 500/5001，领域错误自带的 HTTP 语义被丢掉；
// RespondError 又单独分类，同一个 service 错误经两个 helper 得到两种状态码。
// 收敛后 classifyError 是唯一分类者：状态码跟随领域语义，消息仍是 handler 的 publicMsg
// （领域文案可能是英文，不能在这里替换掉中文 UX 文案）。
func TestFailWithErr_ClassifiesDomainErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantHTTP int
	}{
		{"app_not_found", NewNotFoundError("ticket"), NotFoundCode, http.StatusNotFound},
		{"app_forbidden", NewForbiddenError("no permission"), ForbiddenCode, http.StatusForbidden},
		{"app_bad_request", NewBadRequestError("title is required", nil), BadRequestCode, http.StatusBadRequest},
		{"app_conflict", NewConflictError("ticket", "dup"), ConflictCode, http.StatusConflict},
		{"app_unavailable", NewAppError(ErrCodeInternal, "db down", http.StatusServiceUnavailable, nil), ServiceUnavailableCode, http.StatusServiceUnavailable},
		{"business_error", NewBusinessError(4220, "无法受理", "detail leaks"), UnprocessableEntityCode, http.StatusUnprocessableEntity},
		{"wrapped_app_not_found", fmt.Errorf("load: %w", NewNotFoundError("ticket")), NotFoundCode, http.StatusNotFound},
		{"version_conflict", NewVersionConflictError("工单", 7, 2, 5), ConflictCode, http.StatusConflict},
		{"unclassified", errors.New("pq: duplicate key value violates unique constraint"), InternalErrorCode, http.StatusInternalServerError},
		{"nil", nil, InternalErrorCode, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/test", nil)

			FailWithErr(c, tc.err, "操作失败")

			assert.Equal(t, tc.wantHTTP, w.Code)
			var resp Response
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, "操作失败", resp.Message, "消息必须是 handler 文案，不得透领域/驱动文本")
			assert.NotContains(t, w.Body.String(), "pq:")
			assert.NotContains(t, w.Body.String(), "duplicate key")
			assert.NotContains(t, w.Body.String(), "not found", "领域英文文案也不得替换掉 UX 文案")
			assert.NotContains(t, w.Body.String(), "detail leaks")
		})
	}
}

// 版本冲突除状态码外还要带载荷，客户端才能拿 currentVersion/serverVersion 做冲突处理；
// 只有一段中文文字等于把乐观锁退化成不可编程的错误。
func TestFailWithErr_VersionConflictCarriesPayload(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/tickets/7", nil)

	FailWithErr(c, NewVersionConflictError("工单", 7, 2, 5), "操作失败")

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, ConflictCode, resp.Code)
	assert.Equal(t, "操作失败", resp.Message)
	data, ok := resp.Data.(map[string]interface{})
	is := assert.New(t)
	is.True(ok, "版本冲突必须带 data 载荷，got=%v", resp.Data)
	is.Equal(float64(7), data["resourceId"])
	is.Equal(float64(2), data["currentVersion"])
	is.Equal(float64(5), data["serverVersion"])
}

// RespondError 与 FailWithErr 的唯一差异是消息权威：前者透出领域安全文案。
// 分类结果（业务码 + HTTP 状态）必须完全一致。
func TestRespondError_MatchesFailWithErrClassification(t *testing.T) {
	errs := []error{
		NewNotFoundError("ticket"),
		NewForbiddenError("no permission"),
		NewConflictError("ticket", "dup"),
		NewBusinessError(4220, "无法受理", ""),
		NewVersionConflictError("工单", 7, 2, 5),
		errors.New("pq: duplicate key"),
	}
	for _, err := range errs {
		respondW := httptest.NewRecorder()
		respondC, _ := gin.CreateTestContext(respondW)
		respondC.Request = httptest.NewRequest("GET", "/api/v1/test", nil)
		RespondError(respondC, err, "兜底")

		failW := httptest.NewRecorder()
		failC, _ := gin.CreateTestContext(failW)
		failC.Request = httptest.NewRequest("GET", "/api/v1/test", nil)
		FailWithErr(failC, err, "兜底")

		var r1, r2 Response
		assert.NoError(t, json.Unmarshal(respondW.Body.Bytes(), &r1))
		assert.NoError(t, json.Unmarshal(failW.Body.Bytes(), &r2))
		assert.Equal(t, respondW.Code, failW.Code, "err=%v", err)
		assert.Equal(t, r1.Code, r2.Code, "err=%v", err)
	}
}

// ErrorHandler 中间件此前用类型断言，只认顶层错误；被 %w 包装的业务拒绝掉进 500。
// 分类收敛到 classifyError（errors.As）后，包装同样识别。
func TestErrorHandler_Middleware_ClassifiesWrappedErrors(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantHTTP  int
		wantCode  int
		wantInMsg string
	}{
		{"wrapped_not_found", fmt.Errorf("svc: %w", NewNotFoundError("ticket")), http.StatusNotFound, NotFoundCode, "ticket not found"},
		{"top_level_business", NewBusinessError(4090, "状态冲突", ""), http.StatusConflict, ConflictCode, "状态冲突"},
		{"wrapped_conflict", fmt.Errorf("save: %w", NewVersionConflictError("工单", 7, 2, 5)), http.StatusConflict, ConflictCode, "版本冲突"},
		{"unclassified", errors.New("pq: connection refused"), http.StatusInternalServerError, InternalErrorCode, "内部服务器错误"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(ErrorHandler(zap.NewNop().Sugar()))
			engine.GET("/probe", func(c *gin.Context) {
				// c.Error 记录后由中间件统一分流，handler 不写响应。
				_ = c.Error(tc.err)
			})

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest("GET", "/probe", nil))

			assert.Equal(t, tc.wantHTTP, w.Code)
			var resp Response
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantCode, resp.Code)
			assert.Contains(t, resp.Message, tc.wantInMsg)
			assert.NotContains(t, w.Body.String(), "pq:")
		})
	}
}
