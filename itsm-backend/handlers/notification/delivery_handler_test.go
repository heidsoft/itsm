// Package notification 的投递审计 HTTP 契约测试。
//
// 覆盖点：
//   - 列表按租户隔离（tenantB 的记录对 tenantA 不可见）；
//   - 过滤条件（channel / status）真实生效；
//   - 统计按 channel×status 聚合，成功率只按已定论投递计算；
//   - 重试只允许 failed，重试后生成新命令且旧记录保留；
//   - 非法参数返回 400 而不是 500。
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type deliveryFixture struct {
	client  *ent.Client
	handler *DeliveryHandler
	engine  *gin.Engine
	tenantA int
	tenantB int
	userA   *ent.User
	userB   *ent.User
}

func newDeliveryFixture(t *testing.T) *deliveryFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:notification_delivery_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("Delivery A").SetCode("delivery-a").
		SetDomain("delivery-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Delivery B").SetCode("delivery-b").
		SetDomain("delivery-b.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("delivery-a-user").SetEmail("delivery-a@example.com").
		SetName("Delivery A User").SetPasswordHash("hash").SetRole("admin").
		SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("delivery-b-user").SetEmail("delivery-b@example.com").
		SetName("Delivery B User").SetPasswordHash("hash").SetRole("admin").
		SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	handler := NewDeliveryHandler(client, zaptest.NewLogger(t).Sugar())
	engine := gin.New()
	// 用最小中间件模拟认证层：真实路由里 tenant_id 由 AuthMiddleware 写入。
	withTenant := func(tenantID int) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			c.Set("user_id", 1)
			c.Next()
		}
	}
	engine.GET("/a/deliveries", withTenant(tenantA.ID), handler.ListDeliveries)
	engine.GET("/a/deliveries/stats", withTenant(tenantA.ID), handler.GetDeliveryStats)
	engine.POST("/a/deliveries/:id/retry", withTenant(tenantA.ID), handler.RetryDelivery)
	engine.GET("/b/deliveries", withTenant(tenantB.ID), handler.ListDeliveries)

	return &deliveryFixture{
		client: client, handler: handler, engine: engine,
		tenantA: tenantA.ID, tenantB: tenantB.ID, userA: userA, userB: userB,
	}
}

// seedDelivery 直接落一条投递记录（模拟 commandbus 处理器写入的结果）。
func seedDelivery(t *testing.T, f *deliveryFixture, tenantID, recipientID int, channel, status string) (*ent.NotificationDelivery, *ent.OperationalCommand) {
	t.Helper()
	ctx := context.Background()
	command := f.client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType("notification.deliver").
		SetAggregateType("ticket").
		SetAggregateID(101).
		SetIdempotencyKey(fmt.Sprintf("seed-%d-%s-%d", tenantID, channel, time.Now().UnixNano())).
		SetPayload(map[string]interface{}{
			"resourceType": "ticket", "resourceId": 101,
			"recipientId": recipientID, "channel": channel,
			"type": "created", "content": "hello",
		}).
		SetStatus("succeeded").
		SaveX(ctx)
	delivery := f.client.NotificationDelivery.Create().
		SetTenantID(tenantID).
		SetOperationalCommandID(command.ID).
		SetRecipientID(recipientID).
		SetChannel(channel).
		SetTargetMasked("us****01").
		SetStatus(status).
		SetAttempt(1).
		SaveX(ctx)
	return delivery, command
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), target))
}

