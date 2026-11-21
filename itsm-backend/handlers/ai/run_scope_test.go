package ai

// 运行轮次读面的**租户隔离**守卫（2026-11-21）。
//
// 背景：run_id 是调用方可传入的路径参数。若读面只按 run_id 查，
// 任何人拿到别人的 run_id 就能读出「另一个租户的 Agent 这一轮做了什么」
// （工具名、参数、结果、审批态、策略版本全在里面）。
//
// 用真实 ent 内存库跑：mock 会直接返回零值，谓词写错根本测不出来，
// 必须让 SQL 真的执行——这是本仓反复踩过的坑（读面 fail-open 比报错危险）。

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
)

// runScopeFixture 汇总测试需要的实体 ID，避免用包级 map 传递（包级状态
// 在并行测试下会互相污染，也让失败信息难以定位）。
type runScopeFixture struct {
	repo    *EntRepository
	tenantA int
	tenantB int
	userA   int
	userB   int
}

func newRunScopeRepo(t *testing.T) runScopeFixture {
	t.Helper()
	// DSN 用 t.Name() 区分：同包内多个用例共用 cache=shared 的内存库时，
	// 表结构与数据会互相可见，租户 ID 递增会让断言错位。
	dsn := "file:ai_run_scope_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("tenant-a").SetCode("tenant-a").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("tenant-b").SetCode("tenant-b").Save(ctx)
	require.NoError(t, err)

	// tool_invocations 的 user_id / conversation_id 是真实外键（schema 里有
	// edge.From user / conversation）。写 0 会触发 FOREIGN KEY constraint，
	// 与「未关联」的语义冲突——所以 fixture 必须造真实用户，不能省。
	userA, err := client.User.Create().SetUsername("run-a").SetName("run-a").
		SetEmail("run-a@example.com").SetPasswordHash("x").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	userB, err := client.User.Create().SetUsername("run-b").SetName("run-b").
		SetEmail("run-b@example.com").SetPasswordHash("x").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	return runScopeFixture{
		repo:    &EntRepository{client: client},
		tenantA: tenantA.ID,
		tenantB: tenantB.ID,
		userA:   userA.ID,
		userB:   userB.ID,
	}
}

// seedRunInvocation 在指定租户下写一条带 run_id 的工具调用记录。
func seedRunInvocation(t *testing.T, f runScopeFixture, tenantID, userID int, runID, toolName string) *ToolInvocation {
	t.Helper()
	inv, err := f.repo.CreateToolInvocation(context.Background(), &ToolInvocation{
		TenantID:        tenantID,
		UserID:          userID,
		ToolName:        toolName,
		Arguments:       "{}",
		Status:          "executed",
		ApprovalState:   "auto",
		PermissionCheck: "passed",
		PolicyVersion:   "pv-test-" + runID,
		RunID:           runID,
	})
	require.NoError(t, err)
	return inv
}

func TestListToolInvocationsByRunIsTenantScoped(t *testing.T) {
	f := newRunScopeRepo(t)
	ctx := context.Background()
	const sharedRun = "run_shared_across_tenants"

	// 同一 run_id 字符串在两个租户下各有一条（攻击者拿到 run_id 串就能撞上）。
	seedRunInvocation(t, f, f.tenantA, f.userA, sharedRun, "list_tickets")
	seedRunInvocation(t, f, f.tenantB, f.userB, sharedRun, "list_cis")

	got, err := f.repo.ListToolInvocationsByRun(ctx, f.tenantA, sharedRun)
	require.NoError(t, err)
	require.Len(t, got, 1, "跨租户读到了他人运行记录（谓词缺失）")
	require.Equal(t, "list_tickets", got[0].ToolName)
	require.Equal(t, f.tenantA, got[0].TenantID)

	gotB, err := f.repo.ListToolInvocationsByRun(ctx, f.tenantB, sharedRun)
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	require.Equal(t, "list_cis", gotB[0].ToolName)
}

func TestListToolInvocationsByRunEmptyReturnsNothing(t *testing.T) {
	// 空 run_id 语义是「不属于任何运行轮次」。此时若落进全表查询，
	// 等于把租户内所有历史调用都吐出来——一个空参数就能拉走整份审计。
	f := newRunScopeRepo(t)
	ctx := context.Background()
	seedRunInvocation(t, f, f.tenantA, f.userA, "", "list_tickets")
	seedRunInvocation(t, f, f.tenantA, f.userA, "run_x", "list_cis")

	got, err := f.repo.ListToolInvocationsByRun(ctx, f.tenantA, "")
	require.NoError(t, err)

	require.Empty(t, got, "空 run_id 返回了记录：会退化为全量读取")
}

func TestListToolInvocationsByRunOrdersChronologically(t *testing.T) {
	// 证据链必须按发生顺序读。倒序会把「先判定后执行」读反，
	// 复盘时因果颠倒比缺数据更危险。
	f := newRunScopeRepo(t)
	ctx := context.Background()
	const runID = "run_order"

	first := seedRunInvocation(t, f, f.tenantA, f.userA, runID, "list_tickets")
	seedRunInvocation(t, f, f.tenantA, f.userA, runID, "get_ci_impact")
	last := seedRunInvocation(t, f, f.tenantA, f.userA, runID, "list_kb")

	got, err := f.repo.ListToolInvocationsByRun(ctx, f.tenantA, runID)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, first.ID, got[0].ID)
	require.Equal(t, last.ID, got[2].ID)
}

