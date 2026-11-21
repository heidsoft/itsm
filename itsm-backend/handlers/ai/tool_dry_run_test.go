package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/toolinvocation"
	"itsm-backend/handlers/ai"
	"itsm-backend/internal/authz"
)

// 这组测试锁「预演（dry-run）是判定而非执行」。
//
// 缺陷背景（2026-11-21 审计发现）：`tool_invocations.dry_run` 自建表起就在
// 生产库存在（ent `Schema.Create` 建的，无任何迁移），但 handlers/ai 全域
// **零消费方**：实体无字段、仓储不映射、服务不写入、handler 不暴露。
// 这是「字段即声明」的典型形态——数据库看起来支持预演，实际没有任何代码
// 能产生或读取预演记录，于是任何人问「这个工具调用会不会被允许」都只能
// 真调一次：写工具会留下一条真实待审批待办，只读工具会真的查库/调外部系统。
//
// 四条判据（缺一不可，逐条都有对应的反向验证注入点）：
//
//	① 预演结论与真实执行一致：allowed / needsApproval / minApprovals 三项
//	   必须完全相同（同一份 evaluateToolCall，不允许两处各算一遍）；
//	② 预演不执行工具：执行计数为 0（否则「问一下」就产生了副作用）；
//	③ 预演不建待审批行：审批待办查不到（否则审批人会对一条从未被调度的
//	   记录点「通过」，放行一个不存在的操作）；
//	④ 预演必落审计且 dry_run=true（否则预演成了无法追溯的只读通道）。
//
// 反向验证：分别把 ① 判定分叉、② 改为调用 Execute、③ 去掉待办查询的
// DryRunEQ(false)、④ 写入改为 DryRun=false，四条都应红在断言上。

// countingToolCatalog 是可控的工具目录 fake：只登记探针工具，并记录执行次数。
//
// 用 fake 而不是 service.NewToolRegistry：真实注册表的工具会真的去查库
// （list_tickets 之类），既拖慢测试也无法表达「执行了多少次」。
// 判据 ② 需要一个可观测的计数器，这是 fake 的唯一理由。
type countingToolCatalog struct {
	defs      map[string]dto.ToolDefinition
	execCount int
	execNames []string
}

func newCountingToolCatalog() *countingToolCatalog {
	return &countingToolCatalog{
		defs: map[string]dto.ToolDefinition{
			// 写探针：ReadOnly=false → 无论什么主体都需审批。
			"probe_write_ticket": {
				Name:        "probe_write_ticket",
				Description: "测试用写工具",
				ReadOnly:    false,
				Resource:    "ticket",
				Action:      "write",
			},
			// 只读探针：ReadOnly=true → human 无需审批，直接执行。
			"probe_read_ticket": {
				Name:        "probe_read_ticket",
				Description: "测试用只读工具",
				ReadOnly:    true,
				Resource:    "ticket",
				Action:      "read",
				// 定义完整性检查要求 Resource/Action 非空，这里已满足。
			},
		},
	}
}

func (c *countingToolCatalog) Execute(_ context.Context, _ int, name string, _ map[string]interface{}) (interface{}, error) {
	c.execCount++
	c.execNames = append(c.execNames, name)
	return map[string]interface{}{"probe": name}, nil
}

func (c *countingToolCatalog) GetTool(name string) *dto.ToolDefinition {
	if d, ok := c.defs[name]; ok {
		dCopy := d
		return &dCopy
	}
	return nil
}

func (c *countingToolCatalog) ListTools() []dto.ToolDefinition {
	out := make([]dto.ToolDefinition, 0, len(c.defs))
	for _, d := range c.defs {
		out = append(out, d)
	}
	return out
}

func (c *countingToolCatalog) ListToolsForTenant(_ context.Context, _ int) []dto.ToolDefinition {
	return c.ListTools()
}

type dryRunFixture struct {
	client   *ent.Client
	handler  *ai.Handler
	svc      *ai.Service
	tenantID int
	userID   int
	catalog  *countingToolCatalog
}

