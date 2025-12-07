/**
 * 工单趋势预测 API 服务
 */

import { httpClient } from './http-client';

export interface PredictionRequest {
  ticket_type?: string;
  category?: string;
  time_range: [string, string];
  prediction_period: 'week' | 'month' | 'quarter';
  model_type: 'arima' | 'exponential' | 'linear';
}

export interface PredictionDataPoint {
  date: string;
  actual?: number;
  predicted: number;
  upper_bound?: number;
  lower_bound?: number;
  confidence?: number;
}

export interface PredictionMetrics {
  accuracy: number;
  mape: number;
  rmse: number;
  trend: 'up' | 'down' | 'stable';
  trend_strength: number;
  next_week_prediction: number;
  next_month_prediction: number;
  risk_level: 'low' | 'medium' | 'high';
  risk_factors: string[];
}

export interface PredictionReport {
  period: string;
  summary: string;
  key_findings: string[];
  recommendations: string[];
  metrics: PredictionMetrics;
  data: PredictionDataPoint[];
  generated_at: string;
}

export class TicketPredictionApi {
  // 获取趋势预测
  static async getTrendPrediction(params: PredictionRequest): Promise<PredictionReport> {
    return httpClient.post<PredictionReport>('/api/tickets/prediction/trend', params);
  }

  // 导出预测报告
  static async exportPredictionReport(params: PredictionRequest, format: 'excel' | 'pdf'): Promise<Blob> {
    return httpClient.post<Blob>(`/api/tickets/prediction/export?format=${format}`, params, {
      responseType: 'blob',
    });
  }
}

