import { httpClient } from './http-client';

// 工单通知类型定义
export interface TicketNotification {
  id: number;
  ticketId: number;
  userId: number;
  type:
    | 'created'
    | 'assigned'
    | 'status_changed'
    | 'commented'
    | 'sla_warning'
    | 'resolved'
    | 'closed';
  channel: 'email' | 'in_app' | 'sms';
  content: string;
  sentAt?: string;
  readAt?: string;
  status: 'pending' | 'sent' | 'read';
  createdAt: string;
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenantId?: number;
  };
}

export interface ListTicketNotificationsResponse {
  notifications: TicketNotification[];
  total: number;
}

export interface ListUserNotificationsResponse {
  notifications: TicketNotification[];
  total: number;
  page: number;
  pageSize: number;
}

export interface SendTicketNotificationRequest {
  userIds: number[];
  type: string;
  channel: 'email' | 'in_app' | 'sms';
  content: string;
}

export interface NotificationPreferenceItem {
  eventType: string;
  emailEnabled: boolean;
  inAppEnabled: boolean;
  smsEnabled: boolean;
}

export interface NotificationPreferencesResponse {
  preferences: NotificationPreferenceItem[];
  eventTypes: Array<{
    type: string;
    name: string;
    description: string;
  }>;
}

export interface BulkUpdatePreferencesRequest {
  preferences: NotificationPreferenceItem[];
}

export class TicketNotificationApi {
  // 获取工单通知列表
  static async getTicketNotifications(ticketId: number): Promise<ListTicketNotificationsResponse> {
    return httpClient.get<ListTicketNotificationsResponse>(
      `/api/v1/tickets/${ticketId}/notifications`
    );
  }

  // 发送工单通知
  static async sendTicketNotification(
    ticketId: number,
    data: SendTicketNotificationRequest
  ): Promise<void> {
    return httpClient.post(`/api/v1/tickets/${ticketId}/notifications`, data);
  }

  // 获取用户通知列表
  static async getUserNotifications(params?: {
    page?: number;
    pageSize?: number;
    read?: boolean;
  }): Promise<ListUserNotificationsResponse> {
    return httpClient.get<ListUserNotificationsResponse>('/api/v1/notifications', params);
  }

  // 标记通知为已读
  static async markNotificationRead(notificationId: number): Promise<void> {
    return httpClient.put(`/api/v1/notifications/${notificationId}/read`, {});
  }

  // 标记所有通知为已读
  static async markAllNotificationsRead(): Promise<void> {
    return httpClient.put('/api/v1/notifications/read-all', {});
  }

  // 删除单条通知
  static async deleteNotification(notificationId: number): Promise<void> {
    return httpClient.delete(`/api/v1/notifications/${notificationId}`);
  }

  // 创建通知（仅管理员）
  static async createNotification(payload: {
    type?: string;
    title: string;
    content: string;
    userId?: number;
    level?: 'info' | 'warning' | 'error' | 'success';
    metadata?: Record<string, unknown>;
  }): Promise<unknown> {
    return httpClient.post('/api/v1/notifications', payload);
  }

  // 获取用户通知偏好
  static async getNotificationPreferences(): Promise<NotificationPreferencesResponse> {
    return httpClient.get<NotificationPreferencesResponse>('/api/v1/notification-preferences');
  }

  // 更新用户通知偏好（单个或批量）
  static async updateNotificationPreferences(
    data: BulkUpdatePreferencesRequest
  ): Promise<NotificationPreferencesResponse> {
    return httpClient.put<NotificationPreferencesResponse>(
      '/api/v1/notification-preferences',
      data
    );
  }

  // 重置为默认偏好
  static async resetNotificationPreferences(): Promise<{ reset: boolean }> {
    return httpClient.post<{ reset: boolean }>('/api/v1/notification-preferences/reset', {});
  }

  // 初始化默认通知偏好
  static async initNotificationPreferences(): Promise<{ initialized: boolean }> {
    return httpClient.post<{ initialized: boolean }>('/api/v1/notification-preferences/init', {});
  }
}
