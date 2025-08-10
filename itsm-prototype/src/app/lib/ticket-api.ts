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

  // 获取工单评论
  static async getTicketComments(id: number): Promise<Array<{
    id: number;
    content: string;
    type: string;
    created_by: number;
    created_at: string;
    author?: {
      id: number;
      name: string;
      username: string;
    };
    is_internal: boolean;
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/comments`);
  }

  // 添加工单评论
  static async addTicketComment(id: number, data: {
    content: string;
    type: 'comment' | 'work_note';
    is_internal?: boolean;
  }): Promise<unknown> {
    return httpClient.post(`/api/v1/tickets/${id}/comments`, data);
  }

  // 获取工单附件
  static async getTicketAttachments(id: number): Promise<Array<{
    id: number;
    filename: string;
    original_name: string;
    file_size: number;
    mime_type: string;
    url: string;
    uploaded_by: number;
    uploaded_at: string;
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/attachments`);
  }

  // 上传工单附件
  static async uploadTicketAttachment(id: number, file: File): Promise<unknown> {
    const formData = new FormData();
    formData.append('file', file);
    return httpClient.post(`/api/v1/tickets/${id}/attachments`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  }

  // 删除工单附件
  static async deleteTicketAttachment(ticketId: number, attachmentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/attachments/${attachmentId}`);
  }

  // 获取工单工作流
  static async getTicketWorkflow(id: number): Promise<Array<{
    id: number;
    step_name: string;
    step_order: number;
    status: string;
    assignee_id?: number;
    assignee?: {
      id: number;
      name: string;
    };
    started_at?: string;
    completed_at?: string;
    comments?: string;
    required_approval: boolean;
    approval_status?: string;
    approval_comments?: string;
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/workflow`);
  }

  // 更新工作流步骤状态
  static async updateWorkflowStep(ticketId: number, stepId: number, data: {
    status: string;
    comments?: string;
    assignee_id?: number;
  }): Promise<unknown> {
    return httpClient.patch(`/api/v1/tickets/${ticketId}/workflow/${stepId}`, data);
  }

  // 获取工单SLA信息
  static async getTicketSLA(id: number): Promise<{
    sla_id: number;
    sla_name: string;
    response_time: number;
    resolution_time: number;
    start_time: string;
    due_time: string;
    breach_time?: string;
    status: string;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/sla`);
  }

  // 添加工单标签
  static async addTicketTags(id: number, tags: string[]): Promise<unknown> {
    return httpClient.post(`/api/v1/tickets/${id}/tags`, { tags });
  }

  // 移除工单标签
  static async removeTicketTags(id: number, tags: string[]): Promise<unknown> {
    return httpClient.delete(`/api/v1/tickets/${id}/tags`, { data: { tags } });
  }

  // 获取工单历史记录
  static async getTicketHistory(id: number): Promise<Array<{
    id: number;
    field_name: string;
    old_value: string;
    new_value: string;
    changed_by: number;
    changed_at: string;
    change_reason?: string;
    user?: {
      id: number;
      name: string;
    };
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/history`);
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