package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/processbinding"
	"itsm-backend/ent/processdefinition"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestLegacyApprovalMigration_Migrate 验证旧审批链迁移到 BPMN 绑定的接管逻辑：
//   - dryRun 不产生任何副作用
//   - 首次迁移：部署流程定义 + 创建 (approval, ticketType) 激活绑定
//   - 重复迁移：不重复部署（Skipped），绑定原地更新（BindingReplaced）
//   - 同 ticketType 的新流程迁移会替换旧激活绑定的路由目标（不产生重复绑定）
func TestLegacyApprovalMigration_Migrate(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))
	svc := NewLegacyApprovalMigrationService(client)

	const tenantID = 1001
	wf := createMigrationTestWorkflow(t, ctx, client, tenantID, "normal")

	t.Run("dry run 无副作用", func(t *testing.T) {
		result, err := svc.Migrate(ctx, wf, true)
		require.NoError(t, err)
		require.NotEmpty(t, result.BPMNXML)
		require.False(t, result.Skipped)

		defCount, err := client.ProcessDefinition.Query().Count(ctx)
		require.NoError(t, err)
		require.Zero(t, defCount)
		bindingCount, err := client.ProcessBinding.Query().Count(ctx)
		require.NoError(t, err)
		require.Zero(t, bindingCount)
	})

	t.Run("首次迁移部署流程并创建绑定", func(t *testing.T) {
		result, err := svc.Migrate(ctx, wf, false)
		require.NoError(t, err)
		require.False(t, result.Skipped)
		require.False(t, result.BindingReplaced)

		key := migrationKey(wf.ID)
		exists, err := client.ProcessDefinition.Query().
			Where(processdefinition.Key(key), processdefinition.TenantID(tenantID)).
			Exist(ctx)
		require.NoError(t, err)
		require.True(t, exists, "流程定义应已部署")

		binding, err := client.ProcessBinding.Query().
			Where(
				processbinding.BusinessType("approval"),
				processbinding.BusinessSubType("normal"),
				processbinding.TenantID(tenantID),
				processbinding.IsActive(true),
			).
			Only(ctx)
		require.NoError(t, err)
		require.Equal(t, key, binding.ProcessDefinitionKey)
		require.Equal(t, 50, binding.Priority)
	})

	t.Run("重复迁移幂等且原地更新绑定", func(t *testing.T) {
		result, err := svc.Migrate(ctx, wf, false)
		require.NoError(t, err)
		require.True(t, result.Skipped, "流程定义已存在时应跳过部署")
		require.True(t, result.BindingReplaced, "已存在激活绑定时应原地更新")

		defCount, err := client.ProcessDefinition.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, defCount)
		bindingCount, err := client.ProcessBinding.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, bindingCount, "不得产生重复绑定")
	})

	t.Run("同 ticketType 新流程迁移接管绑定路由", func(t *testing.T) {
		wf2 := createMigrationTestWorkflow(t, ctx, client, tenantID, "normal")
		result, err := svc.Migrate(ctx, wf2, false)
		require.NoError(t, err)
		require.True(t, result.BindingReplaced)

		binding, err := client.ProcessBinding.Query().
			Where(processbinding.TenantID(tenantID), processbinding.IsActive(true)).
			Only(ctx)
		require.NoError(t, err)
		require.Equal(t, migrationKey(wf2.ID), binding.ProcessDefinitionKey, "绑定路由应指向最新迁移的流程")

		bindingCount, err := client.ProcessBinding.Query().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, bindingCount)
	})
}

func migrationKey(workflowID int) string {
	return fmt.Sprintf("legacy_approval_%d", workflowID)
}

func createMigrationTestWorkflow(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, ticketType string) *ent.ApprovalWorkflow {
	t.Helper()
	wf, err := client.ApprovalWorkflow.Create().
		SetName("迁移测试审批流").
		SetDescription("用于验证旧审批链迁移绑定接管").
		SetIsActive(true).
		SetTenantID(tenantID).
		SetTicketType(ticketType).
		SetNodes([]map[string]interface{}{
			{"name": "经理审批", "assignee_type": "user", "assignee_value": "manager", "step_order": 1},
		}).
		Save(ctx)
	require.NoError(t, err)
	return wf
}
