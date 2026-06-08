import { loadCredentials, type Credentials } from './credentials.js';
import type {
  LoginRequest,
  LoginResponse,
  User,
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  PaginationParams,
  Incident,
  IncidentListResponse,
  Change,
  ChangeListResponse,
  CI,
  CIListResponse,
  KnowledgeArticle,
  KnowledgeListResponse,
  Notification,
  NotificationListResponse,
  ConnectorManifest,
  ProcessInstance,
  ProcessInstanceListResponse,
  ApprovalTask,
  ApprovalTaskListResponse,
} from './types.js';

const DEFAULT_BASE_URL = 'http://localhost:8090';

async function request<T>(
  baseURL: string,
  method: string,
  endpoint: string,
  data?: unknown,
  params?: Record<string, string | number | boolean | undefined>
): Promise<T> {
  const cred = loadCredentials();
  const url = new URL(`${baseURL}/api/v1${endpoint}`);
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined) url.searchParams.set(k, String(v));
    });
  }

  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (cred?.token) headers['Authorization'] = `Bearer ${cred.token}`;

  const res = await fetch(url.toString(), {
    method,
    headers,
    body: data ? JSON.stringify(data) : undefined,
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  }

  const json = await res.json() as { code: number; message: string; data: T };
  if (json.code !== 0) {
    throw new Error(json.message || 'API error');
  }
  return json.data;
}

export class ApiClient {
  constructor(private baseURL: string = DEFAULT_BASE_URL) {}

