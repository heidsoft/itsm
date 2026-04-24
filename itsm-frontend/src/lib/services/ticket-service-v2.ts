/**
 * TicketService - 工单服务
 *
 * 继承自 BaseService，提供工单相关的所有 API 操作。
 * 整合了原有 TicketApi 和 ticketService 的功能。
 */

import { BaseService, PaginatedResponse } from './base-service';
import type { Ticket, TicketPriority, TicketStatus } from '@/lib/api/types';

// ==================== 类型定义 ====================

/** 创建工单参数 */
export interface CreateTicketParams {
  title: string;
  description?: string;
  priority: TicketPriority;
  type?: 'incident' | 'problem' | 'change' | 'service_request';
  categoryId?: number;
  tags?: string[];
  assigneeId?: number;
  requesterId: number;
  formFields?: Record<string, unknown>;
  attachments?: string[];
}

/** 更新工单参数 */
export interface UpdateTicketParams {
  title?: string;
  description?: string;
  priority?: TicketPriority;
  status?: TicketStatus;
  categoryId?: number;
  tags?: string[];
  assigneeId?: number;
  resolution?: string;
  version?: number; // 乐观锁
}

/** 工单查询参数 */
export interface TicketQueryParams {
  page?: number;
  pageSize?: number;
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: string;
  requesterId?: number;
  assigneeId?: number;
  categoryId?: number;
  keyword?: string;
  dateFrom?: string;
  dateTo?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

/** 工单统计 */
export interface TicketStats {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  highPriority: number;
  overdue: number;
}

/** SLA 信息 */
export interface TicketSLAInfo {
  ticketId: number;
  slaName: string;
  responseDeadline: string | null;
  resolutionDeadline: string | null;
  isBreached: boolean;
  responseTimeRemaining: number | null;
  resolutionTimeRemaining: number | null;
}

/** 工单评论 */
export interface TicketComment {
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
  };
  createdAt: string;
  updatedAt: string;
}

/** 工单附件 */
export interface TicketAttachment {
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
}

/** 工单活动 */
export interface TicketActivity {
  id: string;
  type: string;
  title: string;
  description: string;
  user: string;
  timestamp: string;
  priority?: string;
  status?: string;
}

// ==================== TicketService 类 ====================

/**
 * TicketService - 工单服务
 *
 * 使用单例模式，提供工单相关的所有 API 操作。
 */
export class TicketService extends BaseService<Ticket, CreateTicketParams, UpdateTicketParams> {
  protected readonly basePath = '/api/v1/tickets';

  // ==================== 工单操作 ====================

  /**
   * 获取工单列表
   */
  async getTickets(params?: TicketQueryParams): Promise<PaginatedResponse<Ticket>> {
    return this.list(params);
  }

  /**
   * 获取工单详情
   */
  async getTicket(id: number): Promise<Ticket> {
    return this.getById(id);
  }

  /**
   * 创建工单
   */
  async createTicket(data: CreateTicketParams): Promise<Ticket> {
    return this.create(data);
  }

  /**
   * 更新工单
   */
  async updateTicket(id: number, data: UpdateTicketParams): Promise<Ticket> {
    return this.update(id, data);
  }

  /**
   * 删除工单
   */
  async deleteTicket(id: number): Promise<void> {
    return this.delete(id);
  }

  // ==================== 状态操作 ====================

  /**
   * 更新工单状态
   */
  async updateStatus(id: number, status: TicketStatus): Promise<Ticket> {
    return this.put<Ticket>(`/${id}/status`, { status });
  }

  /**
   * 分配工单
   */
  async assign(id: number, assigneeId: number, comment?: string): Promise<Ticket> {
    return this.post<Ticket>(`/${id}/assign`, { assignee_id: assigneeId, comment });
  }

  /**
   * 解决工单
   */
  async resolve(id: number, resolution: string, resolutionCode?: string): Promise<Ticket> {
    return this.post<Ticket>(`/${id}/resolve`, {
      resolution,
      resolution_code: resolutionCode,
    });
  }