// newDryRunFixture 构造一套真实链路（ent + 仓储 + 服务 + handler）。
//
// 必须用真实仓储：判据 ③④ 要查 tool_invocations 表，fake 仓储无法证明
// 「dry_run 真的写进了库里」。
func newDryRunFixture(t *testing.T, dbName string) *dryRunFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("DryRun").SetCode("DRY_RUN_" + dbName).SetDomain(dbName + ".test").SetStatus("active").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername(dbName + "-prober").SetEmail(dbName + "@example.com").SetName("Prober").
		SetPasswordHash("hash").SetActive(true).SetTenantID(tenant.ID).
		SaveX(ctx)

	// Gate 2.5 是 fail-closed：DB 里没有 ticket:write / ticket:read 这两行，
	// agent 主体一律被拒。为了让「agent 被授权的场景」可测，必须先播种。
	// 复用 tool_receipt_test.go 的 seedPermission（同一包）。
	seedPermission(t, client, tenant.ID, "ticket", "write", true)
	seedPermission(t, client, tenant.ID, "ticket", "read", true)

	catalog := newCountingToolCatalog()
	svc := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil,
		catalog, nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(client)
	// 注入指向本测试库的注册表，不 ConfigureShared（避免污染同进程其他测试）。
	svc.SetPermissionCatalog(authz.NewRegistry(client))

	return &dryRunFixture{
		client:   client,
		handler:  ai.NewHandler(svc),
		svc:      svc,
		tenantID: tenant.ID,
		userID:   user.ID,
		catalog:  catalog,
	}
}

type dryRunResponse struct {
	Code int                    `json:"code"`
	Data dto.ToolDryRunResponse `json:"data"`
}

// dryRunAs 以指定主体发起一次预演（走真实 handler，含参数绑定与错误映射）。
func dryRunAs(t *testing.T, h *ai.Handler, tenantID, userID int, principal, role, toolName string) (int, dryRunResponse) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/agent/tools/dry-run", func(c *gin.Context) {
		c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", role)
		h.DryRunTool(c)
	})

	body, err := json.Marshal(map[string]interface{}{"name": toolName, "args": map[string]interface{}{"ticketId": 1}})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tools/dry-run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp dryRunResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

// dryRunRows 直接查库读预演行，绕过所有读面：判据 ③④ 必须证明「库里写了什么」，
// 而不是「接口说写了什么」。
func dryRunRows(t *testing.T, client *ent.Client, tenantID int) []*ent.ToolInvocation {
	t.Helper()
	rows, err := client.ToolInvocation.Query().
		Where(toolinvocation.TenantID(tenantID), toolinvocation.DryRunEQ(true)).
		Order(ent.Asc(toolinvocation.FieldID)).
		All(context.Background())
	require.NoError(t, err)
	return rows
}

// 判据 ①（写工具 / human 主体）：预演必须说「需审批、1 人、pending」。
// 且与真实执行落库的 approval_state 一致。
func TestDryRun_MatchesRealExecutionForHumanWriteTool(t *testing.T) {
	f := newDryRunFixture(t, "dry_human_write")

	status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_write_ticket")
	require.Equal(t, http.StatusOK, status)
	require.Zero(t, resp.Code, "预演被拒或参数错误时 code 非零")
	require.True(t, resp.Data.DryRun, "dryRun 必须恒为 true，让调用方一眼确认这不是执行结果")
	require.True(t, resp.Data.Allowed, "super_admin + 写工具应被允许")
	require.True(t, resp.Data.NeedsApproval, "写工具恒需审批")
	require.Equal(t, 1, resp.Data.MinApprovals, "写工具中风险 → 1 人审批")
	require.Equal(t, "pending", resp.Data.ApprovalState, "人发起写工具 → pending（保留既有语义）")

	// 同一输入真调一次，比较落库审批态。
	status2, receipt := executeToolAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman,
		`{"name":"probe_write_ticket","args":{"ticketId":1}}`)
	require.Equal(t, http.StatusOK, status2)
	require.Equal(t, "pending", receipt.Data.ApprovalState,
		"预演的 approvalState 必须等于真实执行落库值：预演说 pending_agent 而真调落 pending，调用方会按错的审批路径编排")
}

// 判据 ①（写工具 / agent 主体）：AI 发起必须报 pending_agent，不得折叠成 pending。
func TestDryRun_MatchesRealExecutionForAgentWriteTool(t *testing.T) {
	f := newDryRunFixture(t, "dry_agent_write")

	status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalAgent, "super_admin", "probe_write_ticket")
	require.Equal(t, http.StatusOK, status)
	require.Zero(t, resp.Code)
	require.True(t, resp.Data.Allowed, "已播种 allowedPrincipals 含 agent，应放行")
	require.True(t, resp.Data.NeedsApproval)
	require.Equal(t, 1, resp.Data.MinApprovals)
	require.Equal(t, "pending_agent", resp.Data.ApprovalState,
		"预演必须如实报 AI 发起态；折叠成 pending 会让调用方以为可以自批")
	require.Contains(t, resp.Data.Reason, "policyVersion=",
		"放行必须给出策略摘要，只回一个 true 等于让调用方猜依据")

	status2, receipt := executeToolAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalAgent,
		`{"name":"probe_write_ticket","args":{"ticketId":1}}`)
	require.Equal(t, http.StatusOK, status2)
	require.Equal(t, "pending_agent", receipt.Data.ApprovalState,
		"预演与真实执行的审批态必须一致（同一份 evaluateToolCall）")
}

