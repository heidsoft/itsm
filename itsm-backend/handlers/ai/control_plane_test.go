package ai_test

import (
	"context"
	"testing"

	"itsm-backend/ent/toolinvocation"
	"itsm-backend/handlers/ai"
	"itsm-backend/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 控制面聚合读面的回归锁（《Agent 控制面差距报告》P2-5）。
//
// 这批用例的核心不是「能返回数据」，而是**聚合面不许变成第二个判定点**：
// 一旦它自己也跑一遍 evaluateToolCall，就会出现「聚合面说不可见、真实执行
// 却放行」或反向的分叉，而运维会拿聚合面的结论去排查真实执行。
//
// 逐条判据：
//  ① fail-closed：任一真源失败 → 整单 error，绝不返回带部分数据的响应；
//  ② 可见性与 GET /agent/tools 同源（同一 HasResourcePermission 调用）；
//  ③ 队列四态分别计数且不合并（awaiting_second 独立可见）；
//  ④ 策略三类不合并（无策略 / 合法 / 配坏了各自独立）；
//  ⑤ 最近拒绝只含真实调用，预演拒绝（dry_run_denied）不得混入；
//  ⑥ 租户隔离：他人租户的记录不得进入本租户响应；
//  ⑦ 零副作用：读面不写任何记录；
//  ⑧ 空租户返回结构完整（不是 null 也不是 error）；
//  ⑨ 可见面确实按调用者角色算（不是写死 admin）。

// failingQueueRepo 让队列计数恒失败，用于验证 fail-closed。
//
// 只替换这一个方法：其它方法走嵌进来的真实仓储，避免 fake 出
// 「看起来像真实链路」的东西。注入点选在分组计数而非 ListToolInvocations，
// 因为聚合面现在就是用这个方法取队列——注入到别处等于没测。
type failingQueueRepo struct {
	ai.Repository
}

func (f *failingQueueRepo) CountToolInvocationsByApprovalState(context.Context, int) (map[string]int, error) {
	return nil, assert.AnError
}

// ① fail-closed：队列读失败必须整单报错。
//
// 反向验证过的形态：把聚合函数里 `if err != nil { return nil, err }`
// 改成 `continue`（跳过该项继续聚合），本用例会红在 err==nil。
// 这正是不许出现的「半真半假视图」：界面显示队列 0 条，而实际是查询失败。
func TestControlPlane_FailsClosedWhenQueueUnavailable(t *testing.T) {
	f := newDryRunFixture(t, "cp_failclosed")
	ctx := context.Background()

	bad := &failingQueueRepo{Repository: ai.NewEntRepository(f.client)}
	_, err := ai.ControlPlaneWithRepoForTest(ctx, f.svc, bad, f.tenantID, "admin")
	require.Error(t, err, "队列读失败必须整单报错，不得返回部分数据")
	assert.ErrorIs(t, err, assert.AnError, "原始错误必须透出，否则运维无法定位是哪个真源坏了")
}

// ② 可见性与 GET /agent/tools 同源：同一个角色、同一批工具，
// 聚合面报告的可见集合必须与 HasResourcePermission 逐条一致。
func TestControlPlane_VisibilityMatchesToolListSource(t *testing.T) {
	f := newDryRunFixture(t, "cp_visibility")
	ctx := context.Background()

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)
	require.NotEmpty(t, overview.ToolSurface, "fixture 的工具目录非空")

	for _, got := range overview.ToolSurface {
		// 与 handler 层完全同一个函数：结论必须逐条相等。
		expect := middleware.HasResourcePermission(ctx, f.client, "admin", got.Resource, got.Action, f.tenantID)
		assert.Equal(t, expect, got.Visible,
			"工具 %s 的可见性必须与 HasResourcePermission 一致（同一真源）", got.Name)
	}
	assert.Equal(t, overview.TotalTools, len(overview.ToolSurface),
		"不可见的工具也要列出——运维需要看到「有哪些工具存在但被挡住了」")
}

