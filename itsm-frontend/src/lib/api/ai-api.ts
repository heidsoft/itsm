import { httpClient } from './http-client';
import type { PaginationResponse } from './types';
import { security } from '@/lib/security';

export interface TriageResult {
  category: string;
  priority: string;
  assigneeId?: number;
  confidence: number;
  explanation: string;
  urgency?: string;
  /** 服务端决策底账行 ID：有值时采纳/拒绝直接 PATCH /ai/audit-logs/:id 翻转 */
  auditId?: number;
  /** prompt/口径版本（triage-v1） */
  promptVersion?: string;
  /** 决策链路：guidance | llm | keyword */
  model?: string;
}

/** 工单调查编排单步溯源（对齐后端 handlers/ai.InvestigationStep） */
export interface InvestigationStep {
  /** triage | knowledge | cmdb | synthesis */
  step: string;
  tool?: string;
  degraded?: boolean;
  degradedReason?: string;
  hitCount?: number;
  detail?: unknown;
}

/** 调查结论里的后续动作建议（只是建议；执行必须人工走各自表单/审批） */
export interface InvestigationSuggestedAction {
  /** 白名单内工具名：create_ticket | update_ticket | link_ticket_ci | create_ci_relationship | delete_ci_relationship */
  action: string;
  description: string;
  tool?: string;
  requiresApproval: boolean;
}

/** 工单调查编排响应（对齐后端 POST /api/v1/ai/tickets/:id/investigate） */
export interface TicketInvestigationResponse {
  ticketId: number;
  ticketNumber?: string;
  /** completed | degraded */
  status: string;
  degraded: boolean;
  degradedReason?: string;
  /** prompt/口径版本（investigation-v1） */
  promptVersion: string;
  model?: string;
  confidence: number;
  latencyMs?: number;
  steps: InvestigationStep[];
  findings?: string[];
  suspectedRootCause?: string;
  impactedServices?: string[];
  suggestedActions?: InvestigationSuggestedAction[];
  /** 服务端决策底账行 ID：有值时有用/不准反馈直接 PATCH /ai/audit-logs/:id 翻转 */
  auditId?: number;
  /**
   * 本次调查编排的运行轮次标识。拿到它即可用 aiGetRunInvocations(runId)
   * 取回这一轮的全部工具调用证据链，无需自行从零散记录里拼接。
   */
  runId?: string;
}

export interface RagAnswer {
  objectType: string;
  id: number;
  title?: string;
  category?: string;
  snippet: string;
  source?: string;
  score?: number;
  // 可信 RAG 可观测字段（后端 L0/L1/L2 已生效，前端透出标签）
  authorityLevel?: number; // 0 普通 / 10 部门推荐 / 20 官方标准 / 30 唯一真相源
  validFrom?: string; // 生效时间 RFC3339，空=立即生效
  validUntil?: string; // 失效时间 RFC3339，空=长期有效
  lastReviewedAt?: string; // 最近复核时间 RFC3339
  reviewIntervalDays?: number; // 复核周期（天）
  isRestricted?: boolean; // 所属分类是否处于受限可见性（分类级守卫 L0）
}

export interface AIFeedbackRequest {
  kind: string;
  query?: string;
  itemType?: string;
  itemId?: number;
  useful: boolean;
  score?: number;
  notes?: string;
}

export interface AIMetrics {
  totalRequests: number;
  totalFeedback: number;
  usefulFeedback: number;
  usefulRate: number;
  byKind: Record<string, number>;
  avgResponseTimeSeconds: number;
  llmCallCount?: number;
  /** 本租户窗口内 LLM 调用的估算 token 总消耗（租户口径，非平台） */
  totalTokens: number;
  responseTimeAvailable?: boolean;
}

export interface AIAuditEntry {
  id: number;
  tenantId: number;
  scenario: string;
  prompt: string;
  model: string;
  latencyMs: number;
  totalTokens: number;
  totalCost: number;
  score: number;
  feedback: string;
  createdAt: string;
}

export interface ConversationSummary {
  id: number;
  title: string;
  userId: number;
  tenantId: number;
  createdAt: string;
}

export interface AIMessage {
  id: number;
  conversationId: number;
  role: string;
  content: string;
  requestId?: string;
  createdAt: string;
}

/**
 * AI 分诊。
 *
 * ticketId 可选：带上后后端按工单幂等——同一工单只分析一次，重复进入详情页
 * 复用既有结论，不会重复调模型、也不会让分类在多次之间漂移（prod 实测 F2）。
 */
export async function aiTriage(
  title: string,
  description: string,
  ticketId?: number
): Promise<TriageResult> {
  const res = await httpClient.post<{
    title: string;
    description: string;
    auditId?: number;
    promptVersion?: string;
    model?: string;
    /** true 表示命中后端缓存的既有结论（本次没有重新调用模型）。 */
    cached?: boolean;
    suggestions?: {
      category?: string;
      priority?: string;
      confidence?: number;
      reasoning?: string;
      urgency?: string;
    };
  }>(`/api/v1/ai/triage`, ticketId && ticketId > 0 ? { title, description, ticketId } : { title, description });

  const suggestions = res?.suggestions || {};
  return {
    category: suggestions.category || 'general',
    priority: suggestions.priority || 'medium',
    confidence: typeof suggestions.confidence === 'number' ? suggestions.confidence : 0,
    explanation: suggestions.reasoning || '',
    urgency: suggestions.urgency,
    assigneeId: 0,
    auditId: res?.auditId,
    promptVersion: res?.promptVersion,
    model: res?.model,
  };
}

