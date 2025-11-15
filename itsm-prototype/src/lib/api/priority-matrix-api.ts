/**
 * 优先级矩阵 API 服务
 * 提供优先级计算、矩阵配置、规则管理等API接口
 */

import { httpClient } from './http-client';
import type {
  PriorityMatrixConfig,
  PriorityMatrixData,
  PriorityCalculationRequest,
  PriorityCalculationResult,
  PrioritySuggestion,
  BatchPrioritySuggestions,
  PriorityRule,
  PriorityRuleQuery,
  PriorityChangeHistory,
  PriorityDistributionAnalysis,
  PriorityAccuracyAnalysis,
  CreateMatrixConfigRequest,
  UpdateMatrixConfigRequest,
  PriorityAnalysisQuery,
} from '@/types/priority-matrix';

export class PriorityMatrixApi {
  // ==================== 优先级计算 ====================

  /**
   * 计算工单优先级
   */
  static async calculatePriority(
    request: PriorityCalculationRequest
  ): Promise<PriorityCalculationResult> {
    return httpClient.post<PriorityCalculationResult>(
      '/api/v1/priority/calculate',
      request
    );
  }

  /**
   * 批量计算优先级
   */
  static async batchCalculatePriority(
    requests: PriorityCalculationRequest[]
  ): Promise<PriorityCalculationResult[]> {
    return httpClient.post<PriorityCalculationResult[]>(
      '/api/v1/priority/batch-calculate',
      { requests }
    );
  }

  /**
   * 获取优先级建议
   */
  static async getPrioritySuggestion(
    ticketId: number
  ): Promise<PrioritySuggestion> {
    return httpClient.get<PrioritySuggestion>(
      `/api/v1/tickets/${ticketId}/priority-suggestion`
    );
  }

  /**
   * 批量获取优先级建议
   */
  static async getBatchPrioritySuggestions(
    ticketIds: number[]
  ): Promise<BatchPrioritySuggestions> {
    return httpClient.post<BatchPrioritySuggestions>(
      '/api/v1/priority/batch-suggestions',
      { ticketIds }
    );
  }

  // ==================== 矩阵配置管理 ====================

  /**
   * 获取矩阵配置列表
   */
  static async getMatrixConfigs(): Promise<PriorityMatrixConfig[]> {
    return httpClient.get<PriorityMatrixConfig[]>(
      '/api/v1/priority/matrix-configs'
    );
  }

  /**
   * 获取活动的矩阵配置
   */
  static async getActiveMatrixConfig(): Promise<PriorityMatrixConfig> {
    return httpClient.get<PriorityMatrixConfig>(
      '/api/v1/priority/matrix-configs/active'
    );
  }

  /**
   * 获取矩阵数据
   */
  static async getMatrixData(
    configId?: string
  ): Promise<PriorityMatrixData> {
    return httpClient.get<PriorityMatrixData>(
      '/api/v1/priority/matrix-data',
      { configId }
    );
  }

  /**
   * 创建矩阵配置
   */
  static async createMatrixConfig(
    request: CreateMatrixConfigRequest
  ): Promise<PriorityMatrixConfig> {
    return httpClient.post<PriorityMatrixConfig>(
      '/api/v1/priority/matrix-configs',
      request
    );
  }

  /**
   * 更新矩阵配置
   */
  static async updateMatrixConfig(
    configId: string,
    request: UpdateMatrixConfigRequest
  ): Promise<PriorityMatrixConfig> {
    return httpClient.put<PriorityMatrixConfig>(
      `/api/v1/priority/matrix-configs/${configId}`,
      request
    );
  }

  /**
   * 删除矩阵配置
   */
  static async deleteMatrixConfig(configId: string): Promise<void> {
    return httpClient.delete(`/api/v1/priority/matrix-configs/${configId}`);
  }

  /**
   * 激活矩阵配置
   */
  static async activateMatrixConfig(configId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/priority/matrix-configs/${configId}/activate`
    );
  }

  // ==================== 规则管理 ====================

  /**
   * 获取优先级规则列表
   */
  static async getPriorityRules(
    query?: PriorityRuleQuery
  ): Promise<{
    rules: PriorityRule[];
    total: number;
  }> {
    return httpClient.get('/api/v1/priority/rules', query);
  }

  /**
   * 获取单个规则
   */
  static async getPriorityRule(ruleId: string): Promise<PriorityRule> {
    return httpClient.get<PriorityRule>(`/api/v1/priority/rules/${ruleId}`);
  }

  /**
   * 创建规则
   */
  static async createPriorityRule(
    rule: Omit<PriorityRule, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<PriorityRule> {
    return httpClient.post<PriorityRule>('/api/v1/priority/rules', rule);
  }

  /**
   * 更新规则
   */
  static async updatePriorityRule(
    ruleId: string,
    rule: Partial<PriorityRule>
  ): Promise<PriorityRule> {
    return httpClient.put<PriorityRule>(
      `/api/v1/priority/rules/${ruleId}`,
      rule
    );
  }

  /**
   * 删除规则
   */
  static async deletePriorityRule(ruleId: string): Promise<void> {
    return httpClient.delete(`/api/v1/priority/rules/${ruleId}`);
  }

  /**
   * 启用/禁用规则
   */
  static async togglePriorityRule(
    ruleId: string,
    enabled: boolean
  ): Promise<void> {
    return httpClient.post(`/api/v1/priority/rules/${ruleId}/toggle`, {
      enabled,
    });
  }

  /**
   * 测试规则
   */
  static async testPriorityRule(
    ruleId: string,
    ticketId: number
  ): Promise<{
    matched: boolean;
    actions: string[];
  }> {
    return httpClient.post(`/api/v1/priority/rules/${ruleId}/test`, {
      ticketId,
    });
  }

  // ==================== 历史和分析 ====================

  /**
   * 获取优先级变更历史
   */
  static async getPriorityHistory(params: {
    ticketId?: number;
    startDate?: string;
    endDate?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    history: PriorityChangeHistory[];
    total: number;
  }> {
    return httpClient.get('/api/v1/priority/history', params);
  }

  /**
   * 获取优先级分布分析
   */
  static async getPriorityDistribution(
    query: PriorityAnalysisQuery
  ): Promise<PriorityDistributionAnalysis> {
    return httpClient.get<PriorityDistributionAnalysis>(
      '/api/v1/priority/analysis/distribution',
      query
    );
  }

  /**
   * 获取优先级准确性分析
   */
  static async getPriorityAccuracy(
    query: PriorityAnalysisQuery
  ): Promise<PriorityAccuracyAnalysis> {
    return httpClient.get<PriorityAccuracyAnalysis>(
      '/api/v1/priority/analysis/accuracy',
      query
    );
  }

  /**
   * 导出优先级报告
   */
  static async exportPriorityReport(params: {
    format: 'excel' | 'pdf';
    startDate: string;
    endDate: string;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/priority/export-report',
      data: params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default PriorityMatrixApi;
export const PriorityMatrixAPI = PriorityMatrixApi;

