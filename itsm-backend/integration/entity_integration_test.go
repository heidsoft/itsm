package integration

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTagServiceIntegration 测试标签服务
func TestTagServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TAGINT").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建标签", func(t *testing.T) {
		tag, err := client.Tag.Create().
			SetName("Bug").
			SetCode("BUG").
			SetColor("#FF0000").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, tag)
		require.Equal(t, "Bug", tag.Name)
	})
}

// TestCategoryServiceIntegration 测试分类服务
func TestCategoryServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("CATINT").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建工单分类", func(t *testing.T) {
		category, err := client.TicketCategory.Create().
			SetName("技术问题").
			SetCode("TECH").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, category)
		require.Equal(t, "技术问题", category.Name)
	})
}

// TestPriorityMatrixIntegration 测试优先级矩阵服务
func TestPriorityMatrixIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("PRIINT").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	_ = tenant

	t.Run("创建优先级矩阵", func(t *testing.T) {
		// Skip - PriorityMatrix entity doesn't exist
		t.SkipNow()
	})
}

// TestSLAIntegration 测试SLA服务
func TestSLAIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("SLAINT").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建SLA定义", func(t *testing.T) {
		sla, err := client.SLADefinition.Create().
			SetName("2小时响应").
			SetPriority("critical").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, sla)
		require.Equal(t, "2小时响应", sla.Name)
	})
}
