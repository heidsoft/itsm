package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupChangeTest(t *testing.T) (*ent.Client, *ChangeService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewChangeService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createChangeTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createChangeTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 创建变更测试 ====================

func TestChangeService_CreateChange_Success(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	plannedStart := time.Now().Add(24 * time.Hour)
	plannedEnd := time.Now().Add(48 * time.Hour)

	req := &dto.CreateChangeRequest{
		Title:              "升级生产服务器",
		Description:        "升级生产环境服务器的操作系统版本",
		Justification:      "安全更新需要",
		Type:               string(dto.ChangeTypeNormal),
		Priority:           string(dto.ChangePriorityHigh),
		ImpactScope:        string(dto.ChangeImpactMedium),
		RiskLevel:          string(dto.ChangeRiskMedium),
		ImplementationPlan: "1. 备份数据\n2. 执行升级\n3. 验证服务",
		RollbackPlan:       "恢复快照",
		PlannedStartDate:   &plannedStart,
		PlannedEndDate:     &plannedEnd,
	}

	response, err := service.CreateChange(ctx, req, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Title, response.Title)
	assert.Equal(t, req.Description, response.Description)
	assert.Equal(t, dto.ChangeStatusDraft, response.Status)
	assert.Equal(t, dto.ChangeType(req.Type), response.Type)
	assert.Equal(t, dto.ChangePriority(req.Priority), response.Priority)
	assert.Equal(t, testUser.ID, response.CreatedBy)
	assert.NotEmpty(t, response.CreatedAt)
}

func TestChangeService_CreateChange_WithAffectedCIs(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "ci")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "ci")
	require.NoError(t, err)

	req := &dto.CreateChangeRequest{
		Title:         "网络设备配置变更",
		Description:   "更新防火墙规则",
		Justification: "业务需求",
		Type:          string(dto.ChangeTypeNormal),
		Priority:      string(dto.ChangePriorityMedium),
		ImpactScope:   string(dto.ChangeImpactLow),
		RiskLevel:     string(dto.ChangeRiskLow),
		AffectedCIs:   []string{"CI-001", "CI-002", "CI-003"},
	}

	response, err := service.CreateChange(ctx, req, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)

	// 验证变更已创建
	assert.Equal(t, req.Title, response.Title)

	// 通过 GetChange 重新获取以验证 AffectedCIs 已存储
	savedChange, err := client.Change.Get(ctx, response.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"CI-001", "CI-002", "CI-003"}, savedChange.AffectedCis)
}

func TestChangeService_CreateChange_EmergencyChange(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "emergency")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "emergency")
	require.NoError(t, err)

	req := &dto.CreateChangeRequest{
		Title:         "紧急安全修复",
		Description:   "修复严重安全漏洞",
		Justification: "安全漏洞需要立即修复",
		Type:          string(dto.ChangeTypeEmergency),
		Priority:      string(dto.ChangePriorityCritical),
		ImpactScope:   string(dto.ChangeImpactHigh),
		RiskLevel:     string(dto.ChangeRiskHigh),
	}

	response, err := service.CreateChange(ctx, req, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, dto.ChangeTypeEmergency, response.Type)
	assert.Equal(t, dto.ChangePriorityCritical, response.Priority)
}

// ==================== 获取变更测试 ====================

func TestChangeService_GetChange_Success(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "get")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "get")
	require.NoError(t, err)

	// 创建测试变更
	testChange, err := client.Change.Create().
		SetTitle("Test Change").
		SetDescription("Test description").
		SetType("normal").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	response, err := service.GetChange(ctx, testChange.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, testChange.ID, response.ID)
	assert.Equal(t, testChange.Title, response.Title)
}

func TestChangeService_GetChange_NotFound(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	_, err = service.GetChange(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "change not found")
}

func TestChangeService_GetChange_TenantMismatch(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant1, err := createChangeTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createChangeTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant1.ID, "tenant")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("Test Change").
		SetDescription("Test description").
		SetType("normal").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// 使用另一个租户ID获取，应该失败
	_, err = service.GetChange(ctx, testChange.ID, testTenant2.ID)
	require.Error(t, err)
}

// ==================== 列出变更测试 ====================

func TestChangeService_ListChanges_Pagination(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建多个测试变更
	for i := 0; i < 15; i++ {
		_, err := client.Change.Create().
			SetTitle(fmt.Sprintf("Test Change %d", i+1)).
			SetDescription("Test description").
			SetType("normal").
			SetStatus("draft").
			SetPriority("medium").
			SetImpactScope("medium").
			SetRiskLevel("low").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试第一页
	response, err := service.ListChanges(ctx, testTenant.ID, 1, 10, "", "")
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Changes, 10)

	// 测试第二页
	response, err = service.ListChanges(ctx, testTenant.ID, 2, 10, "", "")
	require.NoError(t, err)
	assert.Equal(t, 15, response.Total)
	assert.Len(t, response.Changes, 5)
}

