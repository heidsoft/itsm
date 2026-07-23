import { httpClient } from './http-client';
import { handleApiRequest } from './base-api-handler';
import type { Ticket, TicketListResponse, CreateTicketRequest, GetTicketsParams } from './api-config';

export class TicketApi {
  // Get ticket list
  static async getTickets(
    params?: GetTicketsParams & { [key: string]: unknown }
  ): Promise<TicketListResponse> {
    const response = await handleApiRequest(httpClient.get<TicketListResponse>('/api/v1/tickets', params), {
      errorMessage: 'Failed to fetch tickets',
      silent: true,
    });

    return {
      ...response,
      size: response.size ?? response.pageSize ?? params?.pageSize ?? params?.size ?? 20,
    };
  }

  // Create ticket
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return handleApiRequest(httpClient.post<Ticket>('/api/v1/tickets', data), {
      errorMessage: 'Failed to create ticket',
      showSuccess: true,
    });
  }

  // Get ticket details
  static async getTicket(id: number): Promise<Ticket> {
    return handleApiRequest(httpClient.get<Ticket>(`/api/v1/tickets/${id}`), {
      errorMessage: 'Failed to fetch ticket details',
    });
  }

  // Update ticket status
  static async updateTicketStatus(id: number, status: string): Promise<Ticket> {
    return handleApiRequest(httpClient.put<Ticket>(`/api/v1/tickets/${id}/status`, { status }), {
      errorMessage: 'Failed to update ticket status',
      showSuccess: true,
    });
  }

  // Update ticket information
  static async updateTicket(
    id: number,
    data: Partial<Ticket> & { version?: number; force?: boolean }
  ): Promise<Ticket> {
    return handleApiRequest(httpClient.put<Ticket>(`/api/v1/tickets/${id}`, data), {
      errorMessage: 'Failed to update ticket',
      showSuccess: true,
    });
  }

  // Delete ticket
  static async deleteTicket(id: number): Promise<void> {
    return handleApiRequest(httpClient.delete(`/api/v1/tickets/${id}`), {
      errorMessage: 'Failed to delete ticket',
      showSuccess: true,
    });
  }

  // Approve ticket - 使用后端实际的 workflow/approve 端点
  static async approveTicket(
    id: number,
    data: {
      action: 'approve' | 'reject' | 'delegate';
      comment?: string;
      ticketId: number;
      delegateToUserId?: number;
    }
  ): Promise<{
    success: boolean;
    message: string;
  }> {
    return httpClient.post(`/api/v1/tickets/workflow/approve`, data);
  }

  // Add comment
  static async addComment(
    id: number,
    content: string
  ): Promise<{
    id: number;
    ticketId: number;
    content: string;
    createdBy: number;
    createdAt: string;
    author?: {
      id: number;
      name: string;
      username: string;
    };
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/comments`, { content });
  }

  // Assign ticket
  static async assignTicket(
    id: number,
    assigneeIdOrData: number | { assigneeId: number; comment?: string }
  ): Promise<Ticket> {
    const payload =
      typeof assigneeIdOrData === 'number' ? { assigneeId: assigneeIdOrData } : assigneeIdOrData;
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/assign`, payload);
  }

  // Escalate ticket
  static async escalateTicket(
    id: number,
    reasonOrData: string | { level: string; reason: string; assigneeId?: number }
  ): Promise<Ticket> {
    const payload = typeof reasonOrData === 'string' ? { reason: reasonOrData } : reasonOrData;
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/escalate`, payload);
  }

  // Resolve ticket
  static async resolveTicket(
    id: number,
    resolutionOrData: string | { solution: string; resolutionCode?: string }
  ): Promise<Ticket> {
    const payload =
      typeof resolutionOrData === 'string'
        ? { resolution: resolutionOrData }
        : {
            resolution: resolutionOrData.solution,
            resolutionCode: resolutionOrData.resolutionCode,
          };
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/resolve`, payload);
  }

  // Close ticket
  static async closeTicket(
    id: number,
    feedbackOrData?: string | { closeNotes?: string }
  ): Promise<Ticket> {
    const payload =
      typeof feedbackOrData === 'string'
        ? { feedback: feedbackOrData }
        : feedbackOrData
          ? { feedback: feedbackOrData.closeNotes }
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
    const response = await httpClient.get<{ tickets?: Ticket[]; data?: Ticket[] }>(
      `/api/v1/tickets/${parentTicketId}/subtasks`
    );
     
    return (response as any).tickets || (response as any).data || response || [];
  }

  // Create subtask
  static async createSubtask(parentTicketId: number, data: Partial<Ticket>): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${parentTicketId}/subtasks`, {
      ...data,
      parentTicketId: parentTicketId,
    });
  }

  // Update subtask
  static async updateSubtask(
    parentTicketId: number,
    subtaskId: number,
    data: Partial<Ticket>
  ): Promise<Ticket> {
    return httpClient.patch<Ticket>(
      `/api/v1/tickets/${parentTicketId}/subtasks/${subtaskId}`,
      data
    );
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
  static async getTicketActivity(id: number): Promise<
    Array<{
      action: string;
      timestamp: string;
      userId: number;
      details: string;
    }>
  > {
    return httpClient.get(`/api/v1/tickets/${id}/activity`);
  }

  // Get ticket comments
  static async getTicketComments(id: number): Promise<{
    comments: Array<{
      id: number;
      ticketId: number;
      userId: number;
      content: string;
      isInternal: boolean;
      mentions: number[];
      attachments: number[];
      user?: {
        id: number;
        username: string;
        name: string;
        email: string;
        role?: string;
        department?: string;
        tenantId?: number;
      };
      createdAt: string;
      updatedAt: string;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/comments`);
  }

  // Add ticket comment
  static async addTicketComment(
    id: number,
    data: {
      content: string;
      isInternal?: boolean;
      mentions?: number[];
      attachments?: number[];
    }
  ): Promise<{
    id: number;
    ticketId: number;
    userId: number;
    content: string;
    isInternal: boolean;
    mentions: number[];
    attachments: number[];
    user?: {
      id: number;
      username: string;
      name: string;
      email: string;
      role?: string;
      department?: string;
      tenantId?: number;
    };
    createdAt: string;
    updatedAt: string;
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/comments`, data);
  }

  // Update ticket comment
  static async updateTicketComment(
    ticketId: number,
    commentId: number,
    data: {
      content?: string;
      isInternal?: boolean;
      mentions?: number[];
    }
  ): Promise<{
    id: number;
    ticketId: number;
    userId: number;
    content: string;
    isInternal: boolean;
    mentions: number[];
    attachments: number[];
    user?: {
      id: number;
      username: string;
      name: string;
      email: string;
      role?: string;
      department?: string;
      tenantId?: number;
    };
    createdAt: string;
    updatedAt: string;
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
      ticketId: number;
      fileName: string;
      filePath: string;
      fileUrl: string;
      fileSize: number;
      fileType: string;
      mimeType: string;
      uploadedBy: number;
      uploader?: {
        id: number;
        username: string;
        name: string;
        email: string;
        role?: string;
        department?: string;
        tenantId?: number;
      };
      createdAt: string;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/attachments`);
  }

  // Upload ticket attachment
  static async uploadTicketAttachment(
    id: number,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<{
    id: number;
    ticketId: number;
    fileName: string;
    filePath: string;
    fileUrl: string;
    fileSize: number;
    fileType: string;
    mimeType: string;
    uploadedBy: number;
    uploader?: {
      id: number;
      username: string;
      name: string;
      email: string;
    };
    createdAt: string;
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
    ticketId: number;
    currentStatus: string;
    availableActions: Array<{
      action: string;
      label: string;
      requiresComment: boolean;
    }>;
    workflowHistory: Array<{
      fromStatus: string;
      toStatus: string;
      action: string;
      performedAt: string;
      performedBy: number;
      comment?: string;
    }>;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/workflow/state`);
  }

  // Accept ticket (接单)
  static async acceptTicket(ticketId: number): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/accept`, { ticketId: ticketId });
  }

  // Reject ticket (驳回)
  static async rejectTicket(ticketId: number, reason: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/reject`, { ticketId: ticketId, reason });
  }

  // Withdraw ticket (撤回)
  static async withdrawTicket(ticketId: number, reason?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/withdraw`, { ticketId: ticketId, reason });
  }

  // Forward ticket (转发)
  static async forwardTicket(
    ticketId: number,
    toUserId: number,
    comment?: string
  ): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/forward`, {
      ticketId: ticketId,
      toUserId: toUserId,
      comment,
    });
  }

  // CC ticket (抄送)
  static async ccTicket(
    ticketId: number,
    ccUserIds: number[],
    comment?: string,
    notifyChannels?: string[]
  ): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/cc`, {
      ticketId: ticketId,
      ccUsers: ccUserIds,
      comment,
      notifyChannels,
    });
  }

  static async getMyCCRecords(): Promise<{
    records: Array<{
      id: number;
      ticketId: number;
      ticketNumber: string;
      title: string;
      status: string;
      priority: string;
      user: { id: number; name: string; username: string; email: string };
      addedBy: { id: number; name: string; username: string; email: string };
      addedAt: string;
      isActive: boolean;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/cc/my`);
  }

  static async getTicketCCRecords(ticketId: number): Promise<{
    records: Array<{
      id: number;
      ticketId: number;
      ticketNumber: string;
      title: string;
      status: string;
      priority: string;
      user: { id: number; name: string; username: string; email: string };
      addedBy: { id: number; name: string; username: string; email: string };
      addedAt: string;
      isActive: boolean;
    }>;
    total: number;
  }> {
    return httpClient.get(`/api/v1/tickets/${ticketId}/cc`);
  }

  // Reopen ticket (重开)
  static async reopenTicket(ticketId: number, reason?: string): Promise<{ message: string }> {
    return httpClient.post(`/api/v1/tickets/workflow/reopen`, { ticketId: ticketId, reason });
  }

  // Update workflow step
  static async updateWorkflowStep(
    ticketId: number,
    stepId: number,
    data: {
      status: string;
      comments?: string;
      assigneeId?: number;
    }
  ): Promise<{
    id: number;
    stepName: string;
    stepOrder: number;
    status: string;
    assigneeId?: number;
    startedAt?: string;
    completedAt?: string;
    comments?: string;
  }> {
    return httpClient.put(`/api/v1/tickets/${ticketId}/workflow/${stepId}`, data);
  }

  // Add ticket tags
  static async addTicketTags(
    id: number,
    tags: string[]
  ): Promise<{
    success: boolean;
    ticketId: number;
    tags: string[];
    message: string;
  }> {
    return httpClient.post(`/api/v1/tickets/${id}/tags`, { tags });
  }

  // Remove ticket tags
  static async removeTicketTags(
    id: number,
    tags: string[]
  ): Promise<{
    success: boolean;
    ticketId: number;
    removedTags: string[];
    remainingTags: string[];
    message: string;
  }> {
    return httpClient.request({
      method: 'DELETE',
      url: `/api/v1/tickets/${id}/tags`,
      data: { tags },
    });
  }

  // Get ticket history
  static async getTicketHistory(id: number): Promise<
    Array<{
      id: number;
      fieldName: string;
      oldValue: string;
      newValue: string;
      changedBy: number;
      changedAt: string;
      changeReason?: string;
      user?: {
        id: number;
        name: string;
      };
    }>
  > {
    return httpClient.get(`/api/v1/tickets/${id}/history`);
  }

  // Batch delete tickets
  static async batchDeleteTickets(ticketIds: number[]): Promise<void> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/tickets/batch-delete',
      data: { ticketIds: ticketIds },
    });
  }

  // Get ticket statistics
  static async getTicketStats(): Promise<{
    total: number;
    open: number;
    inProgress: number;
    resolved: number;
    highPriority: number;
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
      responseType: 'blob',
    });
    return response as Blob;
  }

  // Batch update tickets
  static async batchUpdateTickets(
    ticketIds: number[],
    action: string,
    data?: Record<string, unknown>
  ): Promise<void> {
    return httpClient.post('/api/v1/tickets/batch-assign', {
      ticketIds: ticketIds,
      action,
      data,
    });
  }

  // Get ticket templates
  static async getTemplates(params?: {
    page?: number;
    pageSize?: number;
    category?: string;
  }): Promise<{
    items: Array<{
      id: number;
      name: string;
      description: string;
      category: string;
      content: Record<string, unknown>;
      createdAt: string;
      updatedAt: string;
    }>;
    total: number;
  }> {
    return httpClient.get('/api/v1/tickets/templates', params);
  }

  // Create ticket template
  static async createTemplate(payload: {
    name: string;
    description?: string;
    category?: string;
    priority?: string;
    formFields?: Record<string, unknown>;
    fields?: Array<Record<string, unknown>>;
    workflowSteps?: Array<Record<string, unknown>>;
    isActive?: boolean;
  }): Promise<unknown> {
    return httpClient.post('/api/v1/tickets/templates', payload);
  }

  // Update ticket template
  static async updateTemplate(
    id: number | string,
    payload: {
      name?: string;
      description?: string;
      category?: string;
      priority?: string;
      formFields?: Record<string, unknown>;
      fields?: Array<Record<string, unknown>>;
      workflowSteps?: Array<Record<string, unknown>>;
      isActive?: boolean;
    }
  ): Promise<unknown> {
    return httpClient.put(`/api/v1/tickets/templates/${id}`, payload);
  }

  // Delete ticket template
  static async deleteTemplate(id: number | string): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/templates/${id}`);
  }

  // Update template status
  static async updateTemplateStatus(
    id: number | string,
    isActive: boolean
  ): Promise<unknown> {
    return httpClient.patch(`/api/v1/tickets/templates/${id}/status`, { isActive });
  }

  // Get ticket SLA info
  static async getTicketSLA(id: number): Promise<{
    ticketId: number;
    slaDefinitionId: number;
    slaName: string;
    serviceType: string;
    priority: string;
    responseTime: number;
    resolutionTime: number;
    responseDeadline: string | null;
    resolutionDeadline: string | null;
    firstResponseAt: string | null;
    resolvedAt: string | null;
    isBreached: boolean;
    responseTimeRemaining: number | null;
    resolutionTimeRemaining: number | null;
  }> {
    return httpClient.get(`/api/v1/tickets/${id}/sla`);
  }
}

// 统一导出别名
export const TicketAPI = TicketApi;
export default TicketAPI;