  /**
   * 关闭工单
   */
  async close(id: number, feedback?: string): Promise<Ticket> {
    return this.post<Ticket>(`/${id}/close`, { feedback });
  }

  /**
   * 重开工单
   */
  async reopen(id: number, reason?: string): Promise<{ message: string }> {
    return this.post('/workflow/reopen', { ticket_id: id, reason });
  }

  /**
   * 升级工单
   */
  async escalate(
    id: number,
    data: { level: string; reason: string; assigneeId?: number }
  ): Promise<Ticket> {
    return this.post<Ticket>(`/${id}/escalate`, data);
  }

  // ==================== 审批操作 ====================

  /**
   * 批准工单
   */
  async approve(
    id: number,
    data: { action: 'approve' | 'reject' | 'delegate'; comment?: string; delegateToUserId?: number }
  ): Promise<{ success: boolean; message: string }> {
    return this.post('/workflow/approve', {
      ticket_id: id,
      ...data,
    });
  }

  /**
   * 拒绝工单
   */
  async reject(ticketId: number, reason: string): Promise<{ message: string }> {
    return this.post('/workflow/reject', { ticket_id: ticketId, reason });
  }

  /**
   * 接单
   */
  async accept(ticketId: number): Promise<{ message: string }> {
    return this.post('/workflow/accept', { ticket_id: ticketId });
  }

  /**
   * 撤回
   */
  async withdraw(ticketId: number, reason?: string): Promise<{ message: string }> {
    return this.post('/workflow/withdraw', { ticket_id: ticketId, reason });
  }

  /**
   * 转发
   */
  async forward(
    ticketId: number,
    toUserId: number,
    comment?: string
  ): Promise<{ message: string }> {
    return this.post('/workflow/forward', {
      ticket_id: ticketId,
      to_user_id: toUserId,
      comment,
    });
  }

  // ==================== 查询操作 ====================

  /**
   * 搜索工单
   */
  async searchTickets(query: string): Promise<Ticket[]> {
    return this.get<Ticket[]>('/search', { q: query });
  }

  /**
   * 获取逾期工单
   */
  async getOverdueTickets(): Promise<Ticket[]> {
    return this.get<Ticket[]>('/overdue');
  }

  /**
   * 获取处理人的工单
   */
  async getTicketsByAssignee(assigneeId: number): Promise<Ticket[]> {
    return this.get<Ticket[]>(`/assignee/${assigneeId}`);
  }

  /**
   * 获取工单统计
   */
  async getStats(): Promise<TicketStats> {
    return this.get<TicketStats>('/stats');
  }

  // ==================== SLA 操作 ====================

  /**
   * 获取工单 SLA 信息
   */
  async getSLA(id: number): Promise<TicketSLAInfo> {
    return this.get<TicketSLAInfo>(`/${id}/sla`);
  }

  // ==================== 评论操作 ====================

  /**
   * 获取工单评论
   */
  async getComments(id: number): Promise<{ comments: TicketComment[]; total: number }> {
    return this.get(`/${id}/comments`);
  }

  /**
   * 添加工单评论
   */
  async addComment(
    id: number,
    data: { content: string; isInternal?: boolean; mentions?: number[]; attachments?: number[] }
  ): Promise<TicketComment> {
    return this.post(`/${id}/comments`, data);
  }

  /**
   * 更新工单评论
   */
  async updateComment(
    ticketId: number,
    commentId: number,
    data: { content?: string; isInternal?: boolean; mentions?: number[] }
  ): Promise<TicketComment> {
    return this.put(`/${ticketId}/comments/${commentId}`, data);
  }

  /**
   * 删除工单评论
   */
  async deleteComment(ticketId: number, commentId: number): Promise<void> {
    return this.del(`/${ticketId}/comments/${commentId}`);
  }

  // ==================== 附件操作 ====================

  /**
   * 获取工单附件
   */
  async getAttachments(id: number): Promise<{ attachments: TicketAttachment[]; total: number }> {
    return this.get(`/${id}/attachments`);
  }

