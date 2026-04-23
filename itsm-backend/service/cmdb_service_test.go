package service

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 测试设置辅助函数 ====================

func setupCMDBTest(t *testing.T) (*ent.Client, *CMDBService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	service := NewCMDBService(client)
	ctx := context.Background()
	return client, service, ctx
}

func createCMDBTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createCMDBTestCIType(ctx context.Context, client *ent.Client, tenantID int, name string) (*ent.CIType, error) {
	return client.CIType.Create().
		SetName(name).
		SetTenantID(tenantID).
		SetIsActive(true).
		Save(ctx)
}

func createTestCI(ctx context.Context, client *ent.Client, tenantID int, ciTypeID int, name string) (*ent.ConfigurationItem, error) {
	return client.ConfigurationItem.Create().
		SetName(name).
		SetCiType("server").
		SetCiTypeID(ciTypeID).
		SetStatus("active").
		SetEnvironment("production").
		SetCriticality("high").
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== 创建配置项测试 ====================

func TestCMDBService_CreateCI_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "ci")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	req := &CreateCIRequest{
		Name:        "生产服务器",
		CiType:      "server",
		CiTypeID:    ciType.ID,
		Status:      "active",
		Environment: "production",
		Criticality: "high",
		TenantID:    testTenant.ID,
	}

	ci, err := service.CreateCI(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, ci)
	assert.Equal(t, req.Name, ci.Name)
	assert.Equal(t, req.CiType, ci.CiType)
	assert.Equal(t, req.Status, ci.Status)
	assert.Equal(t, req.Environment, ci.Environment)
}

func TestCMDBService_CreateCI_WithOptionalFields(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "opt")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	assetTag := "ASSET-001"
	serialNumber := "SN12345"
	location := "数据中心A"

	req := &CreateCIRequest{
		Name:         "数据库服务器",
		CiType:       "server",
		CiTypeID:     ciType.ID,
		Status:       "active",
		Environment:  "production",
		Criticality:  "critical",
		TenantID:     testTenant.ID,
		AssetTag:     &assetTag,
		SerialNumber: &serialNumber,
		Location:     &location,
	}

	ci, err := service.CreateCI(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, ci)
	assert.Equal(t, assetTag, ci.AssetTag)
	assert.Equal(t, serialNumber, ci.SerialNumber)
	assert.Equal(t, location, ci.Location)
}

// ==================== 获取配置项测试 ====================

func TestCMDBService_GetCI_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "get")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建配置项
	created, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "测试服务器")
	require.NoError(t, err)

	ci, err := service.GetCI(ctx, created.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, ci.ID)
	assert.Equal(t, "测试服务器", ci.Name)
}

func TestCMDBService_GetCI_NotFound(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "notfound")
	require.NoError(t, err)

	_, err = service.GetCI(ctx, 99999, testTenant.ID)
	require.Error(t, err)
}

func TestCMDBService_GetCI_TenantMismatch(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant1, err := createCMDBTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createCMDBTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant1.ID, "server")
	require.NoError(t, err)

	// 在租户1创建配置项
	created, err := createTestCI(ctx, client, testTenant1.ID, ciType.ID, "测试服务器")
	require.NoError(t, err)

	// 使用租户2的ID获取，应该失败
	_, err = service.GetCI(ctx, created.ID, testTenant2.ID)
	require.Error(t, err)
}

// ==================== 列出配置项测试 ====================

func TestCMDBService_ListCIs_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "list")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建多个配置项
	for i := 0; i < 5; i++ {
		_, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, fmt.Sprintf("服务器%d", i))
		require.NoError(t, err)
	}

	req := &ListCIsRequest{
		TenantID: testTenant.ID,
		Offset:   0,
		Limit:    10,
	}

	_, total, err := service.ListCIs(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
}

