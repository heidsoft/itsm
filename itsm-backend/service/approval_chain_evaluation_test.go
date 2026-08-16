package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 审批链 fallback 求值引擎测试 ====================
// 覆盖审计标记的缺陷：层级顺序、会签(AND)/或签(OR)、兜底四策略、
// 跨租户隔离（堵「无审批人时自审批 / 跨租户注入」）、非必需层自动通过。

func newEvalClient(t *testing.T) (*ent.Client, *ApprovalChainService, context.Context) {
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewApprovalChainService(client, logger)
	return client, svc, context.Background()
}

func mkEvalTenant(t *testing.T, ctx context.Context, client *ent.Client, suffix string) *ent.Tenant {
	tn, err := client.Tenant.Create().
		SetName("TN-" + suffix).
		SetCode("tn" + suffix).
		SetDomain("tn" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tn
}

func mkEvalUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, role, suffix string) *ent.User {
	u, err := client.User.Create().
		SetUsername("u_" + suffix).
		SetEmail("u_" + suffix + "@example.com").
		SetName("User " + suffix).
		SetPasswordHash("h").
		SetRole(user.Role(role)).
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u
}

// 直接构造一条审批链实体（绕过 service 以便精准构造各层级/会签/兜底字段）。
func mkChainEntity(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, entityType string, steps []schema.ApprovalChainStep) *ent.ApprovalChain {
	c, err := client.ApprovalChain.Create().
		SetName("chain-" + entityType).
		SetEntityType(entityType).
		SetChain(steps).
		SetStatus("active").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return c
}

// ---- 1. 层级顺序：乱序输入应升序求值 ----
func TestEvaluateApprovalChain_LevelOrdering(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "ord")
	mgr := mkEvalUser(t, ctx, client, tn.ID, "manager", "m")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 3, Role: "manager", Name: "L3", IsRequired: true},
		{Level: 1, Role: "manager", Name: "L1", IsRequired: true},
		{Level: 2, Role: "manager", Name: "L2", IsRequired: true},
	})

	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID, RequesterID: mgr.ID}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{res.Levels[0].Level, res.Levels[1].Level, res.Levels[2].Level})
	assert.Equal(t, 1, res.PendingLevel) // 首层未批准 → 待办=1
	assert.False(t, res.Passed)
}

// ---- 2. 会签(AND)：需全部批准 ----
func TestEvaluateApprovalChain_QuorumAND(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "and")
	u1 := mkEvalUser(t, ctx, client, tn.ID, "manager", "a")
	u2 := mkEvalUser(t, ctx, client, tn.ID, "manager", "b")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "会签", IsRequired: true, ApprovalType: "parallel"},
	})

	// 无人批准
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 1)
	assert.Equal(t, 2, res.Levels[0].Threshold, "会签默认阈值=审批人数")
	assert.Equal(t, "pending", res.Levels[0].Status)

	// 仅一人批准 → 仍 pending
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, map[int][]int{1: {u1.ID}})
	require.NoError(t, err)
	assert.Equal(t, "pending", res.Levels[0].Status)

	// 两人都批准 → satisfied
	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, map[int][]int{1: {u1.ID, u2.ID}})
	require.NoError(t, err)
	assert.Equal(t, "satisfied", res.Levels[0].Status)
	assert.True(t, res.Passed)
}

// ---- 3. 或签(OR)：任一批准即可 ----
func TestEvaluateApprovalChain_QuorumOR(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "or")
	u1 := mkEvalUser(t, ctx, client, tn.ID, "manager", "a")
	mkEvalUser(t, ctx, client, tn.ID, "manager", "b")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "manager", Name: "或签", IsRequired: true, ApprovalType: "serial"},
	})

	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Levels[0].Threshold, "或签阈值=1")
	assert.Equal(t, "pending", res.Levels[0].Status)

	res, err = svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, map[int][]int{1: {u1.ID}})
	require.NoError(t, err)
	assert.Equal(t, "satisfied", res.Levels[0].Status)
	assert.True(t, res.Passed)
}

// ---- 4. fallback=block（默认，失败关闭）----
func TestEvaluateApprovalChain_FallbackBlock(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "blk")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "no_such_role", Name: "必需但无人", IsRequired: true, FallbackAction: FallbackBlock},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	require.Len(t, res.Levels, 1)
	assert.True(t, res.Levels[0].FallbackTriggered)
	assert.Equal(t, "blocked", res.Levels[0].Status)
	assert.True(t, res.Blocked)
	assert.False(t, res.Passed)
}

// ---- 5. fallback=auto_approve ----
func TestEvaluateApprovalChain_FallbackAutoApprove(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "aa")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "no_such_role", Name: "必需但无人", IsRequired: true, FallbackAction: FallbackAutoApprove},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].FallbackTriggered)
	assert.Equal(t, "satisfied", res.Levels[0].Status)
	assert.True(t, res.Passed)
}

// ---- 6. fallback=escalate（升级到显式兜底审批人）----
func TestEvaluateApprovalChain_FallbackEscalate(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "esc")
	esc := mkEvalUser(t, ctx, client, tn.ID, "manager", "esc")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{
			Level: 1, Role: "no_such_role", Name: "必需但无人", IsRequired: true,
			FallbackAction: FallbackEscalate, FallbackApproverID: esc.ID,
		},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	assert.True(t, res.Levels[0].FallbackTriggered)
	assert.Equal(t, "pending", res.Levels[0].Status, "升级后转为待新审批人处理")
	assert.True(t, intInSlice(esc.ID, res.Levels[0].ApproverIDs), "兜底审批人应出现在候选中")
	assert.False(t, res.Blocked)
}

