/**
 * 工单关联 API 服务
 * 提供工单关联的完整API接口
 */

import { httpClient } from './http-client';
import type {
  TicketRelation,
  TicketRelationWithDetails,
  CreateRelationRequest,
  BatchCreateRelationsRequest,
  UpdateRelationRequest,
  RelationValidation,
  RelationConflict,
  TicketHierarchy,
  TicketDependency,
  DependencyGraph,
  RelationSearchResult,
  TicketRelationStats,
  RelationImpactAnalysis,
  RelationGraphData,
  SmartRelationRecommendation,
  RelationHistory,
  BatchRelationResult,
  RelationPermissions,
} from '@/types/ticket-relations';

export class TicketRelationsApi {
  // ==================== 基础CRUD操作 ====================

  /**
   * 创建工单关联
   */
  static async createRelation(
    request: CreateRelationRequest
  ): Promise<TicketRelation> {
    return httpClient.post<TicketRelation>(
      '/api/v1/tickets/relations',
      request
    );
  }

  /**
   * 批量创建关联
   */
  static async batchCreateRelations(
    request: BatchCreateRelationsRequest
  ): Promise<BatchRelationResult> {
    return httpClient.post<BatchRelationResult>(
      '/api/v1/tickets/relations/batch',
      request
    );
  }

  /**
   * 获取工单的所有关联
   */
  static async getTicketRelations(
    ticketId: number,
    params?: {
      relationType?: string;
      direction?: string;
      includeDetails?: boolean;
    }
  ): Promise<TicketRelationWithDetails[]> {
    return httpClient.get<TicketRelationWithDetails[]>(
      `/api/v1/tickets/${ticketId}/relations`,
      params
    );
  }

  /**
   * 获取单个关联详情
   */
  static async getRelation(relationId: string): Promise<TicketRelationWithDetails> {
    return httpClient.get<TicketRelationWithDetails>(
      `/api/v1/tickets/relations/${relationId}`
    );
  }

  /**
   * 更新关联
   */
  static async updateRelation(
    relationId: string,
    request: UpdateRelationRequest
  ): Promise<TicketRelation> {
    return httpClient.put<TicketRelation>(
      `/api/v1/tickets/relations/${relationId}`,
      request
    );
  }

