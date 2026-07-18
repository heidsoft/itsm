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
  const url = `${httpClient.getBaseURL()}/api/v1/ai/chat/stream`;
  const token = httpClient.getAuthToken();
  const tenantId = httpClient.getTenantId();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'text/event-stream',
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (tenantId) headers['X-Tenant-ID'] = String(tenantId);

  const response = await fetch(url, {
    method: 'POST',
    headers,
    credentials: 'include',
    signal: req.signal,
    body: JSON.stringify({
      query: req.query,
      limit: req.limit,
      conversationId: req.conversationId,
    }),
  });

  if (!response.ok || !response.body) {
    const message = `AI chat stream failed: HTTP ${response.status}`;
    callbacks.onError?.(message);
    throw new Error(message);
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

  // Standard SSE frames are separated by "\n\n". Buffer until we see one.
  // We support "\r\n\r\n" as well for CRLF servers.
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
  // Flush any trailing partial block (e.g. server closed without final \n\n).
  if (buffer.trim().length > 0) {
    flushBlock(buffer);
  }

  return finalConversationId;
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