// 判据 ①（只读工具 / human）：无需审批，预演与真调都是 allowed 且不建待办。
func TestDryRun_ReadOnlyToolNeedsNoApproval(t *testing.T) {
	f := newDryRunFixture(t, "dry_read_only")

	status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_read_ticket")
	require.Equal(t, http.StatusOK, status)
	require.True(t, resp.Data.Allowed)
	require.False(t, resp.Data.NeedsApproval, "只读工具无需审批")
	require.Zero(t, resp.Data.MinApprovals)
	require.Empty(t, resp.Data.ApprovalState,
		"无需审批时 approvalState 必须为空串：填 pending 会让调用方误以为已受理")
	require.Equal(t, 0, f.catalog.execCount, "预演绝不允许执行工具（判据 ②）")
}

// 判据 ①（被拒）：未把 agent 加入 allowedPrincipals 的码，agent 预演必须 allowed=false。
// 关键：这是**正常返回**而不是 error——预演的职责就是如实说「不行」。
func TestDryRun_DeniedPrincipalReportsNotAllowedWithoutError(t *testing.T) {
	f := newDryRunFixture(t, "dry_denied")
	// 把 ticket:write 播种为「只允许 human」：覆盖同名码，让 agent 落在零信任拒绝分支。
	ctx := context.Background()
	policy := authz.DerivePolicy("ticket", "write")
	policy.AllowedPrincipals = []string{authz.PrincipalHuman}
	encoded, err := authz.EncodeDefinition("ticket write", policy)
	require.NoError(t, err)
	_, err = f.client.Permission.Update().
		Where(permission.CodeEQ("ticket:write"), permission.TenantIDEQ(f.tenantID)).
		SetDescription(encoded).
		Save(ctx)
	require.NoError(t, err)

	status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalAgent, "super_admin", "probe_write_ticket")
	require.Equal(t, http.StatusOK, status,
		"被拒不是系统错误：用 5xx 表达会让调用方重试或告警一个预期结论")
	require.Zero(t, resp.Code)
	require.False(t, resp.Data.Allowed, "allowedPrincipals 不含 agent 必须拒绝")
	require.False(t, resp.Data.NeedsApproval, "被拒时不应声称需审批（那意味着可以被批）")
	require.NotEmpty(t, resp.Data.Reason, "被拒必须给出理由，只回 false 等于让调用方猜")
	require.Equal(t, 0, f.catalog.execCount, "被拒的预演更不允许执行工具")
}

// 判据 ④：预演必落一条 dry_run=true 的审计，且 approval_state="none"。
// 被拒的预演同样要留痕——否则「谁试图调了什么被挡住」完全不可查。
func TestDryRun_WritesAuditRowWithDryRunFlag(t *testing.T) {
	f := newDryRunFixture(t, "dry_audit")

	_, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_write_ticket")
	require.NotZero(t, resp.Data.AuditID, "预演必须返回审计行 ID")

	rows := dryRunRows(t, f.client, f.tenantID)
	require.Len(t, rows, 1, "一次预演恰好落一条 dry_run 行")

	row := rows[0]
	require.Equal(t, resp.Data.AuditID, row.ID, "响应里的 auditId 必须等于落库行 ID")
	require.Equal(t, "probe_write_ticket", row.ToolName)
	require.Equal(t, "dry_run_allowed", row.Status,
		"status 必须与真实执行的 executed/pending 不相交，否则统计会把预演算成调用")
	require.Equal(t, "none", row.ApprovalState,
		"预演不进审批流：approval_state 必须与待办查询的取值集合不相交")
	require.False(t, row.NeedsApproval)
	require.True(t, row.DryRun)
	require.Equal(t, "passed", row.PermissionCheck)
	require.Equal(t, "super_admin", row.RoleSnapshot)
	require.Equal(t, f.userID, row.UserID)
	require.NotEmpty(t, row.RunID, "预演也是审计动作，必须能挂到某个运行轮次")
	// human 主体不查权限目录 → 无策略参与 → policyVersion 必须是空串（三态之一），
	// 不是 "unknown"。两者混淆会让「有多少调用缺策略版本」指标失真：
	// 空串是「本来就不该有」，unknown 是「本该有却读不出」。
	require.Empty(t, row.PolicyVersion,
		"human 主体无策略参与时 policyVersion 必须为空串，不得写 unknown")
}

