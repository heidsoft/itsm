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
    return httpClient.get<TicketListResponse>('/api/tickets', params);
  }

  // 创建工单
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>('/api/tickets', data);
  }

  // 获取工单详情
  static async getTicket(id: number): Promise<Ticket> {
    return httpClient.get<Ticket>(`/api/tickets/${id}`);
  }

  // 更新工单状态
  static async updateTicketStatus(id: number, status: string): Promise<Ticket> {
    return httpClient.put<Ticket>(`/api/tickets/${id}/status`, { status });
  }

  // 更新工单信息
  static async updateTicket(id: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.patch<Ticket>(`/api/tickets/${id}`, data);
  }

  // 审批工单
  static async approveTicket(id: number, data: {
    action: 'approve' | 'reject';
    comment: string;
    step_name: string;
  }): Promise<unknown> {
    return httpClient.post(`/api/tickets/${id}/approve`, data);
  }

  // 添加评论
  static async addComment(id: number, content: string): Promise<unknown> {
    return httpClient.post(`/api/tickets/${id}/comment`, { content });
  }
}

export default TicketApi;