export interface AIAuditRequest {
  scenario: string;
  inputRef: string;
  promptVersion?: string;
  model?: string;
  confidence?: number;
  suggestion: Record<string, unknown>;
  accepted: boolean;
  notes?: string;
}

/**
 * 上报 AI 建议采纳/拒绝审计（GA AI trace 契约）。
 * 落库为 ai_feedbacks 中 item_type='ai_audit' 的记录，由 /ai/audit-logs 读取；
 * 不调用本端点，审计页就没有任何采纳/拒绝数据。
 *
 * 调用约定：fire-and-forget——审计上报失败不得阻塞主交互，
 * 由调用方 catch 后仅记 console（遥测是 best-effort 副作用）。
 */
export async function recordAIAudit(req: AIAuditRequest): Promise<void> {
  await httpClient.post<{ message: string }>(`/api/v1/ai/audit`, req);
}

export async function aiSearchKB(query: string, limit = 5): Promise<{ answers: RagAnswer[] }> {
  // 后端契约：POST /api/v1/ai/rag/search 返回 { items, degraded }（见 handlers/ai KnowledgeSearch）
  const res = await httpClient.post<{ items: RagAnswer[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query,
      limit,
      type: 'kb',
    }
  );
  return { answers: Array.isArray(res?.items) ? res.items : [] };
}

export async function aiSimilarIncidents(
  query: string,
  limit = 5
): Promise<{ incidents: RagAnswer[] }> {
  const res = await httpClient.post<{ items: RagAnswer[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query,
      limit,
      type: 'incident',
    }
  );
  return { incidents: Array.isArray(res?.items) ? res.items : [] };
}

export async function aiSummarize(text: string, maxLen = 200): Promise<{ summary: string }> {
  const res = await httpClient.post<{ answers: unknown[] }>(`/api/v1/ai/chat`, {
    query: `请在${maxLen}字以内总结以下内容：\n\n${text}`,
    limit: 1,
  });
  const answers = Array.isArray(res?.answers) ? res.answers : [];
  const summary = answers
    .map(a => (typeof a === 'string' ? a : JSON.stringify(a)))
    .join('\n')
    .trim();
  return { summary };
}

export async function aiSaveFeedback(feedback: AIFeedbackRequest): Promise<{ message: string }> {
  return httpClient.post<{ message: string }>(`/api/v1/ai/feedback`, feedback);
}

export async function aiGetMetrics(days = 7): Promise<AIMetrics> {
  return httpClient.get<AIMetrics>(`/api/v1/ai/metrics?days=${days}`);
}

// ==================== AI 评估与审计（AI-Native：可观测、可回测） ====================
// 后端契约：GET /api/v1/ai/evaluation、GET /api/v1/ai/audit-logs（见 handlers/ai GetEvaluation/GetAuditLogs）

export interface AICalibrationBucket {
  bucket: string;
  midpoint: number;
  count: number;
  usefulRate: number;
  calibrationError: number;
}

interface AIDecisionMetrics {
  acceptedCount: number;
  rejectedCount: number;
  pendingCount: number;
  acceptedRate: number | null;
}

export interface AIScenarioEval extends AIDecisionMetrics {
  kind: string;
  count: number;
  usefulRate: number;
  avgConfidence: number;
}

export interface AISkillEval extends AIDecisionMetrics {
  skillCode: string;
  skillName?: string;
  count: number;
  usefulRate: number;
  avgConfidence: number;
  healthScore: number | null;
  isPilot: boolean;
}

export interface AIEvaluationReport extends AIDecisionMetrics {
  generatedAt: string;
  lookbackDays: number;
  totalFeedback: number;
  usefulRate: number;
  avgConfidence: number;
  healthScore: number | null;
  minHealthDecisions: number;
  hasData: boolean;
  byScenario: AIScenarioEval[];
  bySkill: AISkillEval[];
  confidenceCalibration: AICalibrationBucket[];
  platform: {
    llmCallCount: number;
    successRate: number;
    avgLatencyMs: number;
  };
}

export interface AIAuditEntry {
  id: number;
  createdAt: string;
  tenantId: number;
  userId: number;
  requestId: string;
  scenario: string;
  inputRef: string;
  promptVersion: string;
  model: string;
  confidence: number;
  accepted: boolean | null;
  suggestion: Record<string, unknown> | null;
  notes: string;
}

// 后端 GET /api/v1/ai/audit-logs 走 common.SuccessWithPagination，
// 信封只有 {items,total,page,pageSize,totalPages} 这一套形状。
export type AIAuditLogsResponse = PaginationResponse<AIAuditEntry>;

export async function aiGetEvaluation(days = 30): Promise<AIEvaluationReport> {
  return httpClient.get<AIEvaluationReport>(`/api/v1/ai/evaluation?days=${days}`);
}

export async function aiGetAuditLogs(params: {
  page?: number;
  pageSize?: number;
  kind?: string;
  days?: number;
} = {}): Promise<AIAuditLogsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', String(params.page));
  if (params.pageSize) query.set('pageSize', String(params.pageSize));
  if (params.kind) query.set('kind', params.kind);
  if (params.days) query.set('days', String(params.days));
  const qs = query.toString();
  // 字面量路径 + 拼接，避免模板内三元表达式，保证 api-contract 测试可静态解析路径。
  // 变量名必须全文件唯一：契约测试的 staticBases 是文件级 name→path 映射，
  // 同名的 `const url` 会后写覆盖前写，让先定义的那个端点永远拿不到真实校验
  // （此前 ai-api.ts 里两个 `url` 已让 /ai/audit-logs 长期处于这种假绿状态）。
  const auditLogsUrl = '/api/v1/ai/audit-logs' + (qs ? `?${qs}` : '');
  return httpClient.get<AIAuditLogsResponse>(auditLogsUrl);
}

