import { loadCredentials, type Credentials } from './credentials.js';
import type {
  LoginRequest,
  LoginResponse,
  User,
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  PaginationParams,
} from './types.js';

const DEFAULT_BASE_URL = 'http://localhost:8090';

async function request<T>(
  baseURL: string,
  method: string,
  endpoint: string,
  data?: unknown,
    params?: Record<string, string | number | undefined>
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

  async logout(): Promise<void> {
    await request(this.baseURL, 'POST', '/auth/logout');
  }

  async getUserInfo(): Promise<User> {
    return request<User>(this.baseURL, 'GET', '/auth/userinfo');
  }

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
    return request<TicketListResponse>(this.baseURL, 'GET', '/tickets/search', undefined, { ...params, search: query } as Record<string, string | number | undefined>);
  }

  async getSLAStats(): Promise<unknown> {
    return request<unknown>(this.baseURL, 'GET', '/sla/stats');
  }

  async getOverdueTickets(): Promise<Ticket[]> {
    return request<Ticket[]>(this.baseURL, 'GET', '/tickets/overdue');
  }
}

export const apiClient = new ApiClient();
