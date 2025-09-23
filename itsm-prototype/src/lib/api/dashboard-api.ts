import { httpClient } from '../../app/lib/http-client';
import {
  DashboardWidget,
  Dashboard,
  DashboardLayout,
  DashboardFilter,
  TicketStats,
  UserStats,
  SystemStats,
  ChartData,
  RealtimeData,
  Report,
  ExportDashboardRequest,
  DashboardTemplate
} from '../../types/dashboard';

/**
 * 仪表盘API客户端
 * 提供仪表盘数据获取和管理相关的API调用方法
 */
export class DashboardAPI {
  /**
   * 获取仪表盘配置
   * @param userId 用户ID
   * @returns 仪表盘配置
   */
  static async getDashboardConfig(userId?: number): Promise<Dashboard> {
    const params = userId ? { user_id: userId } : {};
    return httpClient.get<Dashboard>('/api/v1/dashboard/config', params);
  }

  /**
   * 保存仪表盘配置
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveDashboardConfig(config: Dashboard): Promise<{ success: boolean }> {
    return httpClient.post<{ success: boolean }>('/api/v1/dashboard/config', config);
  }

  /**
   * 获取仪表盘布局
   * @param userId 用户ID
   * @returns 仪表盘布局
   */
  static async getDashboardLayout(userId?: number): Promise<DashboardLayout> {
    const params = userId ? { user_id: userId } : {};
    return httpClient.get<DashboardLayout>('/api/v1/dashboard/layout', params);
  }

  /**
   * 保存仪表盘布局
   * @param layout 仪表盘布局
   * @returns 保存结果
   */
  static async saveDashboardLayout(layout: DashboardLayout): Promise<{ success: boolean }> {
    return httpClient.post<{ success: boolean }>('/api/v1/dashboard/layout', layout);
  }

  /**
   * 获取工单统计数据
   * @param filters 过滤条件
   * @returns 工单统计数据
   */
  static async getTicketStats(filters?: Record<string, unknown>): Promise<TicketStats> {
    return httpClient.get<TicketStats>('/api/v1/dashboard/stats/tickets', filters);
  }

  /**
   * 获取用户统计数据
   * @param filters 过滤条件
   * @returns 用户统计数据
   */
  static async getUserStats(filters?: Record<string, unknown>): Promise<UserStats> {
    return httpClient.get<UserStats>('/api/v1/dashboard/stats/users', filters);
  }

  /**
   * 获取系统统计数据
   * @param filters 过滤条件
   * @returns 系统统计数据
   */
  static async getSystemStats(filters?: Record<string, unknown>): Promise<SystemStats> {
    return httpClient.get<SystemStats>('/api/v1/dashboard/stats/system', filters);
  }

  /**
   * 获取图表数据
   * @param chartType 图表类型
   * @param filters 过滤条件
   * @returns 图表数据
   */
  static async getChartData(chartType: string, filters?: Record<string, unknown>): Promise<ChartData> {
    return httpClient.get<ChartData>(`/api/v1/dashboard/charts/${chartType}`, filters);
  }

  /**
   * 获取实时数据
   * @param dataType 数据类型
   * @returns 实时数据
   */
  static async getRealtimeData(dataType: string): Promise<RealtimeData> {
    return httpClient.get<RealtimeData>(`/api/v1/dashboard/realtime/${dataType}`);
  }

  /**
   * 获取部件数据
   * @param widgetId 部件ID
   * @param filters 过滤条件
   * @returns 部件数据
   */
  static async getWidgetData(widgetId: string, filters?: Record<string, unknown>): Promise<DashboardWidget> {
    return httpClient.get<DashboardWidget>(`/api/v1/dashboard/widgets/${widgetId}`, filters);
  }

  /**
   * 刷新部件数据
   * @param widgetId 部件ID
   * @param filters 过滤条件
   * @returns 部件数据
   */
  static async refreshWidgetData(widgetId: string, filters?: Record<string, unknown>): Promise<DashboardWidget> {
    return httpClient.post<DashboardWidget>(`/api/v1/dashboard/widgets/${widgetId}/refresh`, filters);
  }

  /**
   * 获取可用部件列表
   * @returns 部件列表
   */
  static async getAvailableWidgets(): Promise<DashboardWidget[]> {
    return httpClient.get<DashboardWidget[]>('/api/v1/dashboard/widgets/available');
  }