  /**
   * 删除关联
   */
  static async deleteRelation(
    relationId: string,
    reason?: string
  ): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/relations/${relationId}`, {
      reason,
    });
  }

  /**
   * 批量删除关联
   */
  static async batchDeleteRelations(
    relationIds: string[],
    reason?: string
  ): Promise<BatchRelationResult> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/tickets/relations/batch',
      data: { relationIds, reason },
    });
  }

  // ==================== 父子关系 ====================

  /**
   * 设置父工单
   */
  static async setParent(
    childTicketId: number,
    parentTicketId: number
  ): Promise<TicketRelation> {
    return httpClient.post<TicketRelation>(
      `/api/v1/tickets/${childTicketId}/parent`,
      { parentTicketId }
    );
  }

  /**
   * 移除父工单
   */
  static async removeParent(childTicketId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/${childTicketId}/parent`);
  }

  /**
   * 获取子工单列表
   */
  static async getChildren(parentTicketId: number): Promise<TicketRelation[]> {
    return httpClient.get<TicketRelation[]>(
      `/api/v1/tickets/${parentTicketId}/children`
    );
  }

  /**
   * 获取工单层级结构
   */
  static async getHierarchy(ticketId: number): Promise<TicketHierarchy> {
    return httpClient.get<TicketHierarchy>(
      `/api/v1/tickets/${ticketId}/hierarchy`
    );
  }

  // ==================== 依赖关系 ====================

  /**
   * 添加依赖
   */
  static async addDependency(data: {
    ticketId: number;
    dependsOnTicketId: number;
    dependencyType: 'hard' | 'soft';
  }): Promise<TicketDependency> {
    return httpClient.post<TicketDependency>(
      `/api/v1/tickets/${data.ticketId}/dependencies`,
      { dependsOnTicketId: data.dependsOnTicketId, dependencyType: data.dependencyType }
    );
  }

  /**
   * 移除依赖
   */
  static async removeDependency(
    ticketId: number,
    dependencyId: string
  ): Promise<void> {
    return httpClient.delete(
      `/api/v1/tickets/${ticketId}/dependencies/${dependencyId}`
    );
  }

  /**
   * 获取依赖列表
   */
  static async getDependencies(
    ticketId: number
  ): Promise<TicketDependency[]> {
    return httpClient.get<TicketDependency[]>(
      `/api/v1/tickets/${ticketId}/dependencies`
    );
  }

  /**
   * 获取依赖图
   */
  static async getDependencyGraph(
    ticketId: number,
    maxDepth?: number
  ): Promise<DependencyGraph> {
    return httpClient.get<DependencyGraph>(
      `/api/v1/tickets/${ticketId}/dependency-graph`,
      { maxDepth }
    );
  }

  /**
   * 检查是否可以开始工作
   */
  static async canStartWork(ticketId: number): Promise<{
    canStart: boolean;
    blockingTickets: number[];
    reason?: string;
  }> {
    return httpClient.get(`/api/v1/tickets/${ticketId}/can-start`);
  }

  // ==================== 验证和冲突检测 ====================

  /**
   * 验证关联
   */
  static async validateRelation(
    request: CreateRelationRequest
  ): Promise<RelationValidation> {
    return httpClient.post<RelationValidation>(
      '/api/v1/tickets/relations/validate',
      request
    );
  }

  /**
   * 检测循环依赖
   */
  static async detectCircularDependency(data: {
    sourceTicketId: number;
    targetTicketId: number;
  }): Promise<{
    hasCircular: boolean;
    path?: number[];
  }> {
    return httpClient.post(
      '/api/v1/tickets/relations/detect-circular',
      data
    );
  }

  /**
   * 获取关联冲突
   */
  static async getConflicts(
    request: CreateRelationRequest
  ): Promise<RelationConflict[]> {
    return httpClient.post<RelationConflict[]>(
      '/api/v1/tickets/relations/conflicts',
      request
    );
  }

  // ==================== 搜索和查询 ====================

  /**
   * 搜索关联
   */
  static async searchRelations(params: {
    ticketId?: number;
    relationType?: string;
    direction?: string;
    search?: string;
    page?: number;
    pageSize?: number;
  }): Promise<RelationSearchResult> {
    return httpClient.get<RelationSearchResult>(
      '/api/v1/tickets/relations/search',
      params
    );
  }

  /**
   * 查找相关工单
   */
  static async findRelatedTickets(
    ticketId: number,
    params?: {
      maxResults?: number;
      minSimilarity?: number;
    }
  ): Promise<Array<{
    ticketId: number;
    ticketNumber: string;
    title: string;
    similarity: number;
    reason: string;
  }>> {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/related`,
      params
    );
  }

  /**
   * 查找重复工单
   */
  static async findDuplicates(
    ticketId: number,
    threshold?: number
  ): Promise<Array<{
    ticketId: number;
    ticketNumber: string;
    title: string;
    similarity: number;
  }>> {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/duplicates`,
      { threshold }
    );
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取关联统计
   */
  static async getRelationStats(
    ticketId: number
  ): Promise<TicketRelationStats> {
    return httpClient.get<TicketRelationStats>(
      `/api/v1/tickets/${ticketId}/relations/stats`
    );
  }

  /**
   * 关联影响分析
   */
  static async analyzeImpact(data: {
    ticketId: number;
    action: 'close' | 'delete' | 'change_status';
    newStatus?: string;
  }): Promise<RelationImpactAnalysis> {
    return httpClient.post<RelationImpactAnalysis>(
      `/api/v1/tickets/${data.ticketId}/impact-analysis`,
      { action: data.action, newStatus: data.newStatus }
    );
  }

  /**
   * 获取关键路径
   */
  static async getCriticalPath(ticketId: number): Promise<{
    path: number[];
    estimatedDuration: number;
    bottlenecks: number[];
  }> {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/critical-path`
    );
  }

  // ==================== 关系图谱 ====================

  /**
   * 获取关系图谱数据
   */
  static async getRelationGraph(
    ticketId: number,
    params?: {
      maxDepth?: number;
      relationTypes?: string[];
      direction?: string;
    }
  ): Promise<RelationGraphData> {
    return httpClient.get<RelationGraphData>(
      `/api/v1/tickets/${ticketId}/graph`,
      params
    );
  }

  /**
   * 导出关系图谱
   */
  static async exportGraph(
    ticketId: number,
    format: 'png' | 'svg' | 'json'
  ): Promise<Blob> {
    const response = await httpClient.request({
      method: 'GET',
      url: `/api/v1/tickets/${ticketId}/graph/export`,
      params: { format },
      responseType: 'blob',
    });
    return response as Blob;
  }

  // ==================== 智能推荐 ====================

  /**
   * 获取关联建议
   */
  static async getRelationSuggestions(
    ticketId: number,
    limit?: number
  ): Promise<SmartRelationRecommendation> {
    return httpClient.get<SmartRelationRecommendation>(
      `/api/v1/tickets/${ticketId}/relation-suggestions`,
      { limit }
    );
  }

  /**
   * 基于AI的关联推荐
   */
  static async getAIRecommendations(
    ticketId: number
  ): Promise<Array<{
    targetTicketId: number;
    relationType: string;
    confidence: number;
    reason: string;
  }>> {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/ai-recommendations`
    );
  }

  // ==================== 历史和审计 ====================

  /**
   * 获取关联历史
   */
  static async getRelationHistory(
    ticketId: number,
    params?: {
      page?: number;
      pageSize?: number;
    }
  ): Promise<{
    history: RelationHistory[];
    total: number;
  }> {
    return httpClient.get(
      `/api/v1/tickets/${ticketId}/relations/history`,
      params
    );
  }

  /**
   * 获取关联操作日志
   */
  static async getRelationLogs(params?: {
    ticketId?: number;
    action?: string;
    startDate?: string;
    endDate?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    logs: RelationHistory[];
    total: number;
  }> {
    return httpClient.get('/api/v1/tickets/relations/logs', params);
  }

  // ==================== 权限 ====================

  /**
   * 获取关联权限
   */
  static async getRelationPermissions(
    ticketId: number
  ): Promise<RelationPermissions> {
    return httpClient.get<RelationPermissions>(
      `/api/v1/tickets/${ticketId}/relations/permissions`
    );
  }

  /**
   * 检查是否可以创建关联
   */
  static async canCreateRelation(data: {
    sourceTicketId: number;
    targetTicketId: number;
    relationType: string;
  }): Promise<{ canCreate: boolean; reason?: string }> {
    return httpClient.post(
      '/api/v1/tickets/relations/can-create',
      data
    );
  }

  // ==================== 批量操作 ====================

  /**
   * 批量更新关联描述
   */
  static async batchUpdateDescriptions(data: {
    relationIds: string[];
    description: string;
  }): Promise<BatchRelationResult> {
    return httpClient.put(
      '/api/v1/tickets/relations/batch/descriptions',
      data
    );
  }

  /**
   * 批量转换关联类型
   */
  static async batchConvertRelationType(data: {
    relationIds: string[];
    newRelationType: string;
  }): Promise<BatchRelationResult> {
    return httpClient.put(
      '/api/v1/tickets/relations/batch/convert',
      data
    );
  }

  /**
   * 复制工单关联到另一个工单
   */
  static async copyRelations(data: {
    sourceTicketId: number;
    targetTicketId: number;
    relationTypes?: string[];
  }): Promise<BatchRelationResult> {
    return httpClient.post(
      '/api/v1/tickets/relations/copy',
      data
    );
  }
}

// 导出默认实例和类
export default TicketRelationsApi;

// 导出别名
export const TicketRelationsAPI = TicketRelationsApi;

