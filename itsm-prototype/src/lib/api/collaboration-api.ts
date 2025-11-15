/**
 * 协作 API 服务
 * 提供评论、@提及、通知等协作功能的API接口
 */

import { httpClient } from './http-client';
import type {
  Comment,
  CreateCommentRequest,
  UpdateCommentRequest,
  CommentListQuery,
  CommentListResponse,
  CommentStats,
  Mention,
  MentionSuggestion,
  Notification,
  NotificationListQuery,
  NotificationListResponse,
  NotificationSettings,
  Activity,
  ActivityQuery,
  ActivityFeed,
  Watcher,
  CollaborationStats,
  TypingIndicator,
  OnlinePresence,
} from '@/types/collaboration';

export class CollaborationApi {
  // ==================== 评论管理 ====================

  /**
   * 获取工单评论列表
   */
  static async getComments(
    query: CommentListQuery
  ): Promise<CommentListResponse> {
    return httpClient.get<CommentListResponse>(
      `/api/v1/tickets/${query.ticketId}/comments`,
      {
        type: query.type,
        includeDeleted: query.includeDeleted,
        includeInternal: query.includeInternal,
        sortBy: query.sortBy,
        sortOrder: query.sortOrder,
        page: query.page,
        pageSize: query.pageSize,
      }
    );
  }

  /**
   * 获取单个评论
   */
  static async getComment(commentId: string): Promise<Comment> {
    return httpClient.get<Comment>(`/api/v1/comments/${commentId}`);
  }

  /**
   * 创建评论
   */
  static async createComment(request: CreateCommentRequest): Promise<Comment> {
    return httpClient.post<Comment>(
      `/api/v1/tickets/${request.ticketId}/comments`,
      request
    );
  }

  /**
   * 更新评论
   */
  static async updateComment(
    commentId: string,
    request: UpdateCommentRequest
  ): Promise<Comment> {
    return httpClient.put<Comment>(
      `/api/v1/comments/${commentId}`,
      request
    );
  }

  /**
   * 删除评论
   */
  static async deleteComment(commentId: string): Promise<void> {
    return httpClient.delete(`/api/v1/comments/${commentId}`);
  }

  /**
   * 点赞评论
   */
  static async likeComment(commentId: string): Promise<void> {
    return httpClient.post(`/api/v1/comments/${commentId}/like`);
  }

  /**
   * 取消点赞
   */
  static async unlikeComment(commentId: string): Promise<void> {
    return httpClient.delete(`/api/v1/comments/${commentId}/like`);
  }

  /**
   * 置顶评论
   */
  static async pinComment(commentId: string): Promise<void> {
    return httpClient.post(`/api/v1/comments/${commentId}/pin`);
  }

  /**
   * 取消置顶
   */
  static async unpinComment(commentId: string): Promise<void> {
    return httpClient.delete(`/api/v1/comments/${commentId}/pin`);
  }

  /**
   * 获取评论回复
   */
  static async getCommentReplies(commentId: string): Promise<Comment[]> {
    return httpClient.get<Comment[]>(
      `/api/v1/comments/${commentId}/replies`
    );
  }

  /**
   * 获取评论统计
   */
  static async getCommentStats(ticketId: number): Promise<CommentStats> {
    return httpClient.get<CommentStats>(
      `/api/v1/tickets/${ticketId}/comments/stats`
    );
  }

  // ==================== @提及 ====================

  /**
   * 搜索可提及的用户/团队
   */
  static async searchMentionSuggestions(params: {
    query: string;
    ticketId?: number;
    types?: string[];
    limit?: number;
  }): Promise<MentionSuggestion[]> {
    return httpClient.get<MentionSuggestion[]>(
      '/api/v1/mentions/suggestions',
      params
    );
  }

