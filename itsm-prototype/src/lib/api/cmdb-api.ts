/**
 * CMDB API 服务
 */

import { httpClient } from './http-client';
import type {
  ConfigurationItem,
  CIRelationship,
  CITypeDefinition,
  RelationshipGraph,
  ImpactAnalysisRequest,
  ImpactAnalysisResult,
  CIChangeRecord,
  DiscoveryRule,
  DiscoveryResult,
  CMDBStats,
  CreateCIRequest,
  UpdateCIRequest,
  CreateRelationshipRequest,
  CIQuery,
  GraphQuery,
} from '@/types/cmdb';

export class CMDBApi {
  // ==================== 配置项管理 ====================

  /**
   * 获取CI列表
   */
  static async getCIs(
    query?: CIQuery
  ): Promise<{
    cis: ConfigurationItem[];
    total: number;
  }> {
    return httpClient.get('/api/v1/cmdb/cis', query);
  }

  /**
   * 获取单个CI
   */
  static async getCI(id: string): Promise<ConfigurationItem> {
    return httpClient.get(`/api/v1/cmdb/cis/${id}`);
  }

  /**
   * 创建CI
   */
  static async createCI(
    request: CreateCIRequest
  ): Promise<ConfigurationItem> {
    return httpClient.post('/api/v1/cmdb/cis', request);
  }

  /**
   * 更新CI
   */
  static async updateCI(
    id: string,
    request: UpdateCIRequest
  ): Promise<ConfigurationItem> {
    return httpClient.put(`/api/v1/cmdb/cis/${id}`, request);
  }

