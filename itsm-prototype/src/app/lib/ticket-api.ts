import { httpClient } from './http-client';
import {
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  UpdateStatusRequest,
  GetTicketsParams,
  ApiResponse
} from './api-config';

export class TicketApi {
  // 获取工单列表
  static async getTickets(params?: GetTicketsParams): Promise<ApiResponse<TicketListResponse>> {
    return httpClient.get<TicketListResponse>('/api/tickets', params);
  }

  // 创建工单
  static async createTicket(data: CreateTicketRequest): Promise<ApiResponse<Ticket>> {
    return httpClient.post<Ticket>('/api/tickets', data);
  }

  // 获取工单详情
  static async getTicket(id: number): Promise<ApiResponse<Ticket>> {
    return httpClient.get<Ticket>(`/api/tickets/${id}`);
  }

  // 更新工单状态
  static async updateTicketStatus(id: number, status: string): Promise<ApiResponse<Ticket>> {
    return httpClient.put<Ticket>(`/api/tickets/${id}/status`, { status });
  }

  // 更新工单信息
  static async updateTicket(id: number, data: Partial<Ticket>): Promise<ApiResponse<Ticket>> {
    return httpClient.patch<Ticket>(`/api/tickets/${id}`, data);
  }

  // 审批工单
  static async approveTicket(id: number, data: {
    action: 'approve' | 'reject';
    comment: string;
    step_name: string;
  }): Promise<ApiResponse<any>> {
    return httpClient.post(`/api/tickets/${id}/approve`, data);
  }

  // 添加评论
  static async addComment(id: number, content: string): Promise<ApiResponse<any>> {
    return httpClient.post(`/api/tickets/${id}/comment`, { content });
  }
}

export default TicketApi;