// Audit Console 的 Accept/Revoke：翻转单条 ai_feedbacks 行的 accepted 列，
// 仅 item_type='ai_audit' 的行可改。
export async function aiUpdateAuditAccepted(id: number, accepted: boolean): Promise<{ id: number; accepted: boolean }> {
  return httpClient.patch<{ id: number; accepted: boolean }>(
    `/api/v1/ai/audit-logs/${id}`,
    { accepted },
  );
}

// ==================== 工单调查编排（对应后端 POST /api/v1/ai/tickets/:id/investigate） ====================

/**
 * 触发工单调查编排：分诊 → 知识库检索 → CMDB 影响分析 → LLM 综合。
 * 只读决策支持：suggestedActions 只是建议，写操作由操作员走各自表单与审批流，
 * 本端点不执行任何写操作。每次调用全量重跑并落一条 ai_audit 底账（响应 auditId）。
 */
export async function aiInvestigateTicket(ticketId: number): Promise<TicketInvestigationResponse> {
  return httpClient.post<TicketInvestigationResponse>(
    `/api/v1/ai/tickets/${ticketId}/investigate`,
    {},
  );
}

// ==================== AI 趋势预测（对应后端 POST /api/v1/ai/predictions） ====================

/** 后端 DTO：itsm-backend/dto/ticket_prediction_dto.go */
export type TrendPredictionType = 'volume' | 'type' | 'priority' | 'resource';

export interface TrendPredictionRequest {
  timeRange: [string, string]; // YYYY-MM-DD ×2
  predictionType: TrendPredictionType;
  filters?: Record<string, unknown>;
  model?: string;
}

export interface PredictionDataPoint {
  date: string;
  predictedValue: number;
  lowerBound: number;
  upperBound: number;
  confidence: number;
  category?: string;
  priority?: string;
}

export interface TrendPredictionResponse {
  predictions: PredictionDataPoint[];
  confidence: number;
  model: string;
  generatedAt: string;
}

export async function aiPredict(req: TrendPredictionRequest): Promise<TrendPredictionResponse> {
  return httpClient.post<TrendPredictionResponse>('/api/v1/ai/predictions', req);
}

// ==================== AI 工具调用审批队列（P0：打通人工审批闭环） ====================

export interface ToolApproval {
  id: number;
  toolName: string;
  arguments: string; // JSON 字符串
  status: string;
  needsApproval: boolean;
  approvalState: string; // pending | approved | rejected | auto | pending_agent | pending_agent_two | awaiting_second
  approvalReason?: string;
  permissionCheck?: string;
  permissionReason?: string;
  createdAt: string;
  conversationId: number;
  userId: number;
  /** 该工具对应权限码的运行期风险分级（后端从权限目录实时解析，unknown=目录不可读） */
  riskLevel?: string;
  /** 需要多少个互不相同的人类审批人（来自权限码策略） */
  minApprovals?: number;
  /**
   * 放行判据的策略指纹。三态**不做前端归一化**：
   * 空串 = 该概念在记录产出时还不存在（历史行）；
   * `unknown` = 概念存在但本次读不出策略（需运维介入）；
   * 其余 = 实际放行依据的内容哈希（前 16 字节 hex）。
   * 三者如上不可合并——合并后「有多少次调用拿不到策略版本」这个指标就失效。
   */
  policyVersion?: string;
  /** 逻辑运行轮次（跨请求串联同一轮 Agent 行为）。空串=未归属任何轮次。 */
  runId?: string;
}

/** 运行轮次证据链响应（对齐后端 GET /api/v1/agent/runs/:runId/invocations） */
export interface ToolRunInvocationsResponse {
  items: ToolApproval[];
  runId: string;
  count: number;
}

/**
 * 工具调用预演结论（对齐后端 POST /api/v1/agent/tools/dry-run）。
 *
 * 字段与真实执行判定**同源**（后端只有一个判定函数），因此这里看到的
 * allowed / needsApproval / minApprovals 就是真调时会得到的答案。
 * 唯一的差别是副作用：预演不执行工具、不建待审批行。
 */
export interface ToolDryRunResult {
  /** 恒为 true，用于一眼确认「这不是执行结果」。 */
  dryRun: boolean;
  toolName: string;
  /** Gate 2 与 Gate 2.5 是否都通过。 */
  allowed: boolean;
  /** 若此刻真调，是被受理等待审批（true）还是立刻执行（false）。 */
  needsApproval: boolean;
  /** 需要几名互不相同的人类审批人（0 = 不需要审批）。 */
  minApprovals: number;
  /** 若真调会落到哪个审批态（pending/pending_agent/...），无需审批时为空。 */
  approvalState?: string;
  /** 该工具权限码的运行期风险分级；unknown = 目录不可读。 */
  riskLevel?: string;
  /** 策略版本三态：空串=无策略参与 / unknown=读不出 / 指纹=实际依据。 */
  policyVersion?: string;
  /** 判定理由（拒绝原因或放行时的策略摘要）。 */
  reason?: string;
  /** 本次预演落下的审计行 ID（dry_run=true）。 */
  auditId?: number;

  // --- 以下字段仅「复核既有调用」模式返回 ---