  /**
   * 删除CI
   */
  static async deleteCI(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/cmdb/cis/${id}`);
  }

  /**
   * 批量创建CI
   */
  static async batchCreateCIs(
    requests: CreateCIRequest[]
  ): Promise<ConfigurationItem[]> {
    return httpClient.post('/api/v1/cmdb/cis/batch', { cis: requests });
  }

  /**
   * 批量删除CI
   */
  static async batchDeleteCIs(ids: string[]): Promise<void> {
    return httpClient.post('/api/v1/cmdb/cis/batch-delete', { ids });
  }

  // ==================== CI关系管理 ====================

  /**
   * 获取CI的关系列表
   */
  static async getCIRelationships(
    ciId: string,
    params?: {
      direction?: 'incoming' | 'outgoing' | 'both';
      types?: string[];
    }
  ): Promise<CIRelationship[]> {
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/relationships`, params);
  }

  /**
   * 创建关系
   */
  static async createRelationship(
    request: CreateRelationshipRequest
  ): Promise<CIRelationship> {
    return httpClient.post('/api/v1/cmdb/relationships', request);
  }

  /**
   * 删除关系
   */
  static async deleteRelationship(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/cmdb/relationships/${id}`);
  }

  /**
   * 批量创建关系
   */
  static async batchCreateRelationships(
    requests: CreateRelationshipRequest[]
  ): Promise<CIRelationship[]> {
    return httpClient.post('/api/v1/cmdb/relationships/batch', {
      relationships: requests,
    });
  }

  // ==================== 关系图谱 ====================

  /**
   * 获取关系图谱
   */
  static async getRelationshipGraph(
    query: GraphQuery
  ): Promise<RelationshipGraph> {
    return httpClient.post('/api/v1/cmdb/graph', query);
  }

  /**
   * 导出图谱
   */
  static async exportGraph(
    query: GraphQuery,
    format: 'json' | 'image' | 'pdf'
  ): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/cmdb/graph/export',
      data: { ...query, format },
      responseType: 'blob',
    });
    return response as Blob;
  }

  // ==================== 影响分析 ====================

  /**
   * 分析影响
   */
  static async analyzeImpact(
    request: ImpactAnalysisRequest
  ): Promise<ImpactAnalysisResult> {
    return httpClient.post('/api/v1/cmdb/impact-analysis', request);
  }

  /**
   * 获取依赖链
   */
  static async getDependencyChain(
    ciId: string,
    direction: 'upstream' | 'downstream'
  ): Promise<{
    chain: ConfigurationItem[];
    relationships: CIRelationship[];
  }> {
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/dependency-chain`, {
      direction,
    });
  }

  // ==================== CI类型管理 ====================

  /**
   * 获取CI类型定义列表
   */
  static async getCITypes(): Promise<CITypeDefinition[]> {
    return httpClient.get('/api/v1/cmdb/ci-types');
  }

  /**
   * 获取单个CI类型定义
   */
  static async getCIType(type: string): Promise<CITypeDefinition> {
    return httpClient.get(`/api/v1/cmdb/ci-types/${type}`);
  }

  /**
   * 创建CI类型定义
   */
  static async createCIType(
    data: Omit<CITypeDefinition, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<CITypeDefinition> {
    return httpClient.post('/api/v1/cmdb/ci-types', data);
  }

  /**
   * 更新CI类型定义
   */
  static async updateCIType(
    type: string,
    data: Partial<CITypeDefinition>
  ): Promise<CITypeDefinition> {
    return httpClient.put(`/api/v1/cmdb/ci-types/${type}`, data);
  }

  // ==================== 变更追踪 ====================

  /**
   * 获取CI变更历史
   */
  static async getCIChangeHistory(
    ciId: string,
    params?: {
      startDate?: string;
      endDate?: string;
      page?: number;
      pageSize?: number;
    }
  ): Promise<{
    changes: CIChangeRecord[];
    total: number;
  }> {
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/changes`, params);
  }

  /**
   * 比较CI版本
   */
  static async compareCIVersions(
    ciId: string,
    fromTimestamp: string,
    toTimestamp: string
  ): Promise<{
    changes: {
      field: string;
      oldValue: any;
      newValue: any;
    }[];
  }> {
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/compare`, {
      from: fromTimestamp,
      to: toTimestamp,
    });
  }

  // ==================== CI发现和扫描 ====================

  /**
   * 获取发现规则列表
   */
  static async getDiscoveryRules(): Promise<DiscoveryRule[]> {
    return httpClient.get('/api/v1/cmdb/discovery/rules');
  }

  /**
   * 创建发现规则
   */
  static async createDiscoveryRule(
    rule: Omit<DiscoveryRule, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<DiscoveryRule> {
    return httpClient.post('/api/v1/cmdb/discovery/rules', rule);
  }

  /**
   * 运行发现规则
   */
  static async runDiscoveryRule(ruleId: string): Promise<DiscoveryResult> {
    return httpClient.post(`/api/v1/cmdb/discovery/rules/${ruleId}/run`);
  }

  /**
   * 获取扫描结果
   */
  static async getDiscoveryResult(resultId: string): Promise<DiscoveryResult> {
    return httpClient.get(`/api/v1/cmdb/discovery/results/${resultId}`);
  }

  /**
   * 获取扫描历史
   */
  static async getDiscoveryHistory(
    ruleId?: string
  ): Promise<DiscoveryResult[]> {
    return httpClient.get('/api/v1/cmdb/discovery/history', { ruleId });
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取CMDB统计
   */
  static async getCMDBStats(params?: {
    startDate?: string;
    endDate?: string;
  }): Promise<CMDBStats> {
    return httpClient.get('/api/v1/cmdb/stats', params);
  }

  /**
   * 获取CI健康度
   */
  static async getCIHealth(ciId: string): Promise<{
    score: number;
    issues: Array<{
      type: 'missing_info' | 'outdated' | 'no_relationships' | 'compliance';
      severity: 'low' | 'medium' | 'high';
      message: string;
    }>;
    recommendations: string[];
  }> {
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/health`);
  }

  /**
   * 检测孤立CI
   */
  static async findOrphanedCIs(): Promise<ConfigurationItem[]> {
    return httpClient.get('/api/v1/cmdb/analysis/orphaned');
  }

  /**
   * 检测重复CI
   */
  static async findDuplicateCIs(): Promise<
    Array<{
      cis: ConfigurationItem[];
      similarity: number;
      reason: string;
    }>
  > {
    return httpClient.get('/api/v1/cmdb/analysis/duplicates');
  }

  /**
   * 导出CMDB报告
   */
  static async exportReport(params: {
    format: 'excel' | 'pdf';
    includeRelationships?: boolean;
    types?: string[];
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/cmdb/export-report',
      data: params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default CMDBApi;
export const CMDBAPI = CMDBApi;
