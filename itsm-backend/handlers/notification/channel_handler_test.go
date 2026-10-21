// Package notification 的渠道诊断 HTTP 契约测试。
//
// 覆盖点：
//   - 站内信永远可投递（不依赖连接器），且排在第一位；
//   - 渠道健康 + 用户目标齐备才判定 deliverable，缺一必须给出可读的阻断原因；
//   - 收件目标脱敏，响应里不得出现完整邮箱/手机号；
//   - 查询只使用当前租户，避免诊断接口变成跨租户连通性探针。
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// fakeChannelCatalog 模拟连接器目录，避免测试真的拨号到 SMTP / 飞书。
type fakeChannelCatalog struct {
	configs     []connector.Config
	health      map[string]connector.HealthStatus
	lastTenant  int
	healthCalls int
}

func (f *fakeChannelCatalog) ListByTenant(tenantID int) []connector.Config {
	f.lastTenant = tenantID
	out := make([]connector.Config, 0, len(f.configs))
	for _, cfg := range f.configs {
		if cfg.TenantID == tenantID {
			out = append(out, cfg)
		}
	}
	return out
}

func (f *fakeChannelCatalog) HealthCheckTenant(_ context.Context, tenantID int) map[string]connector.HealthStatus {
	f.lastTenant = tenantID
	f.healthCalls++
	out := map[string]connector.HealthStatus{}
	for key, status := range f.health {
		out[key] = status
	}
	return out
}

type channelFixture struct {
	client  *ent.Client
	engine  *gin.Engine
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	catalog *fakeChannelCatalog
}

func newChannelFixture(t *testing.T) *channelFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:notification_channel_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("Channel A").SetCode("channel-a").
		SetDomain("channel-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Channel B").SetCode("channel-b").
		SetDomain("channel-b.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("channel-a-user").SetEmail("channel-a@example.com").
		SetName("Channel A User").SetPasswordHash("hash").SetRole("admin").
		SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	catalog := &fakeChannelCatalog{
		configs: []connector.Config{
			{TenantID: tenantA.ID, Name: "email", Provider: "standard", Type: connector.TypeEmail},
			{TenantID: tenantA.ID, Name: "feishu", Provider: "feishu", Type: connector.TypeIM},
			{TenantID: tenantB.ID, Name: "email", Provider: "standard", Type: connector.TypeEmail},
		},
		health: map[string]connector.HealthStatus{
			fmt.Sprintf("%d/email/standard", tenantA.ID): {OK: true, CheckedAt: time.Now()},
			fmt.Sprintf("%d/feishu/feishu", tenantA.ID):  {OK: false, Message: "app credentials rejected", CheckedAt: time.Now()},
		},
	}

	gin.SetMode(gin.TestMode)
	handler := NewChannelHandler(client, catalog, zaptest.NewLogger(t).Sugar())
	engine := gin.New()
	engine.GET("/channels", func(c *gin.Context) {
		c.Set("tenant_id", tenantA.ID)
		c.Set("user_id", userA.ID)
		c.Next()
	}, handler.GetChannels)

	return &channelFixture{client: client, engine: engine, tenantA: tenantA, tenantB: tenantB, userA: userA, catalog: catalog}
}

func (f *channelFixture) getChannels(t *testing.T) []ChannelStatus {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/channels", nil)
	f.engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body struct {
		Data struct {
			Items []ChannelStatus `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Data.Items
}

func TestChannelDiagnosticsReportsDeliverability(t *testing.T) {
	fx := newChannelFixture(t)
	items := fx.getChannels(t)
	require.Len(t, items, 3)

	// 站内信是兜底渠道：不依赖连接器，必须始终可投递且排第一。
	assert.Equal(t, "in_app", items[0].Channel)
	assert.True(t, items[0].Deliverable)
	assert.True(t, items[0].Healthy)

	byChannel := map[string]ChannelStatus{}
	for _, item := range items {
		byChannel[item.Channel] = item
	}

	email, ok := byChannel["email"]
	require.True(t, ok)
	assert.True(t, email.Healthy)
	assert.True(t, email.Deliverable)
	// 目标必须脱敏：响应里不能出现完整邮箱。
	assert.NotContains(t, email.RecipientTarget, "@")
	assert.Contains(t, email.RecipientTarget, "****")
	assert.Empty(t, email.BlockedReason)

	feishu, ok := byChannel["feishu"]
	require.True(t, ok)
	assert.False(t, feishu.Healthy)
	assert.False(t, feishu.Deliverable)
	// 该用户没有飞书 open_id：先命中「未配置目标」，这是更根本的原因。
	assert.Contains(t, feishu.BlockedReason, "未配置")
}

func TestChannelDiagnosticsBlockedByHealth(t *testing.T) {
	fx := newChannelFixture(t)
	// 给用户补上飞书 open_id：此时唯一阻断原因应变成「连接器不健康」。
	_, err := fx.client.User.UpdateOneID(fx.userA.ID).SetFeishuOpenID("ou_test_open_id").Save(context.Background())
	require.NoError(t, err)

	items := fx.getChannels(t)
	for _, item := range items {
		if item.Channel != "feishu" {
			continue
		}
		assert.False(t, item.Deliverable)
		assert.Equal(t, "连接器不健康", item.BlockedReason)
		assert.Equal(t, "app credentials rejected", item.HealthMessage)
		return
	}
	t.Fatal("feishu channel missing from diagnostics")
}

func TestChannelDiagnosticsScopesToCurrentTenant(t *testing.T) {
	fx := newChannelFixture(t)
	items := fx.getChannels(t)

	assert.Equal(t, fx.tenantA.ID, fx.catalog.lastTenant, "诊断必须按当前租户查询连接器")
	for _, item := range items {
		// tenantB 只装了 email；若串租户，会出现重复 email 或 tenantB 的实例。
		assert.NotEqual(t, "channel-b-user", item.RecipientTarget)
	}
	assert.Equal(t, 1, fx.catalog.healthCalls)
}
