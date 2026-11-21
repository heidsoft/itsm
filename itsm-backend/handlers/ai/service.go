package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/knowledgeaccess"
	"itsm-backend/metrics"
	"itsm-backend/middleware"
)

type Service struct {
	repo               Repository
	logger             *zap.SugaredLogger
	rag                RAGChatService
	llmGateway         dto.LLMGatewayPort
	tools              ToolCatalog
	analytics          DeepAnalyticsProvider
	prediction         TrendPredictor
	slaForecastSkill   SLAForecastRunner
	triageService      TriageAdvisor
	rca                RootCauseAnalyzer
	aiTelemetryService AITelemetryFacade
	// SummarizeService 装配路径；摘要优先走它，不再绕行 rca.SummarizeTicket。
	summarizeService Summarizer
	// ent client，用于复用 RBAC hasResourcePermission
	entClient *ent.Client
	// catalog 是 AI 主体零信任判定的运行期权限真源（默认 authz.Shared()）。
	// 允许单测注入 fake；不额外引入编译期码表。
	catalog authz.Catalog
	// auditLedger 服务端 AI 决策底账写口（窄接口，生产实现是 AITelemetryService）。
	// nil 时分诊不落底账，响应不携带 auditId（降级不阻塞）。
	auditLedger AuditLedgerWriter
}

// AuditLedgerWriter 是服务端 AI 决策底账的窄接口：产出 AI 建议时自动落一条
// item_type='ai_audit' 行（accepted=NULL 未决策），返回行 ID 供操作者经
// PATCH /ai/audit-logs/:id 翻转采纳/拒绝。生产实现是共享层 AI 遥测服务。
type AuditLedgerWriter interface {
	RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)
}

// ragStreamer 是流式问答的可选能力接口，与 RAGChatService 分开声明。
//
// 网关契约化后，其方法签名只引用 dto 端口与 dto 值类型；本批暂留本文件
// （既有回归锁断言声明位置），AI 域整体归零批次再并入零依赖契约文件。
// 生产实现（共享层 RAG 服务）同时满足 RAGChatService 与 ragStreamer；
// 只实现读面契约的域内实现（单测 fake）不具备流式能力，ChatStream 会走
// 明确的降级错误而不是 panic。
type ragStreamer interface {
	AskWithLLMStreamWithToolsMulti(
		ctx context.Context,
		tenantID int,
		query string,
		supplemental []string,
		gateway dto.LLMGatewayPort,
		maxResults int,
		tools []dto.LLMTool,
		onSources func(sources []map[string]any),
		onDelta func(delta string),
		execTool func(name string, args map[string]any) (any, error),
		extraDocs []map[string]any,
	) error
}

func NewService(
	repo Repository,
	logger *zap.SugaredLogger,
	rag RAGChatService,
	tools ToolCatalog,
	analytics DeepAnalyticsProvider,
	prediction TrendPredictor,
	slaForecastSkill SLAForecastRunner,
	triageService TriageAdvisor,
	rca RootCauseAnalyzer,
	aiTelemetryService AITelemetryFacade,
) *Service {
	s := &Service{
		repo:               repo,
		logger:             logger,
		rag:                rag,
		tools:              tools,
		analytics:          analytics,
		prediction:         prediction,
		slaForecastSkill:   slaForecastSkill,
		triageService:      triageService,
		rca:                rca,
		aiTelemetryService: aiTelemetryService,
	}
	// 遥测服务就绪时默认作为底账写口；测试可用 SetAuditLedger 覆盖为 mock。
	if aiTelemetryService != nil {
		s.auditLedger = aiTelemetryService
	}
	return s
}

// SetAuditLedger 注入/替换决策底账写口（测试用；bootstrap 走 NewService 默认装配）。
func (s *Service) SetAuditLedger(l AuditLedgerWriter) {
	s.auditLedger = l
}

// SetLLMGateway wires an optional LLM gateway for streaming answers.
// Kept as a setter to avoid churning existing NewService call sites.
func (s *Service) SetLLMGateway(gateway dto.LLMGatewayPort) {
	s.llmGateway = gateway
}

// SetSummarizeService wires SummarizeService。
// SummarizeTicket 优先走 SummarizeService.SummarizeWithMetadata，失败兜底回 rca。
func (s *Service) SetSummarizeService(svc Summarizer) {
	s.summarizeService = svc
}

// SetEntClient wires the ent client for RBAC permission checks.
// AI 工具执行前调用 hasResourcePermission 校验用户对工具 Resource/Action 的权限
func (s *Service) SetEntClient(client *ent.Client) {
	s.entClient = client
}

// SetPermissionCatalog 注入运行期权限真源。生产装配用 bootstrap 里
// authz.ConfigureShared 返回的共享实例；单测可注入 fake 覆盖 AI 主体策略。
func (s *Service) SetPermissionCatalog(catalog authz.Catalog) {
	s.catalog = catalog
}

// Tool Methods

func (s *Service) ListTools() []dto.ToolDefinition {
	if s.tools == nil {
		return nil
	}
	return s.tools.ListTools()
}

// ListToolsForTenant 返回按租户动态化的工具清单（list_cis 的 ci_type 枚举来自租户 CIType 表）。
func (s *Service) ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition {
	if s.tools == nil {
		return nil
	}
	return s.tools.ListToolsForTenant(ctx, tenantID)
}

// ErrToolPermissionDenied 工具权限不足。
//
// 单一事实源在 dto：handlers/skill 需要在不 import handlers/ai 的前提下
// 用 errors.Is 判定「拒绝」并映射 403，而不是把它当成内部错误。
var ErrToolPermissionDenied = dto.ErrToolPermissionDenied

// ErrUnknownTool 未知工具
var ErrUnknownTool = fmt.Errorf("unknown tool")

var ErrToolUnavailable = fmt.Errorf("tool authorization dependencies unavailable")

// ErrRAGUnavailable RAG 检索服务未装配。此前 rag 为 nil 时 Chat 直接解引用
// 崩溃；现在返回错误，由 handler 的降级分支接住（返回空结果而不是 500）。
var ErrRAGUnavailable = fmt.Errorf("rag retrieval service unavailable")

// ExecuteTool 执行 AI 工具
// userID 和 role 用于 Gate 2 RBAC 校验
//
// 校验分层：
//
//	Gate 1: 路由级 RBACMiddleware（/api/v1/agent/* 检查 ai:read/ai:write）— 调用前已完成
//	Gate 2: 工具级 RBAC（本方法）— 按 ToolDefinition.Resource/Action 校验
//	Gate 3: 审批流（写工具 !ReadOnly）— 由 NeedsApproval 处理
//
// 身份守卫说明（2026-11-18 修正）：
//
// 守卫的意图是「无身份不得执行工具」，但**不能只看 userID**：
// service 主体是**合法的进程身份**，其 JWT 里 user_id 恒为 0（Phase B1 明确
// service 不是登录用户、不进用户表）。此前这里对 userID<=0 一律拒绝，
// 导致 service 主体永远无法执行任何工具 —— MCP 通道与 sidecar 回连
// 全部被挡在门外，且错误信息只说「tool permission denied」，
// 排障时完全看不出是身份判定把人挡了。
//
// 现在按 principal 区分：
//   - human / agent：userID 必须 > 0（它们是「用户身份」，须有真实 user_id）
//   - service：userID 恒为 0 属正常，身份由签发时的载体角色 + 运行期策略决定
//
// service 主体的实际准入由 Gate 2（载体角色是否持该 resource:action）
// 与 Gate 2.5（allowedPrincipals 是否含 service）共同决定 —— 放行与否
// 仍由运行期策略说了算，本守卫只负责「有没有身份」。
// toolDecision 是一次工具调用判定的**完整结论**。
//
// 存在的唯一理由：预演（DryRunTool）与真实执行（ExecuteTool）必须给出同一个答案。
// 如果两处各写一份判定逻辑，最坏的失败形态是「预演说可以、真调被拒」——
// 那比不提供预演更糟，因为调用方会按预演结论去编排后续动作，然后在真实执行时
// 炸在半路，且中间已经产生了不可回退的副作用。
//
// 因此：判定逻辑只此一处，两个入口都只消费本结构；任何新增判定条件
// （新的 Gate、新的策略字段）只能加在这里，不得在某个入口单独加。
type toolDecision struct {
	// 工具定义（判定依据，也是 needsApproval 的来源）
	Tool *dto.ToolDefinition
	// Gate 2 结论
	PermCheck  string // passed | denied
	PermReason string
	// 综合结论：Gate 2 与 Gate 2.5 都通过才为 true
	Allowed bool
	// Gate 2.5 结论（仅非 human 主体参与）
	NeedsApproval bool
	MinApprovals  int
	RiskLevel     string
	// PolicyVersion 三态：""=无策略参与 / "unknown"=读不出 / 指纹
	PolicyVersion string
}

