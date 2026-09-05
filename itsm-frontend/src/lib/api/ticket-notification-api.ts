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
    | 'closed'
    | 'info';
  channel: 'email' | 'in_app' | 'sms';
  content: string;
  /** Raw title from API (often an i18n key) */
  title?: string;
  /** Deep-link URL to the related entity */
  actionUrl?: string;
  /** Human-readable link text */
  actionText?: string;
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

/**
 * 用户通知契约 —— 与后端 dto.Notification（GET /api/v1/notifications 真实
 * 响应体）逐字段对齐：字段是 message/read，而不是 content/status。
 */
export interface UserNotification {
  id: number;
  title: string;
  message: string;
  type: string;
  read: boolean;
  actionUrl?: string;
  actionText?: string;
  userId: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface ListUserNotificationsResponse {
  notifications: UserNotification[];
  total: number;
  page: number;
  /** 后端分页字段是 size（不是 pageSize） */
  size: number;
}

const TICKET_EVENT_TYPES: ReadonlySet<string> = new Set([
  'created',
  'assigned',
  'status_changed',
  'commented',
  'sla_warning',
  'resolved',
  'closed',
  'info',
]);

/**
 * Maps the raw user-notification contract to the UI-facing TicketNotification
 * shape: message→content, read(bool)→status(enum). Single source of truth for
 * the badge count and read-state rendering (Header / notifications page / WS).
 */
export function toTicketNotification(raw: UserNotification): TicketNotification {
  const type = TICKET_EVENT_TYPES.has(raw.type) ? (raw.type as TicketNotification['type']) : 'info';
  return {
    id: raw.id,
    ticketId: 0,
    userId: raw.userId,
    type,
    channel: 'in_app',
    content: raw.message || '',
    title: raw.title || '',
    actionUrl: raw.actionUrl,
    actionText: raw.actionText,
    readAt: raw.read ? raw.updatedAt : undefined,
    sentAt: raw.createdAt,
    status: raw.read ? 'read' : 'sent',
    createdAt: raw.createdAt,
  };
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

  // 获取用户通知列表（后端 query 契约：page/size/read/type）
  static async getUserNotifications(params?: {
    page?: number;
    size?: number;
    read?: boolean;
    type?: string;
  }): Promise<ListUserNotificationsResponse> {
    return httpClient.get<ListUserNotificationsResponse>('/api/v1/notifications', params);
  }

  // 未读通知数（服务端权威计数，用于 badge）
  static async getUnreadCount(): Promise<{ count: number }> {
    return httpClient.get<{ count: number }>('/api/v1/notifications/unread-count');
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
