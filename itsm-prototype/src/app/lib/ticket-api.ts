import { httpClient } from './http-client';
import {
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  UpdateStatusRequest,
  GetTicketsParams
} from './api-config';

export class TicketApi {
  // 获取工单列表
  static async getTickets(params?: GetTicketsParams): Promise<TicketListResponse> {
    return httpClient.get<TicketListResponse>('/api/v1/tickets', params);
  }

  // 创建工单
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>('/api/v1/tickets', data);
  }

  // 获取工单详情
  static async getTicket(id: number): Promise<Ticket> {
    return httpClient.get<Ticket>(`/api/v1/tickets/${id}`);
  }

  // 更新工单状态
  static async updateTicketStatus(id: number, status: string): Promise<Ticket> {
    return httpClient.put<Ticket>(`/api/v1/tickets/${id}/status`, { status });
  }

  // 更新工单信息
  static async updateTicket(id: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.patch<Ticket>(`/api/v1/tickets/${id}`, data);
  }

  // 删除工单
  static async deleteTicket(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${id}`);
  }

  // 审批工单
  static async approveTicket(id: number, data: {
    action: 'approve' | 'reject';
    comment: string;
    step_name: string;
  }): Promise<unknown> {
    return httpClient.post(`/api/v1/tickets/${id}/approve`, data);
  }

  // 添加评论
  static async addComment(id: number, content: string): Promise<unknown> {
    return httpClient.post(`/api/v1/tickets/${id}/comment`, { content });
  }

  // 分配工单
  static async assignTicket(id: number, assigneeId: number): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/assign`, { assignee_id: assigneeId });
  }

  // 升级工单
  static async escalateTicket(id: number, reason: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/escalate`, { reason });
  }

  // 解决工单
  static async resolveTicket(id: number, resolution: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/resolve`, { resolution });
  }

  // 关闭工单
  static async closeTicket(id: number, feedback?: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/close`, { feedback });
  }

  // 搜索工单
  static async searchTickets(query: string): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/search', { q: query });
  }

  // 获取逾期工单
  static async getOverdueTickets(): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/overdue');
  }

  // 获取指定处理人的工单
  static async getTicketsByAssignee(assigneeId: number): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>(`/api/v1/tickets/assignee/${assigneeId}`);
  }

  // 获取工单活动日志
  static async getTicketActivity(id: number): Promise<Array<{
    action: string;
    timestamp: string;
    user_id: number;
    details: string;
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/activity`);
  }

  // 批量删除工单
  static async batchDeleteTickets(ticketIds: number[]): Promise<void> {
    return httpClient.post('/api/v1/tickets/batch-delete', { ticket_ids: ticketIds });
  }

  // 获取工单统计
  static async getTicketStats(): Promise<{
    total: number;
    open: number;
    in_progress: number;
    resolved: number;
    high_priority: number;
    overdue: number;
  }> {
    return httpClient.get('/api/v1/tickets/stats');
  }

  // 导出工单
  static async exportTickets(params: {
    format: 'excel' | 'csv' | 'pdf';
    filters?: Record<string, unknown>;
  }): Promise<Blob> {
    return httpClient.post('/api/v1/tickets/export', params, {
      responseType: 'blob'
    });
  }

  // 批量操作工单
  static async batchUpdateTickets(ticketIds: number[], action: string, data?: Record<string, unknown>): Promise<void> {
    return httpClient.post('/api/v1/tickets/batch-update', {
      ticket_ids: ticketIds,
      action,
      ...data
    });
  }
}

export default TicketApi;