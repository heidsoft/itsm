package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupIncidentTest(t *testing.T) (*ent.Client, *IncidentService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewIncidentService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createIncidentTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createIncidentTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 创建事件测试 ====================

func TestIncidentService_CreateIncident_Success(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	// 创建测试租户和用户
	testTenant, err := createIncidentTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	// 测试创建事件
	req := &dto.CreateIncidentRequest{
		Title:       "测试事件",
		Description: "这是一个测试事件的描述",
		Priority:    "high",
		Severity:    "medium",
		Category:    "performance",
		Source:      "manual",
	}

	response, err := service.CreateIncident(ctx, req, testTenant.ID, testUser.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Title, response.Title)
	assert.Equal(t, req.Priority, response.Priority)
	assert.Equal(t, req.Severity, response.Severity)
	assert.Equal(t, "new", response.Status)
	assert.NotEmpty(t, response.IncidentNumber)
	assert.Contains(t, response.IncidentNumber, "INC-")
	assert.Equal(t, testTenant.ID, response.TenantID)
}

func TestIncidentService_CreateIncident_WithOptionalFields(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "optional")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "optional")
	require.NoError(t, err)

	assignee, err := createIncidentTestUser(ctx, client, testTenant.ID, "assignee")
	require.NoError(t, err)

	detectedAt := time.Now().Add(-1 * time.Hour)

	req := &dto.CreateIncidentRequest{
		Title:       "带可选字段的事件",
		Description: "描述",
		Priority:    "critical",
		Severity:    "high",
		Category:    "security",
		Subcategory: "intrusion",
		AssigneeID:  &assignee.ID,
		Source:      "monitoring",
		DetectedAt:  &detectedAt,
		Metadata: map[string]interface{}{
			"source_ip": "192.168.1.100",
			"alert_id":  "ALT-001",
			"automated": true,
		},
	}

	response, err := service.CreateIncident(ctx, req, testTenant.ID, testUser.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "security", response.Category)
	assert.NotNil(t, response.AssigneeID)
	assert.Equal(t, assignee.ID, *response.AssigneeID)
}

// ==================== 获取事件测试 ====================

func TestIncidentService_GetIncident_Success(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "get")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "get")
	require.NoError(t, err)

	// 创建测试事件
	testIncident, err := client.Incident.Create().
		SetTitle("Test Incident").
		SetDescription("Test description").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-202401-000001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试获取事件
	response, err := service.GetIncident(ctx, testIncident.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, testIncident.ID, response.ID)
	assert.Equal(t, testIncident.Title, response.Title)
	assert.Equal(t, testIncident.Status, response.Status)
}

