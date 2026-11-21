package dto

import "errors"

// ErrToolPermissionDenied 表示 AI 工具执行被运行期策略或身份守卫拒绝。
//
// 定义在 dto 层是为了让 handlers/skill 能在不依赖 handlers/ai 的前提下用
// errors.Is 识别拒绝语义。此前 registry 用 %v 包装执行错误，链路断开，
// 权限拒绝被统一兜底成 500/5001，操作者无法区分「无授权」与「服务异常」。
var ErrToolPermissionDenied = errors.New("tool permission denied")

// ToolExecutionResponse distinguishes an accepted approval request from execution.
type ToolExecutionResponse struct {
	Status        string      `json:"status"`
	Summary       string      `json:"summary"`
	InvocationID  int         `json:"invocationId"`
	ApprovalState string      `json:"approvalState"`
	NextActions   []string    `json:"nextActions"`
	Artifacts     []string    `json:"artifacts"`
	Data          interface{} `json:"data"`
}

// ToolDryRunResponse 是「预演」的结论：现在调用这个工具会发生什么。
//
// 与 ToolExecutionResponse 刻意分开而不是复用一个结构：
// 预演没有 invocationId（不建待审批行）、没有 data（没执行）、没有 approvalState
// 的流转语义（没进审批流）。塞进同一个结构就必须用零值表达「不存在」，
// 而调用方无法区分「值为零」与「字段不适用」——这正是 dry_run 之前沦为空字段的原因。
//
// 关键约束：Allowed / NeedsApproval / MinApprovals / RiskLevel / PolicyVersion
// 五项必须与真实执行时**完全一致**，因为它们来自同一个 evaluateToolCall。
// 若预演说 allowed=true 而真实执行被拒，预演就成了有害功能。
type ToolDryRunResponse struct {
	// DryRun 恒为 true：让调用方一眼确认「这不是执行结果」，
	// 不必依赖 status 的取值去推断（那种推断只要有一处写错就会酿成误判）。
	DryRun bool `json:"dryRun"`
	// Allowed = Gate 2 与 Gate 2.5 都通过。
	Allowed bool `json:"allowed"`
	// NeedsApproval 为 true 表示当前调用会被受理并等待人工审批，而非立刻执行。
	NeedsApproval bool `json:"needsApproval"`
	// MinApprovals 需要多少名互不相同的人类审批人（0 表示不需要审批）。
	MinApprovals int `json:"minApprovals"`
	// ApprovalState 若此刻真调，会落到哪个审批态（pending/pending_agent/...）。
	// 不需要审批时为空串——用空串而不是造一个 "none" 值，避免与真实落库的
	// approval_state="none"（预演/无审批历史行的值）混淆。
	ApprovalState string `json:"approvalState,omitempty"`
	// RiskLevel 该工具对应权限码的运行期风险分级；unknown=目录不可读。
	// 与 allowed 不同，风险未知不代表被拒（human 路径不查目录）。
	RiskLevel string `json:"riskLevel"`
	// PolicyVersion 三态，语义同 tool_invocations.policy_version：
	// ""=无策略参与判定 / "unknown"=有概念但读不出 / 指纹=实际依据。
	PolicyVersion string `json:"policyVersion"`
	// Reason 判定理由（拒绝原因，或放行时的策略摘要）。预演必须给出理由，
	// 只回一个 false 的布尔值等于让调用方猜自己哪里不满足。
	Reason string `json:"reason"`
	// ToolName 回显，便于编排方在并发预演多个工具时对上号。
	ToolName string `json:"toolName"`
	// AuditID 本次预演落下的审计行 ID（dry_run=true）。
	// 预演也要留痕：管理者事后要能查到「谁在什么时候预演了什么」。
	AuditID int `json:"auditId,omitempty"`

	// --- 以下字段仅在「复核既有调用」模式（按 invocationId 发起）下有意义 ---

	// EvaluatedPrincipal 本次判定实际使用的主体（human/agent/service）。
	//
	// 普通模式恒为请求者自己的主体；复核模式由被复核记录的来源推导
	// （AI 发起的调用按 agent 判定）。必须回显：同一个工具在 human 与 agent
	// 两个主体下的结论可能完全不同（agent 多一道 Gate 2.5），
	// 不说明「按谁判的」，allowed=false 会被读成「这个工具不能用」。
	EvaluatedPrincipal string `json:"evaluatedPrincipal"`
	// InvocationID 被复核的调用记录 ID（普通模式为 0）。
	InvocationID int `json:"invocationId,omitempty"`
	// BaselinePolicyVersion 被复核记录在**当时**记录的策略指纹，供与本次比对。
	// 三态同 tool_invocations.policy_version，此处原样透出不归一化。
	BaselinePolicyVersion string `json:"baselinePolicyVersion,omitempty"`
	// PolicyComparison 两次判定的策略比对结论，取值见下方常量。
	//
	// 刻意由服务端计算而不是交给前端比较字符串：指纹相同 ⟺ 放行判据相同
	// 是本项目的核心不变量，让它在前端各自实现一份比较逻辑，必然出现
	// 「UI 说没变、实际要求变了」的分歧。
	PolicyComparison string `json:"policyComparison,omitempty"`
}

// PolicyComparison 的取值。四态不可合并：
//
//	not_applicable：至少一侧无策略参与（历史上 human 路径的调用），比较无意义；
//	identical     ：两指纹相同 → 放行判据未变，本次复核结论即原判定的结论；
//	changed       ：两指纹不同 → 策略已改，原结论不能直接沿用（审批人须按新策略判断）；
//	unknown       ：至少一侧读不出策略指纹 → 不能断言「没变」，须按「可能已变」处理。
//
// 把 unknown 归入 identical 会让「策略信息缺失」伪装成「策略未变」，
// 这是审计场景里最危险的一类假阴性。
const (
	PolicyComparisonNotApplicable = "not_applicable"
	PolicyComparisonIdentical     = "identical"
	PolicyComparisonChanged       = "changed"
	PolicyComparisonUnknown       = "unknown"
)
