package user

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// rankGuardService 独立最小 mock（不复用 handler_test.go 的 mockUserService，避免与
// 其在途改动耦合），仅实现 rank 防护路径触达的三个方法。
type rankGuardService struct {
	Service // 嵌入接口，未实现的方法被调用即 panic（测试不应触达）

	target           *ent.User
	deleteCalled     bool
	changeStatusCall bool
}

func (m *rankGuardService) GetUserByID(ctx context.Context, id, tenantID int) (*ent.User, error) {
	return m.target, nil
}

func (m *rankGuardService) DeleteUser(ctx context.Context, id, tenantID int) error {
	m.deleteCalled = true
	return nil
}

func (m *rankGuardService) ChangeUserStatus(ctx context.Context, id int, active bool, currentUserID, tenantID int) error {
	m.changeStatusCall = true
	return nil
}

func newRankGuardHandler(svc Service) *UserHandler {
	gin.SetMode(gin.TestMode)
	return NewHandler(svc, zap.NewNop().Sugar())
}

func doRankGuardJSON(h *UserHandler, method, body, callerRole string, callerTenant int) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/api/v1/users/1/status", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 100)
	c.Set("role", callerRole)
	c.Set("tenant_id", callerTenant)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	return w, c
}

// TestDeleteUser_RejectsHigherRankTarget 修复回归：admin 不得删除同租户三员/超管账号——
// 此前 DeleteUser 无 rank 防护。
func TestDeleteUser_RejectsHigherRankTarget(t *testing.T) {
	for _, targetRole := range []string{"super_admin", "sysadmin", "security_admin", "audit_admin"} {
		svc := &rankGuardService{target: &ent.User{ID: 1, TenantID: 1, Role: entuser.Role(targetRole), Active: true}}
		h := newRankGuardHandler(svc)

		w, c := doRankGuardJSON(h, http.MethodDelete, "", "admin", 1)
		h.DeleteUser(c)

		assert.Equal(t, http.StatusForbidden, w.Code, "admin 删除 %s 应被拒绝", targetRole)
		assert.False(t, svc.deleteCalled, "删除 %s 时不得触达 service", targetRole)
	}
}

// TestChangeUserStatus_RejectsHigherRankTarget admin 停用 sysadmin 同样必须 403。
func TestChangeUserStatus_RejectsHigherRankTarget(t *testing.T) {
	svc := &rankGuardService{target: &ent.User{ID: 1, TenantID: 1, Role: entuser.Role("sysadmin"), Active: true}}
	h := newRankGuardHandler(svc)

	w, c := doRankGuardJSON(h, http.MethodPut, `{"active":false}`, "admin", 1)
	h.ChangeUserStatus(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, svc.changeStatusCall, "停用 sysadmin 时不得触达 service")
}

// TestDeleteUser_AllowsLowerRankTarget 同级/低级目标照常放行（admin 删 agent）。
func TestDeleteUser_AllowsLowerRankTarget(t *testing.T) {
	svc := &rankGuardService{target: &ent.User{ID: 1, TenantID: 1, Role: entuser.Role("agent"), Active: true}}
	h := newRankGuardHandler(svc)

	w, c := doRankGuardJSON(h, http.MethodDelete, "", "admin", 1)
	h.DeleteUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.deleteCalled)
}