// evaluateToolCall 跑完 Gate 2 + Gate 2.5，返回判定结论。
//
// 不执行任何副作用（不落审计、不执行工具、不建待审批行）——副作用由调用方
// 按自己的语义决定，因为预演与真实执行的副作用本就不同（前者只审计，后者
// 还要真正执行或建待审批）。把副作用留在外面，是让「判定同源」这件事可验证的前提。
//
// ⚠️ principal 显式作为参数而不是从 ctx 读取，这看似绕远但是必要的：
// 「按哪个主体判定」与「谁发起了这次请求」在复核场景里是**两个不同的值**
// （审批人按 AI 主体复核一条 AI 发起的调用）。若从 ctx 隐式取，复核就会
// 用审批人自己的身份去判定 AI 的调用，得出与真实执行完全无关的结论——
// 而这种错误在代码里看不出来，只有对比测试能发现。
func (s *Service) evaluateToolCall(ctx context.Context, tenantID int, userID int, role, name, principal string) (*toolDecision, error) {
	if tenantID <= 0 {
		return nil, ErrToolPermissionDenied
	}
	// service 主体 userID 恒为 0 属正常（见 ExecuteTool 上方身份守卫说明）。
	if userID <= 0 && principal != authz.PrincipalService {
		return nil, ErrToolPermissionDenied
	}
	if s.tools == nil || s.entClient == nil {
		return nil, ErrToolUnavailable
	}

	toolDef := s.tools.GetTool(name)
	if toolDef == nil {
		return nil, ErrUnknownTool
	}
	// 定义完整性：漏配 Resource/Action 必须 fail-closed，而不是靠下游通配匹配放行。
	if toolDef.Resource == "" || toolDef.Action == "" {
		return nil, fmt.Errorf("%w: tool %s missing resource/action declaration",
			ErrToolPermissionDenied, name)
	}

	d := &toolDecision{
		Tool:      toolDef,
		PermCheck: "passed",
		Allowed:   true,
		RiskLevel: "unknown",
	}

	// Gate 2: 角色 RBAC（human 与 agent 都走这一道，agent 还要再过 Gate 2.5）
	if role != "" && middleware.HasResourcePermission(ctx, s.entClient, role, toolDef.Resource, toolDef.Action, tenantID) {
		d.PermCheck = "passed"
	} else {
		d.PermCheck = "denied"
		d.PermReason = fmt.Sprintf("role=%s lacks %s:%s", role, toolDef.Resource, toolDef.Action)
		d.Allowed = false
	}

	// Gate 2.5: AI 主体零信任（运行期 DB 策略）
	//
	// 人发起的操作继续走角色 RBAC；LLM / skill 自主决定调用工具时调用方用
	// authz.WithPrincipal(ctx, "agent") 标记，这里再做一次判定：
	//   1. 该 resource:action 必须在 DB 权限目录登记且启用；
	//   2. AllowedPrincipals 必须显式包含该主体（默认只有 human → 零权限）；
	//   3. 按风险等级算出需要的人类确认人数。
	// 任何一步读不出来都 deny —— 不回退到编译期引导清单。
	//
	// 这是「AI 安全攻击门槛低」的核心防线：提示词注入只能改变 LLM 想调什么，
	// 改变不了 DB 里管理员没给 AI 开过的权限。
	if principal != authz.PrincipalHuman {
		catalog := s.catalog
		if catalog == nil {
			catalog = authz.Shared()
		}
		code := toolDef.Resource + ":" + toolDef.Action
		needs, min, err := catalog.AuthorizePrincipal(ctx, tenantID, code, principal)
		if err != nil {
			// 被拒时仍写 unknown：审计必须能区分「策略说不许」与「策略读不出来」。
			d.PolicyVersion = authz.PolicyVersionUnknown
			d.PermCheck = "denied"
			d.PermReason = fmt.Sprintf("AI 主体 %s 未获授权 %s: %v", principal, code, err)
			d.Allowed = false
			d.LogDenial(s, tenantID, userID, name, code, principal, err)
		} else {
			d.NeedsApproval, d.MinApprovals = needs, min
			if def, lerr := catalog.Lookup(ctx, tenantID, code); lerr == nil && def != nil {
				d.RiskLevel = def.Policy.RiskLevel
				d.PolicyVersion = def.PolicyVersion()
			} else {
				// 放行已发生但版本取不到：写 unknown 而非空串（空串意为「无策略参与」，
				// 会把这次调用错误归类到 human 路径）。
				d.PolicyVersion = authz.PolicyVersionUnknown
			}
			policyReason := fmt.Sprintf("principal=%s risk=%s minApprovals=%d policyVersion=%s",
				principal, d.RiskLevel, d.MinApprovals, d.PolicyVersion)
			if d.PermReason != "" {
				d.PermReason += " | " + policyReason
			} else {
				d.PermReason = policyReason
			}
		}
	}

	// 写工具恒需审批；agent 主体按风险策略追加审批。
	//
	// MinApprovals 兜底：人的写操作恒为「1 名审批人」——落库的
	// approval_state="pending" 就是这个语义（见 approvalStateFor）。
	// Gate 2.5 只对非 human 主体计算审批人数，若不在此兜底，human 的写工具
	// 会出现「needsApproval=true 但 minApprovals=0」：审批进度分母为 0，
	// 进度条恒满，审批人会以为「已经批够了」，而库里仍是 pending。
	// 这是预演暴露出来的既有判定缺口，兜底放在这里而不是 handler：
	// 两个入口共用同一个结构，修一次即两边同时正确。
	if !toolDef.ReadOnly && d.MinApprovals < 1 {
		d.MinApprovals = 1
	}
	d.NeedsApproval = d.NeedsApproval || !toolDef.ReadOnly
	return d, nil
}

// LogDenial 记录主体策略拒绝（判定函数内只记日志，不落审计——
// 落审计是调用方的副作用，预演与真实执行的审计形态不同）。
func (d *toolDecision) LogDenial(s *Service, tenantID, userID int, name, code, principal string, err error) {
	s.logger.Warnw("AI tool denied by runtime permission catalog",
		"tenant_id", tenantID, "user_id", userID, "tool", name,
		"code", code, "principal", principal, "reason", err)
}

// approvalStateFor 把「是否需要审批 + 需几人 + 主体」映射成落库的审批态。
//
//	pending           = 人发起，单人审批（保留既有语义）
//	pending_agent     = AI 发起，单人审批 + 禁止发起人自批
//	pending_agent_two = AI 发起，双人审批 + 禁止发起人自批 + 两名不同审批人
//
// principal 同样显式传入：复核模式要按被复核调用的原主体推导审批态，
// 而不是按当前请求者（那样会给一条 AI 发起的调用算出 pending）。
func approvalStateFor(principal string, minApprovals int) string {
	if principal == authz.PrincipalHuman {
		return "pending"
	}
	if minApprovals >= 2 {
		return "pending_agent_two"
	}
	return "pending_agent"
}

// ExecuteTool 执行 AI 工具（真实副作用的唯一入口）
//
// userID 和 role 用于 Gate 2 RBAC 校验。判定逻辑本身在 evaluateToolCall，
// 本方法只按判定结论决定副作用：只读直接执行、写工具建待审批行。
// 这样「预演问会不会被允许」与「真调是否被允许」不可能给出两个答案。
func (s *Service) ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error) {
	// 真实执行的主体恒为请求者本人：这条通道不接受外部指定主体。
	principal := authz.PrincipalFromContext(ctx)
	// 判定逻辑集中在 evaluateToolCall：预演与真实执行必须给出同一个答案。
	// 这里只负责「按判定结论执行副作用」，不再自己算判定。
	d, err := s.evaluateToolCall(ctx, tenantID, userID, role, name, principal)
	if err != nil {
		// 未知工具 / 定义漏配 / 依赖缺失：仍要落审计，否则这类失败在
		// tool_invocations 里完全不可见（只剩一条 warn 日志）。
		// 判定函数不碰副作用正是为了让调用方在这里决定记什么。
		switch {
		case errors.Is(err, ErrUnknownTool):
			s.recordToolAudit(ctx, tenantID, userID, role, name, args, "denied", "unknown tool", "", nil, false, "")
		case errors.Is(err, ErrToolPermissionDenied):
			s.recordToolAudit(ctx, tenantID, userID, role, name, args, "denied", err.Error(), "", nil, false, "")
		}
		return nil, 0, err
	}

	// Authorization is mandatory; feature flags must never bypass it.
	if !d.Allowed {
		s.recordToolAudit(ctx, tenantID, userID, role, name, args,
			d.PermCheck, d.PermReason, "", nil, false, d.PolicyVersion)
		return nil, 0, fmt.Errorf("%w: %s", ErrToolPermissionDenied, d.PermReason)
	}

	if !d.NeedsApproval {
		res, execErr := s.tools.Execute(ctx, tenantID, name, args)
		// 只读工具执行也记录审计（AGENTS.md: AI tool invocation must produce audit logs）
		if execErr == nil {
			s.recordToolAudit(ctx, tenantID, userID, role, name, args,
				d.PermCheck, d.PermReason, "executed", nil, false, d.PolicyVersion)
		}
		return res, 0, execErr
	}

	approvalState := approvalStateFor(principal, d.MinApprovals)

	// 写工具：创建 pending invocation，等待审批
	argsStr, _ := json.Marshal(args)
	ticketRef, changeRef := businessRefsFromArgs(args)
	// 待审批记录同样要带 policyVersion / run_id / request_id：审批可能发生在
	// 几分钟后甚至跨进程重启，那时 context 早已消失；只有落库的这三个值能把
	// 「批准执行」这一步挂回原来那次判定与那一轮运行。
	_, runID := EnsureRunID(ctx)
	var requestID string
	if rid, ok := middleware.RequestIDFromContext(ctx); ok {
		requestID = rid
	}
	inv, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:         tenantID,
		ToolName:         name,
		Arguments:        string(argsStr),
		Status:           "pending",
		NeedsApproval:    true,
		ApprovalState:    approvalState,
		UserID:           userID,
		PermissionCheck:  d.PermCheck,
		PermissionReason: d.PermReason,
		RoleSnapshot:     role,
		TicketID:         ticketRef,
		ChangeID:         changeRef,
		PolicyVersion:    d.PolicyVersion,
		RunID:            runID,
		RequestID:        requestID,
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, inv.ID, nil
}

// businessRefsFromArgs 从工具参数中提取业务单据 ID（camelCase）。
// JSON 反序列化后数字为 float64，直接断言 int 会恒失败；这里兼容
// int/int64/float64/json.Number/string 五种形态，非法或负数一律视为无关联（0）。
// Phase B6：这是 ToolInvocation 强关联列的唯一入参解析点，读/写工具共用。
func businessRefsFromArgs(args map[string]interface{}) (ticketID, changeID int) {
	toID := func(v interface{}) int {
		switch n := v.(type) {
		case int:
			if n > 0 {
				return n
			}
		case int64:
			if n > 0 && n <= int64(^uint(0)>>1) {
				return int(n)
			}
		case float64:
			if n > 0 && n <= float64(^uint(0)>>1) && n == float64(int(n)) {
				return int(n)
			}
		case json.Number:
			if i, err := n.Int64(); err == nil && i > 0 && i <= int64(^uint(0)>>1) {
				return int(i)
			}
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && i > 0 {
				return i
			}
		}
		return 0
	}
	return toID(args["ticketId"]), toID(args["changeId"])
}

