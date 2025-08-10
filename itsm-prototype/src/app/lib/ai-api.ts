import { httpClient } from './http-client';

export interface TriageResult {
  category: string;
  priority: string;
  assignee_id: number;
  confidence: number;
  explanation: string;
}

export interface RagAnswer {
  object_type: string;
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
  item_type?: string;
  item_id?: number;
  useful: boolean;
  score?: number;
  notes?: string;
}

export interface AIMetrics {
  total_requests: number;
  total_feedback: number;
  useful_feedback: number;
  useful_rate: number;
  by_kind: Record<string, number>;
  avg_response_time_seconds: number;
}

export async function aiTriage(title: string, description: string): Promise<TriageResult> {
  return httpClient.post<TriageResult>(`/api/v1/ai/triage`, { title, description });
}

export async function aiSearchKB(query: string, limit = 5): Promise<{ answers: RagAnswer[] }> {
  return httpClient.post<{ answers: RagAnswer[] }>(`/api/v1/ai/search`, { query, limit, type: 'kb' });
}

export async function aiSimilarIncidents(query: string, limit = 5): Promise<{ incidents: RagAnswer[] }> {
  return httpClient.post<{ incidents: RagAnswer[] }>(`/api/v1/ai/similar-incidents`, { query, limit });
}

export async function aiSummarize(text: string, maxLen = 200): Promise<{ summary: string }> {
  return httpClient.post<{ summary: string }>(`/api/v1/ai/summarize`, { text, max_len: maxLen });
}

export async function aiSaveFeedback(feedback: AIFeedbackRequest): Promise<{ message: string }> {
  return httpClient.post<{ message: string }>(`/api/v1/ai/feedback`, feedback);
}

export async function aiGetMetrics(days = 7): Promise<AIMetrics> {
  return httpClient.get<AIMetrics>(`/api/v1/ai/metrics?days=${days}`);
}


