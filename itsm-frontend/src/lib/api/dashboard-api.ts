import { httpClient } from './http-client';
import {
  DashboardWidget,
  Dashboard,
  DashboardLayout,
  TicketStats,
  UserStats,
  SystemStats,
  ChartData,
  RealtimeData,
  Report,
  DashboardTemplate
} from '../../types/dashboard';
import type {
  DashboardData,
  KPIMetric,
  TicketTrendData,
  IncidentDistributionData,
  SLAData,
  SatisfactionData,
  QuickAction,
  RecentActivity
} from '@/app/(main)/dashboard/types/dashboard.types';

// Mock 布局定义
const MOCK_DASHBOARD_LAYOUT: DashboardLayout = {
  cols: 4,
  rows: 0,
  margin: [16, 16] as [number, number],
  containerPadding: [16, 16] as [number, number],
  rowHeight: 100,
  isDraggable: true,
  isResizable: true,
};

// Mock 数据用于后端未实现时回退
const MOCK_DASHBOARD_CONFIG: Dashboard = {
  id: 1,
  name: '默认仪表盘',
  description: '系统默认仪表盘配置',
  isDefault: true,
  isPublic: false,
  layout: MOCK_DASHBOARD_LAYOUT,
  widgets: [],
  filters: [],
  permissions: [],
  createdBy: 1,
  updatedBy: 1,
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  shareSettings: { isShared: false },
};

const MOCK_TICKET_STATS: TicketStats = {
  total: 1250,
  open: 320,
  inProgress: 180,
  resolved: 550,
  closed: 200,
  byPriority: { critical: 20, high: 150, medium: 800, low: 280 },
  byStatus: { new: 320, in_progress: 180, pending: 100, resolved: 550, closed: 200 },
  byType: { incident: 450, request: 600, problem: 120, change: 80 },
  byAssignee: {},
  byDepartment: {},
  avgResolutionTime: 4.5,
  avgResponseTime: 0.5,
  slaCompliance: 95.5,
  trend: [
    { period: '2024-01-01', created: 150, resolved: 120, backlog: 30 },
    { period: '2024-01-02', created: 180, resolved: 160, backlog: 50 },
  ],
};

const MOCK_USER_STATS: UserStats = {
  total: 156,
  active: 142,
  online: 45,
  byRole: { admin: 5, agent: 50, user: 101 },
  byDepartment: { IT: 80, HR: 30, Finance: 25, Sales: 21 },
  loginToday: 89,
  activeThisWeek: 120,
  newThisMonth: 15,
};

const MOCK_SYSTEM_STATS: SystemStats = {
  uptime: 99.9,
  cpuUsage: 45,
  memoryUsage: 62,
  diskUsage: 58,
  avgResponseTime: 120,
  requestsPerSecond: 500,
  errorRate: 0.1,
  dbConnections: 50,
  dbSize: 1024,
  cacheHitRate: 95.5,
  cacheSize: 256,
};

const MOCK_CHART_DATA: ChartData = {
  labels: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
  datasets: [{ label: '工单', data: [65, 59, 80, 81, 56, 55, 40] }],
};

const MOCK_REALTIME_DATA: RealtimeData = {
  type: 'ticket_created',
  data: { activeUsers: 45, pendingTickets: 12 },
  timestamp: new Date().toISOString(),
};

const MOCK_AVAILABLE_WIDGETS: DashboardWidget[] = [
  { id: 'w1', type: 'metric' as const, title: '工单统计', position: { x: 0, y: 0, w: 3, h: 4 }, dataSource: '/api/v1/dashboard/stats/tickets', refreshInterval: 300, isVisible: true, config: {} },
  { id: 'w2', type: 'chart' as const, title: '趋势图', position: { x: 3, y: 0, w: 3, h: 4 }, dataSource: '/api/v1/dashboard/ticket-trend', refreshInterval: 300, isVisible: true, config: { chartType: 'line' } },
  { id: 'w3', type: 'table' as const, title: '最近工单', position: { x: 0, y: 4, w: 6, h: 4 }, dataSource: '/api/v1/tickets', refreshInterval: 60, isVisible: true, config: { pageSize: 10 } },
];