// recordToolAudit 统一记录只读工具执行审计，包含 RBAC 校验结果
//
// 审计写入失败不能用 _, _ = 吞掉，走结构化错误日志 +
// itsm_ai_persist_errors_total{operation="create_tool_invocation"} 计数器埋点，
// 保证 DB 抖动时审计丢失可观测、可告警。
//
// policyVersion 是本次判定的策略指纹（见 ExecuteTool 的三态说明）。
// 显式作为参数传入而不是在函数内重新读取目录：判定与记录必须是同一个快照，
// 否则「记录下来的版本」可能已是判定之后又被管理员改过的新版本。
// DryRunTool 预演一次工具调用：跑完真实的判定链，但不执行、不建待审批行。
//
// 为什么需要它（而不是让调用方「试着调一次看看」）：
//   - 写工具真调会产生一条待审批记录，审批人真的会去批，操作真的会发生；
//     调用方想知道「会不会被允许」，却在过程中制造了一个真实待办。
//   - 只读工具真调会真的查库/调外部系统，在编排阶段做可行性判断时是浪费，
//     对某些工具有副作用风险（外部系统调用并非全都幂等）。
//
// 返回值与真实执行的对应关系是本次实现的核心契约：
//
//	Allowed       ← 同 evaluateToolCall（Gate 2 + Gate 2.5 结论）
//	NeedsApproval ← 同上
//	MinApprovals  ← 同上
//
// 因此「预演 allowed=true 而真调被拒」在代码层面不可能出现：只有一份判定。
//
// 唯一允许的差异是副作用：预演只落一条 dry_run=true 的审计行，
// 且**不创建**任何 pending invocation —— 否则审批队列里会出现永远不该被批的行。
func (s *Service) DryRunTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (*dto.ToolDryRunResponse, error) {
	// 普通模式：按请求者自己的主体判定。
	return s.dryRunForPrincipal(ctx, userID, tenantID, role, name, args,
		authz.PrincipalFromContext(ctx), 0, "")
}

// ReviewInvocation 复核一条**既有**工具调用：按它当时的发起主体与当前策略重新判定。
//
// 与 DryRunTool 的区别不是「换个参数」，而是回答一个不同的问题：
//
//	DryRunTool(agent 主体)  → 「AI 现在调用这个工具，会被允许吗」
//	ReviewInvocation        → 「审批队列里这条 AI 发起的调用，现在还该放行吗」
//
// 审批人需要的是后者。批一条几小时前排队的写操作前，必须知道「策略在这段时间
// 有没有改过、改了以后这条还合规吗」——只看落库时的 policy_version 只能知道
// 「当时合规」，不能回答「现在合规」。
//
// 为什么按**原发起主体**判定而不是按审批人自己：审批人对该工具的权限是
// 「有没有资格批」（`ApproveTool` 已单独校验），而这次调用能不能执行取决于
// AI 主体是否仍被授权。按审批人自己的身份判定会得出一个与真实执行无关的结论
// ——审批人通常是管理员，永远 allowed=true，复核就失去意义。
//
// 不做任何越权扩张：复核只读判定，零副作用，且不修改被复核记录的任何字段。
func (s *Service) ReviewInvocation(ctx context.Context, invocationID, tenantID, userID int, role string) (*dto.ToolDryRunResponse, error) {
	if invocationID <= 0 {
		return nil, fmt.Errorf("%w: invocation id is required", ErrToolPermissionDenied)
	}
	// 租户谓词在仓储层强制：只按 id 查会让任何人都能复核别人的调用。
	// 查不到时返回 ent.NotFoundError，由 handler 映射成 404——「记录不存在」
	// 与「记录存在但不可复核」是两种处置，不能合并。
	inv, err := s.repo.GetToolInvocation(ctx, invocationID, tenantID)
	if err != nil {
		return nil, err
	}
	if s.tools == nil || s.tools.GetTool(inv.ToolName) == nil {
		// 工具已下线：复核无法进行，且必须明确说明是「工具没了」而不是「被拒」。
		return nil, fmt.Errorf("%w: tool %s no longer registered", ErrUnknownTool, inv.ToolName)
	}

	// 从落库的审批态推导原发起主体：pending_agent* / awaiting_second 都是
	// AI 发起后才可能出现的态；pending 是人发起。用审批态推导而不是新增一列
	// 「principal」，是因为审批态本就是它的函数——同一事实存两份必然漂移。
	originPrincipal := authz.PrincipalHuman
	switch inv.ApprovalState {
	case "pending_agent", "pending_agent_two", "awaiting_second":
		originPrincipal = authz.PrincipalAgent
	}

	var args map[string]interface{}
	if inv.Arguments != "" {
		if uerr := json.Unmarshal([]byte(inv.Arguments), &args); uerr != nil {
			// 参数已损坏：不能拿一份读不懂的输入去判定，那会给审批人一个
			// 建立在空参数之上的结论（比拒绝更危险）。
			return nil, fmt.Errorf("%w: invocation %d arguments are unreadable",
				ErrToolPermissionDenied, invocationID)
		}
	}
	if args == nil {
		args = map[string]interface{}{}
	}

	// role 用被复核记录的 role_snapshot：Gate 2 判的是「发起时的载体角色是否
	// 持该权限」，用审批人当前角色代替会让结论与执行链不一致。
	return s.dryRunForPrincipal(ctx, userID, tenantID, inv.RoleSnapshot, inv.ToolName, args,
		originPrincipal, inv.ID, inv.PolicyVersion)
}

// dryRunForPrincipal 是预演的唯一实现：普通预演与复核共用，只是判定主体不同。
//
// 单点实现是「预演结论 = 真实执行结论」这条契约的技术保障：如果复核另写一份，
// 两条路径就会各自漂移，而审批人看到的结论与执行链的断言再也无法对齐。
func (s *Service) dryRunForPrincipal(
	ctx context.Context,
	userID, tenantID int,
	role, name string,
	args map[string]interface{},
	principal string,
	reviewedInvocationID int,
	baselinePolicyVersion string,
) (*dto.ToolDryRunResponse, error) {
	d, err := s.evaluateToolCall(ctx, tenantID, userID, role, name, principal)
	if err != nil {
		// 未知工具 / 定义漏配 / 依赖缺失：这些不是「判定结论」，是请求本身不成立。
		// 与 ExecuteTool 一致地落一条 denied 审计，让排查有据可依。
		if errors.Is(err, ErrUnknownTool) || errors.Is(err, ErrToolPermissionDenied) {
			s.recordDryRunAudit(ctx, tenantID, userID, role, name, args, "denied", err.Error(), "")
		}
		return nil, err
	}

	resp := &dto.ToolDryRunResponse{
		DryRun:             true,
		ToolName:           name,
		Allowed:            d.Allowed,
		NeedsApproval:      d.Allowed && d.NeedsApproval,
		RiskLevel:          d.RiskLevel,
		PolicyVersion:      d.PolicyVersion,
		Reason:             d.PermReason,
		EvaluatedPrincipal: principal,
	}
	// 只有「会被允许且需要审批」时才有审批态而言；被拒或无需审批时留空，
	// 避免调用方把 pending_agent 误读成「已经受理了」。
	if resp.NeedsApproval {
		resp.ApprovalState = approvalStateFor(principal, d.MinApprovals)
		resp.MinApprovals = d.MinApprovals
	}
	if !d.Allowed && resp.Reason == "" {
		resp.Reason = "工具调用被拒绝"
	}

	// 复核模式：附带与被复核记录的策略指纹比对结论。
	// 这段逻辑必须由服务端算，不能交给前端拼字符串——「同指纹 ⟺ 同判据」
	// 是本项目的核心不变量，前端各自实现一份比较必然出现口径分歧。
	if reviewedInvocationID > 0 {
		resp.InvocationID = reviewedInvocationID
		resp.BaselinePolicyVersion = baselinePolicyVersion
		resp.PolicyComparison = comparePolicyVersions(baselinePolicyVersion, d.PolicyVersion)
	}

	// 预演必须留痕：审计回答「谁在何时预演了什么、当时判定是什么」。
	// 少了这条，预演就成了一个无法追溯的只读通道——恰恰是审计系统最该覆盖的部分。
	status := "dry_run_allowed"
	if !d.Allowed {
		status = "dry_run_denied"
	}
	auditID := s.recordDryRunAudit(ctx, tenantID, userID, role, name, args,
		status, resp.Reason, d.PolicyVersion)
	resp.AuditID = auditID

	// 被拒不是系统错误：预演的职责就是如实报告「不行」，用 error 表达会让
	// 调用方把「预期结论」当成「调用失败」，从而重试或告警。
	return resp, nil
}

// comparePolicyVersions 比对两次判定的策略指纹，返回四态结论。
//
// 四态的划分依据是「能否据此断言放行判据未变」，而不是「字符串是否相等」：
//
//	两侧都空串 → not_applicable：两次判定都没有策略参与（human 路径），
//	             没有可比对象，断言「未变」与断言「已变」都是错的；
//	任一侧 unknown → unknown：读不出指纹就不能断言未变。**不得**归入
//	             identical——「策略信息缺失」伪装成「策略未变」是审计场景里
//	             最危险的假阴性；
//	相等 → identical：指纹是判定输入的 SHA-256，相等 ⟺ 判据相等；
//	不等 → changed：判据已变，原结论不能沿用。
//
// 注意空串与 unknown 同时出现时返回 unknown 而非 not_applicable：
// 后者要求**两侧**都无策略参与，只要有一侧「本该有却读不出」，就属于信息缺失。
func comparePolicyVersions(baseline, current string) string {
	hasMissing := baseline == authz.PolicyVersionUnknown || current == authz.PolicyVersionUnknown
	if hasMissing {
		return dto.PolicyComparisonUnknown
	}
	if baseline == "" && current == "" {
		return dto.PolicyComparisonNotApplicable
	}
	if baseline == current {
		return dto.PolicyComparisonIdentical
	}
	// 一侧空串另一侧有指纹：发生在「当年无策略参与、如今有策略参与」的
	// 策略体系升级场景，判据实质已变，必须报 changed 而不是 unknown。
	return dto.PolicyComparisonChanged
}

