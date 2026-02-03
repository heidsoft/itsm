import { httpClient } from './http-client';
import { handleApiRequest } from './base-api-handler';
import {
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  GetTicketsParams
} from './api-config';

export class TicketApi {
  // Get ticket list
  static async getTickets(params?: GetTicketsParams & { [key: string]: unknown }): Promise<TicketListResponse> {
    return handleApiRequest(
      httpClient.get<TicketListResponse>('/api/v1/tickets', params),
      { errorMessage: 'Failed to fetch tickets', silent: true }
    );
  }

  // Create ticket
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return handleApiRequest(
      httpClient.post<Ticket>('/api/v1/tickets', data),
      { errorMessage: 'Failed to create ticket', showSuccess: true }
    );
  }

  // Get ticket details
  static async getTicket(id: number): Promise<Ticket> {
    return handleApiRequest(
      httpClient.get<Ticket>(`/api/v1/tickets/${id}`),
      { errorMessage: 'Failed to fetch ticket details' }
    );
  }

  // Update ticket status
  static async updateTicketStatus(id: number, status: string): Promise<Ticket> {
    return handleApiRequest(
      httpClient.put<Ticket>(`/api/v1/tickets/${id}/status`, { status }),
      { errorMessage: 'Failed to update ticket status', showSuccess: true }
    );
  }

  // Update ticket information
  static async updateTicket(id: number, data: Partial<Ticket>): Promise<Ticket> {
    return handleApiRequest(
      httpClient.patch<Ticket>(`/api/v1/tickets/${id}`, data),
      { errorMessage: 'Failed to update ticket', showSuccess: true }
    );
  }

  // Delete ticket
  static async deleteTicket(id: number): Promise<void> {
    return handleApiRequest(
      httpClient.delete(`/api/v1/tickets/${id}`),
      { errorMessage: 'Failed to delete ticket', showSuccess: true }
    );
  }

  // Approve ticket - 使用后端实际的 workflow/approve 端点
  static async approveTicket(id: number, data: {
    action: 'approve' | 'reject' | 'delegate';
    comment?: string;
    ticket_id: number;
    delegate_to_user_id?: number;
  }): Promise<{
    success: boolean;
    message: string;
  }> {
    return httpClient.post(`/api/v1/tickets/workflow/approve`, data);
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
  static async assignTicket(
    id: number,
    assigneeIdOrData: number | { assignee_id: number; comment?: string }
  ): Promise<Ticket> {
    const payload = typeof assigneeIdOrData === 'number' ? { assignee_id: assigneeIdOrData } : assigneeIdOrData;
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/assign`, payload);
  }

  // Escalate ticket
  static async escalateTicket(
    id: number,
    reasonOrData: string | { level: string; reason: string; assignee_id?: number }
  ): Promise<Ticket> {
    const payload = typeof reasonOrData === 'string' ? { reason: reasonOrData } : reasonOrData;
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/escalate`, payload);
  }

  // Resolve ticket
  static async resolveTicket(
    id: number,
    resolutionOrData: string | { solution: string; resolution_code?: string }
  ): Promise<Ticket> {
    const payload =
      typeof resolutionOrData === 'string'
        ? { resolution: resolutionOrData }
        : { resolution: resolutionOrData.solution, resolution_code: resolutionOrData.resolution_code };
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/resolve`, payload);
  }

  // Close ticket
  static async closeTicket(
    id: number,
    feedbackOrData?: string | { close_notes?: string }
  ): Promise<Ticket> {
    const payload =
      typeof feedbackOrData === 'string'
        ? { feedback: feedbackOrData }
        : feedbackOrData
        ? { feedback: feedbackOrData.close_notes }
        : {};
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/close`, payload);
  }

  // Search tickets
  static async searchTickets(query: string): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/search', { q: query });
  }

  // Get overdue tickets
  static async getOverdueTickets(): Promise<Ticket[]> {
    return httpClient.get<Ticket[]>('/api/v1/tickets/overdue');
  }

  // Get subtasks (child tickets)
  static async getSubtasks(parentTicketId: number): Promise<Ticket[]> {
    const response = await httpClient.get<{ tickets?: Ticket[]; data?: Ticket[] }>(`/api/v1/tickets/${parentTicketId}/subtasks`);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return (response as any).tickets || (response as any).data || response || [];
  }

  // Create subtask
  static async createSubtask(parentTicketId: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${parentTicketId}/subtasks`, {
      ...data,
      parent_ticket_id: parentTicketId,
    });
  }

  // Update subtask
  static async updateSubtask(parentTicketId: number, subtaskId: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.patch<Ticket>(`/api/v1/tickets/${parentTicketId}/subtasks/${subtaskId}`, data);
  }

  // Delete subtask
  static async deleteSubtask(parentTicketId: number, subtaskId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${parentTicketId}/subtasks/${subtaskId}`);
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
  static async getTicketComments(id: number): Promise<{
    comments: Array<{
      id: number;
      ticket_id: number;
      user_id: number;
      content: string;
      is_internal: boolean;
      mentions: number[];
      attachments: number[];
      user?: {
        id: number;
        username: string;
        name: string;
        email: string;
        role?: string;
        department?: string;
        tenant_id?: number;
      };
      created_at: string;
      updated_at: string;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/comments`);
  }

  // Add ticket comment
  static async addTicketComment(id: number, data: {
    content: string;
    is_internal?: boolean;
    mentions?: number[];
    attachments?: number[];
  }): Promise<{
    id: number;
    ticket_id: number;
    user_id: number;
    content: string;
    is_internal: boolean;
    mentions: number[];
    attachments: number[];
    user?: {
      id: number;
      username: string;
      name: string;
      email: string;
      role?: string;
      department?: string;
      tenant_id?: number;
    };
    created_at: string;
    updated_at: string;
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/comments`, data);
  }

  // Update ticket comment
  static async updateTicketComment(ticketId: number, commentId: number, data: {
    content?: string;
    is_internal?: boolean;
    mentions?: number[];
  }): Promise<{
    id: number;
    ticket_id: number;
    user_id: number;
    content: string;
    is_internal: boolean;
    mentions: number[];
    attachments: number[];
    user?: {
      id: number;
      username: string;
      name: string;
      email: string;
      role?: string;
      department?: string;
      tenant_id?: number;
    };
    created_at: string;
    updated_at: string;
  }> {
    return httpClient.put(`/api/v1/tickets/${ticketId}/comments/${commentId}`, data);
  }

  // Delete ticket comment
  static async deleteTicketComment(ticketId: number, commentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/comments/${commentId}`);
  }

  // Get ticket attachments
  static async getTicketAttachments(id: number): Promise<{
    attachments: Array<{
      id: number;
      ticket_id: number;
      file_name: string;
      file_path: string;
      file_url: string;
      file_size: number;
      file_type: string;
      mime_type: string;
      uploaded_by: number;
      uploader?: {
        id: number;
        username: string;
        name: string;
        email: string;
        role?: string;
        department?: string;
        tenant_id?: number;
      };
      created_at: string;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/attachments`);
  }

  // Upload ticket attachment
  static async uploadTicketAttachment(id: number, file: File, onProgress?: (progress: number) => void): Promise<{
    id: number;
    ticket_id: number;
    file_name: string;
    file_path: string;
    file_url: string;
    file_size: number;
    file_type: string;
    mime_type: string;
    uploaded_by: number;
    uploader?: {
      id: number;
      username: string;
      name: string;
      email: string;
    };
    created_at: string;
  }> {
    const formData = new FormData();
    formData.append('file', file);
    
    return httpClient.post(`/api/v1/tickets/${id}/attachments`, formData, {
      onUploadProgress: onProgress,
    });
  }

  // Download ticket attachment
  static getAttachmentDownloadUrl(ticketId: number, attachmentId: number): string {
    return `/api/v1/tickets/${ticketId}/attachments/${attachmentId}`;
  }

  // Preview ticket attachment
  static getAttachmentPreviewUrl(ticketId: number, attachmentId: number): string {
    return `/api/v1/tickets/${ticketId}/attachments/${attachmentId}/preview`;
  }

  // Delete ticket attachment
  static async deleteTicketAttachment(ticketId: number, attachmentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/attachments/${attachmentId}`);
  }

  // Get ticket workflow state - 使用后端实际的 workflow/state 端点
  static async getTicketWorkflow(id: number): Promise<{
    ticket_id: number;
    current_status: string;
    available_actions: Array<{
      action: string;
      label: string;
      requires_comment: boolean;
    }>;
    workflow_history: Array<{
      from_status: string;
      to_status: string;
      action: string;
      performed_at: string;
      performed_by: number;
      comment?: string;
    }>;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/workflow/state`);
  }

  // Accept ticket (接单)
  static async acceptTicket(ticketId: number): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/accept`, { ticket_id: ticketId });
  }

  // Reject ticket (驳回)
  static async rejectTicket(ticketId: number, reason: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/reject`, { ticket_id: ticketId, reason });
  }

  // Withdraw ticket (撤回)
  static async withdrawTicket(ticketId: number, reason?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/withdraw`, { ticket_id: ticketId, reason });
  }

  // Forward ticket (转发)
  static async forwardTicket(ticketId: number, toUserId: number, comment?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/forward`, { ticket_id: ticketId, to_user_id: toUserId, comment });
  }

  // CC ticket (抄送)
  static async ccTicket(ticketId: number, ccUserIds: number[], comment?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/cc`, { ticket_id: ticketId, cc_user_ids: ccUserIds, comment });
  }

  // Reopen ticket (重开)
  static async reopenTicket(ticketId: number, reason?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/reopen`, { ticket_id: ticketId, reason });
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