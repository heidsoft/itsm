package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupApprovalChainTest(t *testing.T) (*ent.Client, *ApprovalChainService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent_approval_chain?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewApprovalChainService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createApprovalChainTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createApprovalChainTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 创建审批链测试 ====================

func TestApprovalChainService_CreateApprovalChain_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "create")
	require.NoError(t, err)

	req := &dto.ApprovalChainRequest{
		Name:        "技术审批链",
		Description: "用于技术评审的审批流程",
		EntityType:  "change",
		Status:      "active",
		Chain: []dto.ApprovalChainStepDTO{
			{
				Level:      1,
				ApproverID: testUser.ID,
				Role:       "tech_lead",
				Name:       "技术负责人",
				IsRequired: true,
			},
			{
				Level:      2,
				ApproverID: 0,
				Role:       "manager",
				Name:       "部门经理",
				IsRequired: true,
			},
		},
	}

	response, err := service.CreateApprovalChain(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Name, response.Name)
	assert.Equal(t, req.Description, response.Description)
	assert.Equal(t, req.EntityType, response.EntityType)
	assert.Equal(t, "active", response.Status)
	assert.Equal(t, testTenant.ID, response.TenantID)
	assert.Len(t, response.Chain, 2)
}

func TestApprovalChainService_CreateApprovalChain_WithDefaultStatus(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "default_status")
	require.NoError(t, err)

	req := &dto.ApprovalChainRequest{
		Name:        "默认状态审批链",
		Description: "测试默认状态",
		EntityType:  "ticket",
		Status:      "", // 空状态应该默认为 active
		Chain: []dto.ApprovalChainStepDTO{
			{
				Level:      1,
				ApproverID: 1,
				Role:       "approver",
				Name:       "审批人",
				IsRequired: true,
			},
		},
	}

	response, err := service.CreateApprovalChain(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "active", response.Status)
}

func TestApprovalChainService_CreateApprovalChain_MultipleSteps(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "multi_steps")
	require.NoError(t, err)

	testUser1, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "step1")
	require.NoError(t, err)

	testUser2, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "step2")
	require.NoError(t, err)

	req := &dto.ApprovalChainRequest{
		Name:        "多级审批链",
		Description: "包含多个审批级别的审批流程",
		EntityType:  "incident",
		Status:      "active",
		Chain: []dto.ApprovalChainStepDTO{
			{
				Level:      1,
				ApproverID: testUser1.ID,
				Role:       "l1_support",
				Name:       "一线支持",
				IsRequired: true,
			},
			{
				Level:      2,
				ApproverID: testUser2.ID,
				Role:       "l2_support",
				Name:       "二线支持",
				IsRequired: true,
			},
			{
				Level:      3,
				ApproverID: 0,
				Role:       "manager",
				Name:       "经理审批",
				IsRequired: false,
			},
		},
	}

	response, err := service.CreateApprovalChain(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Chain, 3)
}

// ==================== 获取审批链测试 ====================

func TestApprovalChainService_GetApprovalChain_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "get")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "get")
	require.NoError(t, err)

	// 先创建一个审批链
	created, err := client.ApprovalChain.Create().
		SetName("待获取审批链").
		SetDescription("测试获取功能").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试获取
	response, err := service.GetApprovalChain(ctx, created.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "待获取审批链", response.Name)
}

