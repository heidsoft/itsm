/**
 * 工单评论API
 * 提供工单评论的创建、查询、更新、删除功能
 */

import { httpClient } from './http-client';

export interface TicketComment {
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
}

export interface CreateTicketCommentRequest {
  content: string;
  is_internal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

export interface UpdateTicketCommentRequest {
  content?: string;
  is_internal?: boolean;
  mentions?: number[];
}

export interface ListTicketCommentsResponse {
  comments: TicketComment[];
  total: number;
}

export class TicketCommentApi {
  /**
   * 获取工单评论列表
   */
  static async getComments(ticketId: number): Promise<ListTicketCommentsResponse> {
    const response = await httpClient.get<{
      code: number;
      data: ListTicketCommentsResponse;
    }>(`/api/v1/tickets/${ticketId}/comments`);
    
    // 处理响应格式
    if (response && 'data' in response) {
      return {
        comments: Array.isArray(response.data?.comments) ? response.data.comments : [],
        total: response.data?.total || 0,
      };
    }
    
    // 如果直接返回数据
    if (Array.isArray(response)) {
      return {
        comments: response,
        total: response.length,
      };
    }
    
    return {
      comments: [],
      total: 0,
    };
  }

  /**
   * 创建工单评论
   */
  static async createComment(
    ticketId: number,
    data: CreateTicketCommentRequest
  ): Promise<TicketComment> {
    return httpClient.post<TicketComment>(`/api/v1/tickets/${ticketId}/comments`, data);
  }

  /**
   * 更新工单评论
   */
  static async updateComment(
    ticketId: number,
    commentId: number,
    data: UpdateTicketCommentRequest
  ): Promise<TicketComment> {
    return httpClient.put<TicketComment>(
      `/api/v1/tickets/${ticketId}/comments/${commentId}`,
      data
    );
  }

  /**
   * 删除工单评论
   */
  static async deleteComment(ticketId: number, commentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/comments/${commentId}`);
  }
}

