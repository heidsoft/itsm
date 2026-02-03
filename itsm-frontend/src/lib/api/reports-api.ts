/**
 * 报表系统 API 服务
 */

import { httpClient } from './http-client';
import type {
  ReportDefinition,
  ReportTemplate,
  ReportExecutionResult,
  ReportStats,
  ReportPerformance,
  CreateReportRequest,
  UpdateReportRequest,
  ExecuteReportRequest,
  ReportQuery,
  DatasetDefinition,
} from '@/types/reports';

export class ReportsApi {
  // ==================== 报表管理 ====================

  /**
   * 获取报表列表
   */
  static async getReports(
    query?: ReportQuery
  ): Promise<{
    reports: ReportDefinition[];
    total: number;
  }> {
    return httpClient.get('/api/v1/reports', query);
  }

  /**
   * 获取单个报表
   */
  static async getReport(id: string): Promise<ReportDefinition> {
    return httpClient.get(`/api/v1/reports/${id}`);
  }

  /**
   * 创建报表
   */
  static async createReport(
    request: CreateReportRequest
  ): Promise<ReportDefinition> {
    return httpClient.post('/api/v1/reports', request);
  }

  /**
   * 更新报表
   */
  static async updateReport(
    id: string,
    request: UpdateReportRequest
  ): Promise<ReportDefinition> {
    return httpClient.put(`/api/v1/reports/${id}`, request);
  }

