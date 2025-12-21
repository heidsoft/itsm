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

export interface NotificationPreferences {
  user_id: number;
  email_enabled: boolean;
  in_app_enabled: boolean;
  sms_enabled: boolean;
  sla_warning_time: number;
}

export interface UpdateNotificationPreferencesRequest {
  email_enabled: boolean;
  in_app_enabled: boolean;
  sms_enabled: boolean;
  sla_warning_time: number;
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
  static async getNotificationPreferences(userId: number): Promise<NotificationPreferences> {
    return httpClient.get<NotificationPreferences>(`/api/v1/users/${userId}/notification-preferences`);
  }

  // 更新用户通知偏好
  static async updateNotificationPreferences(
    userId: number,
    data: UpdateNotificationPreferencesRequest
  ): Promise<NotificationPreferences> {
    return httpClient.put<NotificationPreferences>(`/api/v1/users/${userId}/notification-preferences`, data);
  }
}