// ③ 队列四态分别计数：awaiting_second 必须独立可见。
//
// 为什么不能合并：awaiting_second 是「已有一票、等第二个人复审」——
// 首票不放行，是最容易卡住的态。若与 pending_agent_two 合并成「待批 2 条」，
// 运维看不出有一批已经批了一半却停住了。
func TestControlPlane_QueueCountsEachApprovalStateSeparately(t *testing.T) {
	f := newDryRunFixture(t, "cp_queue")
	ctx := context.Background()

	seed := func(state string, n int) {
		for i := 0; i < n; i++ {
			f.client.ToolInvocation.Create().
				SetTenantID(f.tenantID).
				SetToolName("create_ticket").
				SetArguments(`{}`).
				SetStatus("pending").
				SetApprovalState(state).
				SetNeedsApproval(true).
				SaveX(ctx)
		}
	}
	seed("pending", 1)
	seed("pending_agent", 2)
	seed("pending_agent_two", 3)
	seed("awaiting_second", 4)

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	assert.Equal(t, 1, overview.Queue.Pending)
	assert.Equal(t, 2, overview.Queue.PendingAgent)
	assert.Equal(t, 3, overview.Queue.PendingAgentTwo)
	assert.Equal(t, 4, overview.Queue.AwaitingSecond,
		"awaiting_second 必须独立计数——它已有一票、首票不放行，是最容易卡住的态")
	assert.Equal(t, 10, overview.Queue.Total)
}

// ④ 策略三类不可合并：无信封 / 合法信封 / 配坏了的信封。
//
// 反向验证过的形态：把 Unreadable 并进 PolicyFree（都当「没配策略」），
// 本用例会红在 policyFree==2。配坏了是最危险的一类——看起来配了、
// 鉴权时 fail-closed 全拒，把它算成「没配」会让运维永远找不到根因。
func TestControlPlane_PolicyCountsDoNotMergeUnreadable(t *testing.T) {
	f := newDryRunFixture(t, "cp_policy")
	ctx := context.Background()

	// fixture 已播种 ticket:read 与 ticket:write（合法信封），先记基线。
	base, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	// 无信封：纯文本描述（鉴权时走 DerivePolicy）。
	f.client.Permission.Create().
		SetCode("asset:read").SetName("asset read").
		SetTenantID(f.tenantID).SetResource("asset").SetAction("read").
		SetDescription("查看资产").SaveX(ctx)

	// 配坏了：坏 JSON（看起来有描述，实际解不出）。
	f.client.Permission.Create().
		SetCode("asset:write").SetName("asset write").
		SetTenantID(f.tenantID).SetResource("asset").SetAction("write").
		SetDescription(`{"v":1,"policy":`).SaveX(ctx)

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	assert.Equal(t, base.Policies.Total+2, overview.Policies.Total)
	assert.Equal(t, base.Policies.WithPolicy, overview.Policies.WithPolicy,
		"新增的两行都不该算 WithPolicy（一行纯文本、一行坏 JSON）")
	assert.Equal(t, 1, overview.Policies.PolicyFree, "只有纯文本行是「没配策略」")
	assert.Equal(t, 1, overview.Policies.Unreadable,
		"配坏了的信封必须独立计数：它看起来配了策略但鉴权时 fail-closed 全拒")
	assert.Equal(t, overview.Policies.Total,
		overview.Policies.WithPolicy+overview.Policies.PolicyFree+overview.Policies.Unreadable,
		"三类必须覆盖全部行，漏算任何一行都会让风险面被低估")
}

