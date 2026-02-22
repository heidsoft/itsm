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

func TestReleaseService_CreateRelease(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
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
		createdBy    int
		expectedError bool
	}{
		{
			name: "成功创建发布",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "REL-20260222-001",
				Title:        "测试发布",
				Description:  "这是一个测试发布",
				Type:         "minor",
				Environment:  "staging",
				Severity:     "medium",
			},
			tenantID:      testTenant.ID,
			createdBy:    testUser.ID,
			expectedError: false,
		},
		{
			name: "发布编号为空",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "",
				Title:        "测试发布",
			},
			tenantID:      testTenant.ID,
			createdBy:    testUser.ID,
			expectedError: true,
		},
		{
			name: "标题为空",
			request: &dto.CreateReleaseRequest{
				ReleaseNumber: "REL-001",
				Title:        "",
			},
			tenantID:      testTenant.ID,
			createdBy:    testUser.ID,
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
				assert.Equal(t, tt.request.ReleaseNumber, release.ReleaseNumber)
			}
		})
	}
}

func TestReleaseService_GetReleaseByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
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

	// 测试获取发布
	t.Run("获取存在的发布", func(t *testing.T) {
		result, err := releaseService.GetReleaseByID(ctx, release.ID, testTenant.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "REL-001", result.ReleaseNumber)
	})

	// 测试获取不存在的发布
	t.Run("获取不存在的发布", func(t *testing.T) {
		result, err := releaseService.GetReleaseByID(ctx, 9999, testTenant.ID)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestReleaseService_UpdateReleaseStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
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
	t.Run("更新为已计划状态", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "scheduled")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "scheduled", result.Status)
	})

	t.Run("更新为进行中状态", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "in-progress")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "in-progress", result.Status)
	})

	t.Run("更新为已完成状态", func(t *testing.T) {
		result, err := releaseService.UpdateReleaseStatus(ctx, release.ID, testTenant.ID, "completed")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
		assert.NotNil(t, result.ActualReleaseDate)
	})
}

func TestReleaseService_GetReleaseStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
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
