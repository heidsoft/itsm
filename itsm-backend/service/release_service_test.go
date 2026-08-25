package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestReleaseService_CreateRelease(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	releaseService := NewReleaseService(client, logger)

	ctx := context.Background()

	// 创建测试租户
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

	tests := []struct {
		name          string
		request       *dto.CreateReleaseRequest
		tenantID      int
		createdBy     int
		expectedError bool
	}{
		{
			// P1 修复：release_number 现由服务端生成（避免越权/冲突编号），不信任
			// 客户端传入值。故创建成功的断言改为"服务端生成了 REL- 前缀的编号"，
			// 客户传入的 ReleaseNumber 被安全忽略。
			name: "成功创建发布",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "REL-20260222-001", // 客户端输入，服务端忽略
				Title:         "测试发布",
				Description:   "这是一个测试发布",
				Type:          "minor",
				Environment:   "staging",
				Severity:      "medium",
			},
			tenantID:      testTenant.ID,
			createdBy:     testUser.ID,
			expectedError: false,
		},
		{
			// P1 修复：旧实期望客户端必须提供非空 ReleaseNumber，否则报错。
			// 新实服务端总是生成编号，客户端传空也是合法的（服务端会填上）。
			name: "发布编号为空（由服务端自动生成）",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "",
				Title:         "测试发布",
			},
			tenantID:      testTenant.ID,
			createdBy:     testUser.ID,
			expectedError: false,
		},
		{
			name: "标题为空",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "REL-001",
				Title:         "",
			},
			tenantID:      testTenant.ID,
			createdBy:     testUser.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, err := releaseService.CreateRelease(ctx, tt.request, tt.createdBy, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, release)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, release)
				assert.Equal(t, tt.request.Title, release.Title)
				// P1 修复：release_number 由服务端生成，不信任客户端输入。
				// 断言仅校验"服务端生成的编号符合 REL-YYYYMMDD- 格式契约"，不再
				// 硬编码某个具体值，也不再与请求中的 ReleaseNumber 字段强比对。
				assert.Regexp(t, `^REL-\d{8}-[A-Za-z0-9]+$`, release.ReleaseNumber,
					"服务端应该生成 REL-YYYYMMDD-token 格式的发布编号")
				assert.NotEqual(t, "", release.ReleaseNumber, "服务端不应返回空 release_number")
			}
		})
	}
}

func TestReleaseService_GetReleaseByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	releaseService := NewReleaseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

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

	// 创建测试发布
	release, err := releaseService.CreateRelease(ctx, &dto.CreateReleaseRequest{
		ReleaseNumber: "REL-001", // 客户端输入，服务端忽略
		Title:         "测试发布",
		Type:          "minor",
	}, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	require.NotEmpty(t, release.ReleaseNumber, "服务端应生成 release_number")
	originalNumber := release.ReleaseNumber

	// 测试获取发布
	t.Run("获取存在的发布", func(t *testing.T) {
		result, err := releaseService.GetReleaseByID(ctx, release.ID, testTenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// P1 修复：服务端生成的 release_number 不应与客户端传入值强绑定。
		// 断言该编号与创建响应一致，并符合 REL-YYYYMMDD-token 格式。
		assert.Equal(t, originalNumber, result.ReleaseNumber,
			"重复获取同一发布的 release_number 应稳定")
		assert.Regexp(t, `^REL-\d{8}-[A-Za-z0-9]+$`, result.ReleaseNumber)
	})

	// 测试获取不存在的发布
	t.Run("获取不存在的发布", func(t *testing.T) {
		result, err := releaseService.GetReleaseByID(ctx, 9999, testTenant.ID)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestReleaseService_UpdateReleaseStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	releaseService := NewReleaseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

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

	// 创建测试发布
	release, err := releaseService.CreateRelease(ctx, &dto.CreateReleaseRequest{
		ReleaseNumber: "REL-001",
		Title:         "测试发布",
		Type:          "minor",
	}, testUser.ID, testTenant.ID)
	require.NoError(t, err)

	// 测试更新状态
	// D-5 修复后：draft→scheduled 为审批等效动作，必须经 ApplyReleaseApproval（release:approve），
	// 不允许通过 UpdateReleaseStatus（release:write）直达，避免持 write 权限绕过审批门禁。
	t.Run("draft->scheduled 经 status 路由被拒绝（须走审批）", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "scheduled")
		assert.Error(t, err)
		assert.Nil(t, result)
		rel, qErr := client.Release.Get(ctx, release.ID)
		require.NoError(t, qErr)
		assert.Equal(t, "draft", rel.Status)
	})

	t.Run("draft->cancelled 允许（write 可取消草稿）", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "cancelled")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cancelled", result.Status)
		// 复位为 draft 以便后续链路测试
		_, _ = client.Release.UpdateOneID(release.ID).SetStatus("draft").Save(ctx)
	})

	// scheduled / in-progress / completed 链路：先直接置 scheduled 作为"已审批"基线
	t.Run("scheduled->in-progress", func(t *testing.T) {
		_, _ = client.Release.UpdateOneID(release.ID).SetStatus("scheduled").Save(ctx)
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "in-progress")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "in-progress", result.Status)
	})

	t.Run("in-progress->completed", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "completed")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
		assert.NotNil(t, result.ActualReleaseDate)
	})
}

func TestReleaseService_GetReleaseStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	releaseService := NewReleaseService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

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

	// 创建多个测试发布
	_, _ = releaseService.CreateRelease(ctx, &dto.CreateReleaseRequest{
		ReleaseNumber: "REL-001",
		Title:         "发布1",
		Type:          "minor",
	}, testUser.ID, testTenant.ID)

	_, _ = releaseService.CreateRelease(ctx, &dto.CreateReleaseRequest{
		ReleaseNumber: "REL-002",
		Title:         "发布2",
		Type:          "patch",
	}, testUser.ID, testTenant.ID)

	// 测试获取统计
	stats, err := releaseService.GetReleaseStats(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Draft)
}