func TestChangeService_ListChanges_Filters(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "filter")
	require.NoError(t, err)

	// 创建不同状态和类型的变更
	statuses := []string{"draft", "pending", "approved"}
	types := []string{"normal", "emergency"}

	for _, status := range statuses {
		for _, changeType := range types {
			_, err := client.Change.Create().
				SetTitle(fmt.Sprintf("Change %s-%s", status, changeType)).
				SetDescription("Test description").
				SetType(changeType).
				SetStatus(status).
				SetPriority("medium").
				SetImpactScope("medium").
				SetRiskLevel("low").
				SetCreatedBy(testUser.ID).
				SetTenantID(testTenant.ID).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	// 测试状态过滤
	response, err := service.ListChanges(ctx, testTenant.ID, 1, 10, "draft", "")
	require.NoError(t, err)
	assert.Equal(t, 2, response.Total) // 2 种类型 × 1 个状态

	// 测试搜索过滤
	response, err = service.ListChanges(ctx, testTenant.ID, 1, 10, "", "emergency")
	require.NoError(t, err)
	// 搜索会在标题和描述中查找 "emergency"
	assert.GreaterOrEqual(t, response.Total, 3) // 至少 3 个包含 "emergency"
}

// ==================== 更新变更测试 ====================

func TestChangeService_UpdateChange_Success(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "update")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "update")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("Original Title").
		SetDescription("Original description").
		SetType("normal").
		SetStatus("draft").
		SetPriority("low").
		SetImpactScope("low").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	newTitle := "Updated Title"
	newPriority := dto.ChangePriorityHigh

	response, err := service.UpdateChange(ctx, testChange.ID, &dto.UpdateChangeRequest{
		Title:    &newTitle,
		Priority: &newPriority,
	}, testTenant.ID)

	require.NoError(t, err)
	assert.Equal(t, newTitle, response.Title)
	assert.Equal(t, newPriority, response.Priority)
}

func TestChangeService_UpdateChange_StatusTransition(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "status")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "status")
	require.NoError(t, err)

	// 使用 common 包中定义的状态常量，与服务层一致
	// 有效状态转换: draft -> submitted -> approved -> scheduled -> in_progress -> completed

	t.Run("draft -> submitted", func(t *testing.T) {
		testChange, err := client.Change.Create().
			SetTitle("Status Test").
			SetDescription("Test").
			SetType("normal").
			SetStatus("draft").
			SetPriority("medium").
			SetImpactScope("medium").
			SetRiskLevel("low").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		// 使用 common 包中的状态常量
		err = service.UpdateChangeStatus(ctx, testChange.ID, "submitted", testTenant.ID)
		require.NoError(t, err)

		// 验证状态已更新
		response, err := service.GetChange(ctx, testChange.ID, testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, dto.ChangeStatus("submitted"), response.Status)
	})

	t.Run("submitted -> approved", func(t *testing.T) {
		testChange, err := client.Change.Create().
			SetTitle("Status Test 2").
			SetDescription("Test").
			SetType("normal").
			SetStatus("submitted").
			SetPriority("medium").
			SetImpactScope("medium").
			SetRiskLevel("low").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		err = service.UpdateChangeStatus(ctx, testChange.ID, "approved", testTenant.ID)
		require.NoError(t, err)

		response, err := service.GetChange(ctx, testChange.ID, testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, dto.ChangeStatus("approved"), response.Status)
	})

	t.Run("approved -> scheduled", func(t *testing.T) {
		testChange, err := client.Change.Create().
			SetTitle("Status Test 3").
			SetDescription("Test").
			SetType("normal").
			SetStatus("approved").
			SetPriority("medium").
			SetImpactScope("medium").
			SetRiskLevel("low").
			SetCreatedBy(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		err = service.UpdateChangeStatus(ctx, testChange.ID, "scheduled", testTenant.ID)
		require.NoError(t, err)

		response, err := service.GetChange(ctx, testChange.ID, testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, dto.ChangeStatus("scheduled"), response.Status)
	})
}

// ==================== 删除变更测试 ====================

func TestChangeService_DeleteChange_Success(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "delete")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("To Be Deleted").
		SetDescription("Test description").
		SetType("normal").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	err = service.DeleteChange(ctx, testChange.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.Change.Get(ctx, testChange.ID)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

func TestChangeService_DeleteChange_NotFound(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	err = service.DeleteChange(ctx, 99999, testTenant.ID)
	require.Error(t, err)
}

// ==================== 变更统计测试 ====================

func TestChangeService_GetChangeStats(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "stats")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "stats")
	require.NoError(t, err)

	// 创建不同状态的变更
	statuses := []struct {
		status string
		count  int
	}{
		{"draft", 3},
		{"pending", 2},
		{"approved", 4},
		{"completed", 1},
	}

	for _, s := range statuses {
		for i := 0; i < s.count; i++ {
			_, err := client.Change.Create().
				SetTitle(fmt.Sprintf("Stats Test %s %d", s.status, i)).
				SetDescription("Test description").
				SetType("normal").
				SetStatus(s.status).
				SetPriority("medium").
				SetImpactScope("medium").
				SetRiskLevel("low").
				SetCreatedBy(testUser.ID).
				SetTenantID(testTenant.ID).
				Save(ctx)
			require.NoError(t, err)
		}
	}

	stats, err := service.GetChangeStats(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	totalExpected := 3 + 2 + 4 + 1 // 10
	assert.Equal(t, totalExpected, stats.Total)

	// 验证各状态计数
	assert.Equal(t, 2, stats.Pending)
	assert.Equal(t, 4, stats.Approved)
	assert.Equal(t, 1, stats.Completed)
}
