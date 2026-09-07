package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// i3 P0 修复：审批级别动态适配（ConditionPriorities/AmountMin/AmountMax）
//
// 设计目标：
//   - 一个 level 配置了条件（优先级白名单 / 金额区间），当工单上下文（Priority/Amount）
//     不匹配时，整个 level 被跳过（视为自动通过），PendingLevel 跳过该层。
//   - 不匹配时不触发 fallback，避免「优先级不匹配却被 block」。
//   - 全部条件为空时永远匹配（与旧行为兼容）。
//   - 同一 level 内多 step 用「或」语义合并：任一 step 条件匹配即视为本层适用。
// ============================================================

// ---- 1. 优先级白名单：工单优先级不在白名单 → 跳过该层 ----
func TestEvaluateApprovalChain_SkipByPriorityNotMatch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "skip-pri")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		// L1 仅适用 urgent/high；medium 工单应被跳过
		{Level: 1, Role: "manager", Name: "高优先级层", IsRequired: true,
			ConditionPriorities: []string{"urgent", "high"}},
		// L2 无条件，永远适用
		{Level: 2, Role: "manager", Name: "通用层", IsRequired: true},
	})

	// medium 工单 → L1 跳过 → PendingLevel=2
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "medium",
	}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 2)
	assert.True(t, res.Levels[0].Skipped, "L1 应被条件跳过")
	assert.Equal(t, "priority_not_match", res.Levels[0].SkipReason)
	assert.Equal(t, "satisfied", res.Levels[0].Status, "跳过的层视为 satisfied")
	assert.Equal(t, 2, res.PendingLevel, "PendingLevel 应跳到 L2")
	assert.False(t, res.Passed, "L2 仍未批 → Passed=false")
	assert.False(t, res.Blocked)

	// urgent 工单 → L1 不被跳过，正常 pending
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "urgent",
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
	assert.Equal(t, "pending", res.Levels[0].Status)
	assert.Equal(t, 1, res.PendingLevel)
}

// ---- 2. 优先级大小写不敏感 ----
func TestEvaluateApprovalChain_SkipByPriority_CaseInsensitive(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "case")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "高层", IsRequired: true,
			ConditionPriorities: []string{"Urgent", "HIGH"}},
	})

	// 工单优先级 URGENT（不同大小写）也应匹配
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "URGENT",
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped, "大小写不敏感应让 URGENT 匹配")
	assert.Equal(t, "pending", res.Levels[0].Status)
}

// ---- 3. 金额区间：低于最小值 → 跳过 ----
func TestEvaluateApprovalChain_SkipByAmountBelowMin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "amt-min")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		// L1 仅适用金额 >= 10000
		{Level: 1, Role: "manager", Name: "大额层", IsRequired: true,
			ConditionAmountMin: 10000},
		// L2 无条件
		{Level: 2, Role: "manager", Name: "通用层", IsRequired: true},
	})

	// amount=5000 < 10000 → L1 跳过
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Amount:   5000,
	}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].Skipped)
	assert.Equal(t, "amount_below_min", res.Levels[0].SkipReason)
	assert.Equal(t, 2, res.PendingLevel)

	// amount=50000 >= 10000 → L1 不跳过
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Amount:   50000,
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
	assert.Equal(t, 1, res.PendingLevel)
}

// ---- 4. 金额上限：超过最大值 → 跳过 ----
func TestEvaluateApprovalChain_SkipByAmountAboveMax(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "amt-max")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "小额层", IsRequired: true,
			ConditionAmountMax: 1000},
	})

	// amount=50000 > 1000 → L1 跳过
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Amount:   50000,
	}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].Skipped)
	assert.Equal(t, "amount_above_max", res.Levels[0].SkipReason)

	// amount=500 ≤ 1000 → L1 不跳过
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Amount:   500,
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
}

// ---- 5. 金额区间（含）：边界值视为匹配 ----
func TestEvaluateApprovalChain_SkipByAmountRangeBoundary(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "amt-bnd")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "区间层", IsRequired: true,
			ConditionAmountMin: 1000,
			ConditionAmountMax: 5000},
	})

	// amount=1000（等于下界）应匹配
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID, Amount: 1000}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)

	// amount=5000（等于上界）应匹配
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID, Amount: 5000}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)

	// amount=999 低于下界 → 跳过
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID, Amount: 999}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].Skipped)

	// amount=5001 高于上界 → 跳过
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID, Amount: 5001}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].Skipped)
}