// 判据 ④ 的补充：被拒的预演落 status=dry_run_denied + permission_check=denied。
func TestDryRun_DeniedAuditIsDistinguishable(t *testing.T) {
	f := newDryRunFixture(t, "dry_denied_audit")
	_, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "no_such_role", "probe_write_ticket")
	require.False(t, resp.Data.Allowed, "role 无 ticket:write → Gate 2 拒绝")

	rows := dryRunRows(t, f.client, f.tenantID)
	require.Len(t, rows, 1)
	require.Equal(t, "dry_run_denied", rows[0].Status)
	require.Equal(t, "denied", rows[0].PermissionCheck,
		"被拒的预演必须与放行的预演在审计上可区分")
}

// 判据 ③：预演不得进入审批待办。
//
// 这是本组里最危险的一条：审批队列的每一行都对应一次真实待发生的副作用，
// 审批人点「通过」会真的放行一个写操作。若预演行混进来，审批人会对一条
// 从未被调度、也不需要放行的记录点通过——放行一个不存在的操作，与漏批
// 一个真实操作同样危险，且更难发现（列表看起来完全正常）。
func TestDryRun_ExcludedFromApprovalQueue(t *testing.T) {
	f := newDryRunFixture(t, "dry_queue")

	// 仅预演：审批待办必须为空。
	dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_write_ticket")

	status, queue := listPendingInvocations(t, f.handler, f.tenantID)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, queue.Data.Items,
		"预演不得出现在审批待办：审批人会对从未被调度的记录点「通过」")

	// 再真调一次写工具：待办必须恰好多出这一条真实记录。
	_, receipt := executeToolAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman,
		`{"name":"probe_write_ticket","args":{"ticketId":1}}`)
	require.NotZero(t, receipt.Data.InvocationID)

	status2, queue2 := listPendingInvocations(t, f.handler, f.tenantID)
	require.Equal(t, http.StatusOK, status2)
	require.Len(t, queue2.Data.Items, 1,
		"真实写调用必须出现；预演记录仍须被排除（排除的是 dry_run，不是这次请求）")
	require.Equal(t, receipt.Data.InvocationID, queue2.Data.Items[0].ID)
}

// 判据 ③ 的纵深防御：即便某天有代码给预演行写了审批态，dry_run 也必须挡住它。
//
// 为什么必须单独写这一条（而不是满足于上一条的端到端断言）：
// 上一条测试曾经是**假绿**——把仓储里的 `DryRunEQ(false)` 谓词整个删掉，
// 它仍然通过。原因是预演行的 approval_state="none" 本身就不在待办查询的
// 取值集合（pending/pending_agent/pending_agent_two/awaiting_second）里，
// 两道防线被第一道兜住了，第二道从未被任何断言覆盖。
//
// 这正是「注入缺陷仍全绿」的典型形态：断言看起来在验证排除逻辑，
// 实际验证的是另一个字段的取值。要锁住 DryRunEQ(false)，必须构造一条
// approval_state="pending" 且 dry_run=true 的行——这是唯一能区分
// 「靠 approval_state 挡住」与「靠 dry_run 挡住」的输入。
//
// 这个输入不是纯粹的理论构造：把预演做成「可审批的草稿」是一个很自然的
// 产品诉求（审批人先看到预演结论再决定要不要真跑），而实现它最省事的做法
// 就是给预演行的 approval_state 写 pending。那一步一旦发生，第一道防线
// 立刻失效，只剩 dry_run 能挡住。
func TestDryRun_QueueExclusionSurvivesPendingApprovalState(t *testing.T) {
	f := newDryRunFixture(t, "dry_queue_defence")

	// 直插两行：一行预演但带了 pending 审批态，一行真实 pending。
	// 直插而非走服务：这里要验证的是**读面谓词**，不是写入路径。
	ctx := context.Background()
	_, err := f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).
		SetUserID(f.userID).
		SetToolName("probe_write_ticket").
		SetArguments(`{"ticketId":1}`).
		SetStatus("dry_run_allowed").
		SetNeedsApproval(false).
		SetApprovalState("pending"). // 模拟「预演也进了审批态」的历史脏数据
		SetPermissionCheck("passed").
		SetDryRun(true).
		Save(ctx)
	require.NoError(t, err)

	realInv, err := f.client.ToolInvocation.Create().
		SetTenantID(f.tenantID).
		SetUserID(f.userID).
		SetToolName("probe_write_ticket").
		SetArguments(`{"ticketId":2}`).
		SetStatus("pending").
		SetNeedsApproval(true).
		SetApprovalState("pending").
		SetPermissionCheck("passed").
		SetDryRun(false).
		Save(ctx)
	require.NoError(t, err)

	status, queue := listPendingInvocations(t, f.handler, f.tenantID)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, queue.Data.Items, 1,
		"approval_state=pending 的预演行仍必须被排除：dry_run 是真防线，不是装饰")
	require.Equal(t, realInv.ID, queue.Data.Items[0].ID)
	require.False(t, queue.Data.Items[0].DryRun)
}

