/**
 * 问题管理 API 包装器
 * 兼容旧代码使用
 */

import { httpClient } from './http-client';

// ==================== 问题趋势相关类型 ====================

export interface ProblemTrendRequest {
  start_date: string;
  end_date: string;
  category?: string;
}

export interface CategoryCount {
  category: string;
  count: number;
}

export interface MonthlyCount {
  month: string;
  count: number;
  resolved: number;
  open: number;
}

export interface ProblemTrendData {
  period: string;
  total_problems: number;
  resolved_problems: number;
  open_problems: number;
  resolution_rate: number;
  avg_resolution_time_hours: number;
  category_breakdown: Record<string, number>;
  priority_breakdown: Record<string, number>;
  trend_direction: string;
  top_categories: CategoryCount[];
  monthly_trend: MonthlyCount[];
}

export interface ProblemHotspotsData {
  period_start: string;
  period_end: string;
  category_breakdown: Record<string, number>;
  priority_breakdown: Record<string, number>;
  hotspots: string[];
  avg_per_category: number;
}

export interface Problem {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  severity: string;
  category?: string;
  impact?: string;
  assignee_id?: number;
  assigneeId?: number;
  reporter_id?: number;
  reporterId?: number;
  root_cause?: string;
  rootCause?: string;
  workaround?: string;
  resolution?: string;
  affected_incidents?: number[];
  affectedIncidents?: number[];
  related_changes?: number[];
  relatedChanges?: number[];
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;
}

export interface ProblemListResponse {
  problems: Problem[];
  total: number;
  page: number;
  page_size: number;
}

export class ProblemApi {
  /**
   * 获取问题列表
   */
  static async getProblems(params?: any): Promise<ProblemListResponse> {
    return httpClient.get('/api/v1/problems', params);
  }

  /**
   * 获取问题详情
   */
  static async getProblem(id: number): Promise<Problem> {
    return httpClient.get(`/api/v1/problems/${id}`);
  }

  /**
   * 创建问题
   */
  static async createProblem(data: Partial<Problem>): Promise<Problem> {
    return httpClient.post('/api/v1/problems', data);
  }

  /**
   * 更新问题
   */
  static async updateProblem(id: number, data: Partial<Problem>): Promise<Problem> {
    return httpClient.put(`/api/v1/problems/${id}`, data);
  }

  /**
   * 删除问题
   */
  static async deleteProblem(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/problems/${id}`);
  }

  /**
   * 获取问题统计
   */
  static async getProblemStats(params?: any): Promise<any> {
    return httpClient.get('/api/v1/problems/stats', params);
  }

  /**
   * 调查问题
   */
  static async investigateProblem(id: number, data: unknown): Promise<Problem> {
    return httpClient.post(`/api/v1/problems/${id}/investigate`, data);
  }

  /**
   * 记录根本原因
   */
  static async recordRootCause(id: number, rootCause: string): Promise<Problem> {
    return httpClient.post(`/api/v1/problems/${id}/root-cause`, { root_cause: rootCause });
  }

  /**
   * 提供解决方案
   */
  static async provideSolution(id: number, solution: string): Promise<Problem> {
    return httpClient.post(`/api/v1/problems/${id}/solution`, { resolution: solution });
  }

  /**
   * 关闭问题
   */
  static async closeProblem(id: number, resolution: string): Promise<Problem> {
    return httpClient.post(`/api/v1/problems/${id}/close`, { resolution });
  }

  // ==================== 趋势分析 ====================


  /**
   * 获取问题趋势分析
   */
  static async getTrends(params: ProblemTrendRequest): Promise<ProblemTrendData> {
    return httpClient.get<ProblemTrendData>('/api/v1/problems/trend', params);
  }

  /**
   * 获取问题热点分析
   */
  static async getHotspots(params: ProblemTrendRequest): Promise<ProblemHotspotsData> {
    return httpClient.get<ProblemHotspotsData>('/api/v1/problems/hotspots', params);
  }
}

export default ProblemApi;