  /**
   * 获取我被@提及的列表
   */
  static async getMyMentions(params?: {
    isRead?: boolean;
    startDate?: string;
    endDate?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    mentions: Mention[];
    total: number;
  }> {
    return httpClient.get('/api/v1/mentions/me', params);
  }

  /**
   * 标记@提及为已读
   */
  static async markMentionAsRead(mentionId: string): Promise<void> {
    return httpClient.post(`/api/v1/mentions/${mentionId}/read`);
  }

  // ==================== 通知管理 ====================

  /**
   * 获取通知列表
   */
  static async getNotifications(
    query: NotificationListQuery
  ): Promise<NotificationListResponse> {
    return httpClient.get<NotificationListResponse>(
      '/api/v1/notifications',
      query
    );
  }

  /**
   * 获取未读通知数量
   */
  static async getUnreadCount(): Promise<{ count: number }> {
    return httpClient.get<{ count: number }>(
      '/api/v1/notifications/unread-count'
    );
  }

  /**
   * 标记通知为已读
   */
  static async markNotificationAsRead(notificationId: string): Promise<void> {
    return httpClient.post(`/api/v1/notifications/${notificationId}/read`);
  }

  /**
   * 标记所有通知为已读
   */
  static async markAllNotificationsAsRead(): Promise<void> {
    return httpClient.post('/api/v1/notifications/mark-all-read');
  }

  /**
   * 删除通知
   */
  static async deleteNotification(notificationId: string): Promise<void> {
    return httpClient.delete(`/api/v1/notifications/${notificationId}`);
  }

  /**
   * 清空所有通知
   */
  static async clearAllNotifications(): Promise<void> {
    return httpClient.delete('/api/v1/notifications/clear-all');
  }

  /**
   * 获取通知设置
   */
  static async getNotificationSettings(): Promise<NotificationSettings> {
    return httpClient.get<NotificationSettings>(
      '/api/v1/notifications/settings'
    );
  }

  /**
   * 更新通知设置
   */
  static async updateNotificationSettings(
    settings: Partial<NotificationSettings>
  ): Promise<NotificationSettings> {
    return httpClient.put<NotificationSettings>(
      '/api/v1/notifications/settings',
      settings
    );
  }

  // ==================== 协作历史 ====================

  /**
   * 获取工单活动流
   */
  static async getActivities(query: ActivityQuery): Promise<ActivityFeed> {
    return httpClient.get<ActivityFeed>(
      `/api/v1/tickets/${query.ticketId}/activities`,
      {
        types: query.types,
        actorId: query.actorId,
        startDate: query.startDate,
        endDate: query.endDate,
        cursor: query.cursor,
        limit: query.limit,
      }
    );
  }

  /**
   * 获取单个活动详情
   */
  static async getActivity(activityId: string): Promise<Activity> {
    return httpClient.get<Activity>(`/api/v1/activities/${activityId}`);
  }

  /**
   * 获取用户的活动历史
   */
  static async getUserActivities(params: {
    userId: number;
    startDate?: string;
    endDate?: string;
    limit?: number;
  }): Promise<Activity[]> {
    return httpClient.get<Activity[]>(
      `/api/v1/users/${params.userId}/activities`,
      {
        startDate: params.startDate,
        endDate: params.endDate,
        limit: params.limit,
      }
    );
  }

  // ==================== 观察者管理 ====================

  /**
   * 获取观察者列表
   */
  static async getWatchers(ticketId: number): Promise<Watcher[]> {
    return httpClient.get<Watcher[]>(
      `/api/v1/tickets/${ticketId}/watchers`
    );
  }

  /**
   * 添加观察者
   */
  static async addWatcher(data: {
    ticketId: number;
    userId: number;
  }): Promise<Watcher> {
    return httpClient.post<Watcher>(
      `/api/v1/tickets/${data.ticketId}/watchers`,
      { userId: data.userId }
    );
  }

  /**
   * 移除观察者
   */
  static async removeWatcher(ticketId: number, watcherId: string): Promise<void> {
    return httpClient.delete(
      `/api/v1/tickets/${ticketId}/watchers/${watcherId}`
    );
  }

  /**
   * 添加自己为观察者
   */
  static async watchTicket(ticketId: number): Promise<Watcher> {
    return httpClient.post<Watcher>(
      `/api/v1/tickets/${ticketId}/watch`
    );
  }

  /**
   * 取消观察
   */
  static async unwatchTicket(ticketId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${ticketId}/watch`);
  }

  // ==================== 协作统计 ====================

  /**
   * 获取协作统计
   */
  static async getCollaborationStats(
    ticketId: number
  ): Promise<CollaborationStats> {
    return httpClient.get<CollaborationStats>(
      `/api/v1/tickets/${ticketId}/collaboration-stats`
    );
  }

  /**
   * 获取参与者列表
   */
  static async getParticipants(ticketId: number): Promise<
    Array<{
      userId: number;
      userName: string;
      userAvatar?: string;
      commentCount: number;
      lastActivityAt: Date;
    }>
  > {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/participants`
    );
  }

