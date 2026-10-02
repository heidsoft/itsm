import { httpClient } from './http-client';
import { handleApiRequest } from './base-api-handler';
import { TicketStatus, TicketPriority } from '@/constants/taxonomy';

// 枚举本体（含展示配置）在 @/constants/taxonomy；此处以值形式再导出，
// 使契约模块成为 status/priority 词表的唯一对外入口。
export { TicketStatus, TicketPriority };

/**
 * 工单 API 契约的唯一声明处。
 *
 * 字段逐一对齐后端 itsm-backend/dto/ticket_dto.go
 * （TicketResponse / ListTicketsResponse / CreateTicketRequest / UpdateTicketRequest /
 * ListTicketsRequest），状态词表对齐 itsm-backend/common/constants.go 的工单状态机
 * （单一事实来源，见 itsm-backend/repository/ticket/model.go CanTransitionTo）。
 *
 * lib/api/types.ts、lib/api/api-config.ts、types/ticket.ts 只能再导出本模块，
 * 不得另立字段。后端 DTO 把 status/priority/type 声明为 string，前端按词表收窄以便渲染，
 * 未知值仍由调用方的 Record<string, ...> 兜底。
 */

/** 工单 ITIL 类型词表；与后端 binding `oneof=incident service_request change ticket problem improvement` 一致 */
export const TICKET_TYPES = [
  'incident',
  'service_request',
  'change',
  'problem',
  'ticket',
  'improvement',
] as const;

export type TicketType = (typeof TICKET_TYPES)[number];

/** 把后端返回的自由字符串收窄为 TicketType */
export const isTicketType = (value: string): value is TicketType =>
  (TICKET_TYPES as readonly string[]).includes(value);

/** 工单来源；后端 TicketResponse 不返回该字段，仅前端筛选/展示使用 */
export type TicketSource = 'web' | 'email' | 'phone' | 'chat' | 'api' | 'mobile';

/** 对应后端 dto.UserBasicInfo */
export interface UserBasicInfo {
  id: number;
  username: string;
  name: string;
  email: string;
  role: string;
}

/** 对应后端 dto.TicketResponse */
export interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  type: string;
  ticketTypeId?: number;
  ticketTypeCode?: string;
  ticketTypeName?: string;
  formFields?: Record<string, unknown>;
  requesterId: number;
  assigneeId?: number;
  tenantId: number;
  categoryId?: number;
  departmentId?: number;
  parentTicketId?: number;
  templateId?: number | null;
  version: number;
  createdAt: string;
  updatedAt: string;
  requester?: UserBasicInfo;
  assignee?: UserBasicInfo;
  resolution?: string;
  resolutionCategory?: string;
  resolvedAt?: string;
  closedAt?: string;
  firstResponseAt?: string;
  slaResponseDeadline?: string;
  slaResolutionDeadline?: string;
  rating?: number;
}

/** 后端标准列表信封 common.ListResponse：集合只在 items 下 */
export interface TicketListResponse {
  items: Ticket[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/** 对应后端 dto.CreateTicketRequest */
export interface CreateTicketRequest {
  title: string;
  description?: string;
  priority: TicketPriority;
  type?: TicketType;
  ticketTypeId?: number;
  /** legacy 租户配置工单类型编码 */
  typeId?: string;
  category?: string;
  categoryId?: number;
  templateId?: number;
  assigneeId?: number;
  parentTicketId?: number;
  tagIds?: number[];
  tags?: string[];
  formFields?: Record<string, unknown>;
  attachments?: string[];
  workflowDefinitionKey?: string;
}

/** 对应后端 dto.UpdateTicketRequest（PUT/PATCH 共用同一 handler） */
export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  priority?: TicketPriority;
  status?: TicketStatus;
  type?: TicketType;
  category?: string;
  categoryId?: number;
  assigneeId?: number;
  tags?: string[];
  resolution?: string;
  formFields?: Record<string, unknown>;
  /** 乐观锁版本；后端 Force 字段是 json:"-"，客户端不可传 */
  version?: number;
}

/** PUT /tickets/:id/status 请求体（后端 handlers/ticket/handler.go:349 匿名 struct） */
export interface UpdateStatusRequest {
  status: TicketStatus;
}

/** 对应后端 dto.AssignTicketRequest；后端只绑定 assigneeId，没有 reason/comment */
export interface AssignTicketRequest {
  assigneeId: number;
}

/** 对应后端 dto.EscalateTicketRequest；reason 必填，后端没有 level/assigneeId 字段 */
export interface EscalateTicketRequest {
  reason: string;
}

/**
 * 对应后端 dto.ResolveTicketRequest。
 * 后端同时接受 resolution 与 solution 是历史兼容，前端统一只发 resolution，不再双写。
 */
export interface ResolveTicketRequest {
  resolution: string;
  resolutionCategory?: string;
  workNotes?: string;
}

/** 对应后端 dto.CloseTicketRequest */
export interface CloseTicketRequest {
  closeReason?: string;
  closeNotes?: string;
  feedback?: string;
}

/**
 * 对应后端 dto.TicketExportRequest（POST /tickets/export，JSON body）。
 * 后端 handler 目前只消费 format 与 filters.status/priority。
 * format 只列后端真实实现的格式，pdf 会被后端拒绝为参数错误。
 */
export interface TicketExportRequest {
  format: 'csv' | 'excel';
  filters?: Pick<GetTicketsParams, 'status' | 'priority'>;
}

/** 对应后端 dto.ListTicketsRequest */
export interface GetTicketsParams {
  page?: number;
  pageSize?: number;
  status?: string;
  priority?: string;
  type?: string;
  category?: string;
  categoryId?: number;
  assigneeId?: number;
  requesterId?: number;
  parentTicketId?: number;
  templateId?: number;
  keyword?: string;
  dateFrom?: string;
  dateTo?: string;
  isOverdue?: boolean;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

/** 对应后端 dto.TicketStatsResponse */
export interface TicketStatsResponse {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  pending: number;
  highPriority: number;
  overdue: number;
}

/**
 * 对应后端 service.TicketSLAInfo（GET /tickets/:id/sla 直接序列化该结构体）。
 * 后端把响应/解决拆成两套剩余时间与违约标记，前端不得合并成单一 isBreached。
 */
export interface TicketSLAInfo {
  ticketId: number;
  ticketNumber: string;
  priority: string;
  slaDefinitionId: number;
  slaDefinitionName: string;
  responseDeadline: string;
  resolutionDeadline: string;
  responseTimeLeftMinutes: number;
  isResponseBreached: boolean;
  resolutionTimeLeftMinutes: number;
  isResolutionBreached: boolean;
  firstResponseAt?: string;
  resolvedAt?: string;
}

// AI-Native：工单关联的配置项（受影响配置项区块使用）
export interface TicketConfigurationItem {
  id: number;
  name: string;
  ciType: string;
  status: string;
  serialNumber?: string;
}

export class TicketApi {
  // Get ticket list
  static async getTickets(
    params?: GetTicketsParams
  ): Promise<TicketListResponse> {
    return handleApiRequest(httpClient.get<TicketListResponse>('/api/v1/tickets', params), {
      errorMessage: 'Failed to fetch tickets',
      silent: true,
    });
  }