func TestIncidentService_GetIncident_NotFound(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	// 测试获取不存在的事件
	_, err = service.GetIncident(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incident not found")
}

func TestIncidentService_GetIncident_TenantMismatch(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant1, err := createIncidentTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createIncidentTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant1.ID, "tenant")
	require.NoError(t, err)

	// 在 tenant1 下创建事件
	testIncident, err := client.Incident.Create().
		SetTitle("Test Incident").
		SetDescription("Test description").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("INC-TENANT-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant1.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 尝试用 tenant2 获取事件，应该失败
	_, err = service.GetIncident(ctx, testIncident.ID, testTenant2.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incident not found")
}

// ==================== 列出事件测试 ====================

func TestIncidentService_ListIncidents_Pagination(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建多个测试事件
	for i := 0; i < 15; i++ {
		_, err := client.Incident.Create().
			SetTitle(fmt.Sprintf("Test Incident %d", i+1)).
			SetDescription("Test description").
			SetStatus("new").
			SetPriority("medium").
			SetSeverity("low").
			SetIncidentNumber(fmt.Sprintf("INC-LIST-%03d", i+1)).
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetDetectedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	responses, total, err := service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, responses, 10)

	// 测试第二页
	responses, total, err = service.ListIncidents(ctx, testTenant.ID, 2, 10, map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, responses, 5)
}

func TestIncidentService_ListIncidents_Filters(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "filter")
	require.NoError(t, err)

	// 创建不同状态和优先级的事件
	statuses := []string{"new", "in_progress", "resolved"}
	priorities := []string{"low", "medium", "high"}

	for i, status := range statuses {
		for j, priority := range priorities {
			_, err := client.Incident.Create().
				SetTitle(fmt.Sprintf("Incident %s-%s", status, priority)).
				SetDescription("Test description").
				SetStatus(status).
				SetPriority(priority).
				SetSeverity("medium").
				SetIncidentNumber(fmt.Sprintf("INC-FLT-%d%d", i, j)).
				SetReporterID(testUser.ID).
				SetTenantID(testTenant.ID).
				SetDetectedAt(time.Now()).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	// 测试状态过滤
	responses, total, err := service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"status": "new",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total) // 3 个优先级 × 1 个状态
	assert.Len(t, responses, 3)

	// 测试优先级过滤
	_, total, err = service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"priority": "high",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total) // 3 个状态 × 1 个优先级

	// 测试组合过滤
	_, total, err = service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"status":   "in_progress",
		"priority": "high",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestIncidentService_ListIncidents_KeywordSearch(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "search")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "search")
	require.NoError(t, err)

	// 创建带有关键词的事件
	_, err = client.Incident.Create().
		SetTitle("数据库连接失败").
		SetDescription("生产环境数据库无法连接").
		SetStatus("new").
		SetPriority("critical").
		SetSeverity("high").
		SetIncidentNumber("INC-DB-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Incident.Create().
		SetTitle("网络延迟问题").
		SetDescription("用户反馈网络响应缓慢").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber("INC-NET-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 搜索关键词 "数据库"
	responses, total, err := service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"keyword": "数据库",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Contains(t, responses[0].Title, "数据库")

	// 搜索关键词 "网络"
	responses, total, err = service.ListIncidents(ctx, testTenant.ID, 1, 10, map[string]interface{}{
		"keyword": "网络",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Contains(t, responses[0].Title, "网络")
}

// ==================== 更新事件测试 ====================

func TestIncidentService_UpdateIncident_Success(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "update")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "update")
	require.NoError(t, err)

	testIncident, err := client.Incident.Create().
		SetTitle("Original Title").
		SetDescription("Original description").
		SetStatus("new").
		SetPriority("low").
		SetSeverity("low").
		SetIncidentNumber("INC-UPD-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetVersion(1).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试更新
	newTitle := "Updated Title"
	newPriority := "high"

	response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
		Title:    &newTitle,
		Priority: &newPriority,
		Version:  0, // 跳过版本检查
	}, testTenant.ID)

	require.NoError(t, err)
	assert.Equal(t, newTitle, response.Title)
	assert.Equal(t, newPriority, response.Priority)
	assert.Equal(t, 2, response.Version) // 版本号自动 +1
}

// ==================== 乐观锁版本控制测试 ====================

func TestIncidentService_UpdateIncident_VersionControl(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "version")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "version")
	require.NoError(t, err)

	testIncident, err := client.Incident.Create().
		SetTitle("Version Test").
		SetDescription("Test description").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber("INC-VER-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetVersion(1).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	t.Run("版本匹配时更新成功", func(t *testing.T) {
		newTitle := "Updated with correct version"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Title:   &newTitle,
			Version: 1, // 匹配当前版本
			Force:   false,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newTitle, response.Title)
		assert.Equal(t, 2, response.Version) // 版本号自动 +1
	})

	t.Run("版本不匹配时返回冲突错误", func(t *testing.T) {
		// 当前版本应该是 2
		newTitle := "Should fail"
		_, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Title:   &newTitle,
			Version: 1, // 使用旧版本号
			Force:   false,
		}, testTenant.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "版本冲突")

		// 检查是否是 VersionConflictError 类型
		var conflictErr *common.VersionConflictError
		assert.ErrorAs(t, err, &conflictErr)
		assert.Equal(t, testIncident.ID, conflictErr.ResourceID)
		assert.Equal(t, 1, conflictErr.CurrentVersion)
		assert.Equal(t, 2, conflictErr.ServerVersion)
	})

	t.Run("Force=true 忽略版本检查", func(t *testing.T) {
		newTitle := "Force Update"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Title:   &newTitle,
			Version: 1,    // 旧版本号
			Force:   true, // 强制更新
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newTitle, response.Title)
	})

	t.Run("Version=0 跳过版本检查", func(t *testing.T) {
		newTitle := "No Version Check"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Title:   &newTitle,
			Version: 0, // 跳过版本检查
			Force:   false,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newTitle, response.Title)
	})
}

// ==================== 状态转换测试 ====================