// recordDryRunAudit 落一条 dry_run=true 的审计行，返回行 ID（失败返回 0）。
//
// 与 recordToolAudit 分开而不是给它加一个布尔参数：两者写入的
// status / approval_state 语义完全不同（预演恒为 "none"，不参与审批流转），
// 一个参数化函数会让「预演行不小心带上 approval_state=pending」成为可能。
func (s *Service) recordDryRunAudit(ctx context.Context, tenantID, userID int, role, toolName string, args map[string]interface{}, status, reason, policyVersion string) int {
	argsStr, _ := json.Marshal(args)
	ticketRef, changeRef := businessRefsFromArgs(args)
	_, runID := EnsureRunID(ctx)
	var requestID string
	if rid, ok := middleware.RequestIDFromContext(ctx); ok {
		requestID = rid
	}
	permCheck := "passed"
	if strings.HasPrefix(status, "dry_run_denied") {
		permCheck = "denied"
	}
	inv, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:      tenantID,
		ToolName:      toolName,
		Arguments:     string(argsStr),
		Status:        status,
		NeedsApproval: false,
		// 预演不进入审批流：approval_state 必须是与待办查询不相交的取值，
		// 否则一条从未被调度的记录会出现在审批人的待办列表里。
		ApprovalState:    "none",
		UserID:           userID,
		PermissionCheck:  permCheck,
		PermissionReason: reason,
		RoleSnapshot:     role,
		TicketID:         ticketRef,
		ChangeID:         changeRef,
		PolicyVersion:    policyVersion,
		RunID:            runID,
		RequestID:        requestID,
		DryRun:           true,
	})
	if err != nil {
		s.logger.Errorw("AI tool dry-run audit persistence failed",
			"operation", "create_tool_invocation",
			"tenant_id", tenantID,
			"user_id", userID,
			"tool_name", toolName,
			"error", err,
		)
		metrics.AIPersistErrors.WithLabelValues(
			"create_tool_invocation",
			"dry_run",
			strconv.Itoa(tenantID),
		).Inc()
		return 0
	}
	return inv.ID
}

func (s *Service) recordToolAudit(ctx context.Context, tenantID, userID int, role, toolName string, args map[string]interface{}, permCheck, permReason, status string, result *string, needsApproval bool, policyVersion string) {
	argsStr, _ := json.Marshal(args)
	ticketRef, changeRef := businessRefsFromArgs(args)
	// run_id 与 request_id 在审计点补齐，让「同一轮 Agent 的动作」可整体取出。
	// 用 EnsureRunID 而不是只用 RunIDFromContext：单发 HTTP 工具调用没有上游
	// 注入 run_id，此时仍应给它一个独立轮次，而不是留空导致证据链断在这里。
	_, runID := EnsureRunID(ctx)
	var requestID string
	if rid, ok := middleware.RequestIDFromContext(ctx); ok {
		requestID = rid
	}
	if _, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:         tenantID,
		ToolName:         toolName,
		Arguments:        string(argsStr),
		Status:           status,
		NeedsApproval:    needsApproval,
		ApprovalState:    "auto",
		UserID:           userID,
		PermissionCheck:  permCheck,
		PermissionReason: permReason,
		RoleSnapshot:     role,
		TicketID:         ticketRef,
		ChangeID:         changeRef,
		PolicyVersion:    policyVersion,
		RunID:            runID,
		RequestID:        requestID,
	}); err != nil {
		s.logger.Errorw("AI tool audit persistence failed",
			"operation", "create_tool_invocation",
			"tenant_id", tenantID,
			"user_id", userID,
			"tool_name", toolName,
			"approval_state", "auto",
			"error", err,
		)
		metrics.AIPersistErrors.WithLabelValues(
			"create_tool_invocation",
			"",
			strconv.Itoa(tenantID),
		).Inc()
	}
}

// ApproveTool 审批一次工具调用。
// 审批状态与待执行任务在同一事务内落库（见 Repository.DecideToolInvocation），
// 因此通过审批不会再因为进程退出或队列满而丢失执行。
//
// Phase A（2026-11-17）审批人资格矩阵（路由 ai:approve 之外的二次校验）：
// 审批人须持有目标工具的 resource:action 权限——只会点按钮而无权执行
// 该工具的人，不应有权放行 AI 代人执行。拒绝票不改变任何业务状态，
// 不做资格校验（宽进严出：任何人可叫停 AI，放行才需资格）。
func (s *Service) ApproveTool(ctx context.Context, id int, tenantID, userID int, role string, approve bool, reason string) (string, error) {
	if approve {
		inv, err := s.repo.GetToolInvocation(ctx, id, tenantID)
		if err != nil {
			return "", err
		}
		toolDef := s.tools.GetTool(inv.ToolName)
		if toolDef == nil || toolDef.Resource == "" || toolDef.Action == "" {
			// 工具已下线或定义漏配：fail-closed，宁可不批也不能绕过资格校验。
			return "", common.NewBusinessError(common.ConflictCode,
				fmt.Sprintf("工具 %s 定义缺失，无法校验审批资格，拒绝处理", inv.ToolName), "")
		}
		if role != "" && !middleware.HasResourcePermission(ctx, s.entClient, role, toolDef.Resource, toolDef.Action, tenantID) {
			s.logger.Warnw("AI tool approval denied: approver lacks tool permission",
				"user_id", userID, "tenant_id", tenantID, "role", role,
				"tool", inv.ToolName, "resource", toolDef.Resource, "action", toolDef.Action)
			return "", common.NewBusinessError(common.ConflictCode,
				fmt.Sprintf("审批人须持有 %s:%s 权限才能放行该工具", toolDef.Resource, toolDef.Action), "")
		}
	}
	// 返回仓储层的真实状态：双人审批的第一票是 awaiting_second，
	// 不能对外谎报 approved（否则前端会以为操作已放行）。
	decided, err := s.repo.DecideToolInvocation(ctx, id, tenantID, userID, approve, reason)
	if err != nil {
		return "", err
	}
	if decided != nil && decided.ApprovalState != "" {
		return decided.ApprovalState, nil
	}
	if approve {
		return "approved", nil
	}
	return "rejected", nil
}

// Chat and RAG

func (s *Service) Chat(ctx context.Context, tenantID, userID int, query string, limit int, convID int) (interface{}, int, error) {
	s.logger.Infow("AI Chat", "query", query, "tenantID", tenantID)

	// RAG 未装配时返回错误而不是 panic：此前直接 s.rag.Ask(...) 在 rag 为 nil
	// 时解引用崩溃（nil receiver 上的字段访问）。生产装配一定会注入，但装配
	// 缺口不该以 500 + 崩溃的形式暴露——handler 已有降级分支可接住错误。
	if s.rag == nil {
		return nil, 0, ErrRAGUnavailable
	}

	// 记忆层：回读本会话历史，抽出的实体/短语作为**补充检索路**。
	//
	// 此前每轮只用当前 query 检索，「它怎么修」这类指代必然落空——表在、
	// 历史在，只是从没读回。补充路独立检索（不拼接 query：关键词检索是
	// 整串包含，拼接会让当前 query 也命不中），记忆召回的条目带
	// fromMemory 标注，与「库里写着」的权威内容区分开。
	memory := s.recallConversationMemory(ctx, tenantID, userID, convID)

	items, err := s.rag.AskMulti(ctx, tenantID, query, memory.Queries, limit)
	if err != nil {
		return nil, 0, err
	}

	// 情景记忆：历史工单的处置经验（RAG 的关键词检索只覆盖知识文章，
	// 历史结论一直躺在库里不被召回）。
	episodes := s.recallEpisodes(ctx, tenantID, userID, chatRole(ctx), query)
	if len(episodes) > 0 {
		items = mergeMemoryResults(items, episodesToResults(episodes), "")
	}

	// Persist conversation
	if convID == 0 {
		conv, err := s.repo.CreateConversation(ctx, &Conversation{
			Title:    "AI 对话",
			UserID:   userID,
			TenantID: tenantID,
		})
		if err == nil {
			convID = conv.ID
		}
	}

	if convID != 0 {
		// Chat 路径持久化失败必须可见。两类消息分别打点，避免一次失败掩盖另一类错误。
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		}); err != nil {
			s.logger.Errorw("AI chat persist user message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "user",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"user",
				strconv.Itoa(tenantID),
			).Inc()
		}
		payload, _ := json.Marshal(items)
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(payload),
		}); err != nil {
			s.logger.Errorw("AI chat persist assistant message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "assistant",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"assistant",
				strconv.Itoa(tenantID),
			).Inc()
		}
	}

	return items, convID, nil
}

// ragAskPromptVersion 是 RAG 问答的检索口径版本：换 embedding 模型、换召回
// 策略（top-k / 阈值 / 融合权重）都必须换它，否则底账里新旧口径的答案混在一起，
// 事后无法解释「为什么那批答案质量明显不同」。
const ragAskPromptVersion = "rag-ask-v1"

// ChatResult 是带检索溯源的问答结果。
//
// Model 是**检索模式**（hybrid/vector/keyword）而不是生成模型：RAG 问答的答案
// 片段来自知识库检索，不经过生成模型。把它写成某个模型名就是伪造归因。
// Confidence 是命中片段里的最高分（真实检索分，0 表示无命中）。
type ChatResult struct {
	Answers        interface{} `json:"answers"`
	ConversationID int         `json:"conversationId"`
	AuditID        int64       `json:"auditId,omitempty"`
	PromptVersion  string      `json:"promptVersion"`
	Model          string      `json:"model"`
	Confidence     float64     `json:"confidence"`
}

// ChatWithTrace 在 Chat 之上补检索溯源并落服务端决策底账。
//
// 为什么单独一个方法而不是改 Chat 的签名：Chat 的 (answers, convID, error)
// 三元组被 handler、skill、测试多处使用，改动面大；溯源是新增能力，
// 独立入口不影响既有调用。
//
// 底账照落但 confidence 是真实检索分：无命中时为 0——「问了但库里没有」是最
// 需要被记录的一类情况，否则它会被当成「AI 答得不好」而不是「知识库缺内容」。
func (s *Service) ChatWithTrace(ctx context.Context, tenantID, userID int, query string, limit, convID int) (*ChatResult, error) {
	answers, cid, err := s.Chat(ctx, tenantID, userID, query, limit, convID)
	if err != nil {
		return nil, err
	}

	mode := "unknown"
	if s.rag != nil {
		mode = s.rag.Mode()
	}
	confidence := topRetrievalScore(answers)

	out := &ChatResult{
		Answers:        answers,
		ConversationID: cid,
		PromptVersion:  ragAskPromptVersion,
		Model:          mode,
		Confidence:     confidence,
	}
	if s.auditLedger == nil {
		return out, nil
	}
	auditID, ledgerErr := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		ledgerRequestID(ctx, "chat"),
		"chat", fmt.Sprintf("query:%s", truncateForRef(query, 120)),
		ragAskPromptVersion, mode, confidence,
		map[string]interface{}{
			"retrievalMode": mode,
			"hitCount":      hitCount(answers),
		},
	)
	if ledgerErr != nil {
		s.logger.Warnw("Chat: audit ledger write failed (best-effort)", "error", ledgerErr)
		return out, nil
	}
	out.AuditID = auditID
	return out, nil
}