  /**
   * 上传工单附件
   */
  async uploadAttachment(
    id: number,
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<TicketAttachment> {
    const formData = new FormData();
    formData.append('file', file);

    return this.post(`/${id}/attachments`, formData, {
      onUploadProgress: onProgress,
    } as any);
  }

  /**
   * 删除工单附件
   */
  async deleteAttachment(ticketId: number, attachmentId: number): Promise<void> {
    return this.del(`/${ticketId}/attachments/${attachmentId}`);
  }

  // ==================== 工作流操作 ====================

  /**
   * 获取工单工作流状态
   */
  async getWorkflowState(id: number): Promise<{
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
    return this.get(`/${id}/workflow/state`);
  }

  // ==================== 活动记录 ====================

  /**
   * 获取工单活动日志
   */
  async getActivity(id: number): Promise<TicketActivity[]> {
    return this.get(`/${id}/activity`);
  }

  /**
   * 获取工单历史
   */
  async getHistory(id: number): Promise<
    Array<{
      id: number;
      fieldName: string;
      oldValue: string;
      newValue: string;
      changedBy: number;
      changedAt: string;
      user?: { id: number; name: string };
    }>
  > {
    return this.get(`/${id}/history`);
  }

  // ==================== 子任务操作 ====================

  /**
   * 获取子任务
   */
  async getSubtasks(parentTicketId: number): Promise<Ticket[]> {
    const response = await this.get<{ tickets?: Ticket[]; data?: Ticket[] }>(
      `/${parentTicketId}/subtasks`
    );
    return (response as any).tickets || (response as any).data || [];
  }

  /**
   * 创建子任务
   */
  async createSubtask(parentTicketId: number, data: Partial<Ticket>): Promise<Ticket> {
    return this.post(`/${parentTicketId}/subtasks`, {
      ...data,
      parent_ticket_id: parentTicketId,
    });
  }

  /**
   * 更新子任务
   */
  async updateSubtask(
    parentTicketId: number,
    subtaskId: number,
    data: Partial<Ticket>
  ): Promise<Ticket> {
    return this.patch(`/${parentTicketId}/subtasks/${subtaskId}`, data);
  }

  /**
   * 删除子任务
   */
  async deleteSubtask(parentTicketId: number, subtaskId: number): Promise<void> {
    return this.del(`/${parentTicketId}/subtasks/${subtaskId}`);
  }

  // ==================== 标签操作 ====================

  /**
   * 添加工单标签
   */
  async addTags(id: number, tags: string[]): Promise<{ success: boolean; tags: string[] }> {
    return this.post(`/${id}/tags`, { tags });
  }

  /**
   * 移除工单标签
   */
  async removeTags(
    id: number,
    tags: string[]
  ): Promise<{ success: boolean; removedTags: string[]; remainingTags: string[] }> {
    return this.del(`/${id}/tags`, { tags });
  }

  // ==================== 批量操作 ====================

  /**
   * 批量删除工单
   */
  async batchDelete(ticketIds: number[]): Promise<void> {
    return this.del('/batch', { ticket_ids: ticketIds });
  }

  /**
   * 批量更新工单
   */
  async batchUpdate(
    ticketIds: number[],
    action: string,
    data?: Record<string, unknown>
  ): Promise<void> {
    return this.put('/batch', { ticket_ids: ticketIds, action, data });
  }

  // ==================== 导出操作 ====================

  /**
   * 导出工单
   */
  async export(params: {
    format: 'excel' | 'csv' | 'pdf';
    filters?: Record<string, unknown>;
  }): Promise<Blob> {
    const response = await this.get<Blob>('/export', params as any);
    return response;
  }

  // ==================== 模板操作 ====================

  /**
   * 获取工单模板
   */
  async getTemplates(params?: {
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
    return this.get('/templates', params);
  }
}

// ==================== 单例导出 ====================

export const ticketService = new TicketService();
export default ticketService;