  /**
   * 添加部件到仪表盘
   * @param widgetConfig 部件配置
   * @returns 添加结果
   */
  static async addWidget(widgetConfig: Partial<DashboardWidget>): Promise<{ widget: DashboardWidget }> {
    return httpClient.post<{ widget: DashboardWidget }>('/api/v1/dashboard/widgets', widgetConfig);
  }

  /**
   * 更新部件配置
   * @param widgetId 部件ID
   * @param config 部件配置
   * @returns 更新结果
   */
  static async updateWidget(widgetId: string, config: Partial<DashboardWidget>): Promise<{ widget: DashboardWidget }> {
    return httpClient.put<{ widget: DashboardWidget }>(`/api/v1/dashboard/widgets/${widgetId}`, config);
  }

  /**
   * 删除部件
   * @param widgetId 部件ID
   * @returns 删除结果
   */
  static async removeWidget(widgetId: string): Promise<{ success: boolean }> {
    return httpClient.delete<{ success: boolean }>(`/api/v1/dashboard/widgets/${widgetId}`);
  }

  /**
   * 生成报告
   * @param reportType 报告类型
   * @param filters 过滤条件
   * @returns 报告数据
   */
  static async generateReport(reportType: string, filters?: Record<string, unknown>): Promise<Report> {
    return httpClient.post<Report>(`/api/v1/dashboard/reports/${reportType}`, filters);
  }

  /**
   * 获取报告列表
   * @param page 页码
   * @param pageSize 页面大小
   * @returns 报告列表
   */
  static async getReports(page: number = 1, pageSize: number = 20): Promise<{
    reports: DashboardReport[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    return httpClient.get('/api/v1/dashboard/reports', { page, page_size: pageSize });
  }

  /**
   * 下载报告
   * @param reportId 报告ID
   * @returns 报告文件Blob
   */
  static async downloadReport(reportId: string): Promise<Blob> {
    const response = await httpClient.get<ArrayBuffer>(`/api/v1/dashboard/reports/${reportId}/download`);
    return new Blob([response], { type: 'application/octet-stream' });
  }

  /**
   * 导出仪表盘数据
   * @param params 导出参数
   * @returns 导出结果
   */
  static async exportDashboard(params?: Record<string, unknown>): Promise<{ download_url: string }> {
    return httpClient.post<{ download_url: string }>('/api/v1/dashboard/export', params);
  }

  /**
   * 获取仪表盘模板列表
   * @returns 模板列表
   */
  static async getTemplates(): Promise<DashboardTemplate[]> {
    return httpClient.get<DashboardTemplate[]>('/api/v1/dashboard/templates');
  }

  /**
   * 应用仪表盘模板
   * @param templateId 模板ID
   * @returns 应用结果
   */
  static async applyTemplate(templateId: string): Promise<{ success: boolean; config: DashboardConfig }> {
    return httpClient.post<{ success: boolean; config: DashboardConfig }>(`/api/v1/dashboard/templates/${templateId}/apply`);
  }

  /**
   * 保存为模板
   * @param name 模板名称
   * @param description 模板描述
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveAsTemplate(name: string, description: string, config: DashboardConfig): Promise<{ template: DashboardTemplate }> {
    return httpClient.post<{ template: DashboardTemplate }>('/api/v1/dashboard/templates', {
      name,
      description,
      config
    });
  }

  /**
   * 获取仪表盘性能指标
   * @returns 性能指标
   */
  static async getPerformanceMetrics(): Promise<{
    loadTime: number;
    renderTime: number;
    dataFetchTime: number;
    widgetCount: number;
    memoryUsage: number;
  }> {
    return httpClient.get('/api/v1/dashboard/metrics/performance');
  }

  /**
   * 获取仪表盘使用统计
   * @param dateRange 日期范围
   * @returns 使用统计
   */
  static async getUsageStats(dateRange?: { start: string; end: string }): Promise<{
    totalViews: number;
    uniqueUsers: number;
    avgSessionDuration: number;
    mostUsedWidgets: Array<{ widgetId: string; usage: number }>;
    peakUsageHours: number[];
  }> {
    return httpClient.get('/api/v1/dashboard/metrics/usage', dateRange);
  }
}