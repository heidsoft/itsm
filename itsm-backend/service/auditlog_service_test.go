package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupAuditLogTest(t *testing.T) (*ent.Client, *AuditLogService, context.Context) {
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	service := NewAuditLogService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createAuditLogTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createAuditLogTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 列表查询测试 ====================

func TestAuditLogService_ListAuditLogs_Empty(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Total)
	assert.Empty(t, response.Items)
}

func TestAuditLogService_ListAuditLogs_WithLogs(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createAuditLogTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建审计日志
	for i := 0; i < 5; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetUserID(testUser.ID).
			SetRequestID(fmt.Sprintf("req-%d", i)).
			SetIP("127.0.0.1").
			SetResource("ticket").
			SetAction("create").
			SetPath("/api/tickets").
			SetMethod("POST").
			SetStatusCode(200).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, response.Total)
	assert.Len(t, response.Items, 5)
}

func TestAuditLogService_ListAuditLogs_Pagination(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "page")
	require.NoError(t, err)

	// 创建15条审计日志
	for i := 0; i < 15; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetRequestID(fmt.Sprintf("req-page-%d", i)).
			SetIP("127.0.0.1").
			SetResource("ticket").
			SetAction("create").
			SetPath("/api/tickets").
			SetMethod("POST").
			SetStatusCode(200).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Items, 10)

	// 测试第二页
	req.Page = 2
	response, err = service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Items, 5)
}

func TestAuditLogService_ListAuditLogs_FilterByUser(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser1, err := createAuditLogTestUser(ctx, client, testTenant.ID, "filter1")
	require.NoError(t, err)

	testUser2, err := createAuditLogTestUser(ctx, client, testTenant.ID, "filter2")
	require.NoError(t, err)

	// 为两个用户创建审计日志
	for i := 0; i < 3; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetUserID(testUser1.ID).
			SetRequestID(fmt.Sprintf("req-user1-%d", i)).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod("POST").
			Save(ctx)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetUserID(testUser2.ID).
			SetRequestID(fmt.Sprintf("req-user2-%d", i)).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod("POST").
			Save(ctx)
		require.NoError(t, err)
	}

	userID := testUser1.ID
	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		UserID:   &userID,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, response.Total)
}

func TestAuditLogService_ListAuditLogs_FilterByResource(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "resource")
	require.NoError(t, err)

	// 创建不同资源的审计日志
	resources := []string{"ticket", "user", "ticket"}
	for _, resource := range resources {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource(resource).
			SetAction("create").
			SetPath("/api/" + resource).
			SetMethod("POST").
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		Resource: "ticket",
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total)
}

func TestAuditLogService_ListAuditLogs_FilterByAction(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "action")
	require.NoError(t, err)

	// 创建不同操作的审计日志
	actions := []string{"create", "update", "delete", "create"}
	for _, action := range actions {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ticket").
			SetAction(action).
			SetPath("/api/tickets").
			SetMethod("POST").
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		Action:   "create",
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total)
}

func TestAuditLogService_ListAuditLogs_FilterByMethod(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "method")
	require.NoError(t, err)

	// 创建不同方法的审计日志
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod(method).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		Method:   "POST",
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
}

func TestAuditLogService_ListAuditLogs_FilterByStatusCode(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "status")
	require.NoError(t, err)

	// 创建不同状态码的审计日志
	statusCodes := []int{200, 201, 400, 500}
	for _, code := range statusCodes {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod("POST").
			SetStatusCode(code).
			Save(ctx)
		require.NoError(t, err)
	}

	statusCode := 200
	req := &dto.ListAuditLogsRequest{
		Page:       1,
		PageSize:   10,
		StatusCode: &statusCode,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
}

func TestAuditLogService_ListAuditLogs_FilterByPath(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "path")
	require.NoError(t, err)

	// 创建不同路径的审计日志
	paths := []string{"/api/tickets", "/api/users", "/api/tickets/1"}
	for _, path := range paths {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("api").
			SetPath(path).
			SetMethod("GET").
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		Path:     "/api/tickets",
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total) // /api/tickets and /api/tickets/1
}

func TestAuditLogService_ListAuditLogs_FilterByTimeRange(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "time")
	require.NoError(t, err)

	// 创建审计日志
	now := time.Now()
	for i := 0; i < 5; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod("POST").
			SetCreatedAt(now.Add(-time.Duration(i) * time.Hour)).
			Save(ctx)
		require.NoError(t, err)
	}

	// 查询最近2小时的日志
	from := now.Add(-2 * time.Hour).Format(time.RFC3339)
	to := now.Add(1 * time.Hour).Format(time.RFC3339)

	req := &dto.ListAuditLogsRequest{
		Page:     1,
		PageSize: 10,
		From:     from,
		To:       to,
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, response.Total, 1)
}

func TestAuditLogService_ListAuditLogs_FilterByRequestID(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "reqid")
	require.NoError(t, err)

	// 创建审计日志
	_, err = client.AuditLog.Create().
		SetTenantID(testTenant.ID).
		SetRequestID("unique-request-id-123").
		SetResource("ticket").
		SetPath("/api/tickets").
		SetMethod("POST").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.AuditLog.Create().
		SetTenantID(testTenant.ID).
		SetRequestID("another-request-id-456").
		SetResource("ticket").
		SetPath("/api/tickets").
		SetMethod("POST").
		Save(ctx)
	require.NoError(t, err)

	req := &dto.ListAuditLogsRequest{
		Page:      1,
		PageSize:  10,
		RequestID: "unique-request-id-123",
	}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
}