// ---- 6. 优先级 + 金额组合条件（AND 语义）----
func TestEvaluateApprovalChain_SkipByCombinedConditions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "combo")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		// L1 要求 urgent 且 amount >= 10000
		{Level: 1, Role: "manager", Name: "VIP 层", IsRequired: true,
			ConditionPriorities: []string{"urgent"},
			ConditionAmountMin:  10000},
	})

	tests := []struct {
		name        string
		priority    string
		amount      float64
		wantSkipped bool
		wantReason  string
	}{
		{"urgent + 50000 (both match)", "urgent", 50000, false, ""},
		{"urgent + 5000 (amount too low)", "urgent", 5000, true, "amount_below_min"},
		{"medium + 50000 (priority miss)", "medium", 50000, true, "priority_not_match"},
		{"low + 100 (both miss)", "low", 100, true, "priority_not_match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
				TenantID: tn.ID,
				Priority: tt.priority,
				Amount:   tt.amount,
			}, nil)
			require.NoError(t, err)
			if tt.wantSkipped {
				assert.True(t, res.Levels[0].Skipped, "应被跳过")
				assert.Equal(t, tt.wantReason, res.Levels[0].SkipReason)
			} else {
				assert.False(t, res.Levels[0].Skipped, "不应被跳过")
			}
		})
	}
}

// ---- 7. 多层混合：L1 跳过、L2 适用、L3 跳过 → 只有 L2 pending ----
func TestEvaluateApprovalChain_MixedSkipAndApply(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "mix")
	m1 := mkEvalUser(t, ctx, client, tn.ID, "manager", "m1")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m2")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		// L1 仅适用 urgent
		{Level: 1, Role: "manager", Name: "紧急层", IsRequired: true,
			ConditionPriorities: []string{"urgent"}},
		// L2 永远适用
		{Level: 2, Role: "manager", Name: "通用层", IsRequired: true},
		// L3 仅适用 amount > 100000
		{Level: 3, Role: "manager", Name: "巨额层", IsRequired: true,
			ConditionAmountMin: 100000},
	})

	// medium + 5000 → L1 跳、L2 待、L3 跳
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "medium",
		Amount:   5000,
	}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 3)
	assert.True(t, res.Levels[0].Skipped, "L1 skip")
	assert.False(t, res.Levels[1].Skipped, "L2 apply")
	assert.True(t, res.Levels[2].Skipped, "L3 skip")
	assert.Equal(t, "satisfied", res.Levels[0].Status)
	assert.Equal(t, "pending", res.Levels[1].Status)
	assert.Equal(t, "satisfied", res.Levels[2].Status)
	assert.Equal(t, 2, res.PendingLevel, "PendingLevel 应跳到 L2")

	// urgent + 5000 → L1 待、L2 待、L3 跳
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "urgent",
		Amount:   5000,
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
	assert.Equal(t, "pending", res.Levels[0].Status)
	assert.Equal(t, 1, res.PendingLevel)

	// urgent + 200000 → L1 待、L2 待、L3 待
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "urgent",
		Amount:   200000,
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[2].Skipped)
	assert.Equal(t, 1, res.PendingLevel)

	// L1 + L2 都批准（用 L1 的 manager），L3 仍被跳过 → 整体 satisfied
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "medium",
		Amount:   5000,
	}, map[int][]int{
		1: {m1.ID}, // L1 跳过的层忽略
		2: {m1.ID}, // L2 批准
	})
	require.NoError(t, err)
	assert.Equal(t, "satisfied", res.Levels[1].Status)
	assert.True(t, res.Passed, "L1/L3 跳过 + L2 通过 → Passed=true")
	assert.Equal(t, 0, res.PendingLevel)
}