  // ==================== 实时协作 ====================

  /**
   * 获取在线状态
   */
  static async getOnlinePresence(userIds: number[]): Promise<OnlinePresence[]> {
    return httpClient.post<OnlinePresence[]>(
      '/api/v1/presence/batch',
      { userIds }
    );
  }

  /**
   * 更新在线状态
   */
  static async updatePresence(data: {
    status: 'online' | 'away' | 'busy' | 'offline';
    currentActivity?: string;
  }): Promise<void> {
    return httpClient.post('/api/v1/presence/update', data);
  }

  /**
   * 发送正在输入状态
   */
  static async sendTypingIndicator(ticketId: number): Promise<void> {
    return httpClient.post(`/api/v1/tickets/${ticketId}/typing`);
  }

  /**
   * 获取正在输入的用户
   */
  static async getTypingUsers(ticketId: number): Promise<TypingIndicator[]> {
    return httpClient.get<TypingIndicator[]>(
      `/api/v1/tickets/${ticketId}/typing`
    );
  }

  // ==================== 附件管理 ====================

  /**
   * 上传评论附件
   */
  static async uploadAttachment(data: {
    file: File;
    commentId?: string;
    ticketId: number;
  }): Promise<{
    id: string;
    fileName: string;
    fileUrl: string;
    fileSize: number;
    fileType: string;
  }> {
    const formData = new FormData();
    formData.append('file', data.file);
    if (data.commentId) {
      formData.append('commentId', data.commentId);
    }

    return httpClient.post(
      `/api/v1/tickets/${data.ticketId}/attachments`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );
  }

  /**
   * 删除附件
   */
  static async deleteAttachment(attachmentId: string): Promise<void> {
    return httpClient.delete(`/api/v1/attachments/${attachmentId}`);
  }

  // ==================== 批量操作 ====================

  /**
   * 批量删除评论
   */
  static async batchDeleteComments(commentIds: string[]): Promise<{
    deleted: number;
    failed: number;
  }> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/comments/batch',
      data: { commentIds },
    });
  }

  /**
   * 批量标记通知为已读
   */
  static async batchMarkNotificationsAsRead(
    notificationIds: string[]
  ): Promise<void> {
    return httpClient.post('/api/v1/notifications/batch-read', {
      notificationIds,
    });
  }

  // ==================== 搜索和过滤 ====================

  /**
   * 搜索评论
   */
  static async searchComments(params: {
    query: string;
    ticketId?: number;
    authorId?: number;
    startDate?: string;
    endDate?: string;
    page?: number;
    pageSize?: number;
  }): Promise<CommentListResponse> {
    return httpClient.get<CommentListResponse>(
      '/api/v1/comments/search',
      params
    );
  }

  /**
   * 导出评论
   */
  static async exportComments(data: {
    ticketId: number;
    format: 'pdf' | 'excel' | 'markdown';
    includeAttachments?: boolean;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: `/api/v1/tickets/${data.ticketId}/comments/export`,
      data: {
        format: data.format,
        includeAttachments: data.includeAttachments,
      },
      responseType: 'blob',
    });
    return response as Blob;
  }
}

// 导出默认实例和类
export default CollaborationApi;

// 导出别名
export const CollaborationAPI = CollaborationApi;

