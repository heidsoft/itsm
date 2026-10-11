package tenant

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// doJSONAs 按 caller role/tenant 构造请求上下文（doJSON 固定 super_admin，无法测归属校验）。
func doJSONAs(h *Handler, method, path, body string, callerRole string, callerTenant int) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)
	c.Set("role", callerRole)
	c.Set("tenant_id", callerTenant)
	if method != http.MethodPost {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
	}
	return w, c
}

// TestUpdateTenantStatus_RejectsDeletedViaStatusEndpoint 修复回归："deleted" 不得经
// status 接口设置——直写会绕过 DELETE 端点「租户下还有用户拒绝删除」守卫。
func TestUpdateTenantStatus_RejectsDeletedViaStatusEndpoint(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	w, c := doJSONAs(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"deleted"}`, "super_admin", 1)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "UpdateTenantStatus", mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateTenantStatus_RejectsCrossTenantWrite 修复回归：非 super_admin 的租户 admin
// 不得暂停/恢复其他租户（此前 tenant:write 持有者可跨租户管控任意租户）。
func TestUpdateTenantStatus_RejectsCrossTenantWrite(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	// caller 属于租户 2，目标租户 1
	w, c := doJSONAs(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"suspended"}`, "admin", 2)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	m.AssertNotCalled(t, "UpdateTenantStatus", mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateTenantStatus_AllowsSelfTenant 同租户合法暂停仍放行（系统租户守卫照常生效）。
func TestUpdateTenantStatus_AllowsSelfTenant(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)
	m.On("GetTenant", mock.Anything, 1).Return(sampleTenant(), nil)
	m.On("UpdateTenantStatus", mock.Anything, 1, "suspended").Return(nil)

	w, c := doJSONAs(h, http.MethodPut, "/api/v1/tenants/1/status", `{"status":"suspended"}`, "admin", 1)
	h.UpdateTenantStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	m.AssertExpectations(t)
}

// TestDeleteTenant_RejectsCrossTenant 跨租户软删除必须 403。
func TestDeleteTenant_RejectsCrossTenant(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	w, c := doJSONAs(h, http.MethodDelete, "/api/v1/tenants/1", "", "admin", 2)
	h.DeleteTenant(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	m.AssertNotCalled(t, "DeleteTenant", mock.Anything, mock.Anything)
}

// TestUpdateTenantSettings_RejectsStatusField 修复回归：settings 端点必须剥离 Status——
// 此前 default 平台租户的 admin 可经 PUT /system/settings 自 suspend 整站。
func TestUpdateTenantSettings_RejectsStatusField(t *testing.T) {
	m := &mockTenantService{}
	h := newTenantHandler(m)

	w, c := doJSONAs(h, http.MethodPut, "/api/v1/system/settings", `{"name":"X","status":"suspended"}`, "admin", 1)
	h.UpdateTenantSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	m.AssertNotCalled(t, "UpdateTenant", mock.Anything, mock.Anything, mock.Anything)

	var resp common.Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
}