// ---- 8. 无条件 step 与旧行为完全兼容 ----
func TestEvaluateApprovalChain_NoConditionBackwardsCompatible(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "compat")
	m1 := mkEvalUser(t, ctx, client, tn.ID, "manager", "m1")
	m2 := mkEvalUser(t, ctx, client, tn.ID, "manager", "m2")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "L1", IsRequired: true}, // 无条件
		{Level: 2, Role: "manager", Name: "L2", IsRequired: true}, // 无条件
	})

	// 验证 1：均未批准 → 两层都 pending，无 skip
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "low",
		Amount:   1,
	}, nil)
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
	assert.False(t, res.Levels[1].Skipped)
	assert.False(t, res.Passed)
	assert.Equal(t, 1, res.PendingLevel)

	// 验证 2：L1+L2 都批准 → passed
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "low",
		Amount:   1,
	}, map[int][]int{1: {m1.ID}, 2: {m2.ID}})
	require.NoError(t, err)
	assert.False(t, res.Levels[0].Skipped)
	assert.False(t, res.Levels[1].Skipped)
	assert.True(t, res.Passed)
	assert.Equal(t, 0, res.PendingLevel)
}

// ---- 9. stepAppliesToContext 纯函数测试 ----
func TestStepAppliesToContext_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		step        schema.ApprovalChainStep
		evalCtx     ApprovalEvalContext
		wantMatch   bool
		wantReason  string
	}{
		{
			name:      "no conditions always matches",
			step:      schema.ApprovalChainStep{},
			evalCtx:   ApprovalEvalContext{Priority: "anything", Amount: 999},
			wantMatch: true,
		},
		{
			name: "priority in whitelist",
			step: schema.ApprovalChainStep{ConditionPriorities: []string{"urgent", "high"}},
			evalCtx:     ApprovalEvalContext{Priority: "urgent"},
			wantMatch:   true,
		},
		{
			name: "priority not in whitelist",
			step: schema.ApprovalChainStep{ConditionPriorities: []string{"urgent", "high"}},
			evalCtx:     ApprovalEvalContext{Priority: "low"},
			wantMatch:   false,
			wantReason:  "priority_not_match",
		},
		{
			name: "amount within range",
			step: schema.ApprovalChainStep{ConditionAmountMin: 100, ConditionAmountMax: 1000},
			evalCtx:     ApprovalEvalContext{Amount: 500},
			wantMatch:   true,
		},
		{
			name: "amount below min",
			step: schema.ApprovalChainStep{ConditionAmountMin: 100},
			evalCtx:     ApprovalEvalContext{Amount: 99},
			wantMatch:   false,
			wantReason:  "amount_below_min",
		},
		{
			name: "amount above max",
			step: schema.ApprovalChainStep{ConditionAmountMax: 1000},
			evalCtx:     ApprovalEvalContext{Amount: 1001},
			wantMatch:   false,
			wantReason:  "amount_above_max",
		},
		{
			name: "amount zero with min condition set → min check skipped (only checks >0)",
			step: schema.ApprovalChainStep{ConditionAmountMin: 0, ConditionAmountMax: 1000},
			evalCtx:     ApprovalEvalContext{Amount: 5000},
			wantMatch:   false,
			wantReason:  "amount_above_max",
		},
		{
			name: "priority case insensitive",
			step: schema.ApprovalChainStep{ConditionPriorities: []string{"URGENT"}},
			evalCtx:     ApprovalEvalContext{Priority: "urgent"},
			wantMatch:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, reason := stepAppliesToContext(tt.step, tt.evalCtx)
			assert.Equal(t, tt.wantMatch, match)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

// ---- 10. 同一 level 内多 step 用「或」语义：任一 step 条件匹配即视为本层适用 ----
func TestEvaluateApprovalChain_MultiStepConditionOR(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	ctx := context.Background()

	tn := mkEvalTenant(t, ctx, client, "or-step")
	_ = mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		// L1 有两个 step：
		//   step1: 优先级=urgent
		//   step2: 金额 >= 50000
		// 工单 (medium, 60000) 应匹配 step2，不被跳过
		{Level: 1, Role: "manager", Name: "S1 urgent", IsRequired: true,
			ConditionPriorities: []string{"urgent"}},
		{Level: 1, Role: "manager", Name: "S2 amount", IsRequired: true,
			ConditionAmountMin: 50000},
	})

	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{
		TenantID: tn.ID,
		Priority: "medium",
		Amount:   60000,
	}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 1)
	assert.False(t, res.Levels[0].Skipped, "任一 step 条件匹配即视为本层适用（OR 语义）")
}
