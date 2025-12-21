/**
 * ITSM前端架构 - 模块化API管理
 * 
 * 工单管理模块API
 * 展示如何使用React Query进行服务端状态管理
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { httpClient } from '@/lib/api/base-api';
import { Ticket, TicketFilters, Comment, Attachment } from './ticket-store';

// ==================== API端点配置 ====================

const API_ENDPOINTS = {
  TICKETS: '/api/v1/tickets',
  TICKET_DETAIL: (id: number) => `/api/v1/tickets/${id}`,
  TICKET_COMMENTS: (id: number) => `/api/v1/tickets/${id}/comments`,
  TICKET_ATTACHMENTS: (id: number) => `/api/v1/tickets/${id}/attachments`,
  TICKET_ASSIGN: (id: number) => `/api/v1/tickets/${id}/assign`,
  TICKET_RESOLVE: (id: number) => `/api/v1/tickets/${id}/resolve`,
  TICKET_CLOSE: (id: number) => `/api/v1/tickets/${id}/close`,
  TICKET_REOPEN: (id: number) => `/api/v1/tickets/${id}/reopen`,
} as const;

// ==================== 查询键配置 ====================

export const QUERY_KEYS = {
  TICKETS: ['tickets'] as const,
  TICKET_LIST: (filters?: TicketFilters) => ['tickets', 'list', filters] as const,
  TICKET_DETAIL: (id: number) => ['tickets', 'detail', id] as const,
  TICKET_COMMENTS: (id: number) => ['tickets', 'comments', id] as const,
  TICKET_ATTACHMENTS: (id: number) => ['tickets', 'attachments', id] as const,
} as const;

// ==================== API服务类 ====================

export class TicketApiService {
  /**
   * 获取工单列表
   */
  static async getTickets(params?: {
    page?: number;
    pageSize?: number;
    filters?: TicketFilters;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
  }) {
    return httpClient.get<{
      tickets: Ticket[];
      total: number;
      page: number;
      pageSize: number;
      totalPages: number;
    }>(API_ENDPOINTS.TICKETS, params);
  }

  /**
   * 获取工单详情
   */
  static async getTicket(id: number) {
    return httpClient.get<Ticket>(API_ENDPOINTS.TICKET_DETAIL(id));
  }

  /**
   * 创建工单
   */
  static async createTicket(data: Partial<Ticket>) {
    return httpClient.post<Ticket>(API_ENDPOINTS.TICKETS, data);
  }

  /**
   * 更新工单
   */
  static async updateTicket(id: number, data: Partial<Ticket>) {
    return httpClient.put<Ticket>(API_ENDPOINTS.TICKET_DETAIL(id), data);
  }

  /**
   * 删除工单
   */
  static async deleteTicket(id: number) {
    return httpClient.delete<void>(API_ENDPOINTS.TICKET_DETAIL(id));
  }

  /**
   * 批量删除工单
   */
  static async batchDeleteTickets(ids: number[]) {
    return httpClient.post<void>(`${API_ENDPOINTS.TICKETS}/batch-delete`, { ids });
  }

  /**
   * 分配工单
   */
  static async assignTicket(id: number, assigneeId: number, comment?: string) {
    return httpClient.post<void>(API_ENDPOINTS.TICKET_ASSIGN(id), {
      assigneeId,
      comment,
    });
  }

  /**
   * 解决工单
   */
  static async resolveTicket(id: number, resolution: string, category?: string) {
    return httpClient.post<void>(API_ENDPOINTS.TICKET_RESOLVE(id), {
      resolution,
      category,
    });
  }

  /**
   * 关闭工单
   */
  static async closeTicket(id: number, reason?: string, rating?: number, comment?: string) {
    return httpClient.post<void>(API_ENDPOINTS.TICKET_CLOSE(id), {
      reason,
      rating,
      comment,
    });
  }

  /**
   * 重新打开工单
   */
  static async reopenTicket(id: number, reason?: string) {
    return httpClient.post<void>(API_ENDPOINTS.TICKET_REOPEN(id), {
      reason,
    });
  }

  /**
   * 获取工单评论
   */
  static async getTicketComments(id: number) {
    return httpClient.get<Comment[]>(API_ENDPOINTS.TICKET_COMMENTS(id));
  }

  /**
   * 添加评论
   */
  static async addComment(id: number, content: string, isInternal = false) {
    return httpClient.post<Comment>(API_ENDPOINTS.TICKET_COMMENTS(id), {
      content,
      isInternal,
    });
  }

  /**
   * 更新评论
   */
  static async updateComment(id: number, commentId: number, content: string) {
    return httpClient.put<Comment>(`${API_ENDPOINTS.TICKET_COMMENTS(id)}/${commentId}`, {
      content,
    });
  }

  /**
   * 删除评论
   */
  static async deleteComment(id: number, commentId: number) {
    return httpClient.delete<void>(`${API_ENDPOINTS.TICKET_COMMENTS(id)}/${commentId}`);
  }

  /**
   * 获取工单附件
   */
  static async getTicketAttachments(id: number) {
    return httpClient.get<Attachment[]>(API_ENDPOINTS.TICKET_ATTACHMENTS(id));
  }

  /**
   * 上传附件
   */
  static async uploadAttachment(id: number, file: File) {
    const formData = new FormData();
    formData.append('file', file);
    
    return httpClient.post<Attachment>(API_ENDPOINTS.TICKET_ATTACHMENTS(id), formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  }

  /**
   * 删除附件
   */
  static async deleteAttachment(id: number, attachmentId: number) {
    return httpClient.delete<void>(`${API_ENDPOINTS.TICKET_ATTACHMENTS(id)}/${attachmentId}`);
  }
}

