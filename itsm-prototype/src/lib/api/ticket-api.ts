import { httpClient } from '../../app/lib/http-client';

// 工单接口定义
export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  source: string;
  type: string;
  ticket_number: string;
  is_major_incident: boolean;
  created_at: string;
  updated_at: string;
  requester_id?: number;
  assignee_id?: number;
  tenant_id: number;
  form_fields?: Record<string, unknown>;
  resolution?: string;
  resolution_category?: string;
  satisfaction_rating?: number;
  satisfaction_comment?: string;
  escalation_level?: number;
  escalation_reason?: string;
  sla_due_date?: string;
  resolved_at?: string;
  closed_at?: string;
}

// 工单统计接口
export interface TicketStats {
  total: number;
  open: number;
  in_progress: number;
  resolved: number;
  closed: number;
  overdue: number;
  high_priority: number;
  critical_priority: number;
}

// 列表请求参数
export interface ListTicketsRequest {
  page?: number;
  page_size?: number;
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  assignee_id?: number;
  requester_id?: number;
  is_major_incident?: boolean;
  keyword?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
  created_after?: string;
  created_before?: string;
  due_after?: string;
  due_before?: string;
}

// 列表响应
export interface ListTicketsResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// 创建工单请求
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  source?: string;
  type?: string;
  requester_id?: number;
  assignee_id?: number;
  form_fields?: Record<string, unknown>;
  is_major_incident?: boolean;
}

// 更新工单请求
export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  priority?: string;
  status?: string;
  assignee_id?: number;
  form_fields?: Record<string, unknown>;
  is_major_incident?: boolean;
}

// 分配工单请求
export interface AssignTicketRequest {
  assignee_id: number;
  comment?: string;
}

// 升级工单请求
export interface EscalateTicketRequest {
  escalation_level: number;
  reason: string;
  assignee_id?: number;
}

// 解决工单请求
export interface ResolveTicketRequest {
  resolution: string;
  resolution_category?: string;
}

// 关闭工单请求
export interface CloseTicketRequest {
  close_reason?: string;
  satisfaction_rating?: number;
  satisfaction_comment?: string;
}

// API响应包装
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 单个工单响应
export interface TicketResponse {
  ticket: Ticket;
}

// 工单API类
export class TicketAPI {
  // 获取工单列表
  static async listTickets(params: ListTicketsRequest = {}): Promise<ListTicketsResponse> {
    return httpClient.get<ListTicketsResponse>('/api/v1/tickets', params as Record<string, unknown>);
  }

  // 获取单个工单
  static async getTicket(id: number): Promise<TicketResponse> {
    return httpClient.get<TicketResponse>(`/api/v1/tickets/${id}`);
  }

  // 创建工单
  static async createTicket(data: CreateTicketRequest): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>('/api/v1/tickets', data);
  }

  // 更新工单
  static async updateTicket(id: number, data: UpdateTicketRequest): Promise<TicketResponse> {
    return httpClient.put<TicketResponse>(`/api/v1/tickets/${id}`, data);
  }

  // 删除工单
  static async deleteTicket(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${id}`);
  }

  // 批量删除工单
  static async batchDeleteTickets(ids: number[]): Promise<void> {
    return httpClient.post('/api/v1/tickets/batch-delete', { ids });
  }

  // 分配工单
  static async assignTicket(id: number, data: AssignTicketRequest): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>(`/api/v1/tickets/${id}/assign`, data);
  }

  // 升级工单
  static async escalateTicket(id: number, data: EscalateTicketRequest): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>(`/api/v1/tickets/${id}/escalate`, data);
  }

  // 解决工单
  static async resolveTicket(id: number, data: ResolveTicketRequest): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>(`/api/v1/tickets/${id}/resolve`, data);
  }

  // 关闭工单
  static async closeTicket(id: number, data: CloseTicketRequest): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>(`/api/v1/tickets/${id}/close`, data);
  }

  // 重新打开工单
  static async reopenTicket(id: number, reason?: string): Promise<TicketResponse> {
    return httpClient.post<TicketResponse>(`/api/v1/tickets/${id}/reopen`, { reason });
  }

  // 获取工单统计
  static async getTicketStats(): Promise<TicketStats> {
    return httpClient.get<TicketStats>('/api/v1/tickets/stats');
  }

  // 导出工单
  static async exportTickets(params: ListTicketsRequest = {}): Promise<Blob> {
    // 使用get方法获取导出数据，然后转换为Blob
    const response = await httpClient.get<ArrayBuffer>('/api/v1/tickets/export', params as Record<string, unknown>);
    return new Blob([response], { type: 'application/octet-stream' });
  }
}