// topRetrievalScore 取命中片段里的最高分作为检索置信度。
// 取不到（无命中 / 结构不符）时返回 0——「没有可信命中」是实情，不是缺失。
func topRetrievalScore(answers interface{}) float64 {
	items, ok := answers.([]map[string]any)
	if !ok {
		return 0
	}
	var best float64
	for _, it := range items {
		if sc, ok := it["score"].(float64); ok && sc > best {
			best = sc
		}
	}
	return best
}

func hitCount(answers interface{}) int {
	if items, ok := answers.([]map[string]any); ok {
		return len(items)
	}
	return 0
}

// truncateForRef 把用户输入截断后作为底账的 inputRef：原样入库会让一条超长
// query 占满底账字段，截断到 120 字符足够定位又不至于膨胀。
func truncateForRef(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// chatWritableTools 是允许注入聊天路径的写工具白名单。
// 这些工具经由 ExecuteTool 的审批流（创建 pending invocation + 入队等待人工审批），
// 绝不在聊天链路内直接落库，因此可安全暴露给 LLM 自主决策调用。
var chatWritableTools = map[string]bool{
	"create_ticket":      true,
	"update_ticket":      true,
	"create_ticket_type": true,
	// CMDB 本体链路：把工单挂到配置项（走同一审批流）
	"link_ticket_ci": true,
	// CMDB 关系写操作：与上面同一审批流 + Gate2 RBAC（resource=ci_relationship, action=write）
	// LLM 在聊天中建/删 CI 关系必须走人工审批，与原生 UI 路径行为一致。
	"create_ci_relationship": true,
	"delete_ci_relationship": true,
}

// ChatStream streams a RAG answer through onDelta while emitting sources
// separately via onSources. It also persists the resulting conversation and
// messages after the stream completes so history is preserved.
//
// 注入按权限过滤后的只读工具。只有当前角色拥有 resource:action 权限的只读工具
// 才会进入聊天路径；写工具（!ReadOnly）绝不注入。模型发起的工具调用经由 execTool
// 复用 ExecuteTool 的 RBAC Gate 2 校验 + 审计记录，执行结果回填后继续流式生成。
func (s *Service) ChatStream(
	ctx context.Context,
	tenantID, userID int,
	role string,
	query string,
	limit int,
	convID int,
	onSources func([]map[string]any),
	onDelta func(string),
) (int, string, error) {
	s.logger.Infow("AI ChatStream", "query", query, "tenantID", tenantID, "convID", convID, "role", role)

	// 一轮对话 = 一个运行轮次。LLM 在这一轮里可能发起多次工具调用（查单、
	// 检索知识、挂 CI），它们必须共享同一个 run_id，否则「这个 Agent 这一轮
	// 做了什么」在数据层仍然无解——这正是本批要修的结构性缺口。
	//
	// 必须在 execTool 闭包之前注入：闭包捕获的是这里的 ctx，晚一步就只在
	// 部分调用上生效（那种「一半有 run_id 一半没有」的记录比完全没有更难排查）。
	ctx, runID := EnsureRunID(ctx)
	s.logger.Infow("AI ChatStream run started", "run_id", runID, "tenantID", tenantID)

	if s.rag == nil {
		return 0, "", fmt.Errorf("RAG service not initialized")
	}
	// 流式问答是读面契约之外的可选能力：只实现 RAGChatService 的注入方
	// 不具备它，与「未装配」同口径降级，不退回具体指针、也不 panic。
	streamer, hasStream := s.rag.(ragStreamer)
	if !hasStream {
		return 0, "", fmt.Errorf("RAG service not initialized")
	}

	// 按权限过滤只读工具注入聊天路径；写工具仅放行经审批流的白名单（创建工单/更新工单/创建工单类型），
	// 它们经由 ExecuteTool 进入待审批队列，绝不在聊天链路内直接落地写库。
	//
	// Provider 能力闸门：部分 provider（如 MiniMax Anthropic 兼容接口）仅实现了
	// Chat，未实现 ChatStreamWithTools。此时 LLMGateway 会退化为普通 ChatStream，
	// 但 System Prompt 依然告诉模型“必须调用工具”，导致模型在文本中假装调用
	// (“正在调用 list_tickets 工具...”) 但永远拿不到结果。检测 provider
	// 能力，不支持时不注入 tools，让 RAG 层退化为纯知识库问答（合规）。
	var tools []dto.LLMTool
	providerSupportsTools := s.llmGateway != nil && s.llmGateway.SupportsToolCalling()
	if s.tools != nil && providerSupportsTools {
		// 按租户动态化工具参数（list_cis 的 ci_type 枚举来自租户 CIType 表）
		for _, td := range s.tools.ListToolsForTenant(ctx, tenantID) {
			if !td.ReadOnly && !chatWritableTools[td.Name] {
				continue
			}
			if IsToolRBACEnabled() && s.entClient != nil && role != "" && role != "super_admin" {
				if !middleware.HasResourcePermission(ctx, s.entClient, role, td.Resource, td.Action, tenantID) {
					s.logger.Debugw("AI ChatStream: tool filtered by RBAC",
						"tool", td.Name, "resource", td.Resource, "action", td.Action, "role", role, "tenantID", tenantID)
					continue
				}
			}
			tools = append(tools, dto.LLMTool{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.ArgsSchema,
			})
		}
	}

	// 工具执行回调：复用 ExecuteTool 的 RBAC Gate 2 + 审计，与 agent 执行路径一致。
	// 写工具经审批流会返回 (res=nil, invID!=0, err=nil)，这里将其转译为结构化
	// approval_pending 提示，让 LLM 明确告知用户"操作已提交、待人工审批"。
	execTool := func(name string, args map[string]any) (any, error) {
		// LLM 自报的 user_id/requester_id 不可信，一律以认证上下文覆盖。
		args["user_id"] = float64(userID)
		args["requester_id"] = float64(userID)
		// LLM 自主决定的工具调用按 agent 主体判定：进入 DB 运行期策略
		// （默认零权限）+ 风险分级人工审批，而不是继承提问人的全部角色权限。
		agentCtx := authz.WithPrincipal(ctx, authz.PrincipalAgent)
		res, invID, err := s.ExecuteTool(agentCtx, userID, tenantID, role, name, args)
		if err != nil {
			return nil, err
		}
		if res == nil && invID != 0 {
			return map[string]any{
				"status":       "approval_pending",
				"tool":         name,
				"invocationId": invID,
				"message":      fmt.Sprintf("操作已提交，等待人工审批后执行（invocationId=%d）", invID),
			}, nil
		}
		return res, nil
	}

	var (
		captured strings.Builder
		sources  []map[string]any
	)

	wrappedSources := func(items []map[string]any) {
		sources = items
		if onSources != nil {
			onSources(items)
		}
	}
	wrappedDelta := func(delta string) {
		captured.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}

	// 记忆层（与 Chat 同口径）：会话历史抽出的实体/短语作为补充检索路。
	memory := s.recallConversationMemory(ctx, tenantID, userID, convID)

	// 情景记忆：历史工单的处置经验。ChatStream 有显式 role 参数（流式路径
	// 的权限判定走这里），行级范围与 Chat 同口径。
	episodes := s.recallEpisodes(ctx, tenantID, userID, role, query)

	if err := streamer.AskWithLLMStreamWithToolsMulti(
		ctx, tenantID, query, memory.Queries, s.llmGateway, limit, tools,
		wrappedSources, wrappedDelta, execTool, episodesToResults(episodes)); err != nil {
		return 0, "", err
	}

	// Persist conversation and messages after the stream completes so we don't
	// leave partial messages if the client disconnects mid-stream.
	if convID == 0 {
		conv, err := s.repo.CreateConversation(ctx, &Conversation{
			Title:    "AI 对话",
			UserID:   userID,
			TenantID: tenantID,
		})
		if err == nil && conv != nil {
			convID = conv.ID
		}
	}
	if convID != 0 {
		// ChatStream 路径持久化失败同样必须可见，与 Chat 路径共用同一计数器。
		// 流式响应已经写回客户端；如果 DB 写入失败而日志被吞，
		// 用户会看到回复但刷新后历史丢失，难以排查。
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		}); err != nil {
			s.logger.Errorw("AI chatstream persist user message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "user",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"user",
				strconv.Itoa(tenantID),
			).Inc()
		}
		payload := map[string]any{
			"answer":  captured.String(),
			"sources": sources,
		}
		buf, _ := json.Marshal(payload)
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(buf),
		}); err != nil {
			s.logger.Errorw("AI chatstream persist assistant message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "assistant",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"assistant",
				strconv.Itoa(tenantID),
			).Inc()
		}
	}

	return convID, captured.String(), nil
}

// chatRole 从上下文取当前用户的角色，供 DataScope 判定工单行级可见范围。
//
// 角色来自知识访问者身份（handler 已注入 knowledgeaccess.WithViewer）——
// 与 RAG 的分类可见性过滤同一套身份来源，避免「知识按身份过滤、工单按
// 另一套过滤」的口径分裂。
//
// 上下文无身份时返回空串：datascope.ForTicketReader 对空角色返回最保守档
// （本人创建或分配给本人），不会放宽到全租户。权限不可得时收窄是唯一安全的
// 方向——放宽会把用户无权查看的工单处置喂给 LLM。
func chatRole(ctx context.Context) string {
	if v, ok := knowledgeaccess.ViewerFrom(ctx); ok {
		return v.Role
	}
	return ""
}

