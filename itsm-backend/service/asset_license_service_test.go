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

func TestAssetLicenseService_CreateLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

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
		request       *dto.CreateLicenseRequest
		tenantID      int
		expectedError bool
	}{
		{
			name: "成功创建许可证",
			request: &dto.CreateLicenseRequest{
				Name:           "Office 365 许可证",
				Description:    "企业版许可证",
				Vendor:         "Microsoft",
				LicenseType:    "subscription",
				TotalQuantity:  100,
				PurchaseDate:   "2024-01-01",
				ExpiryDate:     "2025-01-01",
				PurchasePrice:  func() *float64 { v := 5000.0; return &v }(),
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "许可证名称为空",
			request: &dto.CreateLicenseRequest{
				Name:        "",
				LicenseType: "subscription",
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "许可证类型为空",
			request: &dto.CreateLicenseRequest{
				Name:        "测试许可证",
				LicenseType: "",
			},
			tenantID:      testTenant.ID,
			expectedError: false, // 许可证类型可以为空
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license, err := licenseService.CreateLicense(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, license)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, license)
				assert.Equal(t, tt.request.Name, license.Name)
				assert.Equal(t, tt.request.LicenseType, license.LicenseType)
			}
		})
	}
}

func TestAssetLicenseService_GetLicenseByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试许可证
	license, err := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:        "测试许可证",
		Vendor:      "Microsoft",
		LicenseType: "subscription",
		TotalQuantity: 50,
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试获取存在的许可证
	t.Run("获取存在的许可证", func(t *testing.T) {
		result, err := licenseService.GetLicenseByID(ctx, license.ID, testTenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "测试许可证", result.Name)
	})

	// 测试获取不存在的许可证
	t.Run("获取不存在的许可证", func(t *testing.T) {
		result, err := licenseService.GetLicenseByID(ctx, 9999, testTenant.ID)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestAssetLicenseService_ListLicenses(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建多个测试许可证
	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "许可证1",
		Vendor:       "Microsoft",
		LicenseType:  "subscription",
		TotalQuantity: 100,
	}, testTenant.ID)

	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "许可证2",
		Vendor:       "Adobe",
		LicenseType:  "perpetual",
		TotalQuantity: 50,
	}, testTenant.ID)

	// 测试获取列表
	t.Run("获取所有许可证", func(t *testing.T) {
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 0, 0, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 2, len(result.Licenses))
	})

	// 测试按类型过滤
	t.Run("按类型过滤", func(t *testing.T) {
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 0, 0, "subscription", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, "subscription", result.Licenses[0].LicenseType)
	})

	// 测试分页
	t.Run("分页查询", func(t *testing.T) {
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 1, 1, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 1, len(result.Licenses))
	})
}

func TestAssetLicenseService_UpdateLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试许可证
	license, err := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "原始名称",
		LicenseType:  "subscription",
		TotalQuantity: 100,
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试更新许可证
	t.Run("更新许可证名称", func(t *testing.T) {
		newName := "更新后的名称"
		result, err := licenseService.UpdateLicense(ctx, license.ID, testTenant.ID, &dto.UpdateLicenseRequest{
			Name: &newName,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "更新后的名称", result.Name)
	})

	t.Run("更新许可证数量", func(t *testing.T) {
		newQuantity := 200
		result, err := licenseService.UpdateLicense(ctx, license.ID, testTenant.ID, &dto.UpdateLicenseRequest{
			TotalQuantity: &newQuantity,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 200, result.TotalQuantity)
	})

	// 测试更新不存在的许可证
	t.Run("更新不存在的许可证", func(t *testing.T) {
		result, err := licenseService.UpdateLicense(ctx, 9999, testTenant.ID, &dto.UpdateLicenseRequest{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestAssetLicenseService_DeleteLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试许可证
	license, err := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "待删除许可证",
		LicenseType:  "subscription",
		TotalQuantity: 10,
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试删除许可证
	t.Run("成功删除许可证", func(t *testing.T) {
		err := licenseService.DeleteLicense(ctx, license.ID, testTenant.ID)
		assert.NoError(t, err)

		// 验证已删除
		result, _ := licenseService.GetLicenseByID(ctx, license.ID, testTenant.ID)
		assert.Nil(t, result)
	})
}

func TestAssetLicenseService_AssignUsers(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试许可证 (可用数量为10)
	license, err := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "测试许可证",
		LicenseType:  "subscription",
		TotalQuantity: 10,
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试分配用户
	t.Run("成功分配用户", func(t *testing.T) {
		userIDs := []int{1, 2, 3}
		result, err := licenseService.AssignUsers(ctx, license.ID, testTenant.ID, userIDs)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 3, result.UsedQuantity)
	})

	// 测试超过可用数量
	t.Run("分配超过可用数量", func(t *testing.T) {
		// 先创建一个新许可证，可用数量为2
		license2, _ := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
			Name:         "限量许可证",
			LicenseType:  "subscription",
			TotalQuantity: 2,
		}, testTenant.ID)

		userIDs := []int{1, 2, 3, 4, 5}
		result, err := licenseService.AssignUsers(ctx, license2.ID, testTenant.ID, userIDs)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "insufficient license quantity")
	})
}

func TestAssetLicenseService_GetLicenseStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建多个测试许可证
	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "许可证1",
		LicenseType:  "subscription",
		TotalQuantity: 100,
	}, testTenant.ID)

	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:         "许可证2",
		LicenseType:  "perpetual",
		TotalQuantity: 50,
	}, testTenant.ID)

	// 测试获取统计
	stats, err := licenseService.GetLicenseStats(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Active)
}