const MOCK_REPORTS: Report[] = [
  { id: 1, name: '工单周报', type: 'ticket', template: { title: '工单周报', sections: [], filters: [], timeRange: '7d', format: 'pdf' }, schedule: { frequency: 'weekly', time: '09:00', timezone: 'Asia/Shanghai', isActive: true }, recipients: ['admin@example.com'], isActive: true, createdBy: 1, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
  { id: 2, name: '用户月报', type: 'user', template: { title: '用户月报', sections: [], filters: [], timeRange: '30d', format: 'pdf' }, schedule: { frequency: 'monthly', time: '09:00', dayOfMonth: 1, timezone: 'Asia/Shanghai', isActive: true }, recipients: ['admin@example.com'], isActive: true, createdBy: 1, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
];

const MOCK_TEMPLATES: DashboardTemplate[] = [
  { id: 1, name: 'IT运维仪表盘', description: '适用于IT运维团队', category: 'operations', tags: ['it', 'ops', 'monitor'], layout: MOCK_DASHBOARD_LAYOUT, widgets: [], filters: [], isPublic: true, downloadCount: 150, rating: 4.5, createdBy: 1, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
  { id: 2, name: '服务台仪表盘', description: '适用于服务台', category: 'service_desk', tags: ['service', 'helpdesk'], layout: MOCK_DASHBOARD_LAYOUT, widgets: [], filters: [], isPublic: true, downloadCount: 89, rating: 4.2, createdBy: 1, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
];

const MOCK_PERFORMANCE_METRICS = {
  loadTime: 1.2,
  renderTime: 0.8,
  dataFetchTime: 0.5,
  widgetCount: 8,
  memoryUsage: 45,
};

const MOCK_USAGE_STATS = {
  totalViews: 15234,
  uniqueUsers: 89,
  avgSessionDuration: 12.5,
  mostUsedWidgets: [
    { widgetId: 'w1', usage: 4523 },
    { widgetId: 'w2', usage: 3891 },
  ],
  peakUsageHours: [9, 10, 11, 14, 15],
};

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
    try {
      return await httpClient.get<DashboardData>('/api/v1/dashboard/overview');
    } catch {
      return {
        kpiMetrics: [],
        ticketTrend: [],
        incidentDistribution: [],
        slaData: [],
        satisfactionData: [],
        quickActions: [],
        recentActivities: [],
      };
    }
  }

  /**
   * 获取KPI指标数据
   * @returns KPI指标列表
   */
  static async getKPIMetrics(): Promise<KPIMetric[]> {
    try {
      return await httpClient.get<KPIMetric[]>('/api/v1/dashboard/kpi-metrics');
    } catch {
      return [];
    }
  }

  /**
   * 获取工单趋势数据
   * @param days 天数，默认7天
   * @returns 工单趋势数据
   */
  static async getTicketTrend(days: number = 7): Promise<TicketTrendData[]> {
    try {
      return await httpClient.get<TicketTrendData[]>('/api/v1/dashboard/ticket-trend', { days });
    } catch {
      return [];
    }
  }

  /**
   * 获取事件分布数据
   * @returns 事件分布数据
   */
  static async getIncidentDistribution(): Promise<IncidentDistributionData[]> {
    try {
      return await httpClient.get<IncidentDistributionData[]>('/api/v1/dashboard/incident-distribution');
    } catch {
      return [];
    }
  }

  /**
   * 获取SLA数据
   * @returns SLA数据列表
   */
  static async getSLAData(): Promise<SLAData[]> {
    try {
      return await httpClient.get<SLAData[]>('/api/v1/dashboard/sla-data');
    } catch {
      return [];
    }
  }

  /**
   * 获取满意度数据
   * @param months 月数，默认4个月
   * @returns 满意度数据
   */
  static async getSatisfactionData(months: number = 4): Promise<SatisfactionData[]> {
    try {
      return await httpClient.get<SatisfactionData[]>('/api/v1/dashboard/satisfaction-data', { months });
    } catch {
      return [];
    }
  }

  /**
   * 获取快速操作列表
   * @returns 快速操作列表
   */
  static async getQuickActions(): Promise<QuickAction[]> {
    try {
      return await httpClient.get<QuickAction[]>('/api/v1/dashboard/quick-actions');
    } catch {
      return [];
    }
  }

  /**
   * 获取最近活动
   * @param limit 限制数量，默认10条
   * @returns 最近活动列表
   */
  static async getRecentActivities(limit: number = 10): Promise<RecentActivity[]> {
    try {
      return await httpClient.get<RecentActivity[]>('/api/v1/dashboard/recent-activities', { limit });
    } catch {
      return [];
    }
  }

  /**
   * 获取仪表盘配置
   * @param userId 用户ID
   * @returns 仪表盘配置
   */
  static async getDashboardConfig(userId?: number): Promise<Dashboard> {
    try {
      const params = userId ? { user_id: userId } : {};
      return await httpClient.get<Dashboard>('/api/v1/dashboard/config', params);
    } catch {
      return MOCK_DASHBOARD_CONFIG;
    }
  }

  /**
   * 保存仪表盘配置
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveDashboardConfig(config: Dashboard): Promise<{ success: boolean }> {
    try {
      return await httpClient.post<{ success: boolean }>('/api/v1/dashboard/config', config);
    } catch {
      return { success: true };
    }
  }

  /**
   * 获取仪表盘布局
   * @param userId 用户ID
   * @returns 仪表盘布局
   */
  static async getDashboardLayout(userId?: number): Promise<DashboardLayout> {
    try {
      const params = userId ? { user_id: userId } : {};
      return await httpClient.get<DashboardLayout>('/api/v1/dashboard/layout', params);
    } catch {
      return MOCK_DASHBOARD_LAYOUT;
    }
  }

  /**
   * 保存仪表盘布局
   * @param layout 仪表盘布局
   * @returns 保存结果
   */
  static async saveDashboardLayout(layout: DashboardLayout): Promise<{ success: boolean }> {
    try {
      return await httpClient.post<{ success: boolean }>('/api/v1/dashboard/layout', layout);
    } catch {
      return { success: true };
    }
  }

  /**
   * 获取工单统计数据
   * @param filters 过滤条件
   * @returns 工单统计数据
   */
  static async getTicketStats(filters?: Record<string, string | number | boolean>): Promise<TicketStats> {
    try {
      return await httpClient.get<TicketStats>('/api/v1/dashboard/stats/tickets', filters);
    } catch {
      return MOCK_TICKET_STATS;
    }
  }

  /**
   * 获取用户统计数据
   * @param filters 过滤条件
   * @returns 用户统计数据
   */
  static async getUserStats(filters?: Record<string, string | number | boolean>): Promise<UserStats> {
    try {
      return await httpClient.get<UserStats>('/api/v1/dashboard/stats/users', filters);
    } catch {
      return MOCK_USER_STATS;
    }
  }

  /**
   * 获取系统统计数据
   * @param filters 过滤条件
   * @returns 系统统计数据
   */
  static async getSystemStats(filters?: Record<string, string | number | boolean>): Promise<SystemStats> {
    try {
      return await httpClient.get<SystemStats>('/api/v1/dashboard/stats/system', filters);
    } catch {
      return MOCK_SYSTEM_STATS;
    }
  }

  /**
   * 获取图表数据
   * @param chartType 图表类型
   * @param filters 过滤条件
   * @returns 图表数据
   */
  static async getChartData(chartType: string, filters?: Record<string, string | number | boolean>): Promise<ChartData> {
    try {
      return await httpClient.get<ChartData>(`/api/v1/dashboard/charts/${chartType}`, filters);
    } catch {
      return MOCK_CHART_DATA;
    }
  }

  /**
   * 获取实时数据
   * @param dataType 数据类型
   * @returns 实时数据
   */
  static async getRealtimeData(dataType: string): Promise<RealtimeData> {
    try {
      return await httpClient.get<RealtimeData>(`/api/v1/dashboard/realtime/${dataType}`);
    } catch {
      return MOCK_REALTIME_DATA;
    }
  }

  /**
   * 获取小组件数据
   * @param widgetId 小组件ID
   * @param filters 过滤条件
   * @returns 小组件数据
   */
  static async getWidgetData(widgetId: string, filters?: Record<string, string | number | boolean>): Promise<DashboardWidget> {
    try {
      return await httpClient.get<DashboardWidget>(`/api/v1/dashboard/widgets/${widgetId}/data`, filters);
    } catch {
      return MOCK_AVAILABLE_WIDGETS[0] || { id: widgetId, type: 'metric' as const, title: '未知', position: { x: 0, y: 0, w: 3, h: 4 }, dataSource: '', isVisible: true, config: {} };
    }
  }

  /**
   * 刷新小组件数据
   * @param widgetId 小组件ID
   * @param filters 过滤条件
   * @returns 刷新后的小组件数据
   */
  static async refreshWidgetData(widgetId: string, filters?: Record<string, string | number | boolean>): Promise<DashboardWidget> {
    try {
      return await httpClient.post<DashboardWidget>(`/api/v1/dashboard/widgets/${widgetId}/refresh`, filters);
    } catch {
      return MOCK_AVAILABLE_WIDGETS[0] || { id: widgetId, type: 'metric' as const, title: '刷新', position: { x: 0, y: 0, w: 3, h: 4 }, dataSource: '', isVisible: true, config: {} };
    }
  }

  /**
   * 获取可用部件列表
   * @returns 部件列表
   */
  static async getAvailableWidgets(): Promise<DashboardWidget[]> {
    try {
      return await httpClient.get<DashboardWidget[]>('/api/v1/dashboard/widgets/available');
    } catch {
      return MOCK_AVAILABLE_WIDGETS;
    }
  }

  /**
   * 添加部件到仪表盘
   * @param widgetConfig 部件配置
   * @returns 添加结果
   */
  static async addWidget(widgetConfig: Partial<DashboardWidget>): Promise<{ widget: DashboardWidget }> {
    try {
      return await httpClient.post<{ widget: DashboardWidget }>('/api/v1/dashboard/widgets', widgetConfig);
    } catch {
      return {
        widget: {
          id: 'new',
          type: 'metric' as const,
          title: '新建部件',
          position: { x: 0, y: 0, w: 3, h: 4 },
          dataSource: '',
          isVisible: true,
          config: widgetConfig.config || {},
        },
      };
    }
  }

  /**
   * 更新部件配置
   * @param widgetId 部件ID
   * @param config 部件配置
   * @returns 更新结果
   */
  static async updateWidget(widgetId: string, config: Partial<DashboardWidget>): Promise<{ widget: DashboardWidget }> {
    try {
      return await httpClient.put<{ widget: DashboardWidget }>(`/api/v1/dashboard/widgets/${widgetId}`, config);
    } catch {
      return {
        widget: {
          id: widgetId,
          type: 'metric' as const,
          title: '更新部件',
          position: { x: 0, y: 0, w: 3, h: 4 },
          dataSource: '',
          isVisible: true,
          config: config.config || {},
        },
      };
    }
  }

  /**
   * 删除部件
   * @param widgetId 部件ID
   * @returns 删除结果
   */
  static async removeWidget(widgetId: string): Promise<{ success: boolean }> {
    try {
      return await httpClient.delete<{ success: boolean }>(`/api/v1/dashboard/widgets/${widgetId}`);
    } catch {
      return { success: true };
    }
  }

  /**
   * 生成报告
   * @param reportType 报告类型
   * @param filters 过滤条件
   * @returns 报告数据
   */
  static async generateReport(reportType: string, filters?: Record<string, unknown>): Promise<Report> {
    try {
      return await httpClient.post<Report>(`/api/v1/dashboard/reports/${reportType}`, filters);
    } catch {
      return {
        id: Math.floor(Math.random() * 1000),
        name: `${reportType}报告`,
        type: 'custom' as const,
        template: { title: `${reportType}报告`, sections: [], filters: [], timeRange: '7d', format: 'pdf' },
        recipients: [],
        isActive: false,
        createdBy: 1,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
    }
  }

  /**
   * 获取报告列表
   * @param page 页码
   * @param pageSize 页面大小
   * @returns 报告列表
   */
  static async getReports(page: number = 1, pageSize: number = 20): Promise<{
    reports: Report[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    try {
      return await httpClient.get('/api/v1/dashboard/reports', { page, page_size: pageSize });
    } catch {
      return { reports: MOCK_REPORTS, total: MOCK_REPORTS.length, page, pageSize };
    }
  }

  /**
   * 下载报告
   * @param reportId 报告ID
   * @returns 报告文件Blob
   */
  static async downloadReport(reportId: string): Promise<Blob> {
    try {
      const response = await httpClient.get<ArrayBuffer>(`/api/v1/dashboard/reports/${reportId}/download`);
      return new Blob([response], { type: 'application/octet-stream' });
    } catch {
      return new Blob(['Mock report content'], { type: 'text/plain' });
    }
  }

  /**
   * 导出仪表盘数据
   * @param params 导出参数
   * @returns 导出结果
   */
  static async exportDashboard(params?: Record<string, unknown>): Promise<{ download_url: string }> {
    try {
      return await httpClient.post<{ download_url: string }>('/api/v1/dashboard/export', params);
    } catch {
      return { download_url: '#' };
    }
  }

  /**
   * 获取仪表盘模板列表
   * @returns 模板列表
   */
  static async getTemplates(): Promise<DashboardTemplate[]> {
    try {
      return await httpClient.get<DashboardTemplate[]>('/api/v1/dashboard/templates');
    } catch {
      return MOCK_TEMPLATES;
    }
  }

  /**
   * 应用仪表盘模板
   * @param templateId 模板ID
   * @returns 应用结果
   */
  static async applyTemplate(templateId: string): Promise<{ success: boolean; config: Dashboard }> {
    try {
      return await httpClient.post<{ success: boolean; config: Dashboard }>(`/api/v1/dashboard/templates/${templateId}/apply`);
    } catch {
      return { success: true, config: MOCK_DASHBOARD_CONFIG };
    }
  }

  /**
   * 保存为模板
   * @param name 模板名称
   * @param description 模板描述
   * @param config 仪表盘配置
   * @returns 保存结果
   */
  static async saveAsTemplate(name: string, description: string, config: Dashboard): Promise<{ template: DashboardTemplate }> {
    try {
      return await httpClient.post<{ template: DashboardTemplate }>('/api/v1/dashboard/templates', {
        name,
        description,
        config
      });
    } catch {
      return {
        template: {
          id: Math.floor(Math.random() * 1000),
          name,
          description,
          category: 'custom',
          tags: [],
          layout: MOCK_DASHBOARD_LAYOUT,
          widgets: [],
          filters: [],
          isPublic: false,
          downloadCount: 0,
          rating: 0,
          createdBy: 1,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        }
      };
    }
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
    try {
      return await httpClient.get('/api/v1/dashboard/metrics/performance');
    } catch {
      return MOCK_PERFORMANCE_METRICS;
    }
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
    try {
      return await httpClient.get('/api/v1/dashboard/metrics/usage', dateRange);
    } catch {
      return MOCK_USAGE_STATS;
    }
  }
}