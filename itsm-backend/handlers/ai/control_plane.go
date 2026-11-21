package ai

import (
	"context"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/toolinvocation"
	"itsm-backend/internal/authz"
	"itsm-backend/middleware"
)

// ControlPlane 聚合读面：一个请求返回「AI 工具控制面」的当前状态。
//
// 为什么需要它（《Agent 控制面差距报告》P2-5）：
// 判断「策略是否真的生效」目前要分别调 /permissions/:id/policy（策略正文）
// 与 /agent/tools/invocations?state=pending（待批队列），还要自己再跑一次
// 工具清单去比「哪些工具可见」。三个真源、三次请求，运维只能靠人工拼装，
// 于是「昨天改了策略到底有没有生效」这个最常见的问题没有确定答案。
//
// 本读面的**唯一职责是聚合**，不是判断：
//   - 工具可见面直接复用 middleware.HasResourcePermission
//     （与 GET /agent/tools 完全同一函数），不另写一套可见性规则；
//   - 待批队列直接复用 repo.ListToolInvocations（与审批页同源，含 dry_run 排除）；
//   - 策略配置面直接读 permission 表的既有列，用与鉴权相同的 catalog 解码。
//
// 关键纪律：**不在这里跑 evaluateToolCall**。一旦聚合面自己判定一次，
// 它就会与真实执行路径形成第二个判定点，而这正是本项目明令禁止的分叉。
// 聚合面只回答「现在是什么样」，不回答「允许不允许」——后者走 dry-run。
//
// 读面 fail-closed：任一数据源读失败就整单报错，不返回「其余真实 + 该项空」。
// 半真半假的运维视图比没有视图更危险：看到「0 个待批」会让人以为队列清空了。

// controlPlaneQueryTimeout 聚合读面的内部超时。
//
// 用 context.WithoutCancel 派生：调用方（运维页面轮询）断开不应让已经
// 跑完一半的查询白费，也不应让慢查询继续占用 DB 连接。5s 与 root_cause
// 落库同一口径，属本项目的既有约定值。
const controlPlaneQueryTimeout = 5 * time.Second

// ControlPlanePolicySummary 策略配置面摘要（描述配置状态，不是判定结论）。
type ControlPlanePolicySummary struct {
	// Total 租户内 permission 行总数（策略可能只覆盖其中一部分）。
	Total int `json:"total"`
	// WithPolicy 带合法策略信封的行数。
	WithPolicy int `json:"withPolicy"`
	// PolicyFree 无策略信封的行数 = Total - WithPolicy - Unreadable。
	//
	// 单独给出而不是让前端减：这些 resource:action 在鉴权时走
	// authz.DerivePolicy 的推导值（AI 主体默认零权限），而不是管理员显式配置的
	// 策略。前端自己减算错就会低估「有多少权限点没被显式治理」。
	PolicyFree int `json:"policyFree"`
	// Unreadable 信封存在但解不出（坏 JSON / 未知版本）。
	//
	// **这是三类里最危险的一类**：它看起来配了策略，但鉴权时严格解码失败
	// 会 fail-closed 一律拒绝。它既不能并进 WithPolicy（会让运维以为已生效），
	// 也不能并进 PolicyFree（会让运维以为只是没配）。
	Unreadable int `json:"unreadable"`
}

// ControlPlaneQueueSummary 待批队列汇总。四项状态各自独立计数，
// 因为它们的**处理人不同**：
//   - pending          → 人发起的调用，任一人审批即可
//   - pending_agent    → AI 发起的单人审批
//   - pending_agent_two→ AI 发起的双人审批，尚无人投第一票
//   - awaiting_second  → 已有一票，等第二个不同的人复审（首票不放行）
//
// 把后两者合并成「待批 2 条」会误导运维：awaiting_second 的那些**已经有人
// 批过一半**，再不处理就卡住了；而 pending_agent_two 是还没人开始的。
type ControlPlaneQueueSummary struct {
	Pending         int `json:"pending"`
	PendingAgent    int `json:"pendingAgent"`
	PendingAgentTwo int `json:"pendingAgentTwo"`
	AwaitingSecond  int `json:"awaitingSecond"`
	Total           int `json:"total"`
}

