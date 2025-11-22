import { httpClient } from './http-client';
import {
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  GetTicketsParams
} from './api-config';

export class TicketApi {
  // Get ticket list
  static async getTickets(params?: GetTicketsParams & { [key: string]: unknown }): Promise<TicketListResponse> {
    return httpClient.get<TicketListResponse>('/api/v1/tickets', params);
  }

  // Create ticket
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>('/api/v1/tickets', data);
  }

  // Get ticket details
  static async getTicket(id: number): Promise<Ticket> {
    return httpClient.get<Ticket>(`/api/v1/tickets/${id}`);
  }

  // Update ticket status
  static async updateTicketStatus(id: number, status: string): Promise<Ticket> {
    return httpClient.put<Ticket>(`/api/v1/tickets/${id}/status`, { status });
  }

  // Update ticket information
  static async updateTicket(id: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.patch<Ticket>(`/api/v1/tickets/${id}`, data);
  }

  // Delete ticket
  static async deleteTicket(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${id}`);
  }

  // Approve ticket
  static async approveTicket(id: number, data: {
    action: 'approve' | 'reject';
    comment: string;
    step_name: string;
  }): Promise<{
    success: boolean;
    ticket: Ticket;
    message: string;
    approval_status: 'approved' | 'rejected' | 'pending';
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/approve`, data);
  }

  // Add comment
  static async addComment(id: number, content: string): Promise<{
    id: number;
    ticket_id: number;
    content: string;
    created_by: number;
    created_at: string;
    author?: {
      id: number;
      name: string;
      username: string;
    };
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/comment`, { content });
  }

  // Assign ticket
  static async assignTicket(id: number, assigneeId: number): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/assign`, { assignee_id: assigneeId });
  }

  // Escalate ticket
  static async escalateTicket(id: number, reason: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/escalate`, { reason });
  }

  // Resolve ticket
  static async resolveTicket(id: number, resolution: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/resolve`, { resolution });
  }

  // Close ticket
  static async closeTicket(id: number, feedback?: string): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/close`, { feedback });
  }

  // Search tickets
  static async searchTickets(query: string): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/search', { q: query });
  }

  // Get overdue tickets
  static async getOverdueTickets(): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/overdue');
  }

  // Get tickets by assignee
  static async getTicketsByAssignee(assigneeId: number): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>(`/api/v1/tickets/assignee/${assigneeId}`);
  }

  // Get ticket activity log
  static async getTicketActivity(id: number): Promise<Array<{
    action: string;
    timestamp: string;
    user_id: number;
    details: string;
  }>> {
    return httpClient.get(`/api/v1/tickets/${id}/activity`);
  }

  // Get ticket comments
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

  // Add ticket comment
  static async addTicketComment(id: number, data: {
    content: string;
    type: 'comment' | 'work_note';
    is_internal?: boolean;
  }): Promise<{
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
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/comments`, data);
  }

  // Get ticket attachments
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

  // Upload ticket attachment
  static async uploadTicketAttachment(id: number, file: File): Promise<{
    id: number;
    filename: string;
    original_name: string;
    file_size: number;
    mime_type: string;
    url: string;
    uploaded_by: number;
    uploaded_at: string;
  }> {
    const formData = new FormData();
    formData.append('file', file);
    
    return httpClient.post(`/api/v1/tickets/${id}/attachments`, formData);
  }

  // Delete ticket attachment
  static async deleteTicketAttachment(ticketId: number, attachmentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/attachments/${attachmentId}`);
  }

  // Get ticket workflow
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

  // Update workflow step
  static async updateWorkflowStep(ticketId: number, stepId: number, data: {
    status: string;
    comments?: string;
    assignee_id?: number;
  }): Promise<{
    id: number;
    step_name: string;
    step_order: number;
    status: string;
    assignee_id?: number;
    started_at?: string;
    completed_at?: string;
    comments?: string;
  }> {
    return httpClient.put(`/api/v1/tickets/${ticketId}/workflow/${stepId}`, data);
  }

  // Get ticket SLA
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

  // Add ticket tags
  static async addTicketTags(id: number, tags: string[]): Promise<{
    success: boolean;
    ticket_id: number;
    tags: string[];
    message: string;
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/tags`, { tags });
  }

  // Remove ticket tags
  static async removeTicketTags(id: number, tags: string[]): Promise<{
    success: boolean;
    ticket_id: number;
    removed_tags: string[];
    remaining_tags: string[];
    message: string;
  }> {
    return httpClient.request({
      method: 'DELETE',
      url: `/api/v1/tickets/${id}/tags`,
      data: { tags }
    });
  }

  // Get ticket history
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

  // Batch delete tickets
  static async batchDeleteTickets(ticketIds: number[]): Promise<void> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/tickets/batch',
      data: { ticket_ids: ticketIds }
    });
  }

  // Get ticket statistics
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

  // Export tickets
  static async exportTickets(params: {
    format: 'excel' | 'csv' | 'pdf';
    filters?: Record<string, unknown>;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'GET',
      url: '/api/v1/tickets/export',
      params,
      responseType: 'blob'
    });
    return response as Blob;
  }

  // Batch update tickets
  static async batchUpdateTickets(ticketIds: number[], action: string, data?: Record<string, unknown>): Promise<void> {
    return httpClient.put('/api/v1/tickets/batch', {
      ticket_ids: ticketIds,
      action,
      data
    });
  }
}

export default TicketApi;

// 导出别名以支持不同的导入方式
export const TicketAPI = TicketApi;