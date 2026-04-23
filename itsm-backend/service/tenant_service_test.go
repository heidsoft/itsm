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

func setupTenantTest(t *testing.T) (*ent.Client, *TenantService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewTenantService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

// ==================== 创建租户测试 ====================

func TestTenantService_CreateTenant_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "测试公司",
		Code: "TEST-COMP",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, req.Name, tenant.Name)
	assert.Equal(t, req.Code, tenant.Code)
	assert.Equal(t, "active", tenant.Status)
}

func TestTenantService_CreateTenant_WithDomain(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	domain := "test-company.com"
	req := &dto.CreateTenantRequest{
		Name:   "测试公司",
		Code:   "TEST-DOMAIN",
		Type:   "standard",
		Domain: &domain,
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, domain, tenant.Domain)
}

func TestTenantService_CreateTenant_DuplicateCode(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "第一个租户",
		Code: "DUP-CODE",
		Type: "standard",
	}

	_, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	// 尝试创建相同代码的租户
	req2 := &dto.CreateTenantRequest{
		Name: "第二个租户",
		Code: "DUP-CODE",
		Type: "standard",
	}

	_, err = service.CreateTenant(ctx, req2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户代码已存在")
}

func TestTenantService_CreateTenant_WithExpiry(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30天后过期
	req := &dto.CreateTenantRequest{
		Name:      "试用租户",
		Code:      "TRIAL-001",
		Type:      "msp",
		ExpiresAt: &expiresAt,
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.NotNil(t, tenant.ExpiresAt)
}

// ==================== 获取租户测试 ====================

func TestTenantService_GetTenantByCode_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "测试租户",
		Code: "GET-BY-CODE",
		Type: "standard",
	}

	_, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	tenant, err := service.GetTenantByCode(ctx, "GET-BY-CODE")
	require.NoError(t, err)
	assert.Equal(t, "测试租户", tenant.Name)
}

func TestTenantService_GetTenantByCode_NotFound(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	_, err := service.GetTenantByCode(ctx, "NOT-EXIST")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户不存在")
}

func TestTenantService_GetTenant_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "测试租户",
		Code: "GET-BY-ID",
		Type: "standard",
	}

	created, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	tenant, err := service.GetTenant(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, tenant.ID)
	assert.Equal(t, "测试租户", tenant.Name)
}

func TestTenantService_GetTenant_NotFound(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	_, err := service.GetTenant(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户不存在")
}

// ==================== 更新租户状态测试 ====================

func TestTenantService_UpdateTenantStatus_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "状态测试租户",
		Code: "STATUS-TEST",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	err = service.UpdateTenantStatus(ctx, tenant.ID, "suspended")
	require.NoError(t, err)

	// 验证状态已更新
	updated, err := service.GetTenant(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", updated.Status)
}

// ==================== 列出租户测试 ====================

func TestTenantService_ListTenants_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建多个租户
	for i := 0; i < 5; i++ {
		req := &dto.CreateTenantRequest{
			Name: fmt.Sprintf("租户%d", i),
			Code: fmt.Sprintf("LIST-%03d", i),
			Type: "standard",
		}
		_, err := service.CreateTenant(ctx, req)
		require.NoError(t, err)
	}

	req := &dto.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
	}

	tenants, total, err := service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, tenants, 5)
}

func TestTenantService_ListTenants_Pagination(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建15个租户
	for i := 0; i < 15; i++ {
		req := &dto.CreateTenantRequest{
			Name: fmt.Sprintf("分页租户%d", i),
			Code: fmt.Sprintf("PAGE-%03d", i),
			Type: "standard",
		}
		_, err := service.CreateTenant(ctx, req)
		require.NoError(t, err)
	}

	// 测试第一页
	req := &dto.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
	}

	tenants, total, err := service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, tenants, 10)

	// 测试第二页
	req.Page = 2
	tenants, total, err = service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, tenants, 5)
}

