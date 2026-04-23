package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/notification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 测试设置辅助函数 ====================

func setupNotificationTest(t *testing.T) (*ent.Client, *NotificationService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	service := NewNotificationService(client)
	ctx := context.Background()
	return client, service, ctx
}

func createNotificationTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createNotificationTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser" + suffix).
		SetEmail("test" + suffix + "@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== 创建通知测试 ====================

func TestNotificationService_CreateNotification_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	req := &dto.CreateNotificationRequest{
		Title:    "测试通知",
		Message:  "这是一条测试通知消息",
		Type:     "system",
		UserID:   testUser.ID,
		TenantID: testTenant.ID,
	}

	response, err := service.CreateNotification(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Title, response.Title)
	assert.Equal(t, req.Message, response.Message)
	assert.Equal(t, req.Type, response.Type)
	assert.Equal(t, req.UserID, response.UserID)
	assert.False(t, response.Read)
}

func TestNotificationService_CreateNotification_WithAction(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "action")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "action")
	require.NoError(t, err)

	actionURL := "/tickets/123"
	actionText := "查看工单"

	req := &dto.CreateNotificationRequest{
		Title:      "新工单分配",
		Message:    "您有一个新的工单需要处理",
		Type:       "ticket",
		UserID:     testUser.ID,
		TenantID:   testTenant.ID,
		ActionURL:  &actionURL,
		ActionText: &actionText,
	}

	response, err := service.CreateNotification(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, actionURL, *response.ActionURL)
	assert.Equal(t, actionText, *response.ActionText)
}

func TestNotificationService_CreateNotification_UserNotFound(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	req := &dto.CreateNotificationRequest{
		Title:    "测试通知",
		Message:  "这是一条测试通知消息",
		Type:     "system",
		UserID:   99999,
		TenantID: testTenant.ID,
	}

	_, err = service.CreateNotification(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
}

// ==================== 获取通知列表测试 ====================

func TestNotificationService_GetNotifications_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建多个通知
	for i := 0; i < 5; i++ {
		_, err := client.Notification.Create().
			SetTitle("测试通知").
			SetMessage("测试消息").
			SetType("system").
			SetUserID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.GetNotificationsRequest{
		UserID:   testUser.ID,
		TenantID: testTenant.ID,
		Page:     1,
		Size:     10,
	}

	response, err := service.GetNotifications(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 5, response.Total)
	assert.Len(t, response.Notifications, 5)
}

func TestNotificationService_GetNotifications_Pagination(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "page")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "page")
	require.NoError(t, err)

	// 创建15个通知
	for i := 0; i < 15; i++ {
		_, err := client.Notification.Create().
			SetTitle("测试通知").
			SetMessage("测试消息").
			SetType("system").
			SetUserID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	req := &dto.GetNotificationsRequest{
		UserID:   testUser.ID,
		TenantID: testTenant.ID,
		Page:     1,
		Size:     10,
	}

	response, err := service.GetNotifications(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Notifications, 10)

	// 测试第二页
	req.Page = 2
	response, err = service.GetNotifications(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Notifications, 5)
}

func TestNotificationService_GetNotifications_FilterByType(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "filter")
	require.NoError(t, err)

	// 创建不同类型的通知
	types := []string{"system", "ticket", "approval"}
	for _, nt := range types {
		for i := 0; i < 3; i++ {
			_, err := client.Notification.Create().
				SetTitle("测试通知").
				SetMessage("测试消息").
				SetType(nt).
				SetUserID(testUser.ID).
				SetTenantID(testTenant.ID).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	req := &dto.GetNotificationsRequest{
		UserID:   testUser.ID,
		TenantID: testTenant.ID,
		Type:     "ticket",
		Page:     1,
		Size:     10,
	}

	response, err := service.GetNotifications(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 3, response.Total)
	assert.Len(t, response.Notifications, 3)
}

// ==================== 标记已读测试 ====================

func TestNotificationService_MarkNotificationRead_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "read")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "read")
	require.NoError(t, err)

	// 创建通知
	notification, err := client.Notification.Create().
		SetTitle("测试通知").
		SetMessage("测试消息").
		SetType("system").
		SetUserID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	req := &dto.MarkNotificationReadRequest{
		NotificationID: notification.ID,
		UserID:         testUser.ID,
		TenantID:       testTenant.ID,
	}

	err = service.MarkNotificationRead(ctx, req)
	require.NoError(t, err)

	// 验证已标记为已读
	updated, err := client.Notification.Get(ctx, notification.ID)
	require.NoError(t, err)
	assert.True(t, updated.Read)
}