func TestListDeliveriesIsTenantScopedAndFilterable(t *testing.T) {
	f := newDeliveryFixture(t)
	seedDelivery(t, f, f.tenantA, f.userA.ID, "email", "sent")
	seedDelivery(t, f, f.tenantA, f.userA.ID, "feishu", "failed")
	seedDelivery(t, f, f.tenantB, f.userB.ID, "email", "sent")

	var all struct {
		Data struct {
			Items []DeliveryItem `json:"items"`
			Total int            `json:"total"`
		} `json:"data"`
	}
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/a/deliveries", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	decodeEnvelope(t, recorder, &all)
	require.Equal(t, 2, all.Data.Total, "tenantB 的记录不得出现在 tenantA 列表里")
	require.Len(t, all.Data.Items, 2)

	var failed struct {
		Data struct {
			Items []DeliveryItem `json:"items"`
			Total int            `json:"total"`
		} `json:"data"`
	}
	recorder = httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/a/deliveries?status=failed&channel=feishu", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	decodeEnvelope(t, recorder, &failed)
	require.Equal(t, 1, failed.Data.Total)
	require.Equal(t, "feishu", failed.Data.Items[0].Channel)
	require.Equal(t, "failed", failed.Data.Items[0].Status)
}

func TestListDeliveriesRejectsInvalidFilters(t *testing.T) {
	f := newDeliveryFixture(t)
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/a/deliveries?from=not-a-date", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestDeliveryStatsAggregatesByChannelAndStatus(t *testing.T) {
	f := newDeliveryFixture(t)
	seedDelivery(t, f, f.tenantA, f.userA.ID, "email", "sent")
	seedDelivery(t, f, f.tenantA, f.userA.ID, "email", "failed")
	seedDelivery(t, f, f.tenantA, f.userA.ID, "in_app", "sent")

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/a/deliveries/stats", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data DeliveryStatsResponse `json:"data"`
	}
	decodeEnvelope(t, recorder, &response)
	require.Equal(t, 3, response.Data.Total)
	require.Equal(t, 2, response.Data.ByStatus["sent"])
	require.Equal(t, 1, response.Data.ByStatus["failed"])
	require.Len(t, response.Data.ByChannel, 2)

	byChannel := map[string]ChannelDeliveryStat{}
	for _, stat := range response.Data.ByChannel {
		byChannel[stat.Channel] = stat
	}
	require.Equal(t, 0.5, byChannel["email"].SuccessRate, "email 1 成功 1 失败 → 50%")
	require.Equal(t, 1.0, byChannel["in_app"].SuccessRate)
}

func TestRetryDeliveryOnlyAcceptsFailedRecords(t *testing.T) {
	f := newDeliveryFixture(t)
	sent, _ := seedDelivery(t, f, f.tenantA, f.userA.ID, "email", "sent")
	failed, original := seedDelivery(t, f, f.tenantA, f.userA.ID, "feishu", "failed")

	// sent 记录拒绝重试
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", sent.ID), nil))
	require.Equal(t, http.StatusConflict, recorder.Code)

	// failed 记录重试成功：新命令入队，旧投递记录保留
	recorder = httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", failed.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			DeliveryID int `json:"deliveryId"`
			CommandID  int `json:"commandId"`
		} `json:"data"`
	}
	decodeEnvelope(t, recorder, &response)
	require.Equal(t, failed.ID, response.Data.DeliveryID)
	require.NotZero(t, response.Data.CommandID)
	require.NotEqual(t, original.ID, response.Data.CommandID)

	requeued := f.client.OperationalCommand.GetX(context.Background(), response.Data.CommandID)
	require.Equal(t, "notification.deliver", requeued.CommandType)
	require.Equal(t, f.tenantA, requeued.TenantID)
	require.Equal(t, original.Payload["content"], requeued.Payload["content"], "payload 必须原样复制")
	require.Contains(t, requeued.IdempotencyKey, ":retry:")

	// 旧记录仍是 failed，审计可追溯
	stillFailed := f.client.NotificationDelivery.GetX(context.Background(), failed.ID)
	require.Equal(t, "failed", stillFailed.Status)
}

