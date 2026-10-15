import { httpClient } from './http-client';
import type {
  DashboardWidget,
  Dashboard,
  DashboardLayout,
  TicketStats,
  UserStats,
  SystemStats,
  ChartData,
  RealtimeData,
  Report,
  DashboardTemplate,
} from '../../types/dashboard';
import type {
  DashboardData,
  KPIMetric,
  TicketTrendData,
  IncidentDistributionData,
  SLAData,
  SatisfactionData,
  QuickAction,
  RecentActivity,
} from '@/app/(main)/dashboard/types/dashboard.types';

/**
 * 仪表盘API客户端
 * 提供仪表盘数据获取和管理相关的API调用方法
 */
export class DashboardAPI {
  /**
   * 获取仪表盘概览数据
   * 包括KPI指标、趋势图表、快速操作等所有Dashboard展示数据
   * @returns Dashboard概览数据
   */
  static async getOverview(): Promise<DashboardData> {
    return await httpClient.get<DashboardData>('/api/v1/dashboard/overview');
  }

  /**
   * 获取KPI指标数据
   * @returns KPI指标列表
   */
  static async getKPIMetrics(): Promise<KPIMetric[]> {
    return await httpClient.get<KPIMetric[]>('/api/v1/dashboard/kpi-metrics');
  }

  /**
   * 获取工单趋势数据
   * @param days 天数，默认7天
   * @returns 工单趋势数据
   */
  static async getTicketTrend(days: number = 7): Promise<TicketTrendData[]> {
    return await httpClient.get<TicketTrendData[]>('/api/v1/dashboard/ticket-trend', { days });
  }

  /**
   * 获取事件分布数据
   * @returns 事件分布数据
   */
  static async getIncidentDistribution(): Promise<IncidentDistributionData[]> {
    return await httpClient.get<IncidentDistributionData[]>(
      '/api/v1/dashboard/incident-distribution'
    );
  }

  /**
   * 获取SLA数据
   * @returns SLA数据列表
   */
  static async getSLAData(): Promise<SLAData[]> {
    return await httpClient.get<SLAData[]>('/api/v1/dashboard/sla-data');
  }

  /**
   * 获取满意度数据
   * @param months 月数，默认4个月
   * @returns 满意度数据
   */
  static async getSatisfactionData(months: number = 4): Promise<SatisfactionData[]> {
    return await httpClient.get<SatisfactionData[]>('/api/v1/dashboard/satisfaction-data', {
      months,
    });
  }

  /**
   * 获取快速操作列表
   * @returns 快速操作列表
   */
  static async getQuickActions(): Promise<QuickAction[]> {
    return await httpClient.get<QuickAction[]>('/api/v1/dashboard/quick-actions');
  }

  /**
   * 获取最近活动
   * @param limit 限制数量，默认10条
   * @returns 最近活动列表
   */
  static async getRecentActivities(limit: number = 10): Promise<RecentActivity[]> {
    return await httpClient.get<RecentActivity[]>('/api/v1/dashboard/recent-activities', { limit });
  }

  /**
   * 获取仪表盘配置
   * @param userId 用户ID
   * @returns 仪表盘配置
   */
  static async getDashboardConfig(userId?: number): Promise<Dashboard> {
    const params = userId ? { userId: userId } : {};
    return await httpClient.get<Dashboard>('/api/v1/dashboard/config', params);
  }

  /**
   * 保存仪表盘配置
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveDashboardConfig(config: Dashboard): Promise<{ success: boolean }> {
    return await httpClient.post<{ success: boolean }>('/api/v1/dashboard/config', config);
  }

  /**
   * 获取仪表盘布局
   * @param userId 用户ID
   * @returns 仪表盘布局
   */
  static async getDashboardLayout(userId?: number): Promise<DashboardLayout> {
    const params = userId ? { userId: userId } : {};
    return await httpClient.get<DashboardLayout>('/api/v1/dashboard/layout', params);
  }

  /**
   * 保存仪表盘布局
   * @param layout 仪表盘布局
   * @returns 保存结果
   */
  static async saveDashboardLayout(layout: DashboardLayout): Promise<{ success: boolean }> {
    return await httpClient.post<{ success: boolean }>('/api/v1/dashboard/layout', layout);
  }

  /**
   * 获取工单统计数据
   * @param filters 过滤条件
   * @returns 工单统计数据
   */
  static async getTicketStats(
    filters?: Record<string, string | number | boolean>
  ): Promise<TicketStats> {
    return await httpClient.get<TicketStats>('/api/v1/dashboard/stats/tickets', filters);
  }

  /**
   * 获取用户统计数据
   * @param filters 过滤条件
   * @returns 用户统计数据
   */
  static async getUserStats(
    filters?: Record<string, string | number | boolean>
  ): Promise<UserStats> {
    return await httpClient.get<UserStats>('/api/v1/dashboard/stats/users', filters);
  }

  /**
   * 获取系统统计数据
   * @param filters 过滤条件
   * @returns 系统统计数据
   */
  static async getSystemStats(
    filters?: Record<string, string | number | boolean>
  ): Promise<SystemStats> {
    return await httpClient.get<SystemStats>('/api/v1/dashboard/stats/system', filters);
  }

  /**
   * 获取可用部件列表
   * @returns 部件列表
   */
  static async getAvailableWidgets(): Promise<DashboardWidget[]> {
    return await httpClient.get<DashboardWidget[]>('/api/v1/dashboard/widgets/available');
  }

  /**
   * 添加部件到仪表盘
   * @param widgetConfig 部件配置
   * @returns 添加结果
   */
  static async addWidget(
    widgetConfig: Partial<DashboardWidget>
  ): Promise<{ widget: DashboardWidget }> {
    return await httpClient.post<{ widget: DashboardWidget }>(
      '/api/v1/dashboard/widgets',
      widgetConfig
    );
  }

  /**
   * 删除部件
   * @param widgetId 部件ID
   * @returns 删除结果
   */
  static async removeWidget(widgetId: string): Promise<{ success: boolean }> {
    return await httpClient.delete<{ success: boolean }>(`/api/v1/dashboard/widgets/${widgetId}`);
  }

  /**
   * 获取仪表盘模板列表
   * @returns 模板列表
   */
  static async getTemplates(): Promise<DashboardTemplate[]> {
    return await httpClient.get<DashboardTemplate[]>('/api/v1/dashboard/templates');
  }

  /**
   * 应用仪表盘模板
   * @param templateId 模板ID
   * @returns 应用结果
   */
  static async applyTemplate(templateId: string): Promise<{ success: boolean; config: Dashboard }> {
    return await httpClient.post<{ success: boolean; config: Dashboard }>(
      `/api/v1/dashboard/templates/${templateId}/apply`
    );
  }

  /**
   * 保存为模板
   * @param name 模板名称
   * @param description 模板描述
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveAsTemplate(
    name: string,
    description: string,
    config: Dashboard
  ): Promise<{ template: DashboardTemplate }> {
    return await httpClient.post<{ template: DashboardTemplate }>('/api/v1/dashboard/templates', {
      name,
      description,
      config,
    });
  }
}