// ControlPlaneRecentDenial 最近一次被拒绝的判定（只读投影）。
//
// 只取「被拒绝」而不取全量：允许的调用量随时间线性增长且绝大多数无异常，
// 聚合面若返回全量就会变成一个需要分页的接口；而拒绝是**稀疏事件**，
// 十条就足以看出「哪个工具一直在被挡」。
//
// 预演行（dry_run=true）**不参与**：它的 status 也会是 dry_run_denied，
// 但那是「假设性拒绝」，把它混进真实拒绝面会让运维以为有实际调用被拦。
type ControlPlaneRecentDenial struct {
	ID              int    `json:"id"`
	ToolName        string `json:"toolName"`
	UserID          int    `json:"userId"`
	RoleSnapshot    string `json:"roleSnapshot"`
	PermissionCheck string `json:"permissionCheck"`
	Reason          string `json:"reason"`
	PolicyVersion   string `json:"policyVersion"`
	RunID           string `json:"runId"`
	CreatedAt       string `json:"createdAt"`
}

// ControlPlaneToolSurface 单个工具的可见性（与 GET /agent/tools 同源）。
type ControlPlaneToolSurface struct {
	Name      string `json:"name"`
	Resource  string `json:"resource"`
	Action    string `json:"action"`
	ReadOnly  bool   `json:"readOnly"`
	Visible   bool   `json:"visible"`
	RiskLevel string `json:"riskLevel"`
}

// ControlPlaneOverview 聚合响应。
type ControlPlaneOverview struct {
	TenantID int `json:"tenantId"`
	// Role 是本次请求的调用者角色——visible 面按它算，必须回显。
	//
	// 不回显会让运维困惑：同一租户、同一时刻，管理员看到 14 个工具可见、
	// 普通技术员看到 6 个，界面上的「可见工具数」到底是哪个角色的？
	Role string `json:"role"`

	ToolSurface  []ControlPlaneToolSurface  `json:"toolSurface"`
	VisibleTools int                        `json:"visibleTools"`
	TotalTools   int                        `json:"totalTools"`
	Queue        ControlPlaneQueueSummary   `json:"queue"`
	Policies     ControlPlanePolicySummary  `json:"policies"`
	RecentDenial []ControlPlaneRecentDenial `json:"recentDenial"`

	// GatheredAt 采集时刻。读面是多个真源的快照，不标时刻就无法解释
	// 「为什么我刷新后数字变了」——各表在此期间可能各自在变。
	GatheredAt string `json:"gatheredAt"`
}

// controlPlaneDenialLimit 最近拒绝的条数上限。
//
// 10 条：够看出「哪个工具被反复拒」，又不会让响应体随历史增长。
// 固定值而非参数——参数化会诱导调用方传 1000 然后在页面上渲染一整屏历史拒绝，
// 那时它就不再是「最近」了。
const controlPlaneDenialLimit = 10

// deniedStatus 判定拒绝写入的 status 值。
//
// 刻意写成精确相等而不是前缀/LIKE：
//   - status=dry_run_denied 是**预演**拒绝（假设性结论），不能混进真实拒绝面；
//   - status=failed 是入箱后执行失败（判定已放行），也不是判定拒绝。
//
// 用 LIKE 'denied%' 会把 dry_run_denied 误吸进来——聚合面的「最近拒绝」
// 一旦掺假的拒绝，运维就会去查一个根本不存在的拦截。
const deniedStatus = "denied"

// GetControlPlaneOverview 组装聚合读面。
//
// 本函数不做任何判定：所有可见性结论来自 middleware.HasResourcePermission
// （与 ListTools 完全同一函数），所有队列数字来自 repo.ListToolInvocations
// （与审批页同一查询），所有策略计数来自 permission 表的既有列。
func (s *Service) GetControlPlaneOverview(ctx context.Context, tenantID int, role string) (*ControlPlaneOverview, error) {
	return s.getControlPlaneOverview(ctx, s.repo, tenantID, role)
}