  /**
   * 本次判定实际使用的主体（human / agent / service）。
   *
   * 必须展示在界面上：同一个工具在 human 与 agent 两个主体下结论可能完全不同
   * （agent 多一道 Gate 2.5）。不说明「按谁判的」，`allowed=false` 会被
   * 审批人读成「这个工具本身不能用」。
   */
  evaluatedPrincipal?: string;
  /** 被复核的调用记录 ID（普通预演为 0）。 */
  invocationId?: number;
  /** 被复核记录**当时**落库的策略指纹（三态，不归一化）。 */
  baselinePolicyVersion?: string;
  /**
   * 两次判定的策略比对结论，由后端计算（四态）：
   * - `not_applicable`：至少一侧无策略参与，比较无意义；
   * - `identical`：判据未变，本次复核结论即原判定结论；
   * - `changed`：策略已改，原结论不能直接沿用；
   * - `unknown`：至少一侧读不出指纹，**不能**断言「未变」。
   */
  policyComparison?: 'not_applicable' | 'identical' | 'changed' | 'unknown';
}

export interface ToolApprovalListResponse {
  items: ToolApproval[];
  state: string;
}

// ==================== 控制面聚合读面 ====================
// 对齐后端 GET /api/v1/agent/control-plane。
// 字段名与后端 ControlPlaneOverview 逐一对齐，不做前端改名——改名会让
// 「后端加了字段但前端没消费」这种漂移无法被 grep 发现。

/**
 * 策略配置面的三类计数。**三类不可合并**，前端必须分别展示：
 *
 * - `withPolicy`  —— 管理员显式配置了合法策略信封，鉴权时会被真正读取；
 * - `policyFree`  —— 没有策略信封（空描述或纯文本旧描述），鉴权走
 *                    DerivePolicy 推导（AI 主体默认零权限）；
 * - `unreadable`  —— 信封存在但解不出（坏 JSON / 未知版本），鉴权时
 *                    fail-closed 一律拒绝。
 *
 * 把 `unreadable` 并进 `withPolicy` 会让运维以为已生效；并进 `policyFree`
 * 会让运维以为只是没配。两者都让**最危险的一类彻底隐身**。
 */
export interface AgentControlPlanePolicySummary {
  total: number;
  withPolicy: number;
  policyFree: number;
  unreadable: number;
}

/**
 * 待批队列四种状态分别计数。**不合并**，因为它们的处理人不同：
 * - `pending`          人发起，任一人审批即可；
 * - `pendingAgent`     AI 发起，单人审批；
 * - `pendingAgentTwo`  AI 发起，双人审批，尚无人投第一票；
 * - `awaitingSecond`   已有一票，等第二个不同的人复审（**首票不放行**）。
 *
 * 合并后两者会误导运维：awaitingSecond 那批已经批了一半、再不处理就卡住了。
 */
export interface AgentControlPlaneQueueSummary {
  pending: number;
  pendingAgent: number;
  pendingAgentTwo: number;
  awaitingSecond: number;
  total: number;
}

/** 最近一次被拒绝的判定（只读投影，不含预演行）。 */
export interface AgentControlPlaneRecentDenial {
  id: number;
  toolName: string;
  userId: number;
  roleSnapshot: string;
  permissionCheck: string;
  reason: string;
  /** 策略版本三态：空串=无策略参与 / unknown=读不出 / 指纹=实际依据。 */
  policyVersion: string;
  runId: string;
  createdAt: string;
}

/** 单个工具的可见性（与 GET /agent/tools 同源）。 */
export interface AgentControlPlaneToolSurface {
  name: string;
  resource: string;
  action: string;
  readOnly: boolean;
  /** 该角色此刻能否看到/调用此工具。false 也要展示——运维需要看到「有哪些工具被挡住」。 */
  visible: boolean;
  /** 风险分级；`unknown` = 权限目录不可读，不是「无风险」。 */
  riskLevel: string;
}

/** 控制面聚合快照。 */
export interface AgentControlPlaneOverview {
  tenantId: number;
  /**
   * 本次请求的调用者角色。**必须展示**：同一租户同一时刻，
   * 管理员看到 14 个工具可见、普通技术员看到 6 个——不说明「按谁算的」，
   * 界面上的「可见工具数」就没有确定含义。
   */
  role: string;
  toolSurface: AgentControlPlaneToolSurface[];
  visibleTools: number;
  totalTools: number;
  queue: AgentControlPlaneQueueSummary;
  policies: AgentControlPlanePolicySummary;
  recentDenial: AgentControlPlaneRecentDenial[];
  /** 采集时刻（RFC3339）。多真源快照不标时间就无法解释「刷新后数字怎么变了」。 */
  gatheredAt: string;
}

export interface ToolApproveRequest {
  approve: boolean;
  reason?: string;
}

// 列出 AI 工具调用审批记录（默认待审批）
// 对应后端 GET /api/v1/agent/tools/invocations?state=pending
export async function aiGetToolApprovals(state = 'pending'): Promise<ToolApprovalListResponse> {
  // 字面量路径 + 拼接，避免模板内三元表达式，保证 api-contract 测试可静态解析路径
  const approvalsUrl = '/api/v1/agent/tools/invocations?state=' + encodeURIComponent(state);
  return httpClient.get<ToolApprovalListResponse>(approvalsUrl);
}

/**
 * 取回某个运行轮次的完整证据链：这一轮 Agent 依次调了哪些工具、每步依据哪版策略放行。
 * 对应后端 GET /api/v1/agent/runs/:runId/invocations
 *
 * 与审批列表的区别：审批列表按「状态」聚合待办，回答「我该批哪几条」；
 * 本接口按「轮次」聚合行为，回答「这条调用背后整轮做了什么」。
 * 审批人只看到孤立一条调用时无法判断该不该批——上下文正是这一轮的前序动作。
 *
 * runId 为空时不发请求：后端对该值返回 400，前端静默请求只会产生无意义错误日志。
 */