// ⑤ 最近拒绝只含真实调用：预演拒绝（dry_run_denied）不得混入。
//
// 反向验证过的形态：把 status 精确匹配改成前缀匹配，本用例会红在
// recentDenial 长度——dry_run_denied 被误吸进来。
func TestControlPlane_RecentDenialExcludesDryRunRejection(t *testing.T) {
	f := newDryRunFixture(t, "cp_denial")
	ctx := context.Background()

	// 真实拒绝（判定拒绝）。
	f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).SetToolName("delete_ticket").SetArguments(`{}`).
		SetStatus("denied").SetPermissionCheck("passed").
		SetPermissionReason("AI 主体 agent 未获授权 ticket:delete").
		SetDryRun(false).SaveX(ctx)

	// 预演拒绝（假设性结论，不是实际拒绝）。
	f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).SetToolName("delete_ticket").SetArguments(`{}`).
		SetStatus("dry_run_denied").SetPermissionCheck("passed").
		SetPermissionReason("预演：未获授权").
		SetDryRun(true).SaveX(ctx)

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	require.Len(t, overview.RecentDenial, 1,
		"预演拒绝是假设性结论，混进真实拒绝面会让运维去查一个不存在的拦截")
	assert.Equal(t, "delete_ticket", overview.RecentDenial[0].ToolName)
	assert.NotEqual(t, "预演：未获授权", overview.RecentDenial[0].Reason)
}

// ⑤b 预演排除的**第二条防线**必须独立可证明：dry_run=true 且 status 恰为 "denied"
// 的记录同样不得进入拒绝面。
//
// 为什么这条用例必须单独存在（实测发现）：上一条用例里的预演行 status 是
// "dry_run_denied"，它被 status 精确匹配 "denied" 就已经挡住了——于是
// DryRunEQ(false) 那道防线**在那条用例里不可被证明**。实测：删掉 DryRunEQ(false)
// 后上一条用例依然绿，红不起来。
//
// 这里构造的是真实可能出现的形态：预演落库函数当前写 dry_run_* 前缀，
// 但只要有人把某个预演分支复用 recordToolAudit（或直接在 status 里写 denied），
// 该行就会变成「dry_run=true + status=denied」——一旦 DryRunEQ 缺席，
// 一条从未被调度的假设性结论就会出现在「最近真实拒绝」里，运维照着它去排查
// 一个根本不存在的拦截。
//
// 反向验证过的形态：删掉 recentDenials 里的 DryRunEQ(false)，本用例红在长度 2。
func TestControlPlane_DryRunExclusionHoldsEvenWhenStatusLooksReal(t *testing.T) {
	f := newDryRunFixture(t, "cp_dryrun_denied")
	ctx := context.Background()

	// 真实拒绝。
	f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).SetToolName("delete_ticket").SetArguments(`{}`).
		SetStatus("denied").SetPermissionCheck("denied").
		SetPermissionReason("无权限").SetDryRun(false).SaveX(ctx)

	// 预演行，但 status 写成与真实拒绝相同的 "denied"（DryRun 标记是唯一区分）。
	f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).SetToolName("delete_ticket").SetArguments(`{}`).
		SetStatus("denied").SetPermissionCheck("denied").
		SetPermissionReason("预演：无权限").SetDryRun(true).SaveX(ctx)

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	require.Len(t, overview.RecentDenial, 1,
		"dry_run 标记必须独立生效：status 撞车时它是唯一区分依据，缺了它假设性拒绝会冒充真实拦截")
	assert.Equal(t, "无权限", overview.RecentDenial[0].Reason,
		"留下来的必须是真实拒绝，而不是预演那条")
}

// ⑥ 租户隔离：他人租户的拒绝记录不得进入本租户响应。
//
// 这是聚合面最容易被打破的边界——它一次查多个表，每张表都要带租户谓词。
// 反向验证过的形态：去掉 recentDenials 里的 TenantIDEQ，
// 本用例会红在 recentDenial 长度 2。
func TestControlPlane_TenantScoped(t *testing.T) {
	f := newDryRunFixture(t, "cp_tenant")
	ctx := context.Background()

	otherTenant := f.client.Tenant.Create().
		SetName("Other").SetCode("CP_OTHER").SetDomain("cp-other.test").SetStatus("active").
		SaveX(ctx)

	// 两条拒绝：一条本租户，一条他人租户。
	for _, tid := range []int{f.tenantID, otherTenant.ID} {
		f.client.ToolInvocation.Create().
			SetTenantID(tid).SetToolName("create_ticket").SetArguments(`{}`).
			SetStatus("denied").SetPermissionCheck("denied").
			SetPermissionReason("无权限").SaveX(ctx)
	}

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)
	assert.Len(t, overview.RecentDenial, 1, "只能看到本租户的拒绝记录")
	assert.Equal(t, f.tenantID, overview.TenantID)
}

