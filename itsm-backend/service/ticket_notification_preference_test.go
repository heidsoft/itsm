package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketnotification"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// notificationPreferenceFixture 最小夹具：tenant + 用户 + 工单。
func notificationPreferenceFixture(t *testing.T) (*ent.Client, context.Context, int, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { _ = client.Close })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Pref Tenant").
		SetCode("pref-tenant").
		SetDomain("pref.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("pref-user").
		SetEmail("pref-user@example.com").
		SetName("Pref User").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("pref test ticket").
		SetDescription("for notification preference test").
		SetTicketNumber("TK-PREF-1").
		SetRequesterID(u.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return client, ctx, tenant.ID, u.ID, tk.ID
}

// TestSendNotification_RespectsPerEventPreference 核心契约：用户对某事件类型
// 显式关闭 email 渠道后，SendNotification 不应再创建该渠道的通知记录。
// （P0-3 修复：发送链路此前消费的是硬编码假聚合偏好，per-event 设置从未生效。）
func TestSendNotification_RespectsPerEventPreference(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationPreferenceFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	// 用户显式关闭 sla_warning 事件的 email 渠道
	_, err := client.NotificationPreference.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		SetEventType("sla_warning").
		SetEmailEnabled(false).
		SetInAppEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	err = svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID},
		Type:    "sla_warning",
		Channel: "email",
		Content: "sla warning via email",
	}, tenantID)
	require.NoError(t, err)

	count, err := client.TicketNotification.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count, "email 渠道被偏好关闭，不应创建通知记录")
}

// TestSendNotification_DefaultAllowsWhenNoPreference 无偏好记录 = 默认放行，
// 与 per-event 偏好 API 的默认语义一致。
func TestSendNotification_DefaultAllowsWhenNoPreference(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationPreferenceFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	err := svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID},
		Type:    "sla_warning",
		Channel: "email",
		Content: "default allow",
	}, tenantID)
	require.NoError(t, err)

	count, err := client.TicketNotification.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "无偏好记录应默认放行")
}

// TestSendNotification_ChannelScopedPreference 渠道隔离：关闭 email 不影响
// 同事件类型的 in_app 发送。
func TestSendNotification_ChannelScopedPreference(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationPreferenceFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	_, err := client.NotificationPreference.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		SetEventType("sla_warning").
		SetEmailEnabled(false).
		SetInAppEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	err = svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID},
		Type:    "sla_warning",
		Channel: "in_app",
		Content: "in app still allowed",
	}, tenantID)
	require.NoError(t, err)

	count, err := client.TicketNotification.Query().
		Where(ticketnotification.ChannelEQ("in_app")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "in_app 渠道不受 email 偏好影响")
}

// TestSendNotification_EventTypeScopedPreference 事件类型隔离：关闭 sla_warning
// 不影响 assigned 事件的发送。
func TestSendNotification_EventTypeScopedPreference(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationPreferenceFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	_, err := client.NotificationPreference.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		SetEventType("sla_warning").
		SetEmailEnabled(false).
		SetInAppEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	err = svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID},
		Type:    "assigned",
		Channel: "email",
		Content: "assigned event unaffected",
	}, tenantID)
	require.NoError(t, err)

	count, err := client.TicketNotification.Query().
		Where(ticketnotification.TypeEQ("assigned")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "assigned 事件不受 sla_warning 偏好影响")
}

// TestSendNotification_ConnectorChannelUsesPushToggle 连接器渠道（feishu 等）
// 按 push_enabled 判定。
func TestSendNotification_ConnectorChannelUsesPushToggle(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationPreferenceFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	_, err := client.NotificationPreference.Create().
		SetUserID(userID).
		SetTenantID(tenantID).
		SetEventType("sla_warning").
		SetEmailEnabled(true).
		SetInAppEnabled(true).
		SetPushEnabled(false).
		Save(ctx)
	require.NoError(t, err)

	err = svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID},
		Type:    "sla_warning",
		Channel: "feishu",
		Content: "connector channel blocked by push toggle",
	}, tenantID)
	require.NoError(t, err)

	count, err := client.TicketNotification.Query().
		Where(ticketnotification.ChannelEQ("feishu")).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count, "push_enabled=false 应拦截连接器渠道")
}
