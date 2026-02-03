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
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const response: any = await httpClient.get(`/api/v1/tickets/${ticketId}/comments`);
    const data = response?.data || response;
    const comments = Array.isArray(data?.comments) ? data.comments : Array.isArray(data) ? data : [];
    return {
      comments,
      total: typeof data?.total === 'number' ? data.total : comments.length,
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