// ---- 7. fallback=auto_reject ----
func TestEvaluateApprovalChain_FallbackAutoReject(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "ar")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "no_such_role", Name: "必需但无人", IsRequired: true, FallbackAction: FallbackAutoReject},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, "blocked", res.Levels[0].Status)
	assert.True(t, res.Blocked)
}

// ---- 8. 跨租户隔离：显式 ApproverID 属于其他租户 → 解析为空 → fallback（堵跨租户注入）----
func TestEvaluateApprovalChain_CrossTenantIsolation(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tnA := mkEvalTenant(t, ctx, client, "A")
	tnB := mkEvalTenant(t, ctx, client, "B")
	uB := mkEvalUser(t, ctx, client, tnB.ID, "manager", "other") // 属于租户 B

	// 在租户 A 的求值上下文里，引用租户 B 的用户作为审批人
	chain := mkChainEntity(t, ctx, client, tnA.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, ApproverID: uB.ID, Role: "", Name: "越权审批人", IsRequired: true, FallbackAction: FallbackBlock},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tnA.ID}, nil)
	require.NoError(t, err)
	// 跨租户用户被拒绝 → 该层无人可审 → 触发 block fallback
	assert.True(t, res.Levels[0].FallbackTriggered)
	assert.Equal(t, "blocked", res.Levels[0].Status)
	assert.True(t, res.Blocked, "跨租户注入必须失败关闭，绝不能默认自审批")
}

// ---- 9. 非必需层无人 → 自动通过 ----
func TestEvaluateApprovalChain_NonRequiredEmpty(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "nr")

	chain := mkChainEntity(t, ctx, client, tn.ID, "ticket", []schema.ApprovalChainStep{
		{Level: 1, Role: "no_such_role", Name: "可选层无人", IsRequired: false},
	})
	res, err := svc.Evaluate(ctx, chain, ApprovalEvalContext{TenantID: tn.ID}, nil)
	require.NoError(t, err)
	assert.Equal(t, "satisfied", res.Levels[0].Status)
	assert.True(t, res.Passed)
}

// ---- 10. 端到端：经 service 建链 + ResolveApprovalPlan 消费（含 DTO 往返校验）----
func TestResolveApprovalPlan_EndToEnd(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "e2e")
	m1 := mkEvalUser(t, ctx, client, tn.ID, "manager", "m1")
	m2 := mkEvalUser(t, ctx, client, tn.ID, "manager", "m2")
	esc := mkEvalUser(t, ctx, client, tn.ID, "security", "esc")

	// 经 DTO 建链（同时验证此前 DTO 丢弃新字段的缺陷已修复）
	created, err := svc.CreateApprovalChain(ctx, &dto.ApprovalChainRequest{
		Name:       "E2E 链",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "主管会签", IsRequired: true, ApprovalType: "parallel", Threshold: 2},
			{
				Level: 2, Role: "no_such_role", Name: "必需无人层", IsRequired: true,
				FallbackAction: FallbackEscalate, FallbackApproverID: esc.ID,
			},
		},
	}, tn.ID)
	require.NoError(t, err)

	// 读回校验：新字段（会签/兜底）应持久化并往返
	got, err := svc.GetApprovalChain(ctx, created.ID, tn.ID)
	require.NoError(t, err)
	require.Len(t, got.Chain, 2)
	assert.Equal(t, "parallel", got.Chain[0].ApprovalType)
	assert.Equal(t, 2, got.Chain[0].Threshold)
	assert.Equal(t, FallbackEscalate, got.Chain[1].FallbackAction)
	assert.Equal(t, esc.ID, got.Chain[1].FallbackApproverID)

	// 经统一入口求值
	plan, err := svc.ResolveApprovalPlan(ctx, tn.ID, "service_request", ApprovalEvalContext{}, nil)
	require.NoError(t, err)
	require.Len(t, plan.Levels, 2)
	assert.Equal(t, 1, plan.PendingLevel)
	assert.False(t, plan.Passed) // L1 会签未批 + L2 升级待办

	// L1 应解析出两名 manager
	assert.ElementsMatch(t, []int{m1.ID, m2.ID}, plan.Levels[0].ApproverIDs)
	assert.Equal(t, 2, plan.Levels[0].Threshold)

	// L2 必需无人 → 升级到 esc
	assert.True(t, plan.Levels[1].FallbackTriggered)
	assert.Equal(t, "pending", plan.Levels[1].Status)
	assert.True(t, intInSlice(esc.ID, plan.Levels[1].ApproverIDs))
}

// ---- 11. 无匹配链 → ResolveApprovalPlan 报错（fail-closed，不默认放行）----
func TestResolveApprovalPlan_NoChain(t *testing.T) {
	client, svc, ctx := newEvalClient(t)
	defer client.Close()
	tn := mkEvalTenant(t, ctx, client, "nc")

	_, err := svc.ResolveApprovalPlan(ctx, tn.ID, "change", ApprovalEvalContext{}, nil)
	require.Error(t, err, "未配置审批链时必须显式报错，绝不能默认免审/自审批")
}