func (s *Service) ListConversations(ctx context.Context, tenantID, userID int) ([]*Conversation, error) {
	return s.repo.ListConversations(ctx, tenantID, userID)
}

func (s *Service) GetConversation(ctx context.Context, id, tenantID, userID int) (*Conversation, error) {
	return s.repo.GetConversation(ctx, id, tenantID, userID)
}

// GetConversationMessages 读会话内的消息。
//
// 先按「租户 + 属主」校验会话存在——会话是用户私有的，同租户内他人无权
// 读取；未命中时返回空切片与错误，由调用方转成 404（不暴露会话是否存在）。
func (s *Service) GetConversationMessages(ctx context.Context, convID, tenantID, userID int) ([]*Message, error) {
	if _, err := s.repo.GetConversation(ctx, convID, tenantID, userID); err != nil {
		return nil, err
	}
	return s.repo.GetMessages(ctx, convID, tenantID)
}

func (s *Service) DeleteConversation(ctx context.Context, id, tenantID, userID int) error {
	return s.repo.DeleteConversation(ctx, id, tenantID, userID)
}

// Root Cause Analysis

func (s *Service) AnalyzeTicket(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	if s.rca == nil {
		return nil, dto.ErrAIAnalysisUnavailable
	}
	return s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
}

func (s *Service) AnalyzeTicketWithAudit(ctx context.Context, ticketID int, tenantID int, userID int) (interface{}, error) {
	// RCA 未装配时必须报错而不是解引用 nil：AnalyzeIncident 早有此守卫，
	// AnalyzeTicket 之前直接 s.rca.AnalyzeTicket(...)，装配缺失时会 panic，
	// 表现为 HTTP 500 而非可诊断的错误分类。
	if s.rca == nil {
		return nil, dto.ErrAIAnalysisUnavailable
	}
	result, err := s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("AnalyzeTicket ticketID=%d", ticketID)
	confidence, promptVersion, _ := aiResultTrace(result)
	degraded, degradedReason := aiResultDegrade(result)
	s.persistAIResult(ctx, "rca", tenantID, userID, prompt+" promptVersion="+promptVersion, result, "", 0, 0, &confidence, degraded, degradedReason)
	return result, nil
}

func (s *Service) AnalyzeIncident(ctx context.Context, incidentID int, tenantID int) (interface{}, error) {
	if s.rca == nil {
		return nil, dto.ErrAIAnalysisUnavailable
	}
	return s.rca.AnalyzeIncident(ctx, incidentID, tenantID)
}

func (s *Service) AnalyzeIncidentWithAudit(ctx context.Context, incidentID int, tenantID int, userID int) (interface{}, error) {
	result, err := s.AnalyzeIncident(ctx, incidentID, tenantID)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("AnalyzeIncident incidentID=%d", incidentID)
	confidence, promptVersion, _ := aiResultTrace(result)
	degraded, degradedReason := aiResultDegrade(result)
	s.persistAIResult(ctx, "incident_impact", tenantID, userID, prompt+" promptVersion="+promptVersion, result, "", 0, 0, &confidence, degraded, degradedReason)
	return result, nil
}

// aiResultDegrade 提取分析结果是否走了降级路径。落库时把降级标记一起写进去，
// 否则 ai_analysis_results 里全是 Degraded=false，事后审计无法区分
// 「模型真产出」与「LLM 超时后的启发式兜底」。
func aiResultDegrade(result interface{}) (degraded bool, reason string) {
	switch r := result.(type) {
	case *dto.TicketRootCauseAnalysisResponse:
		return r.Degraded, r.DegradedReason
	case *dto.IncidentAIAnalysisResponse:
		return r.Degraded, r.DegradedReason
	default:
		return false, ""
	}
}

// aiResultTrace 从 RCA/事件分析响应中提取置信度与 prompt 版本，供持久化与底账使用。
// 未知类型返回 ok=false（confidence=0，不落置信度语义）。
func aiResultTrace(result interface{}) (confidence float64, promptVersion string, ok bool) {
	switch r := result.(type) {
	case *dto.TicketRootCauseAnalysisResponse:
		return r.ConfidenceScore, r.PromptVersion, true
	case *dto.IncidentAIAnalysisResponse:
		return r.ConfidenceScore, r.PromptVersion, true
	default:
		return 0, "", false
	}
}

// CreateTicketByAI 通过 AI 解析自然语言描述，智能分析并返回工单创建建议
//
// 响应字段统一 camelCase（AGENTS.md 命名强制规则）：此前输出 suggested_title /
// tenant_id 等 snake_case 字段。AI 链路不可用的 fallback 分支必须显式置
// degraded=true——降级结果不能伪装成模型正常产出（与 TriageTicket 的降级
// 契约同口径）。
func (s *Service) CreateTicketByAI(ctx context.Context, description string, tenantID int) (map[string]interface{}, error) {
	s.logger.Infow("AI CreateTicketByAI", "tenantID", tenantID)

	// 使用 Triage 服务分析描述，提取分类和优先级
	if s.triageService != nil {
		result := s.triageService.SuggestForTenant(ctx, description, description, tenantID)
		return map[string]interface{}{
			"suggestedTitle":    description,
			"suggestedCategory": result.Category,
			"suggestedPriority": result.Priority,
			"confidence":        result.Confidence,
			"reasoning":         result.Explanation,
			"tenantId":          tenantID,
			"status":            "draft",
			"message":           "AI 已分析描述，请确认后提交工单",
		}, nil
	}

	// fallback：基于关键词简单分析
	return map[string]interface{}{
		"suggestedTitle":    description,
		"suggestedCategory": "general",
		"suggestedPriority": "medium",
		"confidence":        0.5,
		"reasoning":         "基于关键词分析",
		"tenantId":          tenantID,
		"status":            "draft",
		"message":           "AI 服务不可用，已返回默认建议",
		"degraded":          true,
	}, nil
}

// defaultSummaryMaxLen 是摘要的目标字数上限。
const defaultSummaryMaxLen = 200

// SummarizeTicket B9: AI 工单总结。
//
// 此前这里只委托 rca，而 SetSummarizeService 的注释却写着「优先走
// SummarizeService.SummarizeWithMetadata」——装配与实现不一致：summarizeService
// 被注入后从未被调用（死装配）。后果是摘要永远走 rca 那条路，而 rca 的默认
// 产物是**字段拼接**（method=fallback），没有任何字段能区分「模型生成的摘要」
// 与「拼出来的原文」。这就是能力矩阵里「默认摘要仍是原文拼接」的由来。
//
// 现在按注释真正接上：SummarizeService 产出 method/model/promptVersion 三件套
// （llm / direct / fallback 如实区分），失败再兜底 rca。两条路径都落服务端 AI
// 决策底账并带出 auditId，操作者可经 PATCH /ai/audit-logs/:id 翻转采纳/拒绝。
//
// userID 为 0 表示无用户上下文（如 skill 调用），底账照落（user_id=0 是实情，
// 不是缺失），不因拿不到用户就放弃溯源。
func (s *Service) SummarizeTicket(ctx context.Context, ticketID int, tenantID int, userID int) (interface{}, error) {
	if s.summarizeService != nil && s.entClient != nil {
		text, err := s.buildTicketSummaryText(ctx, ticketID, tenantID)
		if err != nil {
			s.logger.Warnw("SummarizeTicket: 组装摘要输入失败，兜底 rca", "error", err, "ticketID", ticketID)
		} else {
			meta, mErr := s.summarizeService.SummarizeWithMetadata(ctx, text, defaultSummaryMaxLen)
			if mErr != nil || meta == nil {
				s.logger.Warnw("SummarizeTicket: SummarizeService 失败，兜底 rca", "error", mErr, "ticketID", ticketID)
			} else {
				return s.withSummaryAudit(ctx, ticketID, tenantID, userID, meta.Summary, meta.Method, meta.PromptVersion, meta.Model, meta.Confidence), nil
			}
		}
	}
	return s.rca.SummarizeTicket(ctx, ticketID, tenantID)
}

// buildTicketSummaryText 组装喂给摘要模型的工单文本。
//
// 与 rca 里的那份拼接是两处独立实现：rca 拼的是「fallback 展示文案」（人读的
// 一行状态描述），这里拼的是「模型输入」（结构化字段）。两者目的不同，不共用——
// 把 fallback 文案当模型输入会让模型复述状态字段而不是概括问题。
func (s *Service) buildTicketSummaryText(ctx context.Context, ticketID, tenantID int) (string, error) {
	if s.entClient == nil {
		return "", fmt.Errorf("ent client not initialized")
	}
	tk, err := s.entClient.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("工单号：")
	b.WriteString(tk.TicketNumber)
	b.WriteString("\n标题：")
	b.WriteString(tk.Title)
	if tk.Description != "" {
		b.WriteString("\n描述：")
		b.WriteString(tk.Description)
	}
	b.WriteString("\n状态：")
	b.WriteString(tk.Status)
	b.WriteString("\n优先级：")
	b.WriteString(tk.Priority)
	return b.String(), nil
}

// withSummaryAudit 给摘要结果补溯源字段并落服务端决策底账。
//
// confidence 对摘要恒为 0（生成类产出没有自评分数，硬填就是编造），底账的
// notes 里显式标注 confidenceNotApplicable + method，让事后看底账的人知道
// 「这条记录的 0 是不适用，不是模型给了零分」。method=direct/fallback 时也照落：
// 恰恰是这些「不是模型产出」的摘要最需要被记录——否则采纳率会把它当成 AI 表现。
func (s *Service) withSummaryAudit(ctx context.Context, ticketID, tenantID, userID int, summary, method, promptVersion, model string, confidence float64) map[string]interface{} {
	resp := map[string]interface{}{
		"ticketId":      ticketID,
		"summary":       summary,
		"method":        method,
		"promptVersion": promptVersion,
		"model":         model,
		"confidence":    confidence,
		"generatedAt":   time.Now().Format(time.RFC3339),
	}
	if s.auditLedger == nil {
		return resp
	}
	auditID, err := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		ledgerRequestID(ctx, "summary"),
		"summary", fmt.Sprintf("ticket:%d", ticketID),
		promptVersion, model, confidence,
		map[string]interface{}{
			"method":                  method,
			"confidenceNotApplicable": true,
			"summaryChars":            len([]rune(summary)),
		},
	)
	if err != nil {
		s.logger.Warnw("SummarizeTicket: audit ledger write failed (best-effort)", "error", err)
		return resp
	}
	resp["auditId"] = auditID
	return resp
}