export async function aiGetRunInvocations(runId: string): Promise<ToolRunInvocationsResponse> {
  const trimmed = (runId ?? '').trim();
  if (!trimmed) {
    // 与后端「空 runId → 400」同语义的本地短路，返回同形状的空结果。
    return { items: [], runId: '', count: 0 };
  }
  // 路径直接写在调用点上（含 `${trimmed}` 动态段），不先拼进 const：
  // 契约测试按文件名级 name→path 静态映射解析，只取 `const x = '字面量'`，
  // 把动态段藏进 const 会让解析器只看到前缀 `/api/v1/agent/runs/`，
  // 于是这条真实路由被判成漂移——这是「写法迁就工具」，不能反过来。
  return httpClient.get<ToolRunInvocationsResponse>(
    `/api/v1/agent/runs/${encodeURIComponent(trimmed)}/invocations`
  );
}

// 通过 / 驳回某条工具调用审批
// 对应后端 POST /api/v1/agent/tools/:id/approve
export async function aiApproveTool(
  id: number,
  req: ToolApproveRequest
): Promise<{ invocationId: number; approvalState: string }> {
  return httpClient.post(`/api/v1/agent/tools/${id}/approve`, req);
}

/**
 * 预演：现在调用这个工具会发生什么。对应后端 POST /api/v1/agent/tools/dry-run
 *
 * 后端只跑判定链（Gate 2 角色 RBAC + Gate 2.5 主体策略 + 审批人数），
 * **不执行工具、不创建待审批行**，只落一条 dry_run=true 的审计。
 * 因此它不会在审批队列里留下任何东西，可以放心在展示层调用。
 *
 * 为什么审批页需要它：审批人看到的每一条待办都缺一个关键信息——
 * 「我点通过，实际会发生什么、还差几个人点头」。这条接口回答的正是后者，
 * 且答案与真实执行同源（后端 evaluateToolCall 单一判定），不会两边不一致。
 *
 * 被拒不是接口错误：后端对「判定为不允许」返回 200 + allowed=false，
 * 所以调用方不需要 try/catch 区分，按 allowed 分支渲染即可。
 */
export async function aiDryRunTool(
  name: string,
  args: Record<string, unknown> = {}
): Promise<ToolDryRunResult> {
  // 路径写成字面量、不拼进 const：契约测试按文件名级 name→path 静态映射解析，
  // 藏进变量会让这条真实路由被判成漂移（同一个坑在 aiGetRunInvocations 上也踩过）。
  return httpClient.post<ToolDryRunResult>('/api/v1/agent/tools/dry-run', {
    name,
    args,
  });
}

/**
 * 复核一条既有调用：按它**当时的发起主体**与**当前策略**重新判定。
 * 对应后端 POST /api/v1/agent/tools/invocations/:id/review
 *
 * 与 aiDryRunTool 的区别不是参数不同，而是回答的问题不同：
 * - aiDryRunTool            → 「我现在调用这个工具会被允许吗」（按调用者自己判定）
 * - aiReviewToolInvocation  → 「队列里这条调用，现在还该放行吗」（按原发起主体判定）
 *
 * 审批人需要的是后者。用前者做审批前的核对有一个隐蔽且致命的缺陷：
 * 审批人通常是管理员，他当然持有该工具的权限，于是预演永远 `allowed=true`
 * ——这条结论回答的是「管理员能不能做」，而审批要判断的是「AI 能不能做」。
 *
 * 零副作用：不执行工具、不推进审批状态；审批动作仍在 aiApproveTool（ai:approve）。
 */
export async function aiReviewToolInvocation(invocationId: number): Promise<ToolDryRunResult> {
  // 动态段直接写在模板串里（与 aiGetRunInvocations 同一纪律）：
  // 把 id 拼进 const 再传会让契约测试只解析到前缀，误判为路由漂移。
  return httpClient.post<ToolDryRunResult>(
    `/api/v1/agent/tools/invocations/${encodeURIComponent(String(invocationId))}/review`,
    {}
  );
}

/**
 * 控制面聚合快照：一次请求拿到「AI 工具控制面」当前的四个事实面。
 * 对应后端 GET /api/v1/agent/control-plane（ai:read）
 *
 * 为什么需要聚合而不是让前端并发调四个接口：
 * - **口径一致性**：待批队列、策略覆盖、工具可见面必须来自同一时刻的同一批谓词。
 *   四次独立请求之间各表仍在变，拼出来的「可见工具 14 / 待批 3 / 策略 20」并不对应
 *   任何真实时刻，运维无法据此判断「改了策略到底生效没有」。
 * - **fail-closed 语义**：后端任一真源读失败就整单报错（500），前端只需要处理
 *   两种结果——拿到完整快照，或者明确失败。部分数据比没有数据更危险：
 *   「待批 0 条」会让运维以为队列清空了，而实际是查询失败。
 *
 * 本接口**不做任何判定**：可见性来自与 GET /agent/tools 同一个权限函数，
 * 队列来自与审批页同源的仓储查询。它只回答「现在是什么样」，
 * 「允许不允许」走 aiDryRunTool / aiReviewToolInvocation。
 */
export async function aiGetControlPlane(): Promise<AgentControlPlaneOverview> {
  // 路径写成字面量、不拼进 const：契约测试按文件名级 name→path 静态映射解析。
  return httpClient.get<AgentControlPlaneOverview>('/api/v1/agent/control-plane');
}

// ==================== 合并自 legacy AIService（src/lib/services/ai-service.ts） ====================
// 语义与旧实现保持一致：aiClassifyTicket → /ai/triage；aiSuggestSolutions → /ai/rag/search(kb)；
// aiIntelligentSearch → /global-search。

export interface TicketAnalysisRequest {
  title: string;
  description: string;
  attachments?: string[];
  userContext?: {
    department: string;
    role: string;
    location: string;
  };
}

