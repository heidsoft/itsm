/**
 * 问题管理 API 包装器
 * 兼容旧代码使用
 */

import { httpClient } from './http-client';

export interface Problem {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  severity: string;
  category?: string;
  assignee_id?: number;
  reporter_id?: number;
  root_cause?: string;
  workaround?: string;
  resolution?: string;
  affected_incidents?: number[];
  related_changes?: number[];
  created_at: string;
  updated_at: string;
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
  static async investigateProblem(id: number, data: any): Promise<Problem> {
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
}

export default ProblemApi;