// getControlPlaneOverview 是聚合的实际实现，仓储作为显式参数。
//
// 仓储显式传参（而不是在函数体里读 s.repo）是为了让「任一真源读失败即
// 整单报错」这条判据可被证明：证明它唯一的办法是让队列查询真的失败。
// 传参形式让失败注入只需在测试里构造一个带毒仓储，不必给 Service 加
// SetRepository——那等于为测试开生产后门，且会让生产路径用哪个仓储
// 变得可被外部替换。
func (s *Service) getControlPlaneOverview(ctx context.Context, repo Repository, tenantID int, role string) (*ControlPlaneOverview, error) {
	// 超时派生：保留 ctx 上的租户/链路值，只替换取消信号。
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), controlPlaneQueryTimeout)
	defer cancel()

	overview := &ControlPlaneOverview{
		TenantID:     tenantID,
		Role:         role,
		GatheredAt:   time.Now().Format(time.RFC3339),
		RecentDenial: []ControlPlaneRecentDenial{},
		ToolSurface:  []ControlPlaneToolSurface{},
	}

	// 1) 工具可见面：与 GET /agent/tools 同源（同一个权限判定函数）。
	allTools := s.ListToolsForTenant(ctx, tenantID)
	overview.TotalTools = len(allTools)
	for _, t := range allTools {
		visible := middleware.HasResourcePermission(ctx, s.entClient, role, t.Resource, t.Action, tenantID)
		overview.ToolSurface = append(overview.ToolSurface, ControlPlaneToolSurface{
			Name:      t.Name,
			Resource:  t.Resource,
			Action:    t.Action,
			ReadOnly:  t.ReadOnly,
			Visible:   visible,
			RiskLevel: s.toolRiskLevel(ctx, tenantID, t.Resource, t.Action),
		})
		if visible {
			overview.VisibleTools++
		}
	}

	// 2) 待批队列：按四种审批状态分别计数。
	//
	// 走 CountToolInvocationsByApprovalState 而不是逐态调 ListToolInvocations：
	// 后者的 state="pending" 是**四方合并**的业务语义（一次返回四种待批态，
	// 审批页依赖这一点），拿它当单态计数会让 Pending 变成四态之和、而其余三态
	// 各自精确，Total 于是被重复累加。这个错误不会崩、只会让数字偏大，
	// 而运维正是靠这些数字判断「队列有没有清空」。
	//
	// 分组计数还顺带避免了「为得到一个数字把整行捞回来」。
	// 未被统计到的状态（approved / rejected / failed 等）不进入队列视图——
	// 它们不是待办。
	counts, err := repo.CountToolInvocationsByApprovalState(ctx, tenantID)
	if err != nil {
		// fail-closed：返回 (nil, err) 而非带部分数据的响应。
		return nil, err
	}
	overview.Queue.Pending = counts["pending"]
	overview.Queue.PendingAgent = counts["pending_agent"]
	overview.Queue.PendingAgentTwo = counts["pending_agent_two"]
	overview.Queue.AwaitingSecond = counts["awaiting_second"]
	overview.Queue.Total = overview.Queue.Pending +
		overview.Queue.PendingAgent +
		overview.Queue.PendingAgentTwo +
		overview.Queue.AwaitingSecond

	// 3) 策略配置面：读 permission 表的既有列并逐行解码。
	policies, err := s.summarizePolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	overview.Policies = policies

	// 4) 最近拒绝：只看真实调用（排除预演）。
	denials, err := s.recentDenials(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	overview.RecentDenial = denials

	return overview, nil
}

// toolRiskLevel 取工具对应权限码的运行期风险等级（只读展示）。
//
// 与 handler 层 invocationPolicyMeta 同口径：catalog 不可用或读不到时返回
// "unknown" 而不是空串——空串在界面上与「没有风险」不可区分，而 unknown
// 明确告诉运维「这个码的策略读不出来」。
func (s *Service) toolRiskLevel(ctx context.Context, tenantID int, resource, action string) string {
	if resource == "" || action == "" {
		return "unknown"
	}
	catalog := authz.Shared()
	if catalog == nil {
		return "unknown"
	}
	def, err := catalog.Lookup(ctx, tenantID, resource+":"+action)
	if err != nil || def == nil {
		return "unknown"
	}
	if def.Policy.RiskLevel == "" {
		return "unknown"
	}
	return def.Policy.RiskLevel
}

