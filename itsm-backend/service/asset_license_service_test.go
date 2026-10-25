package service

import (
	"context"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestAssetLicenseService_CreateLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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
				Name:          "Office 365 许可证",
				Description:   "企业版许可证",
				Vendor:        "Microsoft",
				LicenseType:   "subscription",
				TotalQuantity: 100,
				PurchaseDate:  "2024-01-01",
				ExpiryDate:    "2025-01-01",
				PurchasePrice: func() *float64 { v := 5000.0; return &v }(),
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
	client := enttest.Open(t, "sqlite3", testDSN())
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
		Name:          "测试许可证",
		Vendor:        "Microsoft",
		LicenseType:   "subscription",
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
	client := enttest.Open(t, "sqlite3", testDSN())
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
		Name:          "许可证1",
		Vendor:        "Microsoft",
		LicenseType:   "subscription",
		TotalQuantity: 100,
	}, testTenant.ID)

	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:          "许可证2",
		Vendor:        "Adobe",
		LicenseType:   "perpetual",
		TotalQuantity: 50,
	}, testTenant.ID)

	// 测试获取列表
	t.Run("获取所有许可证", func(t *testing.T) {
		// 0/0 是「调用方没写分页」的形态。原实现在这里连 Offset/Limit 都不加，
		// 直接把该租户整表读出来；现在一律归一后再分页。
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 0, 0, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 2, len(result.Items))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, common.DefaultPageSize, result.PageSize)
		assert.Equal(t, 1, result.TotalPages)
	})

	// 测试按类型过滤
	t.Run("按类型过滤", func(t *testing.T) {
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 0, 0, "subscription", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, "subscription", result.Items[0].LicenseType)
	})

	// 测试分页
	t.Run("分页查询", func(t *testing.T) {
		result, err := licenseService.ListLicenses(ctx, testTenant.ID, 1, 1, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, 2, result.TotalPages, "总页数必须由 total 与生效页长算出")
	})
}

func TestAssetLicenseService_UpdateLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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
		Name:          "原始名称",
		LicenseType:   "subscription",
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

// UpdateLicense 不得绕过席位不变量：已用数只能由分配/替换用户推导。
func TestAssetLicenseService_UpdateLicenseSeatInvariant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	licenseService := NewAssetLicenseService(client, logger)

	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("Tenant A").SetCode("lic-a").
		SetDomain("a.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("Tenant B").SetCode("lic-b").
		SetDomain("b.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	mkUser := func(tenantID int, name string) *ent.User {
		return client.User.Create().SetUsername(name).SetEmail(name + "@test.com").SetName(name).
			SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantID).SaveX(ctx)
	}
	userA1 := mkUser(tenantA.ID, "lic-user-a1")
	userA2 := mkUser(tenantA.ID, "lic-user-a2")
	userB := mkUser(tenantB.ID, "lic-user-b")

	requireBusinessCode := func(t *testing.T, err error, want int) {
		t.Helper()
		var bizErr *common.BusinessError
		require.ErrorAs(t, err, &bizErr)
		require.Equal(t, want, bizErr.Code)
	}

	reload := func(t *testing.T, id int) *ent.AssetLicense {
		t.Helper()
		lic, err := client.AssetLicense.Get(ctx, id)
		require.NoError(t, err)
		return lic
	}

	t.Run("拒绝直接修改已用数量", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("直接改已用").SetTotalQuantity(10).
			SetStatus("active").SetTenantID(tenantA.ID).SaveX(ctx)

		used := 5
		_, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{UsedQuantity: &used})
		requireBusinessCode(t, err, common.BadRequestCode)
		require.Equal(t, 0, reload(t, lic.ID).UsedQuantity)
	})

	t.Run("替换授权用户按去重后人数重算席位", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("替换用户").SetTotalQuantity(10).
			SetStatus("active").SetTenantID(tenantA.ID).SaveX(ctx)

		updated, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{
			Users: &[]int{userA1.ID, userA2.ID, userA1.ID},
		})
		require.NoError(t, err)
		require.Equal(t, 2, updated.UsedQuantity)
		require.Equal(t, 8, updated.AvailableQuantity)
		require.ElementsMatch(t, []int{userA1.ID, userA2.ID}, reload(t, lic.ID).Users)

		// 删掉一个用户后席位回落；不是全量替换，不能靠「不传」清空。
		updated, err = licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{
			Users: &[]int{userA1.ID},
		})
		require.NoError(t, err)
		require.Equal(t, 1, updated.UsedQuantity)

		name := "仅改名称"
		updated, err = licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{Name: &name})
		require.NoError(t, err)
		require.Equal(t, 1, updated.UsedQuantity, "未传 users 不得清空现有授权")
		require.Equal(t, []int{userA1.ID}, updated.Users)

		updated, err = licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{Users: &[]int{}})
		require.NoError(t, err)
		require.Zero(t, updated.UsedQuantity)
		require.Empty(t, updated.Users)
		require.Equal(t, string(dto.LicenseStatusActive), updated.Status)
	})

	t.Run("无效或跨租户用户整单拒绝", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("跨租用户").SetTotalQuantity(10).
			SetStatus("active").SetTenantID(tenantA.ID).SaveX(ctx)

		_, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{
			Users: &[]int{userA1.ID, userB.ID},
		})
		requireBusinessCode(t, err, common.BadRequestCode)

		after := reload(t, lic.ID)
		require.Empty(t, after.Users)
		require.Equal(t, 0, after.UsedQuantity)
	})

	t.Run("替换后超过总数是409且不落库", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("席位不足").SetTotalQuantity(1).
			SetStatus("active").SetTenantID(tenantA.ID).SaveX(ctx)

		_, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{
			Users: &[]int{userA1.ID, userA2.ID},
		})
		requireBusinessCode(t, err, common.ConflictCode)

		after := reload(t, lic.ID)
		require.Empty(t, after.Users)
		require.Equal(t, 0, after.UsedQuantity)
	})

	t.Run("总数不得缩到已分配席位以下", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("缩容").SetTotalQuantity(10).
			SetUsers([]int{userA1.ID, userA2.ID}).SetUsedQuantity(2).
			SetStatus("active").SetTenantID(tenantA.ID).SaveX(ctx)

		shrunk := 1
		_, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{TotalQuantity: &shrunk})
		requireBusinessCode(t, err, common.ConflictCode)
		require.Equal(t, 10, reload(t, lic.ID).TotalQuantity)
	})

	t.Run("扩容后从depleted恢复active", func(t *testing.T) {
		lic := client.AssetLicense.Create().SetName("扩容").SetTotalQuantity(1).
			SetUsers([]int{userA1.ID}).SetUsedQuantity(1).
			SetStatus(string(dto.LicenseStatusDepleted)).SetTenantID(tenantA.ID).SaveX(ctx)

		grown := 5
		updated, err := licenseService.UpdateLicense(ctx, lic.ID, tenantA.ID, &dto.UpdateLicenseRequest{TotalQuantity: &grown})
		require.NoError(t, err)
		require.Equal(t, string(dto.LicenseStatusActive), updated.Status)
		require.Equal(t, 4, updated.AvailableQuantity)
	})
}