export interface TicketClassificationResult {
  category: string;
  subcategory?: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  urgency: 'low' | 'medium' | 'high' | 'critical' | string;
  confidence: number;
  reasoning: string;
  suggestions?: string[];
}

export interface SolutionSuggestion {
  solutionId: string;
  title: string;
  description: string;
  steps?: string[];
  estimatedTime?: number; // 预计解决时间（分钟）
  successRate?: number; // 历史成功率
  relatedKnowledge?: string[];
  confidence: number;
  reasoning: string;
}

export interface SolutionSearchRequest {
  query: string;
  category?: string;
  priority?: string;
  context?: string;
  limit?: number;
}

/**
 * 知识库检索返回的原始条目形状（`POST /api/v1/ai/rag/search` 的 items[]）。
 *
 * 后端对同一语义有过三种字段名（content / snippet / text），前端必须全部兼容，
 * 但此前用 `any[]` 表达，读代码的人看不出到底要兼容哪几个。
 */
interface RagSearchRawItem {
  id?: number | string;
  title?: string;
  source?: string;
  snippet?: string;
  content?: string;
  text?: string;
  score?: number;
}

/** 全局搜索返回的原始结果条目（`GET /api/v1/global-search` 的 results[]）。 */
interface GlobalSearchRawResult {
  id?: number | string;
  type?: string;
  title?: string;
  description?: string;
  status?: string | number;
  ticketNumber?: string | number;
  number?: string | number;
}

/**
 * `POST /api/v1/ai/chat` 的响应。
 *
 * 字段名与后端 `handlers/ai/handler.go` 拼的 `gin.H{"answers", "conversationId"}`
 * 一致。answers 的元素形态不固定（RAG 命中时是对象，无命中时是空数组），
 * 所以声明为 unknown[] 而不是某个具体类型——把它写成 `RagAnswer[]` 会让
 * 「字符串条目」这种合法形态在类型层面消失。
 */
export interface AiChatResponse {
  answers?: unknown[];
  conversationId?: number;
}

export interface SearchResult {
  id: number;
  title: string;
  description: string;
  type: string;
  status?: string;
  number?: string;
}

export interface IntelligentSearchFilters {
  type?: 'tickets' | 'knowledge' | 'incidents' | 'all';
  dateRange?: { start: string; end: string };
  category?: string;
}

export async function aiClassifyTicket(
  request: TicketAnalysisRequest
): Promise<TicketClassificationResult> {
  const res = await httpClient.post<{
    title: string;
    description: string;
    suggestions?: {
      category?: string;
      priority?: string;
      confidence?: number;
      reasoning?: string;
      urgency?: string;
    };
  }>(`/api/v1/ai/triage`, {
    title: request.title,
    description: request.description,
  });

  const s = res?.suggestions || {};
  const priority = (s.priority || 'medium') as TicketClassificationResult['priority'];
  const urgency = (s.urgency || priority) as TicketClassificationResult['urgency'];
  return {
    category: s.category || 'general',
    priority,
    urgency,
    confidence: typeof s.confidence === 'number' ? s.confidence : 0,
    reasoning: s.reasoning || '',
    suggestions: [],
  };
}

export async function aiSuggestSolutions(
  request: SolutionSearchRequest
): Promise<SolutionSuggestion[]> {
  const limit = request.limit && request.limit > 0 ? request.limit : 5;
  const res = await httpClient.post<{ items: RagSearchRawItem[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query: request.query,
      limit,
      type: 'kb',
    }
  );

  const list = Array.isArray(res?.items) ? res.items : [];
  return list.map((item, idx) => {
    const title = String(item?.title || item?.source || `知识项 ${idx + 1}`);
    const snippet = String(item?.snippet || item?.content || item?.text || '');
    const score = typeof item?.score === 'number' ? item.score : undefined;
    return {
      solutionId: String(item?.id ?? `${Date.now()}_${idx}`),
      title,
      description: snippet,
      steps: [],
      relatedKnowledge: item?.source ? [String(item.source)] : [],
      confidence: typeof score === 'number' ? score : 0,
      reasoning: '来自知识库检索结果',
    };
  });
}

export async function aiIntelligentSearch(
  query: string,
  _filters?: IntelligentSearchFilters
): Promise<{
  tickets: SearchResult[];
  knowledge: SearchResult[];
  incidents: SearchResult[];
  suggestions: string[];
}> {
  const res = await httpClient.get<{ results: GlobalSearchRawResult[]; total: number }>(
    `/api/v1/global-search?q=${encodeURIComponent(query)}`
  );
  const results = Array.isArray(res?.results) ? res.results : [];

  const normalize = (r: GlobalSearchRawResult): SearchResult => ({
    id: Number(r?.id || 0),
    type: String(r?.type || ''),
    title: String(r?.title || ''),
    description: String(r?.description || ''),
    status: r?.status ? String(r.status) : undefined,
    number: r?.ticketNumber ? String(r.ticketNumber) : r?.number ? String(r.number) : undefined,
  });

  return {
    tickets: results.filter(r => r?.type === 'ticket').map(normalize),
    knowledge: results.filter(r => r?.type === 'knowledge').map(normalize),
    incidents: results.filter(r => r?.type === 'incident').map(normalize),
    suggestions: [],
  };
}

// ==================== SSE Streaming Chat ====================

/** Server-Sent Event payload types emitted by /ai/chat/stream. */
export type AIChatStreamEvent =
  | { type: 'sources'; sources: RagAnswer[] }
  | { type: 'delta'; content: string }
  | { type: 'done'; conversationId: number }
  | { type: 'error'; message: string };