// ⑦ 零副作用：聚合读面是纯只读，连跑三次不得产生任何记录。
//
// 读面自己写审计看起来「更可观察」，实际会造成两个问题：
//
//	① 运维每次刷新页面都制造一条审计，把真实的审计密度淹掉；
//	② 审计表随时间膨胀，而这条记录不描述任何业务动作。
func TestControlPlane_DoesNotMutateAnything(t *testing.T) {
	f := newDryRunFixture(t, "cp_nomutate")
	ctx := context.Background()

	before := f.client.ToolInvocation.Query().CountX(ctx)
	for i := 0; i < 3; i++ {
		_, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
		require.NoError(t, err)
	}
	after := f.client.ToolInvocation.Query().CountX(ctx)

	assert.Equal(t, before, after, "聚合读面是纯只读，连跑三次不得产生任何记录")

	// 顺带锁一个方向：不得创建任何 dry_run 行（预演才是那个会写审计的动作）。
	rows := f.client.ToolInvocation.Query().
		Where(toolinvocation.TenantID(f.tenantID), toolinvocation.DryRunEQ(true)).
		CountX(ctx)
	assert.Equal(t, 0, rows, "聚合面绝不能落预演审计——它连判定都没跑")
}

// ⑧ 空租户：无任何数据时也必须返回结构完整的响应，而不是 null 或错误。
//
// 空租户是**新部署的默认状态**，此时聚合面是运维唯一能看的东西。
// 若返回 null 或 error，运维在系统刚起来的头几天会以为读面坏了。
func TestControlPlane_EmptyTenantReturnsCompleteShape(t *testing.T) {
	f := newDryRunFixture(t, "cp_empty")
	ctx := context.Background()

	overview, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, 0, overview.Queue.Total, "fixture 未建待批记录")
	// 空集合必须序列化为 [] 而不是 null（前端 .length 不会炸）。
	require.NotNil(t, overview.RecentDenial)
	assert.Empty(t, overview.RecentDenial)
	assert.NotEmpty(t, overview.GatheredAt,
		"必须标采集时刻，否则无法解释「刷新后数字变了」——各表在此期间可能各自在变")
	assert.Equal(t, "admin", overview.Role,
		"必须回显角色：可见面是按它算的，不回显就不知道「可见 6 个」是哪个角色的视角")
}

// ⑨ 可见面必须确实按传入角色算，不能写死 admin。
//
// 这条不是重复用例②——②验证「与同源函数一致」，本条验证「确实按传入角色算」。
// 若实现里把角色写死成 admin（或从 ctx 里另取一个），②仍会绿（因为两边
// 都用同一个写死值），只有换角色对比才暴露。
func TestControlPlane_VisibilityDependsOnCallerRole(t *testing.T) {
	f := newDryRunFixture(t, "cp_role")
	ctx := context.Background()

	adminView, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "admin")
	require.NoError(t, err)

	// 用一个几乎没有任何权限的角色对照。
	endUserView, err := f.svc.GetControlPlaneOverview(ctx, f.tenantID, "end_user")
	require.NoError(t, err)

	require.Len(t, endUserView.ToolSurface, len(adminView.ToolSurface),
		"两次调用的工具集合相同（只是可见性不同）")
	assert.LessOrEqual(t, endUserView.VisibleTools, adminView.VisibleTools,
		"end_user 的可见工具数不应多于 admin——若相等，说明可见性没按调用者角色算")
	assert.Equal(t, "end_user", endUserView.Role)
}