func TestIncidentService_UpdateIncident_StatusTransition(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "status")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "status")
	require.NoError(t, err)

	t.Run("有效状态转换 new -> in_progress", func(t *testing.T) {
		testIncident, err := client.Incident.Create().
			SetTitle("Status Test 1").
			SetDescription("Test description").
			SetStatus("new").
			SetPriority("medium").
			SetSeverity("medium").
			SetIncidentNumber("INC-ST-001").
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetDetectedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "in_progress"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Status:  &newStatus,
			Version: 0,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
	})

	t.Run("有效状态转换 in_progress -> resolved", func(t *testing.T) {
		testIncident, err := client.Incident.Create().
			SetTitle("Status Test 2").
			SetDescription("Test description").
			SetStatus("in_progress").
			SetPriority("medium").
			SetSeverity("medium").
			SetIncidentNumber("INC-ST-002").
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetDetectedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "resolved"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Status:  &newStatus,
			Version: 0,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
		assert.NotNil(t, response.ResolvedAt)
	})

	t.Run("有效状态转换 resolved -> closed", func(t *testing.T) {
		resolvedAt := time.Now().Add(-1 * time.Hour)
		testIncident, err := client.Incident.Create().
			SetTitle("Status Test 3").
			SetDescription("Test description").
			SetStatus("resolved").
			SetPriority("medium").
			SetSeverity("medium").
			SetIncidentNumber("INC-ST-003").
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetDetectedAt(time.Now()).
			SetResolvedAt(resolvedAt).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "closed"
		response, err := service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Status:  &newStatus,
			Version: 0,
		}, testTenant.ID)

		require.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
		assert.NotNil(t, response.ClosedAt)
	})

	t.Run("无效状态转换", func(t *testing.T) {
		testIncident, err := client.Incident.Create().
			SetTitle("Status Test 4").
			SetDescription("Test description").
			SetStatus("new").
			SetPriority("medium").
			SetSeverity("medium").
			SetIncidentNumber("INC-ST-004").
			SetReporterID(testUser.ID).
			SetTenantID(testTenant.ID).
			SetDetectedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		newStatus := "closed" // new 不能直接到 closed
		_, err = service.UpdateIncident(ctx, testIncident.ID, &dto.UpdateIncidentRequest{
			Status:  &newStatus,
			Version: 0,
		}, testTenant.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status transition")
	})
}

// ==================== 删除事件测试 ====================

func TestIncidentService_DeleteIncident_Success(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "delete")
	require.NoError(t, err)

	testIncident, err := client.Incident.Create().
		SetTitle("To Be Deleted").
		SetDescription("Test description").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber("INC-DEL-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 创建关联的事件记录
	_, err = client.IncidentEvent.Create().
		SetIncidentID(testIncident.ID).
		SetEventType("creation").
		SetEventName("事件创建").
		SetDescription("Test event created").
		SetStatus("active").
		SetSeverity("info").
		SetTenantID(testTenant.ID).
		SetOccurredAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试删除
	err = service.DeleteIncident(ctx, testIncident.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.Incident.Get(ctx, testIncident.ID)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))

	// 验证关联的事件记录也被删除
	events, err := client.IncidentEvent.Query().
		Where(incidentevent.IncidentIDEQ(testIncident.ID)).
		All(ctx)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestIncidentService_DeleteIncident_NotFound(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	// 测试删除不存在的事件
	err = service.DeleteIncident(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incident not found")
}

// TestIncidentService_DeleteIncident_CascadeTenantIsolation verifies that
// tenant 2 cannot delete an incident belonging to tenant 1 (cross-tenant access denied)
func TestIncidentService_DeleteIncident_CascadeTenantIsolation(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant1, err := createIncidentTestTenant(ctx, client, "cascade1")
	require.NoError(t, err)

	testTenant2, err := createIncidentTestTenant(ctx, client, "cascade2")
	require.NoError(t, err)

	testUser1, err := createIncidentTestUser(ctx, client, testTenant1.ID, "cascade1")
	require.NoError(t, err)

	testIncident, err := client.Incident.Create().
		SetTitle("Tenant 1 Incident").
		SetDescription("Should not be deletable by Tenant 2").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber("INC-CASCADE-001").
		SetReporterID(testUser1.ID).
		SetTenantID(testTenant1.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// Create cascade records (IncidentEvent, IncidentAlert, IncidentMetric)
	_, err = client.IncidentEvent.Create().
		SetIncidentID(testIncident.ID).
		SetEventType("creation").
		SetEventName("事件创建").
		SetDescription("Test event").
		SetStatus("active").
		SetSeverity("info").
		SetTenantID(testTenant1.ID).
		SetOccurredAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.IncidentAlert.Create().
		SetIncidentID(testIncident.ID).
		SetAlertType("warning").
		SetAlertName("Test Alert").
		SetMessage("Test alert message").
		SetStatus("triggered").
		SetSeverity("medium").
		SetTenantID(testTenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.IncidentMetric.Create().
		SetIncidentID(testIncident.ID).
		SetMetricType("test").
		SetMetricName("test_metric").
		SetMetricValue(100.0).
		SetTenantID(testTenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Tenant 2 tries to delete Tenant 1's incident - should fail with cross-tenant error
	err = service.DeleteIncident(ctx, testIncident.ID, testTenant2.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied", "Expected cross-tenant access denied error")

	// Verify incident still exists (not deleted)
	incident, err := client.Incident.Get(ctx, testIncident.ID)
	require.NoError(t, err)
	assert.Equal(t, testTenant1.ID, incident.TenantID, "Incident should still belong to Tenant 1")

	// Verify cascade records still exist
	events, err := client.IncidentEvent.Query().Where(incidentevent.IncidentIDEQ(testIncident.ID)).All(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, events, "IncidentEvent should not be deleted")

	alerts, err := client.IncidentAlert.Query().Where(incidentalert.IncidentIDEQ(testIncident.ID)).All(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, alerts, "IncidentAlert should not be deleted")

	metrics, err := client.IncidentMetric.Query().Where(incidentmetric.IncidentIDEQ(testIncident.ID)).All(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, metrics, "IncidentMetric should not be deleted")
}

// ==================== 事件活动记录测试 ====================

func TestIncidentService_CreateIncidentEvent_Success(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "event")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "event")
	require.NoError(t, err)

	testIncident, err := client.Incident.Create().
		SetTitle("Event Test").
		SetDescription("Test description").
		SetStatus("new").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber("INC-EVT-001").
		SetReporterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 创建事件记录
	response, err := service.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  testIncident.ID,
		EventType:   "status_change",
		EventName:   "状态变更",
		Description: "事件状态从 new 变更为 in_progress",
		Status:      "active",
		Severity:    "info",
		Source:      "system",
		Data: map[string]interface{}{
			"old_status": "new",
			"new_status": "in_progress",
		},
	}, testTenant.ID)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, testIncident.ID, response.IncidentID)
	assert.Equal(t, "status_change", response.EventType)
	assert.Equal(t, "状态变更", response.EventName)
}

