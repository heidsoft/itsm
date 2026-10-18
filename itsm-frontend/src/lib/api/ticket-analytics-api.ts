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
  //
  // 后端 `POST /api/v1/tickets/analytics/export` 用 ShouldBindJSON 读 dto.DeepAnalyticsRequest，
  // format 走查询参数。这里必须用 request 而不是 post：httpClient.post 对 JSON 请求体
  // 不透传 responseType，blob 会被当 JSON 解析而抛错。失败由 httpClient 按非 2xx 抛出
  // ApiRequestError，服务端业务文案原样透出给调用方。
  static async exportAnalytics(
    config: AnalyticsConfig,
    format: 'excel' | 'pdf' | 'csv'
  ): Promise<Blob> {
    return httpClient.request<Blob>({
      method: 'POST',
      url: '/api/v1/tickets/analytics/export',
      params: { format },
      data: config,
      responseType: 'blob',
    });
  }
}
