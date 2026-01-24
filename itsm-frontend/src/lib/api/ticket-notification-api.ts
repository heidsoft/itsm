import { httpClient } from './http-client';

// 工单通知类型定义
export interface TicketNotification {
  id: number;
  ticket_id: number;
  user_id: number;
  type: 'created' | 'assigned' | 'status_changed' | 'commented' | 'sla_warning' | 'resolved' | 'closed';
  channel: 'email' | 'in_app' | 'sms';
  content: string;
  sent_at?: string;
  read_at?: string;
  status: 'pending' | 'sent' | 'read';
  created_at: string;
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role?: string;
    department?: string;
    tenant_id?: number;
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
  page_size: number;
}

export interface SendTicketNotificationRequest {
  user_ids: number[];
  type: string;
  channel: 'email' | 'in_app' | 'sms';
  content: string;
}

export interface NotificationPreferenceItem {
  event_type: string;
  email_enabled: boolean;
  in_app_enabled: boolean;
  sms_enabled: boolean;
}

export interface NotificationPreferencesResponse {
  preferences: NotificationPreferenceItem[];
  event_types: Array<{
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
    return httpClient.get<ListTicketNotificationsResponse>(`/api/v1/tickets/${ticketId}/notifications`);
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
    page_size?: number;
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

  // 获取用户通知偏好
  static async getNotificationPreferences(): Promise<NotificationPreferencesResponse> {
    return httpClient.get<NotificationPreferencesResponse>('/api/v1/notification-preferences');
  }

  // 更新用户通知偏好（单个或批量）
  static async updateNotificationPreferences(
    data: BulkUpdatePreferencesRequest
  ): Promise<NotificationPreferencesResponse> {
    return httpClient.put<NotificationPreferencesResponse>('/api/v1/notification-preferences', data);
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