// summarizePolicies 统计租户内策略信封的三种状态。
//
// 计数口径刻意**逐行严格解码**而不是查 description 是否为空串：
// 「有 description 但解不出」与「没有 description」是两种完全不同的风险
// （前者是配错了、鉴权 fail-closed 全拒；后者是没配、走 DerivePolicy 推导），
// 用字符串判空会把它们并成一类，于是最危险的那类彻底隐身。
//
// 这里用 DecodeDefinition（严格版）而不是 DecodeDefinitionLenient：
// lenient 版会把坏信封**静默替换为推导策略**，于是 count 里看不出任何异常，
// 而鉴权路径用的是严格版会拒绝——读面与判定面必须都看见同一批坏行。
//
// 分类的三个分支**逐条照抄 DecodeDefinition 自身的分支顺序**，这不是巧合
// 而是一条保真约束：DecodeDefinition 对「空串 / 不以 { 开头」一律返回
// (trimmed, DerivePolicy, nil)——**不报错**。所以若仅以「decode 有没有 err」
// 来判「是否配了策略」，纯文本的旧描述行会被算成 WithPolicy，读面就会
// 报告「N 个权限点已被策略覆盖」而实际上它们全走推导默认值（AI 主体零权限）。
// 那个方向是**乐观的错**：运维看到「都配了」就不会再去治理，而真实覆盖面为零。
func (s *Service) summarizePolicies(ctx context.Context, tenantID int) (ControlPlanePolicySummary, error) {
	sum := ControlPlanePolicySummary{}

	rows, err := s.entClient.Permission.Query().
		Where(permission.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return sum, err
	}
	sum.Total = len(rows)

	for _, row := range rows {
		raw := strings.TrimSpace(row.Description)

		// 分支一：无信封。空串与纯文本同属此类——鉴权走 DerivePolicy 推导，
		// 不是管理员显式配置。判据与 DecodeDefinition 首行完全一致。
		if raw == "" || !strings.HasPrefix(raw, "{") {
			sum.PolicyFree++
			continue
		}

		// 分支二：看起来是信封（以 { 开头）但严格解码失败。
		// 坏 JSON / 未知版本 / 策略非法都落在这里，鉴权时 fail-closed 全拒。
		if _, _, decodeErr := authz.DecodeDefinition(raw, row.Resource, row.Action); decodeErr != nil {
			sum.Unreadable++
			continue
		}

		// 分支三：合法信封，管理员显式配置且鉴权时会真正被读取。
		sum.WithPolicy++
	}

	return sum, nil
}

// recentDenials 取最近被拒绝的真实调用（排除预演）。
//
// 用 status 精确匹配而不是 permission_check=denied：
//   - permission_check=denied 只覆盖 Gate 2 角色 RBAC 的拒绝；
//   - Gate 2.5 主体策略的拒绝会写 status=denied 但 permission_check 可能仍是
//     passed（角色有权限、策略不允许）。
//
// 只按 permission_check 过滤会漏掉**最需要被看见的那一类**——策略生效了、
// 但把调用挡住了，正是运维要确认的事。
func (s *Service) recentDenials(ctx context.Context, tenantID int) ([]ControlPlaneRecentDenial, error) {
	rows, err := s.entClient.ToolInvocation.Query().
		Where(
			toolinvocation.TenantIDEQ(tenantID),
			toolinvocation.DryRunEQ(false),
			toolinvocation.StatusEQ(deniedStatus),
		).
		Order(ent.Desc(toolinvocation.FieldCreatedAt)).
		Limit(controlPlaneDenialLimit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ControlPlaneRecentDenial, 0, len(rows))
	for _, row := range rows {
		out = append(out, ControlPlaneRecentDenial{
			ID:              row.ID,
			ToolName:        row.ToolName,
			UserID:          row.UserID,
			RoleSnapshot:    row.RoleSnapshot,
			PermissionCheck: row.PermissionCheck,
			Reason:          row.PermissionReason,
			PolicyVersion:   row.PolicyVersion,
			RunID:           row.RunID,
			CreatedAt:       row.CreatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}
