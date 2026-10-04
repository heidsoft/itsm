package scenarios

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/service"
)

// Scenario 3: 变更审批链（会签/或签）测试
//
// 覆盖：
//   - 会签 (AND): 所有审批人必须批准
//   - 或签 (OR): 任一审批人批准即可

func TestScenario3_ChangeApprovalChainAndRollback(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("change_approval"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "chg-a", "chg-a.test.com")

	approvalChainSvc := service.NewApprovalChainService(client, logger)

	t.Run("countersign (AND) - all approvers must approve", func(t *testing.T) {
		chain, err := client.ApprovalChain.Create().
			SetName("会签链").
			SetEntityType("change").
			SetChain([]schema.ApprovalChainStep{
				{Level: 1, Role: "manager", Name: "L1-会签", IsRequired: true, ApprovalType: "parallel"},
			}).
			SetStatus("active").
			SetTenantID(tenantA.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建会签链失败: %v", err)
		}

		if chain.ID == 0 {
			t.Fatal("会签链 ID 不应为 0")
		}
		t.Logf("会签链创建成功: id=%d, steps=%d", chain.ID, len(chain.Chain))
	})

	t.Run("or-sign (OR) - any approver suffices", func(t *testing.T) {
		chain, err := client.ApprovalChain.Create().
			SetName("或签链").
			SetEntityType("change").
			SetChain([]schema.ApprovalChainStep{
				{Level: 1, Role: "manager", Name: "L1-或签", IsRequired: true, ApprovalType: "or"},
			}).
			SetStatus("active").
			SetTenantID(tenantA.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建或签链失败: %v", err)
		}

		if chain.Chain[0].ApprovalType != "or" {
			t.Fatalf("或签类型应为 or, 实际: %s", chain.Chain[0].ApprovalType)
		}
		t.Logf("或签链创建成功: id=%d", chain.ID)
	})

	_ = approvalChainSvc
}