// ==================== React Query Hooks ====================

/**
 * 获取工单列表
 */
export function useTickets(params?: {
  page?: number;
  pageSize?: number;
  filters?: TicketFilters;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}) {
  return useQuery({
    queryKey: QUERY_KEYS.TICKET_LIST(params?.filters),
    queryFn: () => TicketApiService.getTickets(params),
    staleTime: 5 * 60 * 1000, // 5分钟
    cacheTime: 10 * 60 * 1000, // 10分钟
    keepPreviousData: true,
  });
}

/**
 * 获取工单详情
 */
export function useTicket(id: number) {
  return useQuery({
    queryKey: QUERY_KEYS.TICKET_DETAIL(id),
    queryFn: () => TicketApiService.getTicket(id),
    enabled: !!id,
    staleTime: 2 * 60 * 1000, // 2分钟
    cacheTime: 5 * 60 * 1000, // 5分钟
  });
}

/**
 * 获取工单评论
 */
export function useTicketComments(id: number) {
  return useQuery({
    queryKey: QUERY_KEYS.TICKET_COMMENTS(id),
    queryFn: () => TicketApiService.getTicketComments(id),
    enabled: !!id,
    staleTime: 1 * 60 * 1000, // 1分钟
    cacheTime: 5 * 60 * 1000, // 5分钟
  });
}

/**
 * 获取工单附件
 */
export function useTicketAttachments(id: number) {
  return useQuery({
    queryKey: QUERY_KEYS.TICKET_ATTACHMENTS(id),
    queryFn: () => TicketApiService.getTicketAttachments(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000, // 5分钟
    cacheTime: 10 * 60 * 1000, // 10分钟
  });
}

// ==================== 变更操作 Hooks ====================

/**
 * 创建工单
 */
export function useCreateTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: TicketApiService.createTicket,
    onSuccess: () => {
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('创建工单失败:', error);
    },
  });
}

/**
 * 更新工单
 */
export function useUpdateTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Ticket> }) =>
      TicketApiService.updateTicket(id, data),
    onSuccess: (data, variables) => {
      // 更新工单详情缓存
      queryClient.setQueryData(QUERY_KEYS.TICKET_DETAIL(variables.id), data);
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('更新工单失败:', error);
    },
  });
}

/**
 * 删除工单
 */
export function useDeleteTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: TicketApiService.deleteTicket,
    onSuccess: (_, id) => {
      // 移除工单详情缓存
      queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(id) });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('删除工单失败:', error);
    },
  });
}

/**
 * 批量删除工单
 */
export function useBatchDeleteTickets() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: TicketApiService.batchDeleteTickets,
    onSuccess: (_, ids) => {
      // 移除相关工单详情缓存
      ids.forEach(id => {
        queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(id) });
      });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('批量删除工单失败:', error);
    },
  });
}

/**
 * 分配工单
 */