func TestTenantService_ListTenants_FilterByStatus(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建租户并设置不同状态
	for i := 0; i < 3; i++ {
		req := &dto.CreateTenantRequest{
			Name: fmt.Sprintf("状态租户%d", i),
			Code: fmt.Sprintf("STATUS-F-%d", i),
			Type: "standard",
		}
		tenant, err := service.CreateTenant(ctx, req)
		require.NoError(t, err)

		if i == 1 {
			err = service.UpdateTenantStatus(ctx, tenant.ID, "suspended")
			require.NoError(t, err)
		}
	}

	req := &dto.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
		Status:   "suspended",
	}

	tenants, total, err := service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tenants, 1)
}

func TestTenantService_ListTenants_FilterByType(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建不同类型的租户
	types := []string{"standard", "msp", "standard"}
	for i, tenantType := range types {
		req := &dto.CreateTenantRequest{
			Name: fmt.Sprintf("类型租户%d", i),
			Code: fmt.Sprintf("TYPE-F-%d", i),
			Type: tenantType,
		}
		_, err := service.CreateTenant(ctx, req)
		require.NoError(t, err)
	}

	req := &dto.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
		Type:     "msp",
	}

	tenants, total, err := service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tenants, 1)
}

func TestTenantService_ListTenants_Search(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建租户
	req1 := &dto.CreateTenantRequest{
		Name: "科技公司",
		Code: "TECH-CO",
		Type: "standard",
	}
	_, err := service.CreateTenant(ctx, req1)
	require.NoError(t, err)

	req2 := &dto.CreateTenantRequest{
		Name: "金融公司",
		Code: "FIN-CO",
		Type: "standard",
	}
	_, err = service.CreateTenant(ctx, req2)
	require.NoError(t, err)

	// 搜索包含"科技"的租户
	req := &dto.ListTenantsRequest{
		Page:     1,
		PageSize: 10,
		Search:   "科技",
	}

	tenants, total, err := service.ListTenants(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, tenants, 1)
	assert.Equal(t, "科技公司", tenants[0].Name)
}

// ==================== 更新租户测试 ====================

func TestTenantService_UpdateTenant_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "原名称",
		Code: "UPDATE-TEST",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	newName := "新名称"
	newStatus := "suspended"
	updateReq := &dto.UpdateTenantRequest{
		Name:   &newName,
		Status: &newStatus,
	}

	updated, err := service.UpdateTenant(ctx, tenant.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, newStatus, updated.Status)
}

func TestTenantService_UpdateTenant_NotFound(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	name := "测试"
	req := &dto.UpdateTenantRequest{
		Name: &name,
	}

	_, err := service.UpdateTenant(ctx, 99999, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户不存在")
}

func TestTenantService_UpdateTenant_PartialUpdate(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "原名称",
		Code: "PARTIAL-TEST",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	// 只更新名称
	newName := "仅更新名称"
	updateReq := &dto.UpdateTenantRequest{
		Name: &newName,
	}

	updated, err := service.UpdateTenant(ctx, tenant.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, "active", updated.Status) // 状态保持不变
}

// ==================== 删除租户测试 ====================

func TestTenantService_DeleteTenant_Success(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	req := &dto.CreateTenantRequest{
		Name: "待删除租户",
		Code: "DEL-TEST",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	err = service.DeleteTenant(ctx, tenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = service.GetTenant(ctx, tenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户不存在")
}

func TestTenantService_DeleteTenant_NotFound(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	err := service.DeleteTenant(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户不存在")
}

func TestTenantService_DeleteTenant_WithUsers(t *testing.T) {
	client, service, ctx := setupTenantTest(t)
	defer client.Close()

	// 创建租户
	req := &dto.CreateTenantRequest{
		Name: "有用户租户",
		Code: "HAS-USERS",
		Type: "standard",
	}

	tenant, err := service.CreateTenant(ctx, req)
	require.NoError(t, err)

	// 创建用户
	_, err = client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 尝试删除有用户的租户
	err = service.DeleteTenant(ctx, tenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户下还有用户")
}
