/**
 * 工单评论API
 * 提供工单评论的创建、查询、更新、删除功能
 */

import { httpClient } from './http-client';

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
    role?: string;
    department?: string;
    tenantId?: number;
  };
  createdAt: string;
  updatedAt: string;
}

export interface CreateTicketCommentRequest {
  content: string;
  isInternal?: boolean;
  mentions?: number[];
  attachments?: number[];
}

export interface UpdateTicketCommentRequest {
  content?: string;
  isInternal?: boolean;
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
    const response = await httpClient.get(`/api/v1/tickets/${ticketId}/comments`);
    // Type guard to extract data from possible wrapped response
    const rawData = ((response as { data?: unknown })?.data ?? response) as
      | { comments?: TicketComment[]; total?: number }
      | TicketComment[];
    
    let comments: TicketComment[] = [];
    let total: number = 0;
    
    if (Array.isArray(rawData)) {
      comments = rawData;
      total = rawData.length;
    } else {
      comments = Array.isArray(rawData?.comments) ? rawData.comments : [];
      total = typeof rawData?.total === 'number' ? rawData.total : comments.length;
    }
    
    return { comments, total };
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
    return httpClient.put<TicketComment>(`/api/v1/tickets/${ticketId}/comments/${commentId}`, data);
  }

  /**
   * 删除工单评论
   */
  static async deleteComment(ticketId: number, commentId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/comments/${commentId}`);
  }
}
