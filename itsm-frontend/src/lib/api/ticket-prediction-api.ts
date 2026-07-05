/**
 * 工单趋势预测 API 服务
 */

import { httpClient } from './http-client';

export interface PredictionRequest {
  ticketType?: string;
  category?: string;
  timeRange: [string, string];
  predictionPeriod: 'week' | 'month' | 'quarter';
  modelType: 'arima' | 'exponential' | 'linear';
}

export interface PredictionDataPoint {
  date: string;
  actual?: number;
  predicted: number;
  upperBound?: number;
  lowerBound?: number;
  confidence?: number;
}

export interface PredictionMetrics {
  accuracy: number;
  mape: number;
  rmse: number;
  trend: 'up' | 'down' | 'stable';
  trendStrength: number;
  nextWeekPrediction: number;
  nextMonthPrediction: number;
  riskLevel: 'low' | 'medium' | 'high';
  riskFactors: string[];
}

export interface PredictionReport {
  period: string;
  summary: string;
  keyFindings: string[];
  recommendations: string[];
  metrics: PredictionMetrics;
  data: PredictionDataPoint[];
  generatedAt: string;
}

export class TicketPredictionApi {
  // 获取趋势预测
  static async getTrendPrediction(params: PredictionRequest): Promise<PredictionReport> {
    return httpClient.post<PredictionReport>('/api/v1/tickets/prediction/trend', params);
  }

  // 导出预测报告
  static async exportPredictionReport(
    params: PredictionRequest,
    format: 'excel' | 'pdf'
  ): Promise<Blob> {
    return httpClient.post<Blob>(`/api/v1/tickets/prediction/export?format=${format}`, params, {
      responseType: 'blob',
    });
  }
}
