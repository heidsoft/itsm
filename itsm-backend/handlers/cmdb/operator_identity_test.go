package cmdb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAuthenticatedGinContext 走真实 AuthMiddleware 生成请求上下文，
// 这样断言的是「认证中间件到底写了哪些键」，而不是测试自己伪造的键名。
// 键名一旦在中间件与消费方之间分叉，本函数就会让用例失败。
func newAuthenticatedGinContext(t *testing.T, secret string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)

	token, err := middleware.GenerateAccessToken(42, "zhang.wei", "admin", 1, secret, time.Minute)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/cis/1/lifecycle", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	Auth := middleware.AuthMiddleware(secret)
	Auth(c)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return c
}

// TestOperatorContextCarriesAuthenticatedUsername 锁定 CMDB 变更历史里的操作人身份。
//
// handlers/cmdb 曾读取 gin 键 "user_name"，而 middleware/auth.go 写入的键是
// "username"，且没有任何地方写 "user_name"：CI 历史、导入/导出任务、版本回滚、
// 保存视图与生命周期流转全部把 operator_name 落成空串，审计只剩数字 ID。
func TestOperatorContextCarriesAuthenticatedUsername(t *testing.T) {
	const secret = "test-jwt-secret-for-cmdb-operator"
	c := newAuthenticatedGinContext(t, secret)

	// 前置事实：认证中间件写入的键名。
	assert.Equal(t, "zhang.wei", c.GetString("username"), "AuthMiddleware 写入的键名已变化，需同步所有消费方")
	assert.Empty(t, c.GetString("user_name"), "user_name 不是 AuthMiddleware 写入的键")

	id, name := service.OperatorFromContext((&ProductionService{}).operatorContext(c))
	assert.Equal(t, 42, id, "操作人 ID 必须来自认证上下文")
	assert.Equal(t, "zhang.wei", name, "操作人姓名必须来自认证上下文，不得静默落成空串")
}
