import { httpClient } from './http-client';

// 事件状态枚举
export const INCIDENT_STATUS = {
  NEW: 'new',
  IN_PROGRESS: 'in_progress',
  RESOLVED: 'resolved',
  CLOSED: 'closed',
  CANCELLED: 'cancelled',
} as const;

// 事件优先级枚举
export const INCIDENT_PRIORITY = {
  LOW: 'low',
  MEDIUM: 'medium',
  HIGH: 'high',
  CRITICAL: 'critical',
} as const;

// 事件来源枚举
export const INCIDENT_SOURCE = {
  EMAIL: 'email',
  PHONE: 'phone',
  WEB: 'web',
  API: 'api',
  MONITORING: 'monitoring',
  CHAT: 'chat',
} as const;

// 事件类型枚举
export const INCIDENT_TYPE = {
  HARDWARE: 'hardware',
  SOFTWARE: 'software',
  NETWORK: 'network',
  SECURITY: 'security',
  ACCESS: 'access',
  OTHER: 'other',
} as const;

// 事件数据接口
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  source: string;
  type: string;
  incident_number: string;
  is_major_incident: boolean;
  requester_id: number;
  requester_name?: string;
  requester_email?: string;
  assignee_id?: number;
  assignee_name?: string;
  assignee_email?: string;
  category_id?: number;
  category_name?: string;
  subcategory_id?: number;
  subcategory_name?: string;
  resolution?: string;
  resolution_time?: string;
  first_response_time?: string;
  form_fields?: Record<string, string | number | boolean>;
  attachments?: string[];
  tags?: string[];
  created_at: string;
  updated_at: string;
}

// 事件列表请求参数
export interface ListIncidentsRequest extends Record<string, unknown> {
  page?: number;
  page_size?: number;
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  assignee_id?: number;
  requester_id?: number;
  category_id?: number;
  is_major_incident?: boolean;
  keyword?: string;
  date_from?: string;
  date_to?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 事件列表响应
export interface ListIncidentsResponse {
  incidents: Incident[];
  total: number;
  page: number;
  page_size: number;
}

// 创建事件请求
export interface CreateIncidentRequest {
  title: string;
  description: string;
  priority: string;
  source: string;
  type: string;
  requester_id?: number;
  assignee_id?: number;
  category_id?: number;
  subcategory_id?: number;
  is_major_incident?: boolean;
  form_fields?: Record<string, string | number | boolean>;
  tags?: string[];
}

// 更新事件请求
export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  status?: string;
  priority?: string;
  assignee_id?: number;
  category_id?: number;
  subcategory_id?: number;
  resolution?: string;
  is_major_incident?: boolean;
  form_fields?: Record<string, string | number | boolean>;
  tags?: string[];
}

// 事件统计数据
export interface IncidentStats {
  total: number;
  by_status: Record<string, number>;
  by_priority: Record<string, number>;
  by_type: Record<string, number>;
  by_source: Record<string, number>;
  avg_resolution_time: number;
  avg_first_response_time: number;
  sla_compliance_rate: number;
}

// 事件时间线条目
export interface IncidentTimelineEntry {
  id: number;
  incident_id: number;
  action: string;
  description: string;
  user_id: number;
  user_name: string;
  created_at: string;
  metadata?: Record<string, string | number | boolean>;
}

// 事件评论
export interface IncidentComment {
  id: number;
  incident_id: number;
  content: string;
  user_id: number;
  user_name: string;
  user_avatar?: string;
  is_internal: boolean;
  created_at: string;
  updated_at: string;
}

// 创建评论请求
export interface CreateCommentRequest {
  content: string;
  is_internal?: boolean;
}

// 事件API类
export class IncidentAPI {
  // 获取事件列表
  static async listIncidents(params: ListIncidentsRequest = {}): Promise<ListIncidentsResponse> {
    try {
      console.log('IncidentAPI.listIncidents called with params:', params);
      const response = await httpClient.get<ListIncidentsResponse>('/api/incidents', params);
      console.log('IncidentAPI.listIncidents response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.listIncidents error:', error);
      throw error;
    }
  }

  // 获取单个事件详情
  static async getIncident(id: number): Promise<Incident> {
    try {
      console.log('IncidentAPI.getIncident called with id:', id);
      const response = await httpClient.get<Incident>(`/api/incidents/${id}`);
      console.log('IncidentAPI.getIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncident error:', error);
      throw error;
    }
  }

  // 创建事件
  static async createIncident(data: CreateIncidentRequest): Promise<Incident> {
    try {
      console.log('IncidentAPI.createIncident called with data:', data);
      const response = await httpClient.post<Incident>('/api/incidents', data);
      console.log('IncidentAPI.createIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.createIncident error:', error);
      throw error;
    }
  }

