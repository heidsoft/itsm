import { httpClient } from './http-client';

/**
 * 用户通知偏好设置接口
 */

// 通知偏好设置类型
export interface NotificationPreference {
  id: number;
  userId: number;
  emailEnabled: boolean;
  smsEnabled: boolean;
  pushEnabled: boolean;
  wechatEnabled: boolean;
  notificationTypes: string[];
  quietHoursStart?: string;
  quietHoursEnd?: string;
  language: string;
  timezone: string;
  createdAt: string;
  updatedAt: string;
}

// 创建/更新通知偏好请求
export interface NotificationPreferenceRequest {
  emailEnabled?: boolean;
  smsEnabled?: boolean;
  pushEnabled?: boolean;
  wechatEnabled?: boolean;
  notificationTypes?: string[];
  quietHoursStart?: string;
  quietHoursEnd?: string;
  language?: string;
  timezone?: string;
}

/**
 * 通知偏好设置 API
 */
export class NotificationPreferenceApi {
  private static baseURL = '/api/v1/notification-preferences';

  /**
   * 获取当前用户的通知偏好设置列表（兼容profile页面）
   * 返回 { preferences: [...], event_types: [...] } 格式
   */
  static async getPreferences(): Promise<{
    preferences: Array<{
      event_type: string;
      email_enabled: boolean;
      in_app_enabled: boolean;
      timezone?: string;
    }>;
    event_types: string[];
  }> {
    return httpClient.get<{
      preferences: Array<{
        event_type: string;
        email_enabled: boolean;
        in_app_enabled: boolean;
        timezone?: string;
      }>;
      event_types: string[];
    }>(`${this.baseURL}`);
  }

  /**
   * 获取当前用户的通知偏好设置
   */
  static async getMyPreferences(): Promise<NotificationPreference> {
    return httpClient.get<NotificationPreference>(`${this.baseURL}`);
  }

  /**
   * 获取指定用户的通知偏好设置
   */
  static async getPreferencesByUserId(userId: number): Promise<NotificationPreference> {
    return httpClient.get<NotificationPreference>(`${this.baseURL}/${userId}`);
  }

  /**
   * 更新当前用户的通知偏好设置
   */
  static async updateMyPreferences(
    preferences: NotificationPreferenceRequest
  ): Promise<NotificationPreference> {
    return httpClient.put<NotificationPreference>(`${this.baseURL}/me`, preferences);
  }

  /**
   * 更新指定用户的通知偏好设置
   */
  static async updatePreferences(
    userId: number,
    preferences: NotificationPreferenceRequest
  ): Promise<NotificationPreference> {
    return httpClient.put<NotificationPreference>(`${this.baseURL}/${userId}`, preferences);
  }

  /**
   * 重置通知偏好设置为默认值
   */
  static async resetToDefault(): Promise<NotificationPreference> {
    return httpClient.post<NotificationPreference>(`${this.baseURL}/me/reset`);
  }

  /**
   * 获取通知偏好设置模板列表
   */
  static async getTemplates(): Promise<NotificationPreference[]> {
    return httpClient.get<NotificationPreference[]>(`${this.baseURL}/templates`);
  }

  /**
   * 应用通知偏好设置模板
   */
  static async applyTemplate(templateId: number): Promise<NotificationPreference> {
    return httpClient.post<NotificationPreference>(`${this.baseURL}/me/apply-template`, {
      templateId,
    });
  }

  /**
   * 批量更新通知偏好设置
   */
  static async bulkUpdate(data: {
    preferences: Array<{
      event_type: string;
      email_enabled: boolean;
      in_app_enabled: boolean;
      timezone?: string;
    }>;
  }): Promise<{ preferences: unknown[] }> {
    return httpClient.put<{ preferences: unknown[] }>(`${this.baseURL}`, data);
  }
}