  // Create ticket
  static async createTicket(data: CreateTicketRequest): Promise<Ticket> {
    return handleApiRequest(httpClient.post<Ticket>('/api/v1/tickets', data), {
      errorMessage: 'Failed to create ticket',
      showSuccess: true,
    });
  }

  // Get ticket details
  // id 支持数字 ID 与业务工单号(TKT-202609-000010):
  // 后端 GET /tickets/:id 在 Atoi 失败时会 fallback 到 GetTicketByNumber。
  static async getTicket(id: number | string): Promise<Ticket> {
    return handleApiRequest(httpClient.get<Ticket>(`/api/v1/tickets/${id}`), {
      errorMessage: 'Failed to fetch ticket details',
    });
  }

  // AI-Native：工单→配置项反向查询（受影响配置项）
  // 对应后端 GET /api/v1/tickets/:id/configuration-items
  static async getTicketConfigurationItems(id: number): Promise<TicketConfigurationItem[]> {
    return handleApiRequest(httpClient.get<TicketConfigurationItem[]>(`/api/v1/tickets/${id}/configuration-items`), {
      errorMessage: 'Failed to fetch ticket configuration items',
      silent: true,
    });
  }

  // Update ticket status
  static async updateTicketStatus(id: number, status: TicketStatus): Promise<Ticket> {
    return handleApiRequest(httpClient.put<Ticket>(`/api/v1/tickets/${id}/status`, { status }), {
      errorMessage: 'Failed to update ticket status',
      showSuccess: true,
    });
  }

  // Update ticket information
  static async updateTicket(id: number, data: UpdateTicketRequest): Promise<Ticket> {
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
  static async assignTicket(id: number, data: AssignTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/assign`, data);
  }

  // Escalate ticket
  static async escalateTicket(id: number, data: EscalateTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/escalate`, data);
  }

  // Resolve ticket
  static async resolveTicket(id: number, data: ResolveTicketRequest): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/resolve`, data);
  }

  // Close ticket
  static async closeTicket(id: number, data: CloseTicketRequest = {}): Promise<Ticket> {
    return httpClient.post<Ticket>(`/api/v1/tickets/${id}/close`, data);
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
    const response = await httpClient.get<Ticket[]>(
      `/api/v1/tickets/${parentTicketId}/subtasks`
    );
    return response ?? [];
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
      action: string;
      details?: string;
      createdAt: string;
      userId?: number;
      userName?: string;
      oldValue?: string | null;
      newValue?: string | null;
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
  static async getTicketStats(): Promise<TicketStatsResponse> {
    return httpClient.get('/api/v1/tickets/stats');
  }

  // Export tickets
  static async exportTickets(data: TicketExportRequest): Promise<Blob> {
    const response = await httpClient.request<Blob>({
      method: 'POST',
      url: '/api/v1/tickets/export',
      data,
      responseType: 'blob',
    });
    return response;
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

  static async getTemplate(id: number | string): Promise<{
    id: number;
    name: string;
    description: string;
    category: string;
    priority: string;
    fields?: Array<Record<string, unknown>>;
    formFields: Record<string, unknown>;
    workflowSteps?: Array<Record<string, unknown>>;
    isActive: boolean;
    createdAt: string;
    updatedAt: string;
  }> {
    return httpClient.get(`/api/v1/tickets/templates/${id}`);
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
  static async getTicketSLA(id: number): Promise<TicketSLAInfo> {
    return httpClient.get<TicketSLAInfo>(`/api/v1/tickets/${id}/sla`);
  }
}

// 统一导出别名
export const TicketAPI = TicketApi;
export default TicketAPI;
