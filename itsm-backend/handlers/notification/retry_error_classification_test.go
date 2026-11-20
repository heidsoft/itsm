package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// retryFaultFixture 自建带 DSN 的 SQLite 夹具：需要另开连接 DROP 表来制造真实 DB 故障。
type retryFaultFixture struct {
	client  *ent.Client
	engine  *gin.Engine
	dsn     string
	tenantA int
	tenantB int
}

func newRetryFaultFixture(t *testing.T) *retryFaultFixture {
	t.Helper()
	dsn := fmt.Sprintf("file:retry_fault_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	a := client.Tenant.Create().SetName("Retry A").SetCode("retry-a").
		SetDomain("retry-a.example.com").SetStatus("active").SaveX(ctx)
	b := client.Tenant.Create().SetName("Retry B").SetCode("retry-b").
		SetDomain("retry-b.example.com").SetStatus("active").SaveX(ctx)

	gin.SetMode(gin.TestMode)
	h := NewDeliveryHandler(client, zaptest.NewLogger(t).Sugar())
	engine := gin.New()
	withTenant := func(id int) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("tenant_id", id)
			c.Set("user_id", 1)
			c.Next()
		}
	}
	engine.POST("/deliveries/:id/retry", withTenant(a.ID), h.RetryDelivery)
	return &retryFaultFixture{client: client, engine: engine, dsn: dsn, tenantA: a.ID, tenantB: b.ID}
}

func (f *retryFaultFixture) seedFailedDelivery(t *testing.T, tenantID int) (*ent.NotificationDelivery, *ent.OperationalCommand) {
	t.Helper()
	ctx := context.Background()
	command := f.client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType("notification.deliver").
		SetAggregateType("ticket").
		SetAggregateID(101).
		SetIdempotencyKey(fmt.Sprintf("retry-fault-%d-%d", tenantID, time.Now().UnixNano())).
		SetPayload(map[string]interface{}{"type": "created", "content": "hello"}).
		SetStatus("succeeded").SaveX(ctx)
	delivery := f.client.NotificationDelivery.Create().
		SetTenantID(tenantID).
		SetOperationalCommandID(command.ID).
		SetRecipientID(1).
		SetChannel("email").
		SetTargetMasked("us****01").
		SetStatus("failed").
		SetAttempt(1).SaveX(ctx)
	return delivery, command
}

// TestRetryDelivery_DBFaultIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁。
// 旧实现把 OperationalCommand.Get 的**所有**错误与跨租户不匹配一起合并成 404：
//
//	if err != nil || command.TenantID != tenantID { common.NotFound(...) }
//
// DB 故障（如表缺失/连接抖动）被伪装成「原投递命令不存在」，driver 错误被丢弃。
// 正确语义：ent.NotFoundError 或跨租户不匹配 → 404（保留跨租户隐藏语义）；
// 其余 DB 故障 → 500/5001。
func TestRetryDelivery_DBFaultIsNotReportedAsNotFound(t *testing.T) {
	f := newRetryFaultFixture(t)
	delivery, _ := f.seedFailedDelivery(t, f.tenantA)

	// 独立连接删表：命令查询拿到 "no such table"，而投递记录查询仍正常。
	raw, err := sql.Open("sqlite3", f.dsn)
	require.NoError(t, err)
	t.Cleanup(func() { raw.Close() })
	_, err = raw.ExecContext(context.Background(), "DROP TABLE operational_commands")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/deliveries/%d/retry", delivery.ID), nil)
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.InternalErrorCode, resp.Code, "DB 故障业务码必须是 5001")
	assert.NotEqual(t, common.NotFoundCode, resp.Code, "不得误报 404 not found")
}

// TestRetryDelivery_CrossTenantCommandStill404 锁住跨租户隐藏语义：
// 命令存在但属于其他租户时，仍须维持 404，不得泄露其存在性。
func TestRetryDelivery_CrossTenantCommandStill404(t *testing.T) {
	f := newRetryFaultFixture(t)
	delivery, _ := f.seedFailedDelivery(t, f.tenantB) // 命令属于 tenantB，请求用 tenantA

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/deliveries/%d/retry", delivery.ID), nil)
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.NotFoundCode, resp.Code)
}
