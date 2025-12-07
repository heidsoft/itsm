/**
 * 工单数据分析 API 服务
 */

import { httpClient } from './http-client';

export interface AnalyticsConfig {
  dimensions: string[];
  metrics: string[];
  chart_type: 'line' | 'bar' | 'pie' | 'area' | 'table';
  time_range: [string, string];
  filters: Record<string, any>;
  group_by?: string;
}

export interface AnalyticsDataPoint {
  name: string;
  value: number;
  count?: number;
  avg_time?: number;
  metadata?: Record<string, any>;
}

export interface AnalyticsSummary {
  total: number;
  resolved: number;
  avg_response_time: number;
  avg_resolution_time: number;
  sla_compliance: number;
  customer_satisfaction: number;
}

export interface AnalyticsResponse {
  data: AnalyticsDataPoint[];
  summary: AnalyticsSummary;
  generated_at: string;
}

export class TicketAnalyticsApi {
  // 获取深度分析数据
  static async getDeepAnalytics(config: AnalyticsConfig): Promise<AnalyticsResponse> {
    return httpClient.post<AnalyticsResponse>('/api/tickets/analytics/deep', config);
  }

  // 导出分析数据
  static async exportAnalytics(config: AnalyticsConfig, format: 'excel' | 'pdf' | 'csv'): Promise<Blob> {
    return httpClient.post<Blob>(`/api/tickets/analytics/export?format=${format}`, config, {
      responseType: 'blob',
    });
  }
}