func TestAuditLogService_ListAuditLogs_DefaultPagination(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "default")
	require.NoError(t, err)

	// 创建审计日志
	for i := 0; i < 25; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ticket").
			SetPath("/api/tickets").
			SetMethod("POST").
			Save(ctx)
		require.NoError(t, err)
	}

	// 使用默认分页
	req := &dto.ListAuditLogsRequest{}

	response, err := service.ListAuditLogs(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 25, response.Total)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.PageSize) // 默认值
}

// ==================== CI审计日志测试 ====================

func TestAuditLogService_GetCIAuditLogs_Empty(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "ciempty")
	require.NoError(t, err)

	response, err := service.GetCIAuditLogs(ctx, testTenant.ID, 99999, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, response.Total)
	assert.Empty(t, response.Items)
}

func TestAuditLogService_GetCIAuditLogs_WithLogs(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "ci")
	require.NoError(t, err)

	// 创建CI相关的审计日志
	_, err = client.AuditLog.Create().
		SetTenantID(testTenant.ID).
		SetResource("ci_123").
		SetAction("update").
		SetPath("/api/cmdb/ci/123").
		SetMethod("PUT").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.AuditLog.Create().
		SetTenantID(testTenant.ID).
		SetResource("ci_123").
		SetAction("delete").
		SetPath("/api/cmdb/ci/123").
		SetMethod("DELETE").
		Save(ctx)
	require.NoError(t, err)

	// 创建不相关的审计日志
	_, err = client.AuditLog.Create().
		SetTenantID(testTenant.ID).
		SetResource("ticket").
		SetAction("create").
		SetPath("/api/tickets").
		SetMethod("POST").
		Save(ctx)
	require.NoError(t, err)

	response, err := service.GetCIAuditLogs(ctx, testTenant.ID, 123, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total)
}

func TestAuditLogService_GetCIAuditLogs_Pagination(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()

	testTenant, err := createAuditLogTestTenant(ctx, client, "cipage")
	require.NoError(t, err)

	// 创建15条CI审计日志
	for i := 0; i < 15; i++ {
		_, err := client.AuditLog.Create().
			SetTenantID(testTenant.ID).
			SetResource("ci_456").
			SetAction("update").
			SetPath("/api/cmdb/ci/456").
			SetMethod("PUT").
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	response, err := service.GetCIAuditLogs(ctx, testTenant.ID, 456, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Items, 10)

	// 测试第二页
	response, err = service.GetCIAuditLogs(ctx, testTenant.ID, 456, 2, 10)
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Items, 5)
}

// L7：时间参数非法/倒挂/超长窗口此前只打一行 WARN 然后**跳过谓词**，
// 调用方以为筛了时间，实际仍是全租户 Count + 排序。现在必须显式 400。
func TestAuditLogService_ListAuditLogs_RejectsInvalidTimeWindow(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()
	tenant, err := createAuditLogTestTenant(ctx, client, "window")
	require.NoError(t, err)

	cases := []struct {
		name, from, to, want string
	}{
		{"非法 from", "not-a-date", "", "from 时间格式非法"},
		{"非法 to", "", "not-a-date", "to 时间格式非法"},
		{"倒挂", "2026-11-01T00:00:00Z", "2026-10-01T00:00:00Z", "to 不得早于 from"},
		{"超长窗口", "2020-01-01T00:00:00Z", "2026-11-01T00:00:00Z", "查询窗口不得超过"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.ListAuditLogs(ctx, &dto.ListAuditLogsRequest{
				Page: 1, PageSize: 20, From: tc.from, To: tc.to,
			}, tenant.ID)
			require.Error(t, err)
			var appErr *common.AppError
			require.True(t, errors.As(err, &appErr), "必须是 AppError 才能透传 400，实际 %T", err)
			assert.Equal(t, 400, appErr.HTTPStatus)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestAuditLogService_ListAuditLogs_RejectsDeepPage(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()
	tenant, err := createAuditLogTestTenant(ctx, client, "deeppage")
	require.NoError(t, err)

	_, err = service.ListAuditLogs(ctx, &dto.ListAuditLogsRequest{
		Page: maxAuditLogPage + 1, PageSize: 20,
	}, tenant.ID)
	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// 正常窗口不能被误伤：合法筛选仍要下推谓词并返回结果。
func TestAuditLogService_ListAuditLogs_ValidWindowStillWorks(t *testing.T) {
	client, service, ctx := setupAuditLogTest(t)
	defer client.Close()
	tenant, err := createAuditLogTestTenant(ctx, client, "validwindow")
	require.NoError(t, err)
	_, err = client.AuditLog.Create().
		SetTenantID(tenant.ID).SetResource("ticket").SetAction("create").
		SetPath("/api/v1/tickets").SetMethod("POST").
		SetCreatedAt(time.Now().Add(-24 * time.Hour)).Save(ctx)
	require.NoError(t, err)

	resp, err := service.ListAuditLogs(ctx, &dto.ListAuditLogsRequest{
		Page: 1, PageSize: 20,
		From: time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		To:   time.Now().UTC().Format(time.RFC3339),
	}, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
}