// 判据 ③ 的读面补充：按运行轮次读面**包含**预演行，但必须靠 dryRun 字段可辨。
//
// 两个读面语义相反，这是设计意图不是疏漏：
//   - 审批待办回答「我现在该批什么」→ 预演不是待批项，排除；
//   - 运行回放回答「这一轮发生了什么」→ 预演是一次真实的判定动作，保留。
//
// 但保留的前提是消费方能分辨，因此 dryRun 必须出现在序列化结果里。
// 漏掉它，预演就会在回放里被当成真实动作——「字段级漂移」比彻底缺更危险。
func TestDryRun_VisibleInRunReplayWithFlag(t *testing.T) {
	f := newDryRunFixture(t, "dry_replay")
	ctx := ai.WithRunID(context.Background(), "run_dry_replay")

	dryRunAsCtx(t, f.handler, ctx, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_write_ticket")
	executeToolAsCtx(t, f.handler, ctx, f.tenantID, f.userID, authz.PrincipalHuman,
		`{"name":"probe_write_ticket","args":{"ticketId":1}}`)

	status, replay := getRunInvocations(t, f.handler, f.tenantID, "run_dry_replay")
	require.Equal(t, http.StatusOK, status)
	require.Len(t, replay.Data.Items, 2, "同一 run_id 下预演与真实调用都要能取到")

	// 第一条是预演、第二条是真实调用（按创建时间正序）。
	require.True(t, replay.Data.Items[0].DryRun,
		"预演行必须带 dryRun=true，否则回放会把「问一下」读成「做过」")
	require.False(t, replay.Data.Items[1].DryRun)
	require.Equal(t, "dry_run_allowed", replay.Data.Items[0].Status)
	require.Equal(t, "pending", replay.Data.Items[1].Status)

	// 详情读面同样必须暴露该字段：只在一处暴露会造成「列表有、详情缺」的漂移。
	detailStatus, detail := getInvocationWithDryRun(t, f.handler, f.tenantID, replay.Data.Items[0].ID)
	require.Equal(t, http.StatusOK, detailStatus)
	require.True(t, detail.Data.DryRun, "详情读面必须同样返回 dryRun")
}

// invocationDetailWithDryRun 是详情读面的解析结构。
// 与 tool_receipt_test.go 的 invocationDetail 分开声明而不是复用它：
// 那一个锁的是审批态语义，这一个锁的是预演标记，两者被删改的理由不同，
// 共用一个结构会让「某天有人精简字段」同时破坏两组无关的契约。
type invocationDetailWithDryRun struct {
	Code int `json:"code"`
	Data struct {
		DryRun        bool   `json:"dryRun"`
		Status        string `json:"status"`
		ApprovalState string `json:"approvalState"`
	} `json:"data"`
}

func getInvocationWithDryRun(t *testing.T, h *ai.Handler, tenantID, id int) (int, invocationDetailWithDryRun) {
	t.Helper()
	r := gin.New()
	r.GET("/api/v1/agent/tools/:id", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		h.GetToolInvocation(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/agent/tools/%d", id), nil)
	r.ServeHTTP(w, req)
	var d invocationDetailWithDryRun
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d), w.Body.String())
	return w.Code, d
}

// 判据 ④ 的计数盲区：预演不得被当成「调用次数」。
// 真实的执行计数只有 fake catalog 能观测，这里顺带锁住「预演多次也不执行」。
func TestDryRun_RepeatedPreflightNeverExecutes(t *testing.T) {
	f := newDryRunFixture(t, "dry_repeat")

	for i := 0; i < 3; i++ {
		status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "probe_write_ticket")
		require.Equal(t, http.StatusOK, status)
		require.True(t, resp.Data.Allowed)
	}
	require.Equal(t, 0, f.catalog.execCount, "连续预演不得累积任何执行")
	require.Empty(t, f.catalog.execNames)

	rows := dryRunRows(t, f.client, f.tenantID)
	require.Len(t, rows, 3, "每次预演都应留痕，而不是合并成一条")
}