  // ---------- Auth ----------
  async login(loginData: LoginRequest): Promise<LoginResponse> {
    const url = `${this.baseURL}/api/v1/auth/login`;
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginData),
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = await res.json() as { code: number; message: string; data: LoginResponse };
    if (json.code !== 0) throw new Error(json.message || 'Login failed');
    return json.data;
  }
  async logout(): Promise<void> { await request(this.baseURL, 'POST', '/auth/logout'); }
  async getUserInfo(): Promise<User> { return request<User>(this.baseURL, 'GET', '/auth/userinfo'); }

  // ---------- Tickets ----------
  async listTickets(params?: PaginationParams): Promise<TicketListResponse> {
    return request<TicketListResponse>(this.baseURL, 'GET', '/tickets', undefined, params as Record<string, string | number | undefined>);
  }
  async getTicket(id: number): Promise<Ticket> {
    return request<Ticket>(this.baseURL, 'GET', `/tickets/${id}`);
  }
  async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return request<Ticket>(this.baseURL, 'POST', '/tickets', data);
  }
  async searchTickets(query: string, params?: PaginationParams): Promise<TicketListResponse> {
    return request<TicketListResponse>(this.baseURL, 'GET', '/tickets/search', undefined, { ...params, search: query } as unknown as Record<string, string | number | boolean | undefined>);
  }
  async getOverdueTickets(): Promise<Ticket[]> {
    return request<Ticket[]>(this.baseURL, 'GET', '/tickets/overdue');
  }
  async globalSearch(q: string, page?: number): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'GET', '/search', undefined, { q, page });
  }

  // ---------- Incidents ----------
  async listIncidents(params?: PaginationParams & { status?: string; priority?: string }): Promise<IncidentListResponse> {
    return request<IncidentListResponse>(this.baseURL, 'GET', '/incidents', undefined, params as Record<string, string | number | undefined>);
  }
  async getIncident(id: number): Promise<Incident> {
    return request<Incident>(this.baseURL, 'GET', `/incidents/${id}`);
  }
  async aiTriage(payload: { title: string; description: string }): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', '/ai/triage', payload);
  }

  // ---------- Changes ----------
  async listChanges(params?: PaginationParams & { status?: string }): Promise<ChangeListResponse> {
    return request<ChangeListResponse>(this.baseURL, 'GET', '/changes', undefined, params as Record<string, string | number | undefined>);
  }
  async getChange(id: number): Promise<Change> {
    return request<Change>(this.baseURL, 'GET', `/changes/${id}`);
  }

  // ---------- CMDB ----------
  async listCIs(params?: PaginationParams & { type?: string; status?: string }): Promise<CIListResponse> {
    return request<CIListResponse>(this.baseURL, 'GET', '/cmdb/cis', undefined, params as Record<string, string | number | undefined>);
  }
  async getCI(id: number): Promise<CI> {
    return request<CI>(this.baseURL, 'GET', `/cmdb/cis/${id}`);
  }

  // ---------- Knowledge ----------
  async listKnowledge(params?: PaginationParams & { category?: string }): Promise<KnowledgeListResponse> {
    return request<KnowledgeListResponse>(this.baseURL, 'GET', '/knowledge/articles', undefined, params as Record<string, string | number | undefined>);
  }
  async searchKnowledge(q: string): Promise<KnowledgeListResponse> {
    return request<KnowledgeListResponse>(this.baseURL, 'GET', '/knowledge/articles', undefined, { search: q, pageSize: 20 });
  }
  async getKnowledgeArticle(id: number): Promise<KnowledgeArticle> {
    return request<KnowledgeArticle>(this.baseURL, 'GET', `/knowledge/articles/${id}`);
  }

  // ---------- SLA ----------
  async getSLAStats(): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'GET', '/sla/stats');
  }
  async getSLAOverdue(): Promise<Ticket[]> {
    return request<Ticket[]>(this.baseURL, 'GET', '/sla/overdue');
  }

  // ---------- Workflow ----------
  async listProcessInstances(params?: PaginationParams): Promise<ProcessInstanceListResponse> {
    return request<ProcessInstanceListResponse>(this.baseURL, 'GET', '/workflow/instances', undefined, params as Record<string, string | number | undefined>);
  }
  async getProcessInstance(id: string): Promise<ProcessInstance> {
    return request<ProcessInstance>(this.baseURL, 'GET', `/workflow/instances/${id}`);
  }
  async listUserTasks(): Promise<ApprovalTaskListResponse> {
    return request<ApprovalTaskListResponse>(this.baseURL, 'GET', '/workflow/tasks');
  }
  async completeTask(id: string, outcome: string, comment?: string): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/workflow/tasks/${id}/complete`, { outcome, comment });
  }

  // ---------- Approvals ----------
  async listApprovals(status?: string): Promise<ApprovalTaskListResponse> {
    return request<ApprovalTaskListResponse>(this.baseURL, 'GET', '/approvals', undefined, { status });
  }
  async approveTask(id: string, comment?: string): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/approvals/${id}/approve`, { comment });
  }
  async rejectTask(id: string, comment?: string): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/approvals/${id}/reject`, { comment });
  }

  // ---------- Notifications ----------
  async listNotifications(params?: PaginationParams & { unread?: boolean }): Promise<NotificationListResponse> {
    return request<NotificationListResponse>(this.baseURL, 'GET', '/notifications', undefined, params as unknown as Record<string, string | number | boolean | undefined>);
  }
  async markNotificationRead(id: number): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/notifications/${id}/read`);
  }

  // ---------- Connectors / IM / Plugin market ----------
  async listConnectors(): Promise<{ items: ConnectorManifest[]; total: number }> {
    return request<{ items: ConnectorManifest[]; total: number }>(this.baseURL, 'GET', '/connectors');
  }
  async listConnectorConfigs(): Promise<{ items: unknown[]; total: number }> {
    return request<{ items: unknown[]; total: number }>(this.baseURL, 'GET', '/connectors/configs');
  }
  async provisionConnector(cfg: { name: string; provider: string; enabled: boolean; credentials: Record<string, string>; settings: Record<string, unknown> }): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', '/connectors/configs', cfg);
  }
  async testConnector(name: string): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/connectors/${name}/test`);
  }
  async sendViaConnector(name: string, payload: { channel: string; type: string; title?: string; content: string; card?: unknown }): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'POST', `/connectors/${name}/send`, payload);
  }
  async connectorHealth(): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'GET', '/connectors/health');
  }

  // ---------- Dashboard ----------
  async getDashboardOverview(): Promise<Record<string, unknown>> {
    return request<Record<string, unknown>>(this.baseURL, 'GET', '/dashboard/overview');
  }
}

export const apiClient = new ApiClient();
