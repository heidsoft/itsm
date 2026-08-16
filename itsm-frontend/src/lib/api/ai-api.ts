import { httpClient } from './http-client';
import { security } from '@/lib/security';

export interface TriageResult {
  category: string;
  priority: string;
  assigneeId?: number;
  confidence: number;
  explanation: string;
  urgency?: string;
}

export interface RagAnswer {
  objectType: string;
  id: number;
  title?: string;
  category?: string;
  snippet: string;
  source?: string;
  score?: number;
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
}

export async function aiTriage(title: string, description: string): Promise<TriageResult> {
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
  }>(`/api/v1/ai/triage`, { title, description });

  const suggestions = res?.suggestions || {};
  return {
    category: suggestions.category || 'general',
    priority: suggestions.priority || 'medium',
    confidence: typeof suggestions.confidence === 'number' ? suggestions.confidence : 0,
    explanation: suggestions.reasoning || '',
    urgency: suggestions.urgency,
    assigneeId: 0,
  };
}

export async function aiSearchKB(query: string, limit = 5): Promise<{ answers: RagAnswer[] }> {
  // 后端契约：POST /api/v1/ai/rag/search 返回 { results, degraded }（见 handlers/ai KnowledgeSearch）
  const res = await httpClient.post<{ results: RagAnswer[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query,
      limit,
      type: 'kb',
    }
  );
  return { answers: Array.isArray(res?.results) ? res.results : [] };
}

export async function aiSimilarIncidents(
  query: string,
  limit = 5
): Promise<{ incidents: RagAnswer[] }> {
  const res = await httpClient.post<{ results: RagAnswer[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query,
      limit,
      type: 'incident',
    }
  );
  return { incidents: Array.isArray(res?.results) ? res.results : [] };
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

export interface AIScenarioEval {
  kind: string;
  count: number;
  usefulRate: number;
  acceptedRate: number;
  avgConfidence: number;
}

export interface AIEvaluationReport {
  generatedAt: string;
  lookbackDays: number;
  totalFeedback: number;
  usefulRate: number;
  acceptedRate: number;
  avgConfidence: number;
  healthScore: number;
  hasData: boolean;
  byScenario: AIScenarioEval[];
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
  accepted: boolean;
  suggestion: Record<string, unknown> | null;
  notes: string;
}

export interface AIAuditLogsResponse {
  items: AIAuditEntry[];
  total: number;
  page: number;
  pageSize: number;
}

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
  // 字面量路径 + 拼接，避免模板内三元表达式，保证 api-contract 测试可静态解析路径
  const url = '/api/v1/ai/audit-logs' + (qs ? `?${qs}` : '');
  return httpClient.get<AIAuditLogsResponse>(url);
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
  const res = await httpClient.post<{ results: any[]; degraded?: boolean }>(
    `/api/v1/ai/rag/search`,
    {
      query: request.query,
      limit,
      type: 'kb',
    }
  );

  const list = Array.isArray(res?.results) ? res.results : [];
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
  const res = await httpClient.get<{ results: Array<any>; total: number }>(
    `/api/v1/global-search?keyword=${encodeURIComponent(query)}`
  );
  const results = Array.isArray(res?.results) ? res.results : [];

  const normalize = (r: any): SearchResult => ({
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

// ==================== 兼容类包装器 ====================

export class AIApi {
  static async triage(title: string, description: string): Promise<TriageResult> {
    return aiTriage(title, description);
  }

  static async chat(params: {
    query: string;
    conversationId?: number;
    limit?: number;
  }): Promise<any> {
    return httpClient.post(`/api/v1/ai/chat`, {
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

  static async saveFeedback(feedback: AIFeedbackRequest): Promise<{ message: string }> {
    return aiSaveFeedback(feedback);
  }

  static async getMetrics(days = 7): Promise<AIMetrics> {
    return aiGetMetrics(days);
  }

  static chatStream(
    req: AIChatStreamRequest,
    callbacks: AIChatStreamCallbacks = {}
  ): Promise<number> {
    return aiChatStream(req, callbacks);
  }
}