func TestAssetLicenseService_DeleteLicense(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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
		Name:          "待删除许可证",
		LicenseType:   "subscription",
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
	client := enttest.Open(t, "sqlite3", testDSN())
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

	// 创建测试用户：AssignUsers 现在只保留本租户真实存在的用户 id
	// （B8.2 修复：跨租户/无效 id 静默跳过），旧用例传入的 1..5 是幽灵 id。
	mkUser := func(name string) *ent.User {
		return client.User.Create().SetUsername(name).SetEmail(name + "@test.com").SetName(name).
			SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(testTenant.ID).SaveX(ctx)
	}
	// 创建测试许可证 (可用数量为10)
	license, err := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:          "测试许可证",
		LicenseType:   "subscription",
		TotalQuantity: 10,
	}, testTenant.ID)
	require.NoError(t, err)

	// 测试分配用户
	t.Run("成功分配用户", func(t *testing.T) {
		u1, u2, u3 := mkUser("assignee1"), mkUser("assignee2"), mkUser("assignee3")
		userIDs := []int{u1.ID, u2.ID, u3.ID, u1.ID} // 重复 id 应去重
		result, err := licenseService.AssignUsers(ctx, license.ID, testTenant.ID, userIDs)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 3, result.UsedQuantity)
	})

	// 测试超过可用数量
	t.Run("分配超过可用数量", func(t *testing.T) {
		// 先创建一个新许可证，可用数量为2
		license2, _ := licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
			Name:          "限量许可证",
			LicenseType:   "subscription",
			TotalQuantity: 2,
		}, testTenant.ID)

		u4, u5, u6, u7 := mkUser("assignee4"), mkUser("assignee5"), mkUser("assignee6"), mkUser("assignee7")
		userIDs := []int{u4.ID, u5.ID, u6.ID, u7.ID}
		result, err := licenseService.AssignUsers(ctx, license2.ID, testTenant.ID, userIDs)
		// B8.2：席位不足现在是业务冲突 sentinel（handler 层映射 409），不再伪装 500。
		assert.ErrorIs(t, err, ErrLicenseInsufficient)
		assert.Nil(t, result)
	})
}

func TestAssetLicenseService_GetLicenseStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
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
		Name:          "许可证1",
		LicenseType:   "subscription",
		TotalQuantity: 100,
	}, testTenant.ID)

	_, _ = licenseService.CreateLicense(ctx, &dto.CreateLicenseRequest{
		Name:          "许可证2",
		LicenseType:   "perpetual",
		TotalQuantity: 50,
	}, testTenant.ID)

	// 测试获取统计
	stats, err := licenseService.GetLicenseStats(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Active)
}