export function useAssignTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, assigneeId, comment }: { id: number; assigneeId: number; comment?: string }) =>
      TicketApiService.assignTicket(id, assigneeId, comment),
    onSuccess: (_, variables) => {
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('分配工单失败:', error);
    },
  });
}

/**
 * 解决工单
 */
export function useResolveTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, resolution, category }: { id: number; resolution: string; category?: string }) =>
      TicketApiService.resolveTicket(id, resolution, category),
    onSuccess: (_, variables) => {
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('解决工单失败:', error);
    },
  });
}

/**
 * 关闭工单
 */
export function useCloseTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason, rating, comment }: { 
      id: number; 
      reason?: string; 
      rating?: number; 
      comment?: string; 
    }) => TicketApiService.closeTicket(id, reason, rating, comment),
    onSuccess: (_, variables) => {
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('关闭工单失败:', error);
    },
  });
}

/**
 * 重新打开工单
 */
export function useReopenTicket() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason?: string }) =>
      TicketApiService.reopenTicket(id, reason),
    onSuccess: (_, variables) => {
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
      
      // 使工单列表缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKETS });
    },
    onError: (error) => {
      console.error('重新打开工单失败:', error);
    },
  });
}

/**
 * 添加评论
 */
export function useAddComment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, content, isInternal }: { id: number; content: string; isInternal?: boolean }) =>
      TicketApiService.addComment(id, content, isInternal),
    onSuccess: (data, variables) => {
      // 更新评论缓存
      queryClient.setQueryData(QUERY_KEYS.TICKET_COMMENTS(variables.id), (oldData: Comment[] = []) => [
        ...oldData,
        data,
      ]);
      
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
    },
    onError: (error) => {
      console.error('添加评论失败:', error);
    },
  });
}

/**
 * 上传附件
 */
export function useUploadAttachment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) =>
      TicketApiService.uploadAttachment(id, file),
    onSuccess: (data, variables) => {
      // 更新附件缓存
      queryClient.setQueryData(QUERY_KEYS.TICKET_ATTACHMENTS(variables.id), (oldData: Attachment[] = []) => [
        ...oldData,
        data,
      ]);
      
      // 使工单详情缓存失效
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(variables.id) });
    },
    onError: (error) => {
      console.error('上传附件失败:', error);
    },
  });
}

// ==================== 缓存管理工具 ====================

/**
 * 缓存管理工具
 */
export class TicketCacheManager {
  private queryClient: any;

  constructor(queryClient: any) {
    this.queryClient = queryClient;
  }

  /**
   * 预加载工单详情
   */
  async prefetchTicket(id: number) {
    await this.queryClient.prefetchQuery({
      queryKey: QUERY_KEYS.TICKET_DETAIL(id),
      queryFn: () => TicketApiService.getTicket(id),
      staleTime: 2 * 60 * 1000,
    });
  }

  /**
   * 预加载工单列表
   */
  async prefetchTickets(filters?: TicketFilters) {
    await this.queryClient.prefetchQuery({
      queryKey: QUERY_KEYS.TICKET_LIST(filters),
      queryFn: () => TicketApiService.getTickets({ filters }),
      staleTime: 5 * 60 * 1000,
    });
  }

  /**
   * 清除所有工单相关缓存
   */
  clearAllTicketCache() {
    this.queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKETS });
  }

  /**
   * 清除特定工单的缓存
   */
  clearTicketCache(id: number) {
    this.queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKET_DETAIL(id) });
    this.queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKET_COMMENTS(id) });
    this.queryClient.removeQueries({ queryKey: QUERY_KEYS.TICKET_ATTACHMENTS(id) });
  }

  /**
   * 更新工单缓存
   */
  updateTicketCache(id: number, data: Partial<Ticket>) {
    const queryKey = QUERY_KEYS.TICKET_DETAIL(id);
    const oldData = this.queryClient.getQueryData(queryKey);
    
    if (oldData) {
      this.queryClient.setQueryData(queryKey, { ...oldData, ...data });
    }
  }
}

export default {
  TicketApiService,
  useTickets,
  useTicket,
  useTicketComments,
  useTicketAttachments,
  useCreateTicket,
  useUpdateTicket,
  useDeleteTicket,
  useBatchDeleteTickets,
  useAssignTicket,
  useResolveTicket,
  useCloseTicket,
  useReopenTicket,
  useAddComment,
  useUploadAttachment,
  TicketCacheManager,
};