func (s *Service) GetAnalysisReport(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.GetAnalysisReport(ctx, ticketID, tenantID)
}

// Analytics and Prediction

// analyticsMetricVersion 是深度分析的统计口径版本。
const analyticsMetricVersion = "deep-analytics-v1"

// GetDeepAnalytics 深度分析。
//
// 它是 SQL 聚合，不是 AI 生成——所以这里只补口径版本与 method="aggregate"，
// **不补 model**。此前能力矩阵把 analytics 与 chat/摘要并列为「缺溯源字段」，
// 但给聚合结果加一个 model 字段只会制造假溯源：客户端会以为这是某模型的建议，
// 而实际没有任何模型参与。诚实标注「这是聚合，口径版本是 X」才是它该有的溯源。
func (s *Service) GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int, viewer dto.DashboardViewer) (interface{}, error) {
	res, err := s.analytics.GetDeepAnalytics(ctx, req, tenantID, viewer)
	if err != nil {
		return nil, err
	}
	if res != nil {
		res.MetricVersion = analyticsMetricVersion
		res.Method = "aggregate"
	}
	return res, nil
}

// slaForecastPromptVersion 是 SLA 趋势预测的口径版本。
//
// 预测值会被拿去做容量与风险决策，「这组数按什么口径算的」必须可追：换了
// 预测算法或平滑参数而不换版本号，底账里新旧口径的预测就无法区分。
const slaForecastPromptVersion = "sla-forecast-v1"

func (s *Service) GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, userID int, viewer dto.DashboardViewer) (interface{}, error) {
	// 前端不带 timeRange 时默认为「过去 6 个月 → 今天」。
	// 否则 req.TimeRange[0/1] 会 panic,与共享层预测服务行为对齐。
	if len(req.TimeRange) != 2 {
		now := time.Now()
		req.TimeRange = []string{
			now.AddDate(0, -6, 0).Format("2006-01-02"),
			now.Format("2006-01-02"),
		}
	}

	// Try to use AI-Native SLAForecastSkill first
	if s.slaForecastSkill != nil {
		input := &dto.ForecastInput{
			TenantID:  tenantID,
			StartDate: parseDate(req.TimeRange[0]),
			EndDate:   parseDate(req.TimeRange[1]),
			Metrics:   []string{req.PredictionType},
			Viewer:    viewer,
		}
		output, err := s.slaForecastSkill.Execute(ctx, input)
		if err == nil {
			// slaForecastSkill 的 PredictionPoint 用
			// "predicted" 字段，与 dto.TrendPredictionResponse.Predictions 契约
			// "predictedValue" 不一致。出口做一层转译，保留 SLAForecastSkill 的
			// 全部增强字段（insights / seasonality / trend / anomalyDates），
			// 同时让前端 SLATrendPredictionCard 的 predictedValue 字段拿到真值。
			model, confidence := "", 0.0
			if output != nil {
				model, confidence = output.Model, output.Confidence
			}
			auditID := s.recordForecastAudit(ctx, tenantID, userID,
				fmt.Sprintf("trend:%s", req.PredictionType), model, confidence)
			return normalizeForecastOutput(output, slaForecastPromptVersion, auditID), nil
		}
		// Fall back to legacy prediction service on error
		s.logger.Warnw("SLAForecastSkill failed, falling back to legacy", "error", err)
	}

	res, err := s.prediction.GetTrendPrediction(ctx, req, tenantID, viewer)
	if err != nil {
		return nil, err
	}
	// 兜底路同样补溯源：legacy 预测服务产出的数同样会被拿去决策，
	// 没有口径版本就等于有一批预测来源不明。
	if res != nil {
		res.PromptVersion = slaForecastPromptVersion
		res.AuditID = s.recordForecastAudit(ctx, tenantID, userID,
			fmt.Sprintf("trend:%s", req.PredictionType), res.Model, res.Confidence)
	}
	return res, nil
}

// recordForecastAudit 落一条预测决策底账并返回行 ID；底账不可用时返回 0
// （best-effort：预测结果照常返回，只是这次没有底账 ID）。
//
// 预测与分诊同源地落 ai_audit：采纳/拒绝统一走 PATCH /ai/audit-logs/:id，
// 不再由前端自报。confidence 直接取 ForecastOutput 的真实计算值——它与分诊
// 不同，预测本来就有可计算的置信度（不是生成类产出）。
func (s *Service) recordForecastAudit(ctx context.Context, tenantID, userID int, inputRef, model string, confidence float64) int64 {
	if s.auditLedger == nil {
		return 0
	}
	auditID, err := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		ledgerRequestID(ctx, "forecast"),
		"forecast", inputRef, slaForecastPromptVersion, model, confidence,
		map[string]interface{}{"model": model},
	)
	if err != nil {
		s.logger.Warnw("forecast: audit ledger write failed (best-effort)", "error", err)
		return 0
	}
	return auditID
}

// normalizeForecastOutput 把 SLAForecastSkill.ForecastOutput 转译为 dto 契约
// （predictions[i].predictedValue 而非 predicted）。同时保留 SLAForecastSkill
// 的 AI 增强字段（insights/seasonality/trend/anomalyDates）作为额外元数据。
func normalizeForecastOutput(output *dto.ForecastOutput, promptVersion string, auditID int64) interface{} {
	if output == nil {
		return output
	}
	type predictionDTO struct {
		Date           string  `json:"date"`
		PredictedValue float64 `json:"predictedValue"`
		LowerBound     float64 `json:"lowerBound"`
		UpperBound     float64 `json:"upperBound"`
		Confidence     float64 `json:"confidence"`
		Anomaly        bool    `json:"anomaly"`
	}
	type forecastDTO struct {
		Predictions  []predictionDTO `json:"predictions"`
		Confidence   float64         `json:"confidence"`
		Model        string          `json:"model"`
		GeneratedAt  string          `json:"generatedAt"`
		Insights     string          `json:"insights"`
		Seasonality  map[string]bool `json:"seasonality"`
		Trend        string          `json:"trend"`
		AnomalyDates []string        `json:"anomalyDates"`
		// 溯源：口径版本 + 服务端决策底账 ID（0 = 底账不可用，如实不填）
		PromptVersion string `json:"promptVersion,omitempty"`
		AuditID       int64  `json:"auditId,omitempty"`
	}
	points := make([]predictionDTO, 0, len(output.Predictions))
	for _, p := range output.Predictions {
		points = append(points, predictionDTO{
			Date:           p.Date.Format("2006-01-02"),
			PredictedValue: p.Predicted,
			LowerBound:     p.LowerBound,
			UpperBound:     p.UpperBound,
			Confidence:     p.Confidence,
			Anomaly:        p.Anomaly,
		})
	}
	return forecastDTO{
		Predictions:   points,
		Confidence:    output.Confidence,
		Model:         output.Model,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Insights:      output.Insights,
		Seasonality:   output.Seasonality,
		Trend:         output.Trend,
		AnomalyDates:  output.AnomalyDates,
		PromptVersion: promptVersion,
		AuditID:       auditID,
	}
}

// GetForecastInsights returns AI-generated insights for SLA forecasting
func (s *Service) GetForecastInsights(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, userID int, viewer dto.DashboardViewer) (interface{}, error) {
	if s.slaForecastSkill == nil {
		return nil, fmt.Errorf("SLAForecastSkill not initialized")
	}

	input := &dto.ForecastInput{
		TenantID:  tenantID,
		StartDate: parseDate(req.TimeRange[0]),
		EndDate:   parseDate(req.TimeRange[1]),
		Metrics:   []string{req.PredictionType},
		Viewer:    viewer,
	}

	output, err := s.slaForecastSkill.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	// Return insights-focused response（同样带溯源字段：洞察会被直接展示给
	// 管理者并据此做决策，没有口径版本就没有办法解释「为什么上个月这么说、
	// 这个月又那么说」）。
	auditID := s.recordForecastAudit(ctx, tenantID, userID,
		fmt.Sprintf("insights:%s", req.PredictionType), output.Model, output.Confidence)
	return map[string]interface{}{
		"confidence":    output.Confidence,
		"model":         output.Model,
		"insights":      output.Insights,
		"seasonality":   output.Seasonality,
		"trend":         output.Trend,
		"anomaly_dates": output.AnomalyDates,
		"promptVersion": slaForecastPromptVersion,
		"auditId":       auditID,
	}, nil
}

// Telemetry

// SaveFeedback delegates to AITelemetryService.
// accepted (*bool) for ai_audit accept/reject decisions to be first-class
// (was overloaded onto useful).
func (s *Service) SaveFeedback(ctx context.Context, tenantID, userID int, requestID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string, accepted *bool) error {
	if s.aiTelemetryService == nil {
		return fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes, accepted)
}

