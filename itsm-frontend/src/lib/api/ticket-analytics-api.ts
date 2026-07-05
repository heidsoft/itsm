/**
 * 工单数据分析 API 服务
 */

import { httpClient } from './http-client';

export interface AnalyticsConfig {
  dimensions: string[];
  metrics: string[];
  chartType: 'line' | 'bar' | 'pie' | 'area' | 'table';
  timeRange: [string, string];
  filters: Record<string, unknown>;
  groupBy?: string;
}

export interface AnalyticsDataPoint {
  name: string;
  value: number;
  count?: number;
  avgTime?: number;
  metadata?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface AnalyticsSummary {
  total: number;
  resolved: number;
  avgResponseTime: number;
  avgResolutionTime: number;
  slaCompliance: number;
  customerSatisfaction: number;
}

export interface AnalyticsResponse {
  data: AnalyticsDataPoint[];
  summary: AnalyticsSummary;
  generatedAt: string;
}

export class TicketAnalyticsApi {
  // 获取深度分析数据
  static async getDeepAnalytics(config: AnalyticsConfig): Promise<AnalyticsResponse> {
    // 修正: 确保路径与后端一致，后端可能是 /api/v1/tickets/analytics/deep
    // 根据 PRD，工单相关接口通常在 /api/v1/tickets 下
    return httpClient.post<AnalyticsResponse>('/api/v1/tickets/analytics/deep', config);
  }

  // 导出分析数据
  static async exportAnalytics(
    config: AnalyticsConfig,
    format: 'excel' | 'pdf' | 'csv'
  ): Promise<Blob> {
    return httpClient.post<Blob>(`/api/v1/tickets/analytics/export?format=${format}`, config, {
      responseType: 'blob',
    });
  }
}