// ==================== 事件统计测试 ====================

func TestIncidentService_GetIncidentStats(t *testing.T) {
	client, service, ctx := setupIncidentTest(t)
	defer client.Close()

	testTenant, err := createIncidentTestTenant(ctx, client, "stats")
	require.NoError(t, err)

	testUser, err := createIncidentTestUser(ctx, client, testTenant.ID, "stats")
	require.NoError(t, err)

	// 创建不同状态的事件
	statuses := []struct {
		status   string
		priority string
		severity string
		count    int
	}{
		{"new", "critical", "critical", 2},
		{"in_progress", "high", "high", 3},
		{"resolved", "medium", "medium", 4},
		{"closed", "low", "low", 1},
	}

	for _, s := range statuses {
		for i := 0; i < s.count; i++ {
			incidentBuilder := client.Incident.Create().
				SetTitle(fmt.Sprintf("Stats Test %s %d", s.status, i)).
				SetDescription("Test description").
				SetStatus(s.status).
				SetPriority(s.priority).
				SetSeverity(s.severity).
				SetIncidentNumber(fmt.Sprintf("INC-STATS-%s-%d", s.status, i)).
				SetReporterID(testUser.ID).
				SetTenantID(testTenant.ID).
				SetDetectedAt(time.Now())

			if s.status == "resolved" {
				incidentBuilder.SetResolvedAt(time.Now())
			} else if s.status == "closed" {
				incidentBuilder.SetResolvedAt(time.Now().Add(-1 * time.Hour))
				incidentBuilder.SetClosedAt(time.Now())
			}

			_, err := incidentBuilder.Save(ctx)
			require.NoError(t, err)
		}
	}

	// 获取统计
	stats, err := service.GetIncidentStats(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	// 验证统计数据
	totalExpected := 2 + 3 + 4 + 1 // 10
	assert.Equal(t, totalExpected, stats.TotalIncidents)

	// open incidents = new + in_progress
	openExpected := 2 + 3 // 5
	assert.Equal(t, openExpected, stats.OpenIncidents)

	// critical incidents
	assert.Equal(t, 2, stats.CriticalIncidents)
}