func TestNotificationService_MarkNotificationRead_NotFound(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "readnotfound")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "readnotfound")
	require.NoError(t, err)

	req := &dto.MarkNotificationReadRequest{
		NotificationID: 99999,
		UserID:         testUser.ID,
		TenantID:       testTenant.ID,
	}

	err = service.MarkNotificationRead(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "通知不存在或无权限")
}

// ==================== 标记全部已读测试 ====================

func TestNotificationService_MarkAllNotificationsRead_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "allread")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "allread")
	require.NoError(t, err)

	// 创建多个未读通知
	for i := 0; i < 5; i++ {
		_, err := client.Notification.Create().
			SetTitle("测试通知").
			SetMessage("测试消息").
			SetType("system").
			SetUserID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetRead(false).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.MarkAllNotificationsReadRequest{
		UserID:   testUser.ID,
		TenantID: testTenant.ID,
	}

	err = service.MarkAllNotificationsRead(ctx, req)
	require.NoError(t, err)

	// 验证所有通知已标记为已读
	count, err := client.Notification.Query().
		Where(notification.UserID(testUser.ID)).
		Where(notification.TenantID(testTenant.ID)).
		Where(notification.Read(false)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ==================== 删除通知测试 ====================

func TestNotificationService_DeleteNotification_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "delete")
	require.NoError(t, err)

	// 创建通知
	notification, err := client.Notification.Create().
		SetTitle("测试通知").
		SetMessage("测试消息").
		SetType("system").
		SetUserID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	req := &dto.DeleteNotificationRequest{
		NotificationID: notification.ID,
		UserID:         testUser.ID,
		TenantID:       testTenant.ID,
	}

	err = service.DeleteNotification(ctx, req)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.Notification.Get(ctx, notification.ID)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

func TestNotificationService_DeleteNotification_NotFound(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "delnotfound")
	require.NoError(t, err)

	req := &dto.DeleteNotificationRequest{
		NotificationID: 99999,
		UserID:         testUser.ID,
		TenantID:       testTenant.ID,
	}

	err = service.DeleteNotification(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "通知不存在或无权限")
}

// ==================== 获取未读数量测试 ====================

func TestNotificationService_GetUnreadCount_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "unread")
	require.NoError(t, err)

	testUser, err := createNotificationTestUser(ctx, client, testTenant.ID, "unread")
	require.NoError(t, err)

	// 创建已读通知
	for i := 0; i < 3; i++ {
		_, err := client.Notification.Create().
			SetTitle("已读通知").
			SetMessage("测试消息").
			SetType("system").
			SetUserID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetRead(true).
			Save(ctx)
		require.NoError(t, err)
	}

	// 创建未读通知
	for i := 0; i < 5; i++ {
		_, err := client.Notification.Create().
			SetTitle("未读通知").
			SetMessage("测试消息").
			SetType("system").
			SetUserID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetRead(false).
			Save(ctx)
		require.NoError(t, err)
	}

	count, err := service.GetUnreadCount(ctx, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

// ==================== 创建系统通知测试 ====================

func TestNotificationService_CreateSystemNotification_Success(t *testing.T) {
	client, service, ctx := setupNotificationTest(t)
	defer client.Close()

	testTenant, err := createNotificationTestTenant(ctx, client, "sys")
	require.NoError(t, err)

	testUser1, err := createNotificationTestUser(ctx, client, testTenant.ID, "sys1")
	require.NoError(t, err)

	testUser2, err := createNotificationTestUser(ctx, client, testTenant.ID, "sys2")
	require.NoError(t, err)

	userIDs := []int{testUser1.ID, testUser2.ID}

	err = service.CreateSystemNotification(ctx, "系统维护通知", "系统将于今晚进行维护", "system", userIDs, testTenant.ID)
	require.NoError(t, err)

	// 验证通知已创建
	count, err := client.Notification.Query().
		Where(notification.TenantID(testTenant.ID)).
		Where(notification.Title("系统维护通知")).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