export interface AIChatStreamCallbacks {
  onSources?: (sources: RagAnswer[]) => void;
  onDelta?: (delta: string) => void;
  onDone?: (conversationId: number) => void;
  onError?: (message: string) => void;
}

export interface AIChatStreamRequest {
  query: string;
  conversationId?: number;
  limit?: number;
  signal?: AbortSignal;
}

/**
 * Stream a RAG-backed answer over SSE. Falls back to the plain /ai/chat
 * endpoint if streaming is unsupported by the environment (e.g. legacy
 * browsers without ReadableStream). Returns the final conversationId once the
 * stream finishes.
 */
export async function aiChatStream(
  req: AIChatStreamRequest,
  callbacks: AIChatStreamCallbacks = {}
): Promise<number> {
  const baseUrl = httpClient.getBaseURL();
  const token = httpClient.getAuthToken();
  const tenantId = httpClient.getTenantId();

  // 优先走前端同源代理，避开直连后端的 CORS/CSRF 问题。
  const candidates: Array<{ url: string; useCredentials: boolean; appendCSRF: boolean }> = [
    { url: '/api/v1/ai/chat/stream', useCredentials: true, appendCSRF: true },
  ];
  if (baseUrl && !baseUrl.startsWith('/')) {
    candidates.push({
      url: `${baseUrl}/api/v1/ai/chat/stream`,
      useCredentials: true,
      appendCSRF: true,
    });
  }

  // 获取 CSRF token。后端的 csrf cookie 是 HttpOnly，document.cookie 读不到，
  // 必须与 httpClient 一致走 security 抽象（GET /api/v1/csrf-token）。
  const getCsrfToken = async (): Promise<string | null> => {
    try {
      return await security.csrf.getToken();
    } catch {
      return null;
    }
  };
  // 后端会在每次写请求成功后轮换 CSRF token，缓存值可能已过期；
  // 403 时丢弃缓存重新获取。
  const refreshCsrfToken = async (): Promise<string | null> => {
    security.csrf.clearToken();
    return getCsrfToken();
  };

  const csrfToken = await getCsrfToken();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'text/event-stream',
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (tenantId) headers['X-Tenant-ID'] = String(tenantId);
  if (csrfToken) headers['X-CSRF-Token'] = csrfToken;

  const body = JSON.stringify({
    query: req.query,
    limit: req.limit,
    conversationId: req.conversationId,
  });

  let lastError: Error | null = null;
  for (const candidate of candidates) {
    const controller = new AbortController();
    if (req.signal) {
      if (req.signal.aborted) {
        controller.abort();
      } else {
        req.signal.addEventListener('abort', () => controller.abort(), { once: true });
      }
    }

    try {
      const doFetch = () =>
        fetch(candidate.url, {
          method: 'POST',
          headers,
          credentials: candidate.useCredentials ? 'include' : 'omit',
          signal: controller.signal,
          body,
        });

      let response = await doFetch();

      // CSRF token 可能已被后端轮换导致 403，取新 token 原地重试一次。
      if (response.status === 403) {
        const freshToken = await refreshCsrfToken();
        if (freshToken) {
          headers['X-CSRF-Token'] = freshToken;
          response = await doFetch();
        }
      }

      if (!response.ok || !response.body) {
        // 4xx/5xx 走 fallback 而非抛错
        lastError = new Error(`AI chat stream failed: HTTP ${response.status}`);
        continue;
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let buffer = '';
      let finalConversationId = 0;

      const dispatch = (event: string, dataRaw: string) => {
        let data: unknown;
        try {
          data = JSON.parse(dataRaw);
        } catch {
          return;
        }
        switch (event) {
          case 'sources': {
            const sources = Array.isArray(data) ? (data as RagAnswer[]) : [];
            callbacks.onSources?.(sources);
            break;
          }
          case 'delta': {
            const payload = data as { content?: string };
            if (payload && typeof payload.content === 'string') {
              callbacks.onDelta?.(payload.content);
            }
            break;
          }
          case 'done': {
            const payload = data as { conversationId?: number };
            finalConversationId = payload?.conversationId ?? finalConversationId;
            callbacks.onDone?.(finalConversationId);
            break;
          }
          case 'error': {
            const payload = data as { message?: string };
            callbacks.onError?.(payload?.message ?? 'unknown stream error');
            break;
          }
          default:
            break;
        }
      };

      const flushBlock = (block: string) => {
        let event = 'message';
        const dataLines: string[] = [];
        for (const line of block.split('\n')) {
          if (line.startsWith('event:')) {
            event = line.slice(6).trim();
          } else if (line.startsWith('data:')) {
            dataLines.push(line.slice(5).trim());
          }
        }
        if (dataLines.length > 0) {
          dispatch(event, dataLines.join('\n'));
        }
      };

      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        let boundary = buffer.indexOf('\n\n');
        while (boundary !== -1) {
          const block = buffer.slice(0, boundary);
          buffer = buffer.slice(boundary + 2);
          if (block.trim().length > 0) {
            flushBlock(block);
          }
          boundary = buffer.indexOf('\n\n');
        }
      }
      if (buffer.trim().length > 0) {
        flushBlock(buffer);
      }

      return finalConversationId;
    } catch (err) {
      const aborted = (err as Error)?.name === 'AbortError';
      if (aborted) throw err;
      lastError = err instanceof Error ? err : new Error('stream request failed');
      // 继续尝试下一个 candidate
    }
  }

  // 走到这里表示所有 candidate 都失败，抛给调用方做 fallback
  const message = lastError?.message || 'AI chat stream failed: no candidate succeeded';
  callbacks.onError?.(message);
  throw lastError || new Error(message);
}