func TestCreateToolInvocationPersistsPolicyVersionAndRunID(t *testing.T) {
	// 四层映射守卫：schema → 实体 → 仓储 → 响应。
	// 漏掉 Set 会让字段永远是空串，而 HTTP 依然 200——本仓的典型假绿形态。
	f := newRunScopeRepo(t)
	ctx := context.Background()

	created, err := f.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:      f.tenantA,
		UserID:        f.userA,
		ToolName:      "update_ticket",
		Arguments:     "{}",
		Status:        "pending",
		ApprovalState: "pending_agent",
		RunID:         "run_persist_check",
		PolicyVersion: "9f86d081884c7d659a2feaa0c55ad015",
		RequestID:     "req_persist_check",
	})
	require.NoError(t, err)
	require.Equal(t, "run_persist_check", created.RunID)
	require.Equal(t, "9f86d081884c7d659a2feaa0c55ad015", created.PolicyVersion)

	// 读回来再核一次：写入成功不代表读映射也对（toToolInvocationDomain 漏字段
	// 同样会让 API 层拿不到值）。
	loaded, err := f.repo.GetToolInvocation(ctx, created.ID, f.tenantA)
	require.NoError(t, err)
	require.Equal(t, "run_persist_check", loaded.RunID)
	require.Equal(t, "9f86d081884c7d659a2feaa0c55ad015", loaded.PolicyVersion)
	require.Equal(t, "pending_agent", loaded.ApprovalState, "审批态不能被新字段挤掉")
}

// CountToolInvocationsByApprovalState 的仓储层守卫（2026-11-21）。
//
// 为什么要单独测仓储而不是只经聚合面间接验证：聚合面用例用的是 sqlite 内存库
// + 真实仓储，但队列断言只覆盖「四态分离」；分组计数的**谓词**（dry_run 排除、
// 租户隔离）若写错，聚合面那边只有 fail-closed 用例能碰上，精确性无锁。
//
// 另外这层必须用真实 SQL 跑：ent 的 GroupBy().Scan() 在零行时**不写目标切片**，
// 且列名映射靠 struct tag（`json:"count"`）——mock 永远看不出这两类错。
func TestCountToolInvocationsByApprovalStateIsExactAndScoped(t *testing.T) {
	f := newRunScopeRepo(t)
	ctx := context.Background()

	seedApproval := func(tenantID, userID int, state string, dryRun bool) {
		_, err := f.repo.CreateToolInvocation(ctx, &ToolInvocation{
			TenantID:        tenantID,
			UserID:          userID,
			ToolName:        "create_ticket",
			Arguments:       "{}",
			Status:          "pending",
			ApprovalState:   state,
			PermissionCheck: "passed",
			DryRun:          dryRun,
		})
		require.NoError(t, err)
	}

	// 本租户：四种待批态各按不同条数播种，故意让 pending=1 与其它态不同，
	// 这样「pending 被算成四态之和」的错误会立刻显形。
	seedApproval(f.tenantA, f.userA, "pending", false)
	seedApproval(f.tenantA, f.userA, "pending_agent", false)
	seedApproval(f.tenantA, f.userA, "pending_agent", false)
	seedApproval(f.tenantA, f.userA, "pending_agent_two", false)
	seedApproval(f.tenantA, f.userA, "pending_agent_two", false)
	seedApproval(f.tenantA, f.userA, "pending_agent_two", false)
	seedApproval(f.tenantA, f.userA, "awaiting_second", false)
	seedApproval(f.tenantA, f.userA, "awaiting_second", false)
	seedApproval(f.tenantA, f.userA, "awaiting_second", false)
	seedApproval(f.tenantA, f.userA, "awaiting_second", false)

	// 预演行：dry_run=true，不得计入任何队列。
	// 用 pending 态播种是因为那是最容易被误算进 Pending 的一类。
	seedApproval(f.tenantA, f.userA, "pending", true)
	seedApproval(f.tenantA, f.userA, "awaiting_second", true)

	// 他人租户：全部四态各一条，一条都不该出现在本租户计数里。
	seedApproval(f.tenantB, f.userB, "pending", false)
	seedApproval(f.tenantB, f.userB, "pending_agent", false)
	seedApproval(f.tenantB, f.userB, "pending_agent_two", false)
	seedApproval(f.tenantB, f.userB, "awaiting_second", false)

	counts, err := f.repo.CountToolInvocationsByApprovalState(ctx, f.tenantA)
	require.NoError(t, err)

	require.Equal(t, 1, counts["pending"],
		"pending 必须是单态精确计数；若等于四态之和，说明用了 state=\"pending\" 的合并查询")
	require.Equal(t, 2, counts["pending_agent"])
	require.Equal(t, 3, counts["pending_agent_two"])
	require.Equal(t, 4, counts["awaiting_second"],
		"awaiting_second 独立计数：已有一票、首票不放行，最容易卡住")

	// 预演行与跨租户行都不得出现：总数必须等于 1+2+3+4。
	total := 0
	for _, n := range counts {
		total += n
	}
	require.Equal(t, 10, total,
		"总数 10 = 本租户真实待批；多出来就是预演行或他人租户混入")
}

// 空租户必须返回空 map 而不是 nil：nil 序列化成 null，{} 序列化成 {}，
// 调用方无法区分「查询没跑」与「确实没有待批」。
func TestCountToolInvocationsByApprovalStateEmptyTenantReturnsEmptyMap(t *testing.T) {
	f := newRunScopeRepo(t)
	ctx := context.Background()

	counts, err := f.repo.CountToolInvocationsByApprovalState(ctx, f.tenantA)
	require.NoError(t, err)
	require.NotNil(t, counts, "返回 nil map 会让空队列与查询失败在序列化后不可区分")
	require.Empty(t, counts)
	require.Equal(t, 0, counts["pending"], "空 map 读缺失键必须得 0，而不是 panic")
}