func TestApprovalChainService_GetApprovalChain_NotFound(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	_, err = service.GetApprovalChain(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

func TestApprovalChainService_GetApprovalChain_TenantMismatch(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant1, err := createApprovalChainTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createApprovalChainTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant1.ID, "cross_tenant")
	require.NoError(t, err)

	// 在租户1中创建审批链
	created, err := client.ApprovalChain.Create().
		SetName("租户1审批链").
		SetDescription("测试跨租户访问").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant1.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 租户2无法访问租户1的审批链
	_, err = service.GetApprovalChain(ctx, created.ID, testTenant2.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

// ==================== 列表查询审批链测试 ====================

func TestApprovalChainService_ListApprovalChains_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "list")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "list")
	require.NoError(t, err)

	// 创建多个审批链
	for i := 1; i <= 3; i++ {
		_, err = client.ApprovalChain.Create().
			SetName("审批链" + string(rune('A'+i-1))).
			SetDescription("测试列表查询").
			SetEntityType("ticket").
			SetChain([]schema.ApprovalChainStep{
				{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
			}).
			SetStatus("active").
			SetTenantID(testTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试列表查询
	chains, total, err := service.ListApprovalChains(ctx, testTenant.ID, "", "", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, chains, 3)
}

func TestApprovalChainService_ListApprovalChains_Empty(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	chains, total, err := service.ListApprovalChains(ctx, testTenant.ID, "", "", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, chains, 0)
}

func TestApprovalChainService_ListApprovalChains_TenantIsolation(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant1, err := createApprovalChainTestTenant(ctx, client, "list_tenant1")
	require.NoError(t, err)

	testTenant2, err := createApprovalChainTestTenant(ctx, client, "list_tenant2")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant1.ID, "list_iso")
	require.NoError(t, err)

	// 在租户1中创建审批链
	_, err = client.ApprovalChain.Create().
		SetName("租户1独有审批链").
		SetDescription("测试租户隔离").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant1.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 租户2应该看不到租户1的审批链
	chains, total, err := service.ListApprovalChains(ctx, testTenant2.ID, "", "", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, chains, 0)

	// 租户1可以看到自己的审批链
	chains, total, err = service.ListApprovalChains(ctx, testTenant1.ID, "", "", 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, chains, 1)
}

// ==================== 更新审批链测试 ====================

func TestApprovalChainService_UpdateApprovalChain_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "update")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "update")
	require.NoError(t, err)

	// 先创建一个审批链
	created, err := client.ApprovalChain.Create().
		SetName("待更新审批链").
		SetDescription("原描述").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 更新请求
	updateReq := &dto.ApprovalChainRequest{
		Name:        "更新后的审批链",
		Description: "新描述",
		EntityType:  "change",
		Status:      "inactive",
		Chain: []dto.ApprovalChainStepDTO{
			{
				Level:      1,
				ApproverID: testUser.ID,
				Role:       "updated_role",
				Name:       "更新后的审批人",
				IsRequired: true,
			},
			{
				Level:      2,
				ApproverID: testUser.ID,
				Role:       "another_role",
				Name:       "第二个审批人",
				IsRequired: false,
			},
		},
	}

	response, err := service.UpdateApprovalChain(ctx, created.ID, updateReq, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "更新后的审批链", response.Name)
	assert.Equal(t, "新描述", response.Description)
	assert.Equal(t, "change", response.EntityType)
	assert.Equal(t, "inactive", response.Status)
	assert.Len(t, response.Chain, 2)
}

