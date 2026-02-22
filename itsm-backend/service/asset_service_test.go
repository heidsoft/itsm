package service

import (
	"context"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestAssetService_CreateAsset(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		request       *dto.CreateAssetRequest
		tenantID      int
		expectedError bool
	}{
		{
			name: "成功创建资产",
			request: &dto.CreateAssetRequest{
				AssetNumber: "ASSET-001",
				Name:        "测试资产",
				Description: "这是一个测试资产",
				Type:        "hardware",
				Category:    "computer",
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "资产编号为空",
			request: &dto.CreateAssetRequest{
				AssetNumber: "",
				Name:        "测试资产",
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "资产名称为空",
			request: &dto.CreateAssetRequest{
				AssetNumber: "ASSET-002",
				Name:        "",
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := assetService.CreateAsset(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, asset)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, asset)
				assert.Equal(t, tt.request.Name, asset.Name)
				assert.Equal(t, tt.request.AssetNumber, asset.AssetNumber)
			}
		})
	}
}

func TestAssetService_GetAssetByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试资产
	asset, err := assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-001",
		Name:        "测试资产",
		Type:        "hardware",
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试获取资产
	t.Run("获取存在的资产", func(t *testing.T) {
		result, err := assetService.GetAssetByID(ctx, asset.ID, testTenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ASSET-001", result.AssetNumber)
	})

	// 测试获取不存在的资产
	t.Run("获取不存在的资产", func(t *testing.T) {
		result, err := assetService.GetAssetByID(ctx, 9999, testTenant.ID)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestAssetService_UpdateAssetStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试资产
	asset, err := assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-001",
		Name:        "测试资产",
		Type:        "hardware",
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试更新状态
	t.Run("更新为使用中状态", func(t *testing.T) {
		result, err := assetService.UpdateAssetStatus(ctx, asset.ID, testTenant.ID, "in-use", nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "in-use", result.Status)
	})

	t.Run("更新为维护中状态", func(t *testing.T) {
		result, err := assetService.UpdateAssetStatus(ctx, asset.ID, testTenant.ID, "maintenance", nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "maintenance", result.Status)
	})
}

func TestAssetService_AssignAsset(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("admin").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试资产
	asset, err := assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-001",
		Name:        "测试资产",
		Type:        "hardware",
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试分配资产
	t.Run("成功分配资产", func(t *testing.T) {
		result, err := assetService.AssignAsset(ctx, asset.ID, testTenant.ID, testUser.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "in-use", result.Status)
		assert.NotNil(t, result.AssignedTo)
		assert.Equal(t, testUser.ID, *result.AssignedTo)
	})
}

func TestAssetService_RetireAsset(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试资产
	asset, err := assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-001",
		Name:        "测试资产",
		Type:        "hardware",
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试退役资产
	t.Run("成功退役资产", func(t *testing.T) {
		result, err := assetService.RetireAsset(ctx, asset.ID, testTenant.ID, "测试退役")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "retired", result.Status)
	})
}

func TestAssetService_GetAssetStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	assetService := NewAssetService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建多个测试资产
	_, _ = assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-001",
		Name:        "资产1",
		Type:        "hardware",
	}, testTenant.ID)

	_, _ = assetService.CreateAsset(ctx, &dto.CreateAssetRequest{
		AssetNumber: "ASSET-002",
		Name:        "资产2",
		Type:        "hardware",
	}, testTenant.ID)

	// 测试获取统计
	stats, err := assetService.GetAssetStats(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Available)
}
