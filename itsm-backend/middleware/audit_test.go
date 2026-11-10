package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 用户删除/变更等 CRUD 操作必须落审计记录。
// AuditMiddleware 挂载在 tenant 组（router.go），用户路由位于该组内，
// 本测试证明 POST/PUT/DELETE /api/v1/users* 会产生持久化审计行，
// 且请求体中的密码字段被掩码。
func TestAuditMiddleware_UserCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()

	run := func(method, path, body string) {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		var req *http.Request
		if body != "" {
			req, _ = http.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(method, path, nil)
		}
		c.Request = req
		c.Set("tenant_id", 7)
		c.Set("user_id", 42)

		AuditMiddleware(client)(c)
		// 中间件已在 c.Next() 后写审计；无需真实 handler
	}

	t.Run("CreateUser Persists Audit With Masked Password", func(t *testing.T) {
		run("POST", "/api/v1/users", `{"username":"alice","password":"SuperSecret123!","role":"agent"}`)

		logs, err := client.AuditLog.Query().
			Where(auditlog.TenantIDEQ(7), auditlog.ActionEQ(string(ActionCreate))).
			All(context.Background())
		assert.NoError(t, err)
		if assert.Len(t, logs, 1) {
			assert.Equal(t, "users", logs[0].Resource)
			assert.Equal(t, 42, logs[0].UserID)
			assert.Equal(t, "POST", logs[0].Method)
			if assert.NotNil(t, logs[0].RequestBody) {
				assert.Contains(t, *logs[0].RequestBody, "alice")
				assert.NotContains(t, *logs[0].RequestBody, "SuperSecret123!")
			}
		}
	})

	t.Run("UpdateUser Persists Audit", func(t *testing.T) {
		run("PUT", "/api/v1/users/12", `{"name":"Alice Chen"}`)

		logs, err := client.AuditLog.Query().
			Where(auditlog.TenantIDEQ(7), auditlog.PathEQ("/api/v1/users/12")).
			All(context.Background())
		assert.NoError(t, err)
		if assert.Len(t, logs, 1) {
			assert.Equal(t, string(ActionUpdate), logs[0].Action)
			assert.Equal(t, "users", logs[0].Resource)
		}
	})

	t.Run("DeleteUser Persists Audit", func(t *testing.T) {
		run("DELETE", "/api/v1/users/13", "")

		logs, err := client.AuditLog.Query().
			Where(auditlog.TenantIDEQ(7), auditlog.ActionEQ(string(ActionDelete)), auditlog.PathEQ("/api/v1/users/13")).
			All(context.Background())
		assert.NoError(t, err)
		if assert.Len(t, logs, 1) {
			assert.Equal(t, "users", logs[0].Resource)
		}
	})

	t.Run("ResetPassword Persists Audit With Masked Body", func(t *testing.T) {
		run("PUT", "/api/v1/users/14/reset-password", `{"password":"NewPass456!"}`)

		logs, err := client.AuditLog.Query().
			Where(auditlog.TenantIDEQ(7), auditlog.PathEQ("/api/v1/users/14/reset-password")).
			All(context.Background())
		assert.NoError(t, err)
		if assert.Len(t, logs, 1) {
			assert.NotNil(t, logs[0].RequestBody)
			if logs[0].RequestBody != nil {
				assert.NotContains(t, *logs[0].RequestBody, "NewPass456!")
			}
		}
	})
}

// L6：敏感资源词表用下划线、路由用连字符，旧判定 strings.Contains 永不命中，
// 于是「查看审计日志」这个最该留痕的动作零记录；各类导出/下载 GET 同理。
func TestAuditMiddleware_SensitiveGETPathsAreAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		path       string
		wantAudit  bool
		wantAction string
	}{
		{"/api/v1/audit-logs", true, string(ActionSearch)},
		{"/api/v1/audit-logs/12", true, string(ActionView)},
		{"/api/v1/tickets/export", true, string(ActionExport)},
		{"/api/v1/reports/download?id=1", true, string(ActionExport)},
		{"/api/v1/users", true, string(ActionSearch)},
		{"/api/v1/tickets", false, ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodGet, tc.path, nil)
			assert.Equal(t, tc.wantAudit, shouldAuditRequest(c), "GET 敏感资源判定")
			if tc.wantAction != "" {
				action, _, _ := determineActionAndResource(c)
				assert.Equal(t, tc.wantAction, action)
			}
		})
	}
}