func TestApprovalChainService_UpdateApprovalChain_NotFound(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "update_notfound")
	require.NoError(t, err)

	updateReq := &dto.ApprovalChainRequest{
		Name:       "不存在的更新",
		EntityType: "ticket",
		Chain:      []dto.ApprovalChainStepDTO{},
	}

	_, err = service.UpdateApprovalChain(ctx, 99999, updateReq, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

func TestApprovalChainService_UpdateApprovalChain_TenantMismatch(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant1, err := createApprovalChainTestTenant(ctx, client, "up_tenant1")
	require.NoError(t, err)

	testTenant2, err := createApprovalChainTestTenant(ctx, client, "up_tenant2")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant1.ID, "up_mismatch")
	require.NoError(t, err)

	// 在租户1中创建审批链
	created, err := client.ApprovalChain.Create().
		SetName("租户1审批链").
		SetDescription("测试").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant1.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 租户2尝试更新租户1的审批链
	updateReq := &dto.ApprovalChainRequest{
		Name:       "跨租户更新",
		EntityType: "ticket",
		Chain:      []dto.ApprovalChainStepDTO{},
	}

	_, err = service.UpdateApprovalChain(ctx, created.ID, updateReq, testTenant2.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

// ==================== 删除审批链测试 ====================

func TestApprovalChainService_DeleteApprovalChain_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "delete")
	require.NoError(t, err)

	// 先创建一个审批链
	created, err := client.ApprovalChain.Create().
		SetName("待删除审批链").
		SetDescription("测试删除").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 测试删除
	err = service.DeleteApprovalChain(ctx, created.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = service.GetApprovalChain(ctx, created.ID, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

func TestApprovalChainService_DeleteApprovalChain_NotFound(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "del_notfound")
	require.NoError(t, err)

	err = service.DeleteApprovalChain(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

func TestApprovalChainService_DeleteApprovalChain_TenantMismatch(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant1, err := createApprovalChainTestTenant(ctx, client, "del_tenant1")
	require.NoError(t, err)

	testTenant2, err := createApprovalChainTestTenant(ctx, client, "del_tenant2")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant1.ID, "del_mismatch")
	require.NoError(t, err)

	// 在租户1中创建审批链
	created, err := client.ApprovalChain.Create().
		SetName("租户1审批链").
		SetDescription("测试").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant1.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 租户2尝试删除租户1的审批链
	err = service.DeleteApprovalChain(ctx, created.ID, testTenant2.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链不存在")
}

// ==================== 审批链统计测试 ====================

func TestApprovalChainService_GetApprovalChainStats_Success(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "stats")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "stats")
	require.NoError(t, err)

	// 创建不同状态的审批链
	statuses := []string{"active", "active", "inactive"}
	for i, status := range statuses {
		_, err = client.ApprovalChain.Create().
			SetName("审批链统计" + string(rune('A'+i))).
			SetDescription("测试统计").
			SetEntityType("ticket").
			SetChain([]schema.ApprovalChainStep{
				{Level: 1, ApproverID: testUser.ID, Role: "test", Name: "测试", IsRequired: true},
			}).
			SetStatus(status).
			SetTenantID(testTenant.ID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试统计
	stats, err := service.GetApprovalChainStats(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats.Total)
}

// ==================== 边界情况测试 ====================

func TestApprovalChainService_CreateApprovalChain_EmptyChain(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "empty_chain")
	require.NoError(t, err)

	req := &dto.ApprovalChainRequest{
		Name:        "空审批链",
		Description: "测试空审批步骤",
		EntityType:  "ticket",
		Status:      "active",
		Chain:       []dto.ApprovalChainStepDTO{}, // 空链
	}

	response, err := service.CreateApprovalChain(ctx, req, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Chain, 0)
}

func TestApprovalChainService_CreateApprovalChain_AllEntityTypes(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "entity_types")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "entity_types")
	require.NoError(t, err)

	entityTypes := []string{"ticket", "incident", "change", "problem", "service_request"}

	for _, entityType := range entityTypes {
		req := &dto.ApprovalChainRequest{
			Name:        "审批链-" + entityType,
			Description: "测试实体类型",
			EntityType:  entityType,
			Status:      "active",
			Chain: []dto.ApprovalChainStepDTO{
				{
					Level:      1,
					ApproverID: testUser.ID,
					Role:       "test",
					Name:       "测试",
					IsRequired: true,
				},
			},
		}

		response, err := service.CreateApprovalChain(ctx, req, testTenant.ID)
		require.NoError(t, err)
		assert.Equal(t, entityType, response.EntityType)
	}
}

func TestApprovalChainService_UpdateApprovalChain_PartialUpdate(t *testing.T) {
	client, service, ctx := setupApprovalChainTest(t)
	defer client.Close()

	testTenant, err := createApprovalChainTestTenant(ctx, client, "partial")
	require.NoError(t, err)

	testUser, err := createApprovalChainTestUser(ctx, client, testTenant.ID, "partial")
	require.NoError(t, err)

	// 先创建一个审批链
	created, err := client.ApprovalChain.Create().
		SetName("原始名称").
		SetDescription("原始描述").
		SetEntityType("ticket").
		SetChain([]schema.ApprovalChainStep{
			{Level: 1, ApproverID: testUser.ID, Role: "original", Name: "原始审批人", IsRequired: true},
		}).
		SetStatus("active").
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 部分更新 - 只更新名称
	updateReq := &dto.ApprovalChainRequest{
		Name:        "更新后的名称",
		Description: "", // 空值 - 不更新描述
		EntityType:  "", // 空值 - 不更新实体类型
		Status:      "", // 空值 - 不更新状态
		Chain:       nil, // nil - 不更新审批链
	}

	response, err := service.UpdateApprovalChain(ctx, created.ID, updateReq, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的名称", response.Name)
	// 其他字段保持不变
	assert.Equal(t, "原始描述", response.Description)
	assert.Equal(t, "ticket", response.EntityType)
	assert.Equal(t, "active", response.Status)
}
