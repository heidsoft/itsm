/**
 * 批量操作 API 服务
 * 提供工单批量操作的完整API接口
 */

import { httpClient } from './http-client';
import type {
  BatchOperationRequest,
  BatchOperationResponse,
  BatchOperationProgress,
  BatchOperationLog,
  BatchOperationValidation,
  BatchOperationPreview,
  BatchOperationStats,
  BatchOperationPermissions,
  BatchExportConfig,
} from '@/types/batch-operations';

export class BatchOperationsApi {
  // ==================== 批量操作执行 ====================

  /**
   * 执行批量操作
   */
  static async executeBatchOperation(
    request: BatchOperationRequest
  ): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/execute',
      request
    );
  }

  /**
   * 验证批量操作
   */
  static async validateBatchOperation(
    request: BatchOperationRequest
  ): Promise<BatchOperationValidation> {
    return httpClient.post<BatchOperationValidation>(
      '/api/v1/tickets/batch/validate',
      request
    );
  }

  /**
   * 预览批量操作
   */
  static async previewBatchOperation(
    request: BatchOperationRequest
  ): Promise<BatchOperationPreview> {
    return httpClient.post<BatchOperationPreview>(
      '/api/v1/tickets/batch/preview',
      request
    );
  }

  // ==================== 批量分配 ====================

  /**
   * 批量分配工单
   */
  static async batchAssignTickets(data: {
    ticketIds: number[];
    assigneeId?: number;
    teamId?: number;
    assignmentRule?: 'round_robin' | 'load_balance' | 'manual';
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/assign',
      data
    );
  }

  /**
   * 批量轮流分配
   */
  static async batchAssignRoundRobin(data: {
    ticketIds: number[];
    teamId: number;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/assign/round-robin',
      data
    );
  }

  /**
   * 批量负载均衡分配
   */
  static async batchAssignLoadBalance(data: {
    ticketIds: number[];
    teamId: number;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/assign/load-balance',
      data
    );
  }

  // ==================== 批量状态更新 ====================

  /**
   * 批量更新工单状态
   */
  static async batchUpdateStatus(data: {
    ticketIds: number[];
    status: string;
    resolution?: string;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/status',
      data
    );
  }

  /**
   * 批量关闭工单
   */
  static async batchCloseTickets(data: {
    ticketIds: number[];
    closureReason?: string;
    resolution?: string;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/close',
      data
    );
  }

  /**
   * 批量重新打开工单
   */
  static async batchReopenTickets(data: {
    ticketIds: number[];
    reason?: string;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/reopen',
      data
    );
  }

  // ==================== 批量字段更新 ====================

  /**
   * 批量更新优先级
   */
  static async batchUpdatePriority(data: {
    ticketIds: number[];
    priority: string;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/priority',
      data
    );
  }

  /**
   * 批量更新类型
   */
  static async batchUpdateType(data: {
    ticketIds: number[];
    type: string;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/type',
      data
    );
  }

  /**
   * 批量更新分类
   */
  static async batchUpdateCategory(data: {
    ticketIds: number[];
    categoryId: number;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/category',
      data
    );
  }

  /**
   * 批量更新自定义字段
   */
  static async batchUpdateFields(data: {
    ticketIds: number[];
    customFields: Record<string, any>;
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/fields',
      data
    );
  }

  // ==================== 批量标签操作 ====================

  /**
   * 批量添加标签
   */
  static async batchAddTags(data: {
    ticketIds: number[];
    tags: string[];
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/tags/add',
      data
    );
  }

  /**
   * 批量删除标签
   */
  static async batchRemoveTags(data: {
    ticketIds: number[];
    tags: string[];
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/tags/remove',
      data
    );
  }

  /**
   * 批量替换标签
   */
  static async batchReplaceTags(data: {
    ticketIds: number[];
    tags: string[];
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/tags/replace',
      data
    );
  }

  // ==================== 批量删除和归档 ====================

  /**
   * 批量删除工单
   */
  static async batchDeleteTickets(data: {
    ticketIds: number[];
    reason?: string;
    hardDelete?: boolean;
  }): Promise<BatchOperationResponse> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/tickets/batch/delete',
      data,
    });
  }

  /**
   * 批量归档工单
   */
  static async batchArchiveTickets(data: {
    ticketIds: number[];
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/archive',
      data
    );
  }

  /**
   * 批量取消归档
   */
  static async batchUnarchiveTickets(data: {
    ticketIds: number[];
    comment?: string;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/unarchive',
      data
    );
  }

  // ==================== 批量导出 ====================

  /**
   * 批量导出工单
   */
  static async batchExportTickets(data: {
    ticketIds?: number[];
    filters?: Record<string, any>;
    config: BatchExportConfig;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/tickets/batch/export',
      data,
      responseType: 'blob',
    });
    return response as Blob;
  }

  /**
   * 获取导出状态
   */
  static async getExportStatus(
    exportId: string
  ): Promise<{ status: string; downloadUrl?: string; progress?: number }> {
    return httpClient.get(`/api/v1/tickets/batch/export/${exportId}/status`);
  }

  /**
   * 下载导出文件
   */
  static async downloadExport(exportId: string): Promise<Blob> {
    const response = await httpClient.request({
      method: 'GET',
      url: `/api/v1/tickets/batch/export/${exportId}/download`,
      responseType: 'blob',
    });
    return response as Blob;
  }

  // ==================== 批量操作进度 ====================

  /**
   * 获取批量操作进度
   */
  static async getBatchOperationProgress(
    operationId: string
  ): Promise<BatchOperationProgress> {
    return httpClient.get<BatchOperationProgress>(
      `/api/v1/tickets/batch/operations/${operationId}/progress`
    );
  }

  /**
   * 暂停批量操作
   */
  static async pauseBatchOperation(operationId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/tickets/batch/operations/${operationId}/pause`
    );
  }

  /**
   * 恢复批量操作
   */
  static async resumeBatchOperation(operationId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/tickets/batch/operations/${operationId}/resume`
    );
  }

  /**
   * 取消批量操作
   */
  static async cancelBatchOperation(operationId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/tickets/batch/operations/${operationId}/cancel`
    );
  }

  // ==================== 批量操作日志 ====================

  /**
   * 获取批量操作日志列表
   */
  static async getBatchOperationLogs(params?: {
    page?: number;
    pageSize?: number;
    operationType?: string;
    status?: string;
    operatorId?: number;
    startDate?: string;
    endDate?: string;
  }): Promise<{
    logs: BatchOperationLog[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    return httpClient.get('/api/v1/tickets/batch/operations/logs', params);
  }

  /**
   * 获取批量操作日志详情
   */
  static async getBatchOperationLog(
    logId: string
  ): Promise<BatchOperationLog> {
    return httpClient.get<BatchOperationLog>(
      `/api/v1/tickets/batch/operations/logs/${logId}`
    );
  }

  /**
   * 删除批量操作日志
   */
  static async deleteBatchOperationLog(logId: string): Promise<void> {
    return httpClient.delete(
      `/api/v1/tickets/batch/operations/logs/${logId}`
    );
  }

  // ==================== 批量操作统计 ====================

  /**
   * 获取批量操作统计
   */
  static async getBatchOperationStats(params?: {
    startDate?: string;
    endDate?: string;
    groupBy?: 'day' | 'week' | 'month';
  }): Promise<BatchOperationStats> {
    return httpClient.get<BatchOperationStats>(
      '/api/v1/tickets/batch/operations/stats',
      params
    );
  }

  /**
   * 获取当前用户批量操作统计
   */
  static async getMyBatchOperationStats(): Promise<{
    totalOperations: number;
    successRate: number;
    recentOperations: BatchOperationLog[];
  }> {
    return httpClient.get('/api/v1/tickets/batch/operations/stats/me');
  }

  // ==================== 批量操作权限 ====================

  /**
   * 获取批量操作权限
   */
  static async getBatchOperationPermissions(): Promise<BatchOperationPermissions> {
    return httpClient.get<BatchOperationPermissions>(
      '/api/v1/tickets/batch/operations/permissions'
    );
  }

  /**
   * 检查是否可以执行批量操作
   */
  static async canExecuteBatchOperation(data: {
    operationType: string;
    ticketIds: number[];
  }): Promise<{ canExecute: boolean; reason?: string }> {
    return httpClient.post(
      '/api/v1/tickets/batch/operations/can-execute',
      data
    );
  }

  // ==================== 批量操作模板 ====================

  /**
   * 保存批量操作为模板
   */
  static async saveBatchOperationTemplate(data: {
    name: string;
    description?: string;
    operationType: string;
    defaultData: any;
  }): Promise<{ id: string }> {
    return httpClient.post(
      '/api/v1/tickets/batch/operations/templates',
      data
    );
  }

  /**
   * 获取批量操作模板列表
   */
  static async getBatchOperationTemplates(): Promise<
    Array<{
      id: string;
      name: string;
      description: string;
      operationType: string;
      usageCount: number;
    }>
  > {
    return httpClient.get('/api/v1/tickets/batch/operations/templates');
  }

  /**
   * 使用模板执行批量操作
   */
  static async executeBatchOperationFromTemplate(data: {
    templateId: string;
    ticketIds: number[];
    overrides?: any;
  }): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      '/api/v1/tickets/batch/operations/execute-template',
      data
    );
  }

  // ==================== 批量操作撤销 ====================

  /**
   * 撤销批量操作
   */
  static async undoBatchOperation(
    operationId: string
  ): Promise<BatchOperationResponse> {
    return httpClient.post<BatchOperationResponse>(
      `/api/v1/tickets/batch/operations/${operationId}/undo`
    );
  }

  /**
   * 检查批量操作是否可以撤销
   */
  static async canUndoBatchOperation(operationId: string): Promise<{
    canUndo: boolean;
    reason?: string;
    affectedTickets?: number;
  }> {
    return httpClient.get(
      `/api/v1/tickets/batch/operations/${operationId}/can-undo`
    );
  }

  // ==================== 批量操作调度 ====================

  /**
   * 调度批量操作（定时执行）
   */
  static async scheduleBatchOperation(data: {
    request: BatchOperationRequest;
    scheduledAt: string;
    recurrence?: {
      frequency: 'daily' | 'weekly' | 'monthly';
      interval: number;
      endDate?: string;
    };
  }): Promise<{ scheduleId: string }> {
    return httpClient.post('/api/v1/tickets/batch/operations/schedule', data);
  }

  /**
   * 取消调度的批量操作
   */
  static async cancelScheduledOperation(scheduleId: string): Promise<void> {
    return httpClient.delete(
      `/api/v1/tickets/batch/operations/schedule/${scheduleId}`
    );
  }

  /**
   * 获取调度的批量操作列表
   */
  static async getScheduledOperations(): Promise<
    Array<{
      id: string;
      operationType: string;
      scheduledAt: string;
      status: string;
      ticketCount: number;
    }>
  > {
    return httpClient.get('/api/v1/tickets/batch/operations/schedule');
  }
}

// 导出默认实例和类
export default BatchOperationsApi;

// 导出别名以支持不同的导入方式
export const BatchOperationsAPI = BatchOperationsApi;