func TestCMDBService_ListCIs_FilterByType(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "type")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建不同类型的配置项
	ciTypes := []string{"server", "network", "server"}
	for i, ciTypeName := range ciTypes {
		_, err := client.ConfigurationItem.Create().
			SetName(fmt.Sprintf("CI%d", i)).
			SetCiType(ciTypeName).
			SetCiTypeID(ciType.ID).
			SetStatus("active").
			SetEnvironment("production").
			SetCriticality("medium").
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &ListCIsRequest{
		TenantID: testTenant.ID,
		CiType:   "server",
		Offset:   0,
		Limit:    10,
	}

	_, total, err := service.ListCIs(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

func TestCMDBService_ListCIs_FilterByStatus(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "status")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建不同状态的配置项
	statuses := []string{"active", "inactive", "active"}
	for i, status := range statuses {
		_, err := client.ConfigurationItem.Create().
			SetName(fmt.Sprintf("CI%d", i)).
			SetCiType("server").
			SetCiTypeID(ciType.ID).
			SetStatus(status).
			SetEnvironment("production").
			SetCriticality("medium").
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &ListCIsRequest{
		TenantID: testTenant.ID,
		Status:   "active",
		Offset:   0,
		Limit:    10,
	}

	_, total, err := service.ListCIs(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
}

// ==================== 统计配置项测试 ====================

func TestCMDBService_CountCIs_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "count")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建配置项
	for i := 0; i < 10; i++ {
		_, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, fmt.Sprintf("CI%d", i))
		require.NoError(t, err)
	}

	count, err := service.CountCIs(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

// ==================== 创建CI关系测试 ====================

func TestCMDBService_CreateRelationship_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "rel")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	// 创建两个配置项
	ci1, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "Web服务器")
	require.NoError(t, err)

	ci2, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "数据库服务器")
	require.NoError(t, err)

	req := &CreateRelationshipRequest{
		Type:       "depends_on",
		SourceCIID: ci1.ID,
		TargetCIID: ci2.ID,
	}

	rel, err := service.CreateRelationship(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, "depends_on", rel.RelationshipType)
	assert.Equal(t, ci1.ID, rel.SourceCiID)
	assert.Equal(t, ci2.ID, rel.TargetCiID)
}

func TestCMDBService_CreateRelationship_WithDescription(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "reldesc")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	ci1, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "应用服务器")
	require.NoError(t, err)

	ci2, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "负载均衡器")
	require.NoError(t, err)

	description := "应用服务器通过负载均衡器接收流量"
	req := &CreateRelationshipRequest{
		Type:        "connects_to",
		SourceCIID:  ci1.ID,
		TargetCIID:  ci2.ID,
		Description: &description,
	}

	rel, err := service.CreateRelationship(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, rel)
	assert.Equal(t, description, rel.Description)
}

// ==================== 更新CI关系测试 ====================

func TestCMDBService_UpdateRelationship_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "relupd")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	ci1, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI1")
	require.NoError(t, err)

	ci2, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI2")
	require.NoError(t, err)

	// 创建关系
	rel, err := client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		Save(ctx)
	require.NoError(t, err)

	// 更新关系
	newType := "connects_to"
	req := &UpdateRelationshipRequest{
		Type: newType,
	}

	updated, err := service.UpdateRelationship(ctx, rel.ID, req)
	require.NoError(t, err)
	assert.Equal(t, newType, updated.RelationshipType)
}

// ==================== 删除CI关系测试 ====================

func TestCMDBService_DeleteRelationship_Success(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "reldel")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	ci1, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI1")
	require.NoError(t, err)

	ci2, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI2")
	require.NoError(t, err)

	// 创建关系
	rel, err := client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		Save(ctx)
	require.NoError(t, err)

	// 删除关系
	err = service.DeleteRelationship(ctx, rel.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.CIRelationship.Get(ctx, rel.ID)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

// ==================== 列出关系测试 ====================

func TestCMDBService_ListRelationships_ByCIID(t *testing.T) {
	client, service, ctx := setupCMDBTest(t)
	defer client.Close()

	testTenant, err := createCMDBTestTenant(ctx, client, "relci")
	require.NoError(t, err)

	ciType, err := createCMDBTestCIType(ctx, client, testTenant.ID, "server")
	require.NoError(t, err)

	ci1, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI1")
	require.NoError(t, err)

	ci2, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI2")
	require.NoError(t, err)

	ci3, err := createTestCI(ctx, client, testTenant.ID, ciType.ID, "CI3")
	require.NoError(t, err)

	// 创建关系：ci1 -> ci2, ci3 -> ci1
	_, err = client.CIRelationship.Create().
		SetRelationshipType("depends_on").
		SetSourceCiID(ci1.ID).
		SetTargetCiID(ci2.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.CIRelationship.Create().
		SetRelationshipType("connects_to").
		SetSourceCiID(ci3.ID).
		SetTargetCiID(ci1.ID).
		Save(ctx)
	require.NoError(t, err)

	// 查询与ci1相关的关系
	rels, err := service.ListRelationships(ctx, testTenant.ID, &ci1.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(rels)) // ci1作为source和target各有一个
}