/** 列出当前用户的 AI 会话历史 */
export async function listConversations(): Promise<ConversationSummary[]> {
  const res = await httpClient.get<{ items: ConversationSummary[] }>('/api/v1/ai/conversations');
  return Array.isArray(res?.items) ? res.items : [];
}

/** 获取指定会话的所有消息 */
export async function getConversationMessages(conversationId: number): Promise<AIMessage[]> {
  const res = await httpClient.get<{ items: AIMessage[] }>(`/api/v1/ai/conversations/${conversationId}`);
  return Array.isArray(res?.items) ? res.items : [];
}

/** 删除指定会话 */
export async function deleteConversation(conversationId: number): Promise<void> {
  await httpClient.delete(`/api/v1/ai/conversations/${conversationId}`);
}

// ==================== AI 分析结果持久化 ====================

export interface AIAnalysisResultDTO {
  id: number;
  tenantId: number;
  userId: number;
  analysisType: string; // triage | summary | rca | deep_analytics | trend_prediction | incident_impact
  ticketId?: number;
  incidentId?: number;
  ticketNumber?: string;
  ticketTitle?: string;
  requestPrompt: string;
  resultJson: string;
  model?: string;
  latencyMs?: number;
  totalTokens?: number;
  costUsd?: number;
  confidenceScore?: number;
  degraded: boolean;
  createdAt: string;
}

export interface AIAnalysisResultListResponse {
  items: AIAnalysisResultDTO[];
}

/** 列出分析历史（可按 type 过滤） */
export async function listAnalysisResults(params?: {
  type?: string;
  limit?: number;
}): Promise<AIAnalysisResultDTO[]> {
  const res = await httpClient.get<AIAnalysisResultListResponse>('/api/v1/ai/analysis-results', { params });
  return res?.items ?? [];
}

/** 获取单条分析结果 */
export async function getAnalysisResult(id: number): Promise<AIAnalysisResultDTO> {
  return httpClient.get<AIAnalysisResultDTO>(`/api/v1/ai/analysis-results/${id}`);
}

/** 删除分析结果记录 */
export async function deleteAnalysisResult(id: number): Promise<void> {
  await httpClient.delete(`/api/v1/ai/analysis-results/${id}`);
}

// ==================== 兼容类包装器 ====================

export class AIApi {
  static async triage(title: string, description: string): Promise<TriageResult> {
    return aiTriage(title, description);
  }

  static async chat(params: {
    query: string;
    conversationId?: number;
    limit?: number;
  }): Promise<AiChatResponse> {
    return httpClient.post<AiChatResponse>(`/api/v1/ai/chat`, {
      query: params.query,
      limit: params.limit,
      conversationId: params.conversationId,
    });
  }

  static async searchKB(query: string, limit = 5): Promise<{ answers: RagAnswer[] }> {
    return aiSearchKB(query, limit);
  }

  static async similarIncidents(query: string, limit = 5): Promise<{ incidents: RagAnswer[] }> {
    return aiSimilarIncidents(query, limit);
  }

  static async summarize(text: string, maxLen = 200): Promise<{ summary: string }> {
    return aiSummarize(text, maxLen);
  }

  /**
   * 工单/事件 AI 摘要（基于持久化数据，走 LLM Gateway，失败有降级）
   * 对应后端 GET /api/v1/ai/tickets/:id/summary
   */
  static async summarizeTicket(id: number | string): Promise<{
    degraded?: boolean;
    message?: string;
    summary?: string;
  }> {
    return httpClient.get(`/api/v1/ai/tickets/${id}/summary`);
  }

  /**
   * 工单/事件 AI 分析（影响分析、关联推荐等）
   * 对应后端 POST /api/v1/ai/tickets/:id/analyze
   */
  static async analyzeTicket(id: number | string): Promise<unknown> {
    return httpClient.post(`/api/v1/ai/tickets/${id}/analyze`, {});
  }

  /** AI-assisted analysis for an Incident aggregate. */
  static async analyzeIncident(id: number | string): Promise<unknown> {
    return httpClient.post(`/api/v1/ai/incidents/${id}/analyze`, {});
  }

  static async saveFeedback(feedback: AIFeedbackRequest): Promise<{ message: string }> {
    return aiSaveFeedback(feedback);
  }

  static async getMetrics(days = 7): Promise<AIMetrics> {
    return aiGetMetrics(days);
  }

  /** 列出 AI 工具调用审批记录（默认待审批）。对应 GET /api/v1/agent/tools/invocations */
  static async getToolApprovals(state = 'pending'): Promise<ToolApprovalListResponse> {
    return aiGetToolApprovals(state);
  }

  /** 通过 / 驳回工具调用审批。对应 POST /api/v1/agent/tools/:id/approve */
  static async approveTool(id: number, req: ToolApproveRequest): Promise<{ invocationId: number; approvalState: string }> {
    return aiApproveTool(id, req);
  }

  /** 取回某运行轮次的完整证据链。对应 GET /api/v1/agent/runs/:runId/invocations */
  static async getRunInvocations(runId: string): Promise<ToolRunInvocationsResponse> {
    return aiGetRunInvocations(runId);
  }

  /** 预演工具调用的判定结论（不执行、不建待办）。对应 POST /api/v1/agent/tools/dry-run */
  static async dryRunTool(
    name: string,
    args: Record<string, unknown> = {}
  ): Promise<ToolDryRunResult> {
    return aiDryRunTool(name, args);
  }

  static chatStream(
    req: AIChatStreamRequest,
    callbacks: AIChatStreamCallbacks = {}
  ): Promise<number> {
    return aiChatStream(req, callbacks);
  }
}