  // 更新事件
  static async updateIncident(id: number, data: UpdateIncidentRequest): Promise<Incident> {
    try {
      console.log('IncidentAPI.updateIncident called with id:', id, 'data:', data);
      const response = await httpClient.put<Incident>(`/api/incidents/${id}`, data);
      console.log('IncidentAPI.updateIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateIncident error:', error);
      throw error;
    }
  }

  // 删除事件
  static async deleteIncident(id: number): Promise<void> {
    try {
      console.log('IncidentAPI.deleteIncident called with id:', id);
      await httpClient.delete(`/api/incidents/${id}`);
      console.log('IncidentAPI.deleteIncident completed');
    } catch (error) {
      console.error('IncidentAPI.deleteIncident error:', error);
      throw error;
    }
  }

  // 分配事件
  static async assignIncident(id: number, assigneeId: number): Promise<Incident> {
    try {
      console.log('IncidentAPI.assignIncident called with id:', id, 'assigneeId:', assigneeId);
      const response = await httpClient.put<Incident>(`/api/incidents/${id}/assign`, { assignee_id: assigneeId });
      console.log('IncidentAPI.assignIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.assignIncident error:', error);
      throw error;
    }
  }

  // 关闭事件
  static async closeIncident(id: number, resolution: string): Promise<Incident> {
    try {
      console.log('IncidentAPI.closeIncident called with id:', id, 'resolution:', resolution);
      const response = await httpClient.put<Incident>(`/api/incidents/${id}/close`, { resolution });
      console.log('IncidentAPI.closeIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.closeIncident error:', error);
      throw error;
    }
  }

  // 重新打开事件
  static async reopenIncident(id: number, reason: string): Promise<Incident> {
    try {
      console.log('IncidentAPI.reopenIncident called with id:', id, 'reason:', reason);
      const response = await httpClient.put<Incident>(`/api/incidents/${id}/reopen`, { reason });
      console.log('IncidentAPI.reopenIncident response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.reopenIncident error:', error);
      throw error;
    }
  }

  // 获取事件统计
  static async getIncidentStats(params?: { date_from?: string; date_to?: string }): Promise<IncidentStats> {
    try {
      console.log('IncidentAPI.getIncidentStats called with params:', params);
      const response = await httpClient.get<IncidentStats>('/api/incidents/stats', params);
      console.log('IncidentAPI.getIncidentStats response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncidentStats error:', error);
      throw error;
    }
  }

  // 获取事件时间线
  static async getIncidentTimeline(id: number): Promise<IncidentTimelineEntry[]> {
    try {
      console.log('IncidentAPI.getIncidentTimeline called with id:', id);
      const response = await httpClient.get<IncidentTimelineEntry[]>(`/api/incidents/${id}/timeline`);
      console.log('IncidentAPI.getIncidentTimeline response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncidentTimeline error:', error);
      throw error;
    }
  }

  // 获取事件评论
  static async getIncidentComments(id: number): Promise<IncidentComment[]> {
    try {
      console.log('IncidentAPI.getIncidentComments called with id:', id);
      const response = await httpClient.get<IncidentComment[]>(`/api/incidents/${id}/comments`);
      console.log('IncidentAPI.getIncidentComments response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncidentComments error:', error);
      throw error;
    }
  }

  // 添加事件评论
  static async addIncidentComment(id: number, data: CreateCommentRequest): Promise<IncidentComment> {
    try {
      console.log('IncidentAPI.addIncidentComment called with id:', id, 'data:', data);
      const response = await httpClient.post<IncidentComment>(`/api/incidents/${id}/comments`, data);
      console.log('IncidentAPI.addIncidentComment response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.addIncidentComment error:', error);
      throw error;
    }
  }

  // 批量操作事件
  static async bulkUpdateIncidents(ids: number[], data: Partial<UpdateIncidentRequest>): Promise<void> {
    try {
      console.log('IncidentAPI.bulkUpdateIncidents called with ids:', ids, 'data:', data);
      await httpClient.put('/api/incidents/bulk', { ids, ...data });
      console.log('IncidentAPI.bulkUpdateIncidents completed');
    } catch (error) {
      console.error('IncidentAPI.bulkUpdateIncidents error:', error);
      throw error;
    }
  }

  // 导出事件数据
  static async exportIncidents(params: ListIncidentsRequest & { format?: 'csv' | 'excel' }): Promise<Blob> {
    try {
      console.log('IncidentAPI.exportIncidents called with params:', params);
      const response = await httpClient.get('/api/incidents/export', params);
      console.log('IncidentAPI.exportIncidents completed');
      return response as Blob;
    } catch (error) {
      console.error('IncidentAPI.exportIncidents error:', error);
      throw error;
    }
  }
}

export default IncidentAPI;