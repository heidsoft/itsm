/**
 * 问题管理 API 包装器
 * 兼容旧代码使用
 */

import { httpClient } from './http-client';

// ==================== 问题趋势相关类型 ====================

export interface ProblemTrendRequest {
  startDate: string;
  endDate: string;
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
  totalProblems: number;
  resolvedProblems: number;
  openProblems: number;
  resolutionRate: number;
  avgResolutionTimeHours: number;
  categoryBreakdown: Record<string, number>;
  priorityBreakdown: Record<string, number>;
  trendDirection: string;
  topCategories: CategoryCount[];
  monthlyTrend: MonthlyCount[];
}

export interface ProblemHotspotsData {
  periodStart: string;
  periodEnd: string;
  categoryBreakdown: Record<string, number>;
  priorityBreakdown: Record<string, number>;
  hotspots: string[];
  avgPerCategory: number;
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
  assigneeId?: number;
  reporterId?: number;
  rootCause?: string;
  workaround?: string;
  resolution?: string;
  affectedIncidents?: number[];
  relatedChanges?: number[];
  createdAt: string;
  updatedAt: string;
  // P0 修复暴露的预先 TS 错误：ProblemSLACard 引用 problem.slaStatus
  slaStatus?: 'ok' | 'warning' | 'breached';
  responseDeadline?: string;
  resolutionDeadline?: string;
}

export interface ProblemListResponse {
  problems: Problem[];
  items?: Problem[];
  total: number;
  page: number;
  pageSize: number;
}

// ==================== 问题关联 ====================

export type RelatedType = 'ticket' | 'incident' | 'change';

export interface AssociatedItem {
  id: number;
  type: RelatedType;
  title: string;
  status: string;
  number?: string;
  createdAt?: string;
}

export interface ProblemAssociations {
  tickets: AssociatedItem[];
  incidents: AssociatedItem[];
  changes: AssociatedItem[];
}

export interface ProblemAssociationRequest {
  relatedType: RelatedType;
  relatedIds: number[];
}

export interface ProblemRemoveAssociationRequest {
  relatedType: RelatedType;
  relatedId: number;
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
  static async investigateProblem(_id: number, _data: unknown): Promise<Problem> {
    throw new Error('功能开发中');
  }

  /**
   * 记录根本原因
   */
  static async recordRootCause(_id: number, _rootCause: string): Promise<Problem> {
    throw new Error('功能开发中');
  }

  /**
   * 提供解决方案
   */
  static async provideSolution(_id: number, _solution: string): Promise<Problem> {
    throw new Error('功能开发中');
  }

  /**
   * 关闭问题
   */
  static async closeProblem(_id: number, _resolution: string): Promise<Problem> {
    throw new Error('功能开发中');
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

  // ==================== 关联管理（P0 修复暴露的 TS 错误） ====================

  /**
   * 获取问题关联（工单/事件/变更）
   */
  static async getAssociations(problemId: number): Promise<ProblemAssociations> {
    return httpClient.get<ProblemAssociations>(`/api/v1/problems/${problemId}/associations`);
  }

  /**
   * 添加问题关联
   */
  static async addAssociation(problemId: number, req: ProblemAssociationRequest): Promise<void> {
    return httpClient.post(`/api/v1/problems/${problemId}/associations`, req);
  }

  /**
   * 移除问题关联
   */
  static async removeAssociation(problemId: number, req: ProblemRemoveAssociationRequest): Promise<void> {
    return httpClient.request({
      method: 'DELETE',
      url: `/api/v1/problems/${problemId}/associations`,
      data: req,
    });
  }

  // ==================== SLA（P0 修复暴露的 TS 错误） ====================

  /**
   * 获取问题 SLA 信息
   */
  static async getProblemSLA(problemId: number): Promise<{
    responseDeadline?: string;
    resolutionDeadline?: string;
    responseTimeUsed: number;
    resolutionTimeUsed: number;
    responseBreached: boolean;
    resolutionBreached: boolean;
    slaStatus: string;
  }> {
    return httpClient.get(`/api/v1/problems/${problemId}/sla`);
  }
}

export default ProblemApi;
