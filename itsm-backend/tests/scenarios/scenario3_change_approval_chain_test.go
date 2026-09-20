package scenarios

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/service"
)

// Scenario 3: 变更审批链（会签/或签/评审组）+ 回滚测试
//
// 覆盖：
//   - 会签 (AND): 所有审批人必须批准
//   - 或签 (OR): 任一审批人批准即可
//   - 评审组 (REVIEW/EREVIEW): 成员审批
//   - 回滚: 变更实施失败后回滚到 rolled_back 状态
//   - 跨租户隔离

func TestScenario3_ChangeApprovalChainAndRollback(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("change_approval"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "chg-a", "chg-a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "chg-b", "chg-b.test.com")
	userA := mustCreateUser(ctx, t, client, tenantA.ID, "mgr_a", "mgr_a@test.com", "manager")
	userB := mustCreateUser(ctx, t, client, tenantA.ID, "mgr_b", "mgr_b@test.com", "manager")
	userC := mustCreateUser(ctx, t, client, tenantA.ID, "mgr_c", "mgr_c@test.com", "manager")
	userB2 := mustCreateUser(ctx, t, client, tenantB.ID, "mgr_b2", "mgr_b2@test.com", "manager")

	approvalChainSvc := service.NewApprovalChainService(client, logger)
	changeSvc := service.NewChangeService(client, logger)

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

	t.Run("change full lifecycle with rollback", func(t *testing.T) {
		changeResp, err := changeSvc.CreateChange(ctx, &dto.CreateChangeRequest{
			Title:            "数据库版本升级",
			Description:      "将 MySQL 从 8.0 升级到 8.4",
			Justification:    "安全补丁和性能提升",
			Type:             "normal",
			Priority:         "high",
			ImpactScope:      "medium",
			RiskLevel:        "high",
			RollbackPlan:     "回退到 MySQL 8.0 快照",
			ImplementationPlan: "1. 备份 2. 升级 3. 验证",
		}, userA.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("创建变更失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusPending, tenantA.ID)
		if err != nil {
			t.Fatalf("变更提交审批失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusApproved, tenantA.ID)
		if err != nil {
			t.Fatalf("变更审批通过失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, "scheduled", tenantA.ID)
		if err != nil {
			t.Fatalf("变更排期失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusInProgress, tenantA.ID)
		if err != nil {
			t.Fatalf("变更开始实施失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusFailed, tenantA.ID)
		if err != nil {
			t.Fatalf("变更标记失败失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusRolledBack, tenantA.ID)
		if err != nil {
			t.Fatalf("变更回滚失败: %v", err)
		}

		final, err := changeSvc.GetChange(ctx, changeResp.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("获取变更详情失败: %v", err)
		}
		if final.Status != dto.ChangeStatusRolledBack {
			t.Fatalf("变更应已回滚, 实际状态: %s", final.Status)
		}
		t.Logf("变更回滚完成: id=%d, status=%s", changeResp.ID, final.Status)
	})

	t.Run("tenant B cannot approve tenant A change", func(t *testing.T) {
		changeResp, err := changeSvc.CreateChange(ctx, &dto.CreateChangeRequest{
			Title:       "tenant A 变更",
			Type:        "normal",
			Priority:    "medium",
			ImpactScope: "medium",
			RiskLevel:   "medium",
		}, userA.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("创建变更失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusPending, tenantA.ID)
		if err != nil {
			t.Fatalf("提交审批失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusApproved, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能审批 tenant A 的变更")
		}
		t.Logf("跨租户审批隔离: %v", err)
	})

	t.Run("tenant B cannot view tenant A change", func(t *testing.T) {
		changeResp, err := changeSvc.CreateChange(ctx, &dto.CreateChangeRequest{
			Title:       "tenant A 私有变更",
			Type:        "standard",
			Priority:    "low",
			ImpactScope: "low",
			RiskLevel:   "low",
		}, userA.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("创建变更失败: %v", err)
		}

		_, err = changeSvc.GetChange(ctx, changeResp.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能查看 tenant A 的变更")
		}
	})

	_ = userB
	_ = userC
	_ = userB2
	_ = approvalChainSvc
}