// L4：审计落库失败不能只留一行无上下文日志；至少要有可读取的失败计数，
// 让监控能发现「合规记录正在丢」。刻意不重试（append-only 无幂等键会重复行）。
func TestAuditMiddleware_WriteFailureIsCounted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:audit_write_failure?mode=memory&_fk=1")
	// 先正常落一条，确认夹具可用，再关闭客户端强制后续写入失败。
	warm := httptest.NewRecorder()
	c0, _ := gin.CreateTestContext(warm)
	c0.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"username":"warmup"}`))
	c0.Set("tenant_id", 7)
	c0.Set("user_id", 42)
	AuditMiddleware(client)(c0)
	before := AuditWriteFailureCount()
	require.NoError(t, client.Close())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"username":"after-close"}`))
	c.Set("tenant_id", 7)
	c.Set("user_id", 42)
	AuditMiddleware(client)(c)

	assert.Greater(t, AuditWriteFailureCount(), before, "审计写入失败必须被计数，不能静默吞掉")
	assert.Equal(t, 200, w.Code, "审计失败不得把业务请求改成 5xx")
}

// L7：旧实现 io.ReadAll 把整个请求体读进内存后再截断到 4096，
// 绕过了排在前面的 RequestSizeMiddleware。现在只窥视前 4097 字节，
// 但下游 handler 仍必须拿到完整 body（MultiReader 拼回）。
func TestAuditMiddleware_LargeBodyIsBoundedButDownstreamGetsFullBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:audit_large_body?mode=memory&_fk=1")
	defer client.Close()

	body := `{"password":"SuperSecret123!","payload":"` + strings.Repeat("x", 12000) + `"}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// 旧实现 io.ReadAll 会把整个 body 读进内存后再截断，审计正文长度与
	// 新实现完全一样，单看落库内容区分不出来。用字节探针锁住读取量本身。
	counted := &countingReadCloser{Reader: strings.NewReader(body)}
	c.Request.Body = counted
	c.Set("tenant_id", 7)
	c.Set("user_id", 42)

	AuditMiddleware(client)(c)

	assert.LessOrEqual(t, counted.read, maxAuditBodyBytes+1,
		"审计中间件最多只窥视 4097 字节，不得把整个请求体读进内存")
	downstream, err := io.ReadAll(c.Request.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(downstream), "审计不得把业务请求体截断")

	logs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(7)).All(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.NotNil(t, logs[0].RequestBody)
	assert.LessOrEqual(t, len(*logs[0].RequestBody), maxAuditBodyBytes+len("...[truncated]"))
	assert.Contains(t, *logs[0].RequestBody, "...[truncated]")
	assert.NotContains(t, *logs[0].RequestBody, "SuperSecret123!", "敏感字段仍须脱敏")
}

// L5：MSP 跨租户操作必须记在被操作的客户租户名下，
// 且保留操作者 MSP 租户与用户，否则客户租户审计视图永远看不到这些动作。
func TestAuditMiddleware_MSPOperationAttributedToCustomerTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:audit_msp_customer?mode=memory&_fk=1")
	defer client.Close()

	customer := 9
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/msp/tickets/1/assign", strings.NewReader(`{"assigneeId":3}`))
	c.Set("tenant_id", 7) // MSP 自身租户
	c.Set("user_id", 42)
	c.Set(MSPContextKey, &MSPContext{
		IsMSP: true, MSPUserID: 42,
		CustomerTenantID: &customer, AllowedCustomers: []int{customer},
	})

	AuditMiddleware(client)(c)

	logs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(customer)).All(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 1, "客户租户审计视图必须能看到 MSP 对自己的操作")
	require.NotNil(t, logs[0].RequestBody)
	assert.Contains(t, *logs[0].RequestBody, "msp_operator_tenant_id=7")
	assert.Contains(t, *logs[0].RequestBody, "msp_user_id=42")
	assert.Contains(t, *logs[0].RequestBody, "msp_scope=customer")

	own, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(7)).All(context.Background())
	require.NoError(t, err)
	assert.Empty(t, own, "已明确目标客户时不应再记到 MSP 自身租户")
}

func TestAuditMiddleware_MSPAggregateStaysInOperatorTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:audit_msp_aggregate?mode=memory&_fk=1")
	defer client.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/msp/allocations", strings.NewReader(`{}`))
	c.Set("tenant_id", 7)
	c.Set("user_id", 42)
	c.Set(MSPContextKey, &MSPContext{IsMSP: true, MSPUserID: 42, AllowedCustomers: []int{9}})

	AuditMiddleware(client)(c)

	logs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(7)).All(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 1, "没有明确目标客户的聚合操作仍归 MSP 自身租户")
	require.NotNil(t, logs[0].RequestBody)
	assert.Contains(t, *logs[0].RequestBody, "msp_scope=aggregate")
}

// countingReadCloser 统计审计中间件实际从请求体读取的字节数（L7 回归锁）。
type countingReadCloser struct {
	io.Reader
	read int
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.read += n
	return n, err
}

func (c *countingReadCloser) Close() error { return nil }
