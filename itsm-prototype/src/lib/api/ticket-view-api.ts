/**
 * 工单视图API
 * 提供工单视图的创建、查询、更新、删除和共享功能
 */

import { httpClient } from './http-client';

// 筛选配置类型
export type FilterValue = string | number | boolean | string[] | number[] | null | undefined;
export type FilterConfig = Record<string, FilterValue>;

// 排序配置类型
export interface SortConfig {
  field: string;
  order: 'asc' | 'desc';
}

// 分组配置类型
export interface GroupConfig {
  field: string;
  order?: 'asc' | 'desc';
}

export interface TicketView {
  id: number;
  name: string;
  description?: string;
  filters: FilterConfig;
  columns: string[];
  sort_config: SortConfig;
  group_config?: GroupConfig;
  is_shared: boolean;
  created_by: number;
  creator?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role: string;
  };
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface CreateTicketViewRequest {
  name: string;
  description?: string;
  filters: FilterConfig;
  columns: string[];
  sort_config: SortConfig;
  group_config?: GroupConfig;
  is_shared: boolean;
}

export interface UpdateTicketViewRequest {
  name?: string;
  description?: string;
  filters?: FilterConfig;
  columns?: string[];
  sort_config?: SortConfig;
  group_config?: GroupConfig;
  is_shared?: boolean;
}

export interface ShareTicketViewRequest {
  team_ids: number[];
}

export interface ListTicketViewsResponse {
  views: TicketView[];
  total: number;
}

export class TicketViewApi {
  /**
   * 获取工单视图列表
   */
  static async listViews(): Promise<ListTicketViewsResponse> {
    return httpClient.get<ListTicketViewsResponse>('/api/v1/tickets/views');
  }

  /**
   * 获取工单视图详情
   */
  static async getView(viewId: number): Promise<TicketView> {
    return httpClient.get<TicketView>(`/api/v1/tickets/views/${viewId}`);
  }

  /**
   * 创建工单视图
   */
  static async createView(data: CreateTicketViewRequest): Promise<TicketView> {
    return httpClient.post<TicketView>('/api/v1/tickets/views', data);
  }

  /**
   * 更新工单视图
   */
  static async updateView(viewId: number, data: UpdateTicketViewRequest): Promise<TicketView> {
    return httpClient.put<TicketView>(`/api/v1/tickets/views/${viewId}`, data);
  }

  /**
   * 删除工单视图
   */
  static async deleteView(viewId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/views/${viewId}`);
  }

  /**
   * 共享工单视图
   */
  static async shareView(viewId: number, data: ShareTicketViewRequest): Promise<void> {
    return httpClient.post(`/api/v1/tickets/views/${viewId}/share`, data);
  }
}

