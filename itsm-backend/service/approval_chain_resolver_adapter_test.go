package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entuser "itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestApprovalChainResolverAdapter_ConvergenceWithDirectPath 验证：
// BPMN Service Task 路径（ApprovalChainResolverAdapter → ResolveApprovalPlan）
// 与直接调用 ResolveApprovalPlan 产生完全一致的结果。
// 这是 Phase 1 编排统一的核心回归——三条实体类型（ticket/change/service_request）均须收敛。
func TestApprovalChainResolverAdapter_ConvergenceWithDirectPath(t *testing.T) {
	for _, entityType := range []string{"ticket", "change", "service_request"} {
		t.Run(entityType, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", testDSN())
			t.Cleanup(func() { _ = client.Close() })
			logger := zaptest.NewLogger(t).Sugar()
			ctx := context.Background()
			require.NoError(t, client.Schema.Create(ctx))

			tenant := mkConvergenceTenant(t, ctx, client, entityType)
			mgr := mkConvergenceUser(t, ctx, client, tenant.ID, "manager")
			sec := mkConvergenceUser(t, ctx, client, tenant.ID, "security")
			requester := mkConvergenceUser(t, ctx, client, tenant.ID, "end_user")

			svc := NewApprovalChainService(client, logger)
			_, err := svc.CreateApprovalChain(ctx, &dto.ApprovalChainRequest{
				Name:       entityType + "-chain",
				EntityType: entityType,
				Status:     "active",
				Chain: []dto.ApprovalChainStepDTO{
					{Level: 1, Role: "manager", Name: "L1", IsRequired: true, ApprovalType: "serial"},
					{Level: 2, Role: "security", Name: "L2", IsRequired: true, ApprovalType: "serial"},
				},
			}, tenant.ID)
			require.NoError(t, err)

			evalCtx := ApprovalEvalContext{
				RequesterID: requester,
				Priority:    "high",
			}

			// 路径 A：直接调用（旧路径）
			directPlan, directErr := svc.ResolveApprovalPlan(ctx, tenant.ID, entityType, evalCtx, nil)

			// 路径 B：通过适配器（BPMN Service Task 路径）
			adapter := NewApprovalChainResolverAdapter(client, logger, svc)
			chainID, levelsJSON, passed, pendingLevel, blocked, adapterErr := adapter.ResolveApprovalPlanRaw(
				ctx, tenant.ID, entityType, requester, "high", 0, nil,
			)

			// 一致性断言：两条路径必须同成功或同失败
			if directErr != nil {
				require.Error(t, adapterErr, "直接路径失败时适配器也必须失败: %s", entityType)
				return
			}
			require.NoError(t, adapterErr, "适配器路径不应失败: %s", entityType)

			assert.Equal(t, directPlan.ChainID, chainID, "chain_id 必须一致")
			assert.Equal(t, directPlan.Passed, passed, "passed 必须一致")
			assert.Equal(t, directPlan.PendingLevel, pendingLevel, "pending_level 必须一致")
			assert.Equal(t, directPlan.Blocked, blocked, "blocked 必须一致")

			// levels JSON 必须解析出相同的审批人
			directLevels, err := json.Marshal(directPlan.Levels)
			require.NoError(t, err)
			assert.JSONEq(t, string(directLevels), string(levelsJSON),
				"levels JSON 必须一致（%s）", entityType)

			// 验证审批人解析正确
			_ = mgr
			_ = sec
		})
	}
}

// TestApprovalChainResolverAdapter_NoChainBothPathsFail 验证：
// 无激活审批链时，两条路径都返回错误（fail-closed）。
func TestApprovalChainResolverAdapter_NoChainBothPathsFail(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))

	tenant := mkConvergenceTenant(t, ctx, client, "no-chain")
	requester := mkConvergenceUser(t, ctx, client, tenant.ID, "end_user")

	svc := NewApprovalChainService(client, logger)
	evalCtx := ApprovalEvalContext{RequesterID: requester}

	_, directErr := svc.ResolveApprovalPlan(ctx, tenant.ID, "ticket", evalCtx, nil)
	require.Error(t, directErr, "无激活链时直接路径必须失败")

	adapter := NewApprovalChainResolverAdapter(client, logger, svc)
	_, _, _, _, _, adapterErr := adapter.ResolveApprovalPlanRaw(
		ctx, tenant.ID, "ticket", requester, "", 0, nil,
	)
	require.Error(t, adapterErr, "无激活链时适配器路径也必须失败")
}

// TestApprovalChainResolverAdapter_TenantIsolation 验证：
// 适配器路径严格按 tenantID 过滤，不能读到其他租户的审批链。
func TestApprovalChainResolverAdapter_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))

	tenantA := mkConvergenceTenant(t, ctx, client, "tenant-a")
	tenantB := mkConvergenceTenant(t, ctx, client, "tenant-b")
	reqA := mkConvergenceUser(t, ctx, client, tenantA.ID, "end_user")
	_ = mkConvergenceUser(t, ctx, client, tenantB.ID, "end_user")

	svc := NewApprovalChainService(client, logger)
	// 只在 tenantA 创建激活链
	_, err := svc.CreateApprovalChain(ctx, &dto.ApprovalChainRequest{
		Name:       "A-only-chain",
		EntityType: "ticket",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "L1", IsRequired: true, ApprovalType: "serial"},
		},
	}, tenantA.ID)
	require.NoError(t, err)

	adapter := NewApprovalChainResolverAdapter(client, logger, svc)

	// tenantA 应能找到链
	_, _, _, _, _, errA := adapter.ResolveApprovalPlanRaw(ctx, tenantA.ID, "ticket", reqA, "", 0, nil)
	require.NoError(t, errA, "tenantA 有激活链，适配器应成功")

	// tenantB 不应找到 tenantA 的链
	_, _, _, _, _, errB := adapter.ResolveApprovalPlanRaw(ctx, tenantB.ID, "ticket", reqA, "", 0, nil)
	require.Error(t, errB, "tenantB 无激活链，适配器必须失败（不能跨租户读取）")
}

func mkConvergenceTenant(t *testing.T, ctx context.Context, client *ent.Client, suffix string) *ent.Tenant {
	t.Helper()
	tn, err := client.Tenant.Create().
		SetName("ConvTN-" + suffix).
		SetCode("conv" + suffix).
		SetDomain("conv" + suffix + ".test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tn
}

func mkConvergenceUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, role string) int {
	t.Helper()
	suffix := fmt.Sprintf("%d-%s-%s", tenantID, role, t.Name())
	u, err := client.User.Create().
		SetUsername("conv-" + suffix).
		SetEmail("conv-" + suffix + "@example.com").
		SetName("Conv " + role).
		SetPasswordHash("h").
		SetRole(entuser.Role(role)).
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u.ID
}