// 未知工具：请求本身不成立，返回业务错误而不是「allowed=false」。
//
// 理由：allowed=false 的语义是「这个调用会被拒」，而未知工具根本没形成调用。
// 混为一谈会让编排方以为「等策略放开就能用」，而实际要改的是工具名。
func TestDryRun_UnknownToolIsRequestErrorNotDenial(t *testing.T) {
	f := newDryRunFixture(t, "dry_unknown")
	status, resp := dryRunAs(t, f.handler, f.tenantID, f.userID, authz.PrincipalHuman, "super_admin", "no_such_tool")
	require.NotEqual(t, http.StatusOK, status, "未知工具必须是错误响应，不能伪装成一次正常判定")
	require.NotZero(t, resp.Code)
	require.Equal(t, 0, f.catalog.execCount)
}

// service 主体（userID=0）：预演拒绝。
// 预演是「替某个用户问会发生什么」，service 主体没有可归属的用户，
// 它的结论也无法挂到任何人身上——允许它只会污染审计。
func TestDryRun_ServicePrincipalWithoutUserRejected(t *testing.T) {
	f := newDryRunFixture(t, "dry_service")
	status, resp := dryRunAs(t, f.handler, f.tenantID, 0, authz.PrincipalService, "super_admin", "probe_write_ticket")
	require.NotEqual(t, http.StatusOK, status)
	require.NotZero(t, resp.Code)
	require.Empty(t, dryRunRows(t, f.client, f.tenantID),
		"被身份守卫挡下的请求不应留预演审计（它没有可归属的用户）")
}

// ----- 辅助：带 context 的执行入口与读面 -----

func dryRunAsCtx(t *testing.T, h *ai.Handler, ctx context.Context, tenantID, userID int, principal, role, toolName string) (int, dryRunResponse) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/agent/tools/dry-run", func(c *gin.Context) {
		c.Request = c.Request.WithContext(authz.WithPrincipal(ctx, principal))
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", role)
		h.DryRunTool(c)
	})
	body, _ := json.Marshal(map[string]interface{}{"name": toolName, "args": map[string]interface{}{"ticketId": 1}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tools/dry-run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp dryRunResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

func executeToolAsCtx(t *testing.T, h *ai.Handler, ctx context.Context, tenantID, userID int, principal, body string) (int, toolReceiptResponse) {
	t.Helper()
	r := gin.New()
	r.POST("/api/v1/agent/tools/execute", func(c *gin.Context) {
		c.Request = c.Request.WithContext(authz.WithPrincipal(ctx, principal))
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Set("role", "super_admin")
		h.ExecuteTool(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/tools/execute", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp toolReceiptResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

type invocationListItem struct {
	ID       int    `json:"id"`
	DryRun   bool   `json:"dryRun"`
	Status   string `json:"status"`
	ToolName string `json:"toolName"`
}

type invocationList struct {
	Code int `json:"code"`
	Data struct {
		Items []invocationListItem `json:"items"`
		State string               `json:"state"`
	} `json:"data"`
}

func listPendingInvocations(t *testing.T, h *ai.Handler, tenantID int) (int, invocationList) {
	t.Helper()
	r := gin.New()
	r.GET("/api/v1/agent/tools/invocations", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		h.ListToolInvocations(c)
	})
	w := httptest.NewRecorder()
	// 不传 state → handler 默认 "pending"，正是审批人待办视角。
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/tools/invocations", nil)
	r.ServeHTTP(w, req)
	var d invocationList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d), w.Body.String())
	return w.Code, d
}

func getRunInvocations(t *testing.T, h *ai.Handler, tenantID int, runID string) (int, invocationList) {
	t.Helper()
	r := gin.New()
	r.GET("/api/v1/agent/runs/:runId/invocations", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		h.ListRunInvocations(c)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/runs/"+runID+"/invocations", nil)
	r.ServeHTTP(w, req)
	var d invocationList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d), w.Body.String())
	return w.Code, d
}