  /**
   * 删除报表
   */
  static async deleteReport(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/reports/${id}`);
  }

  /**
   * 复制报表
   */
  static async cloneReport(
    id: string,
    name: string
  ): Promise<ReportDefinition> {
    return httpClient.post(`/api/v1/reports/${id}/clone`, { name });
  }

  /**
   * 更新报表状态
   */
  static async updateReportStatus(
    id: string,
    status: 'active' | 'inactive' | 'archived'
  ): Promise<ReportDefinition> {
    return httpClient.patch(`/api/v1/reports/${id}/status`, { status });
  }

  // ==================== 报表执行 ====================

  /**
   * 执行报表
   */
  static async executeReport(
    request: ExecuteReportRequest
  ): Promise<ReportExecutionResult> {
    return httpClient.post('/api/v1/reports/execute', request);
  }

  /**
   * 获取报表执行历史
   */
  static async getExecutionHistory(
    reportId: string,
    params?: {
      page?: number;
      pageSize?: number;
      startDate?: string;
      endDate?: string;
    }
  ): Promise<{
    executions: ReportExecutionResult[];
    total: number;
  }> {
    return httpClient.get(`/api/v1/reports/${reportId}/executions`, params);
  }

  /**
   * 获取执行结果
   */
  static async getExecutionResult(
    executionId: string
  ): Promise<ReportExecutionResult> {
    return httpClient.get(`/api/v1/reports/executions/${executionId}`);
  }

  /**
   * 取消报表执行
   */
  static async cancelExecution(executionId: string): Promise<void> {
    return httpClient.post(`/api/v1/reports/executions/${executionId}/cancel`);
  }

  // ==================== 报表导出 ====================

  /**
   * 导出报表
   */
  static async exportReport(params: {
    reportId: string;
    format: 'pdf' | 'excel' | 'csv' | 'html';
    filters?: Record<string, unknown>;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: `/api/v1/reports/${params.reportId}/export`,
      data: {
        format: params.format,
        filters: params.filters,
      },
      responseType: 'blob',
    });
    return response as Blob;
  }

  /**
   * 发送报表到邮箱
   */
  static async emailReport(params: {
    reportId: string;
    recipients: string[];
    format: 'pdf' | 'excel' | 'csv' | 'html';
    subject?: string;
    message?: string;
    filters?: Record<string, any>;
  }): Promise<void> {
    return httpClient.post(`/api/v1/reports/${params.reportId}/email`, params);
  }

  // ==================== 报表调度 ====================

  /**
   * 创建报表调度
   */
  static async createSchedule(
    reportId: string,
    schedule: Parameters<typeof ReportsApi.updateSchedule>[1]
  ): Promise<ReportDefinition> {
    return httpClient.post(`/api/v1/reports/${reportId}/schedule`, schedule);
  }

  /**
   * 更新报表调度
   */
  static async updateSchedule(
    reportId: string,
    schedule: {
      enabled: boolean;
      frequency: 'once' | 'daily' | 'weekly' | 'monthly' | 'cron';
      time?: string;
      dayOfWeek?: number[];
      dayOfMonth?: number;
      cronExpression?: string;
      recipients: Array<{
        type: 'user' | 'group' | 'email';
        id?: number;
        email?: string;
      }>;
      outputFormat: 'pdf' | 'excel' | 'csv' | 'html';
    }
  ): Promise<ReportDefinition> {
    return httpClient.put(`/api/v1/reports/${reportId}/schedule`, schedule);
  }

  /**
   * 删除报表调度
   */
  static async deleteSchedule(reportId: string): Promise<void> {
    return httpClient.delete(`/api/v1/reports/${reportId}/schedule`);
  }

  /**
   * 立即执行调度
   */
  static async runScheduleNow(reportId: string): Promise<ReportExecutionResult> {
    return httpClient.post(`/api/v1/reports/${reportId}/schedule/run`);
  }

  // ==================== 报表模板 ====================

  /**
   * 获取报表模板列表
   */
  static async getTemplates(params?: {
    category?: string;
    tags?: string[];
    page?: number;
    pageSize?: number;
  }): Promise<{
    templates: ReportTemplate[];
    total: number;
  }> {
    return httpClient.get('/api/v1/reports/templates', params);
  }

  /**
   * 获取模板详情
   */
  static async getTemplate(id: string): Promise<ReportTemplate> {
    return httpClient.get(`/api/v1/reports/templates/${id}`);
  }

  /**
   * 从模板创建报表
   */
  static async createFromTemplate(
    templateId: string,
    name: string
  ): Promise<ReportDefinition> {
    return httpClient.post(
      `/api/v1/reports/templates/${templateId}/create`,
      { name }
    );
  }

  /**
   * 保存为模板
   */
  static async saveAsTemplate(
    reportId: string,
    params: {
      name: string;
      category: string;
      description?: string;
      isPublic?: boolean;
    }
  ): Promise<ReportTemplate> {
    return httpClient.post(`/api/v1/reports/${reportId}/save-as-template`, params);
  }

  // ==================== 数据源和数据集 ====================

  /**
   * 获取数据集列表
   */
  static async getDatasets(): Promise<DatasetDefinition[]> {
    return httpClient.get('/api/v1/reports/datasets');
  }

  /**
   * 获取数据集详情
   */
  static async getDataset(id: string): Promise<DatasetDefinition> {
    return httpClient.get(`/api/v1/reports/datasets/${id}`);
  }

  /**
   * 预览数据
   */
  static async previewData(params: {
    dataSource: {
      type: string;
      query?: string;
      apiEndpoint?: string;
      datasetId?: string;
    };
    filters?: Record<string, unknown>;
    limit?: number;
  }): Promise<{
    columns: string[];
    rows: unknown[][];
    total: number;
  }> {
    return httpClient.post('/api/v1/reports/preview', params);
  }

  /**
   * 验证查询
   */
  static async validateQuery(query: string): Promise<{
    valid: boolean;
    error?: string;
    columns?: string[];
  }> {
    return httpClient.post('/api/v1/reports/validate-query', { query });
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取报表统计
   */
  static async getStats(): Promise<ReportStats> {
    return httpClient.get('/api/v1/reports/stats');
  }

  /**
   * 获取报表性能分析
   */
  static async getPerformance(
    reportId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<ReportPerformance> {
    return httpClient.get(`/api/v1/reports/${reportId}/performance`, params);
  }

  /**
   * 获取我的最近报表
   */
  static async getRecentReports(limit = 10): Promise<ReportDefinition[]> {
    return httpClient.get('/api/v1/reports/recent', { limit });
  }

  /**
   * 获取我收藏的报表
   */
  static async getFavoriteReports(): Promise<ReportDefinition[]> {
    return httpClient.get('/api/v1/reports/favorites');
  }

  /**
   * 收藏报表
   */
  static async favoriteReport(reportId: string): Promise<void> {
    return httpClient.post(`/api/v1/reports/${reportId}/favorite`);
  }

  /**
   * 取消收藏
   */
  static async unfavoriteReport(reportId: string): Promise<void> {
    return httpClient.delete(`/api/v1/reports/${reportId}/favorite`);
  }

  // ==================== 分享和协作 ====================

  /**
   * 分享报表
   */
  static async shareReport(
    reportId: string,
    params: {
      userIds?: number[];
      groupIds?: number[];
      permission: 'view' | 'edit';
    }
  ): Promise<void> {
    return httpClient.post(`/api/v1/reports/${reportId}/share`, params);
  }

  /**
   * 取消分享
   */
  static async unshareReport(
    reportId: string,
    userId: number
  ): Promise<void> {
    return httpClient.delete(`/api/v1/reports/${reportId}/share/${userId}`);
  }

  /**
   * 获取分享列表
   */
  static async getSharedUsers(reportId: string): Promise<
    Array<{
      userId: number;
      userName: string;
      permission: 'view' | 'edit';
      sharedAt: Date;
    }>
  > {
    return httpClient.get(`/api/v1/reports/${reportId}/shares`);
  }
}

export default ReportsApi;
export const ReportsAPI = ReportsApi;