// UpdateAuditAccepted flips accepted on an ai_feedbacks row whose item_type
// is 'ai_audit'. Returns ErrFeedbackNotFound (dto sentinel) when the row is
// missing, belongs to another tenant, or is not an audit row.
func (s *Service) UpdateAuditAccepted(ctx context.Context, tenantID int, id int64, accepted bool) error {
	if s.aiTelemetryService == nil {
		return fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.UpdateAuditAccepted(ctx, tenantID, id, accepted)
}

func (s *Service) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (interface{}, error) {
	if s.aiTelemetryService == nil {
		return nil, fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.GetMetrics(ctx, tenantID, lookbackDays)
}

// SearchKnowledge handles RAG search over knowledge base
func (s *Service) SearchKnowledge(ctx context.Context, tenantID int, query string, searchType string, limit int) (interface{}, error) {
	s.logger.Infow("Knowledge Search", "query", query, "type", searchType, "tenantID", tenantID)

	if s.rag == nil {
		return nil, fmt.Errorf("RAG 服务未配置")
	}

	results, err := s.rag.Ask(ctx, tenantID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("RAG 搜索失败: %w", err)
	}

	return results, nil
}

// TriageTicket provides ticket classification and recommendations.
//
// 移除内联关键词 fallback，统一委托共享层分诊服务（Guidance → LLM → keyword 三级链）。
// triageService 不可用时返回 unclassified + confidence=0，不再维护第二套分类枚举。
//
// 置信度/溯源强制化：结果携带 promptVersion/model 溯源字段，并自动落一条服务端
// AI 决策底账（item_type='ai_audit'，accepted=NULL 未决策）。响应携带 auditId，
// 前端采纳/拒绝直接 PATCH /ai/audit-logs/:id 翻转底账，不再需要伪造 model
// 二次 POST 自报。底账写失败不阻塞建议返回（best-effort），仅去掉 auditId。
//
// ticketID > 0 时**按工单幂等**：已有分诊结论就直接复用，不重复调 LLM、不重复落底账。
// 此前每次进入工单详情都会重跑一次分诊，同一工单的分类在「存储」与「网络」之间漂移，
// 且每次都新增一条底账（prod 实测 F2）。复用结论时同样返回 auditId，
// 前端据此显示「已采纳/未采纳」。
func (s *Service) TriageTicket(ctx context.Context, tenantID int, userID int, title, description, category, priority string, ticketID int) (interface{}, error) {
	s.logger.Infow("Ticket Triage", "title", title, "tenantID", tenantID, "ticket_id", ticketID)

	// 按工单复用既有结论：同一工单只分析一次。
	if ticketID > 0 {
		if cached := s.cachedTriageForTicket(ctx, tenantID, ticketID); cached != nil {
			return cached, nil
		}
	}

	if s.triageService != nil {
		// 用租户版 SuggestForTenant：此前调 Suggest 内部硬编码 tenant=1，
		// Guidance sidecar 拿到的租户上下文是错的。
		result := s.triageService.SuggestForTenant(ctx, title, description, tenantID)
		response := map[string]interface{}{
			"title":         title,
			"description":   description,
			"promptVersion": result.PromptVersion,
			"model":         result.Model,
			"suggestions": map[string]interface{}{
				"category":   result.Category,
				"priority":   result.Priority,
				"confidence": result.Confidence,
				"reasoning":  result.Explanation,
				"urgency":    s.determineUrgency(result.Priority),
			},
		}
		// 降级结果必须自报家门：前端/审计据此区分「模型结论」与「关键词凑数」，
		// 否则 sidecar 把 LLM 故障包成 200 之后，界面与底账都看不出异常。
		if result.Degraded {
			response["degraded"] = true
			response["degradedReason"] = result.DegradedReason
		}
		if s.auditLedger != nil {
			metadata := map[string]interface{}{
				"category": result.Category,
				"priority": result.Priority,
				"degraded": result.Degraded,
			}
			if result.Degraded {
				metadata["degradedReason"] = result.DegradedReason
			}
			auditID, ledgerErr := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
				ledgerRequestID(ctx, "triage"),
				"triage", fmt.Sprintf("ticket:%s", title),
				result.PromptVersion, result.Model, result.Confidence,
				metadata,
			)
			if ledgerErr != nil {
				s.logger.Warnw("TriageTicket: audit ledger write failed (best-effort)", "error", ledgerErr)
			} else {
				response["auditId"] = auditID
			}
		}
		// 落库分诊结论，供下次按工单复用（幂等的持久化载体）。
		if ticketID > 0 {
			s.persistTriageForTicket(ctx, tenantID, userID, ticketID, title, result, response)
		}
		return response, nil
	}

	return map[string]interface{}{
		"title":       title,
		"description": description,
		"suggestions": map[string]interface{}{
			"category":   "unclassified",
			"priority":   "unclassified",
			"confidence": 0,
			"reasoning":  "triage service unavailable",
			"urgency":    "medium",
		},
	}, nil
}

// triageAnalysisType 是分诊结论在 ai_analysis_results 里的分析类型。
const triageAnalysisType = "triage"

// cachedTriageForTicket 读取该工单既有的分诊结论并还原成响应。
//
// 返回 nil 表示「首次分析」，调用方照常跑一次分诊。
// 复用时会把历史结论原样返回（含 category/priority/confidence），
// 保证同一工单的分类不再在多次进入详情页之间漂移。
func (s *Service) cachedTriageForTicket(ctx context.Context, tenantID, ticketID int) map[string]interface{} {
	if s.repo == nil {
		return nil
	}
	record, err := s.repo.GetLatestAIAnalysisResultByTicket(ctx, ticketID, tenantID, triageAnalysisType)
	if err != nil || record == nil || record.ResultJSON == "" {
		return nil
	}
	var stored map[string]interface{}
	if err := json.Unmarshal([]byte(record.ResultJSON), &stored); err != nil {
		return nil
	}
	suggestions, _ := stored["suggestions"].(map[string]interface{})
	if suggestions == nil {
		return nil
	}
	response := map[string]interface{}{
		"title":         stored["title"],
		"description":   stored["description"],
		"promptVersion": stored["promptVersion"],
		"model":         stored["model"],
		"suggestions":   suggestions,
		"cached":        true,
	}
	if auditID, ok := stored["auditId"]; ok {
		response["auditId"] = auditID
	}
	if degraded, ok := stored["degraded"]; ok {
		response["degraded"] = degraded
	}
	if degradedReason, ok := stored["degradedReason"]; ok {
		response["degradedReason"] = degradedReason
	}
	s.logger.Infow("复用既有分诊结论（不再重复调用模型）", "tenant_id", tenantID, "ticket_id", ticketID)
	return response
}

// persistTriageForTicket 把分诊结论落库，作为该工单下次复用的载体。
//
// best-effort：落库失败不影响本次建议返回，只是下次会重新分析。
func (s *Service) persistTriageForTicket(ctx context.Context, tenantID, userID, ticketID int, title string, result dto.TriageResult, response map[string]interface{}) {
	if s.repo == nil {
		return
	}
	payload, err := json.Marshal(response)
	if err != nil {
		s.logger.Warnw("persistTriageForTicket: marshal failed", "error", err)
		return
	}
	record := &AIAnalysisResult{
		TenantID:        tenantID,
		UserID:          userID,
		AnalysisType:    triageAnalysisType,
		TicketID:        ticketID,
		TicketTitle:     title,
		RequestPrompt:   title,
		ResultJSON:      string(payload),
		Model:           result.Model,
		ConfidenceScore: result.Confidence,
		Degraded:        result.Degraded,
	}
	if _, err := s.repo.SaveAIAnalysisResult(ctx, record); err != nil {
		s.logger.Warnw("persistTriageForTicket: save failed", "error", err)
	}
}

func (s *Service) determineUrgency(priority string) string {
	switch priority {
	case "critical":
		return "high"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

// Evaluate delegates to AITelemetryService.Evaluate (nil-safe for unit tests).
func (s *Service) Evaluate(ctx context.Context, tenantID int, days int) (*dto.AIEvaluationReport, error) {
	if s.aiTelemetryService == nil {
		return &dto.AIEvaluationReport{GeneratedAt: time.Now().Format(time.RFC3339), LookbackDays: days}, nil
	}
	return s.aiTelemetryService.Evaluate(ctx, tenantID, days)
}

// ListAuditLogs delegates to AITelemetryService.ListAuditLogs (nil-safe for unit tests).
func (s *Service) ListAuditLogs(ctx context.Context, tenantID, page, pageSize int, kind string, days int) ([]dto.AIAuditEntry, int, error) {
	if s.aiTelemetryService == nil {
		return []dto.AIAuditEntry{}, 0, nil
	}
	return s.aiTelemetryService.ListAuditLogs(ctx, tenantID, page, pageSize, kind, days)
}

// ExposeResultTraceForTest 供包外测试验证 aiResultTrace 的类型提取逻辑。
func ExposeResultTraceForTest(result interface{}) (float64, string, bool) {
	return aiResultTrace(result)
}

// parseDate parses date string in YYYY-MM-DD format
func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Now()
	}
	return t
}

// persistAIResult serializes and saves an AI analysis result to the database.
// Errors are logged but not propagated — analysis results should still be returned
// to the caller even if persistence fails.
func (s *Service) persistAIResult(ctx context.Context, analysisType string, tenantID, userID int, prompt string, result interface{}, model string, latencyMs, totalTokens int, confidenceScore *float64, degraded bool, degradedReason string) {
	if s.repo == nil {
		return
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		s.logger.Warnw("persistAIResult: marshal failed", "error", err)
		return
	}
	record := &AIAnalysisResult{
		TenantID:      tenantID,
		UserID:        userID,
		AnalysisType:  analysisType,
		RequestPrompt: prompt,
		ResultJSON:    string(resultJSON),
		Model:         model,
		// 降级标记如实入库：此处曾硬编码 false，使启发式兜底结果在审计底账里
		// 表现为模型正常产出。降级原因不进列（表无 degraded_reason 字段，
		// 加列需改 schema + 迁移），改随日志留存，避免为一条标记动表结构。
		Degraded: degraded,
	}
	if degraded {
		s.logger.Warnw("persistAIResult: degraded analysis persisted",
			"analysis_type", analysisType, "tenant_id", tenantID,
			"user_id", userID, "degraded_reason", degradedReason)
	}
	if latencyMs > 0 {
		record.LatencyMs = latencyMs
	}
	if totalTokens > 0 {
		record.TotalTokens = totalTokens
	}
	if confidenceScore != nil {
		record.ConfidenceScore = *confidenceScore
	}
	_, err = s.repo.SaveAIAnalysisResult(ctx, record)
	if err != nil {
		s.logger.Warnw("persistAIResult: save failed", "error", err)
	}
}

// ListAIAnalysisResults returns analysis history for the tenant.
func (s *Service) ListAIAnalysisResults(ctx context.Context, tenantID int, analysisType string, limit int) ([]*AIAnalysisResult, error) {
	return s.repo.ListAIAnalysisResults(ctx, tenantID, analysisType, limit)
}

// GetAIAnalysisResult returns a single analysis result by ID.
func (s *Service) GetAIAnalysisResult(ctx context.Context, id int, tenantID int) (*AIAnalysisResult, error) {
	return s.repo.GetAIAnalysisResult(ctx, id, tenantID)
}

// DeleteAIAnalysisResult deletes an analysis result record.
func (s *Service) DeleteAIAnalysisResult(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteAIAnalysisResult(ctx, id, tenantID)
}