func TestRetryDeliveryRejectsCrossTenantAccess(t *testing.T) {
	f := newDeliveryFixture(t)
	foreign, _ := seedDelivery(t, f, f.tenantB, f.userB.ID, "email", "failed")

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", foreign.ID), nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// 人工重放的幂等性：同一条 failed 投递被连点两次不能发两条通知。
//
// 旧记录刻意保持 failed（失败证据），界面上「重试」按钮永远可点，所以防重复
// 只能落在服务端：第二次点击必须 409，并且不能真的再入一条重试命令。
func TestRetryDeliveryRejectsDuplicateWhileRetryInFlight(t *testing.T) {
	f := newDeliveryFixture(t)
	failed, _ := seedDelivery(t, f, f.tenantA, f.userA.ID, "feishu", "failed")
	target := fmt.Sprintf("/a/deliveries/%d/retry", failed.ID)

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, target, nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, target, nil))
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "在途重试")

	retries := f.client.OperationalCommand.Query().
		Where(operationalcommand.IdempotencyKeyContains(":retry:")).CountX(context.Background())
	require.Equal(t, 1, retries, "重复点击不得产生第二条重试命令")
}

func TestRetryDeliveryRejectsAfterRetryAlreadySucceeded(t *testing.T) {
	f := newDeliveryFixture(t)
	failed, original := seedDelivery(t, f, f.tenantA, f.userA.ID, "email", "failed")
	// 模拟一次已被 worker 执行完的重试（命令 succeeded）。
	f.client.OperationalCommand.Create().
		SetTenantID(f.tenantA).SetCommandType("notification.deliver").
		SetAggregateType("ticket").SetAggregateID(101).
		SetIdempotencyKey(original.IdempotencyKey + ":retry:1").
		SetPayload(original.Payload).SetStatus("succeeded").SaveX(context.Background())

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", failed.ID), nil))
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "已重试成功")
}

// 上一次重试也失败了（dead_letter）说明消息确实没送出去，必须允许再试。
func TestRetryDeliveryAllowsRetryAfterPreviousRetryExhausted(t *testing.T) {
	f := newDeliveryFixture(t)
	failed, original := seedDelivery(t, f, f.tenantA, f.userA.ID, "feishu", "failed")
	f.client.OperationalCommand.Create().
		SetTenantID(f.tenantA).SetCommandType("notification.deliver").
		SetAggregateType("ticket").SetAggregateID(101).
		SetIdempotencyKey(original.IdempotencyKey + ":retry:1").
		SetPayload(original.Payload).SetStatus("dead_letter").SaveX(context.Background())

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", failed.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
}

// 前缀判定走 SQL LIKE，幂等键里的 `_` 会被当通配符：一个形态相近但并非派生的键
// 不能被误判成「已有在途重试」，否则这条失败投递永远重试不了。
func TestRetryDeliveryIgnoresLookalikeIdempotencyKeys(t *testing.T) {
	f := newDeliveryFixture(t)
	ctx := context.Background()
	command := f.client.OperationalCommand.Create().
		SetTenantID(f.tenantA).SetCommandType("notification.deliver").
		SetAggregateType("ticket").SetAggregateID(101).
		SetIdempotencyKey("ticket_42_created").
		SetPayload(map[string]interface{}{
			"resourceType": "ticket", "resourceId": 101, "recipientId": f.userA.ID,
			"channel": "email", "type": "created", "content": "hello",
		}).SetStatus("succeeded").SaveX(ctx)
	delivery := f.client.NotificationDelivery.Create().
		SetTenantID(f.tenantA).SetOperationalCommandID(command.ID).
		SetRecipientID(f.userA.ID).SetChannel("email").SetTargetMasked("a@****om").
		SetStatus("failed").SetAttempt(1).SaveX(ctx)
	// LIKE 'ticket_42_created:retry:%' 同样命中这一条，但它不是本投递的重试。
	f.client.OperationalCommand.Create().
		SetTenantID(f.tenantA).SetCommandType("notification.deliver").
		SetAggregateType("ticket").SetAggregateID(101).
		SetIdempotencyKey("ticketX42Xcreated:retry:1").
		SetPayload(command.Payload).SetStatus("pending").SaveX(ctx)

	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/a/deliveries/%d/retry", delivery.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
}
