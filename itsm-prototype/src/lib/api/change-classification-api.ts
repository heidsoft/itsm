/**
 * 变更分类 API 服务
 */

import { httpClient } from './http-client';
import type {
  ChangeClassification,
  ClassificationSuggestion,
  RiskAssessment,
  ImpactAnalysis,
  ClassificationRule,
  ChangeTemplate,
  ApprovalMatrix,
  ClassificationStats,
  ClassificationAnalysis,
  CreateClassificationRequest,
  UpdateClassificationRequest,
  AssessRiskRequest,
  AnalyzeImpactRequest,
  ClassificationQuery,
  ClassificationHistory,
} from '@/types/change-classification';

export class ChangeClassificationApi {
  // ==================== 分类管理 ====================

  /**
   * 获取分类列表
   */
  static async getClassifications(
    query?: ClassificationQuery
  ): Promise<{
    classifications: ChangeClassification[];
    total: number;
  }> {
    return httpClient.get('/api/v1/change-classifications', query);
  }

  /**
   * 获取单个分类
   */
  static async getClassification(
    id: string
  ): Promise<ChangeClassification> {
    return httpClient.get(`/api/v1/change-classifications/${id}`);
  }

  /**
   * 创建分类
   */
  static async createClassification(
    request: CreateClassificationRequest
  ): Promise<ChangeClassification> {
    return httpClient.post('/api/v1/change-classifications', request);
  }

  /**
   * 更新分类
   */
  static async updateClassification(
    id: string,
    request: UpdateClassificationRequest
  ): Promise<ChangeClassification> {
    return httpClient.put(`/api/v1/change-classifications/${id}`, request);
  }

  /**
   * 删除分类
   */
  static async deleteClassification(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/change-classifications/${id}`);
  }

  // ==================== 风险评估 ====================

  /**
   * 评估风险
   */
  static async assessRisk(
    request: AssessRiskRequest
  ): Promise<RiskAssessment> {
    return httpClient.post('/api/v1/changes/assess-risk', request);
  }

  /**
   * 获取风险评估历史
   */
  static async getRiskAssessmentHistory(
    changeId: number
  ): Promise<RiskAssessment[]> {
    return httpClient.get(`/api/v1/changes/${changeId}/risk-assessments`);
  }

  // ==================== 影响分析 ====================

  /**
   * 分析影响
   */
  static async analyzeImpact(
    request: AnalyzeImpactRequest
  ): Promise<ImpactAnalysis> {
    return httpClient.post('/api/v1/changes/analyze-impact', request);
  }

  /**
   * 获取影响分析历史
   */
  static async getImpactAnalysisHistory(
    changeId: number
  ): Promise<ImpactAnalysis[]> {
    return httpClient.get(`/api/v1/changes/${changeId}/impact-analyses`);
  }

  // ==================== 分类建议 ====================

  /**
   * 获取分类建议
   */
  static async getClassificationSuggestion(
    changeId: number
  ): Promise<ClassificationSuggestion> {
    return httpClient.get(
      `/api/v1/changes/${changeId}/classification-suggestion`
    );
  }

  /**
   * 应用分类建议
   */
  static async applyClassificationSuggestion(
    changeId: number,
    classificationId: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/changes/${changeId}/apply-classification`,
      { classificationId }
    );
  }

  // ==================== 分类规则 ====================

  /**
   * 获取分类规则列表
   */
  static async getClassificationRules(): Promise<ClassificationRule[]> {
    return httpClient.get('/api/v1/classification-rules');
  }

  /**
   * 创建分类规则
   */
  static async createClassificationRule(
    rule: Omit<ClassificationRule, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<ClassificationRule> {
    return httpClient.post('/api/v1/classification-rules', rule);
  }

  /**
   * 更新分类规则
   */
  static async updateClassificationRule(
    ruleId: string,
    rule: Partial<ClassificationRule>
  ): Promise<ClassificationRule> {
    return httpClient.put(`/api/v1/classification-rules/${ruleId}`, rule);
  }

  /**
   * 删除分类规则
   */
  static async deleteClassificationRule(ruleId: string): Promise<void> {
    return httpClient.delete(`/api/v1/classification-rules/${ruleId}`);
  }

  // ==================== 变更模板 ====================

  /**
   * 获取变更模板列表
   */
  static async getChangeTemplates(params?: {
    classificationId?: string;
  }): Promise<ChangeTemplate[]> {
    return httpClient.get('/api/v1/change-templates', params);
  }

  /**
   * 获取单个变更模板
   */
  static async getChangeTemplate(id: string): Promise<ChangeTemplate> {
    return httpClient.get(`/api/v1/change-templates/${id}`);
  }

  /**
   * 创建变更模板
   */
  static async createChangeTemplate(
    template: Omit<ChangeTemplate, 'id' | 'usageCount' | 'createdAt' | 'updatedAt'>
  ): Promise<ChangeTemplate> {
    return httpClient.post('/api/v1/change-templates', template);
  }

  // ==================== 审批矩阵 ====================

  /**
   * 获取审批矩阵
   */
  static async getApprovalMatrix(
    classificationId: string
  ): Promise<ApprovalMatrix> {
    return httpClient.get(
      `/api/v1/change-classifications/${classificationId}/approval-matrix`
    );
  }

  /**
   * 更新审批矩阵
   */
  static async updateApprovalMatrix(
    classificationId: string,
    matrix: Omit<ApprovalMatrix, 'classificationId'>
  ): Promise<ApprovalMatrix> {
    return httpClient.put(
      `/api/v1/change-classifications/${classificationId}/approval-matrix`,
      matrix
    );
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取分类统计
   */
  static async getClassificationStats(params: {
    startDate: string;
    endDate: string;
  }): Promise<ClassificationStats[]> {
    return httpClient.get('/api/v1/change-classifications/stats', params);
  }

  /**
   * 获取分类分析
   */
  static async getClassificationAnalysis(params: {
    startDate: string;
    endDate: string;
    groupBy?: 'day' | 'week' | 'month';
  }): Promise<ClassificationAnalysis> {
    return httpClient.get('/api/v1/change-classifications/analysis', params);
  }

  /**
   * 获取分类历史
   */
  static async getClassificationHistory(
    changeId: number
  ): Promise<ClassificationHistory[]> {
    return httpClient.get(`/api/v1/changes/${changeId}/classification-history`);
  }
}

export default ChangeClassificationApi;
export const ChangeClassificationAPI = ChangeClassificationApi;

