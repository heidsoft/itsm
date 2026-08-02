import { httpClient } from './http-client';

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
  const answers = await httpClient.post<RagAnswer[]>(`/api/v1/ai/knowledge/search`, {
    query,
    limit,
    type: 'kb',
  });
  return { answers: Array.isArray(answers) ? answers : [] };
}

export async function aiSimilarIncidents(
  query: string,
  limit = 5
): Promise<{ incidents: RagAnswer[] }> {
  const incidents = await httpClient.post<RagAnswer[]>(`/api/v1/ai/knowledge/search`, {
    query,
    limit,
    type: 'incident',
  });
  return { incidents: Array.isArray(incidents) ? incidents : [] };
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

  // 读取 CSRF token（可能不存在于某些环境，缺失时不会阻塞请求）。
  let csrfToken: string | null = null;
  try {
    // httpClient 在内部从 cookie 提取，下面这段只用于为 fetch 头补充
    // 与 httpClient 自身一致：避免再次调用 security 抽象以免循环依赖。
    csrfToken =
      typeof document !== 'undefined'
        ? (document.cookie
            .split(';')
            .map(c => c.trim())
            .find(c => c.startsWith('csrf_token=') || c.startsWith('XSRF-TOKEN='))
            ?.split('=')[1] ?? null)
        : null;
  } catch {
    csrfToken = null;
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'text/event-stream',
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (tenantId) headers['X-Tenant-ID'] = String(tenantId);
  if (csrfToken) headers['X-CSRF-Token'] = decodeURIComponent(csrfToken);

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
      const response = await fetch(candidate.url, {
        method: 'POST',
        headers,
        credentials: candidate.useCredentials ? 'include' : 'omit',
        signal: controller.signal,
        body,
      });

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
