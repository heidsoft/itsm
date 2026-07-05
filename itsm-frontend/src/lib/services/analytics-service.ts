import { httpClient } from '@/lib/api/http-client';

// 工单分析响应
export interface TicketAnalyticsResponse {
  totalTickets: number;
  openTickets: number;
  resolvedTickets: number;
  closedTickets: number;
  overdueTickets: number;

  // 趋势数据
  dailyTrend: Array<{
    date: string;
    created: number;
    resolved: number;
    open: number;
  }>;

  // 状态分布
  statusDistribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 优先级分布
  priorityDistribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 类型分布
  typeDistribution: Array<{
    name: string;
    value: number;
  }>;

  // 处理时间统计
  processingTimeStats: {
    avgProcessingTime: number;
    avgResolutionTime: number;
    slaComplianceRate: number;
  };

  // 团队表现
  teamPerformance: Array<{
    assigneeName: string;
    totalHandled: number;
    resolved: number;
    avgResponseTime: number;
    avgResolutionTime: number;
  }>;

  // 热门类别
  hotCategories: Array<{
    category: string;
    count: number;
    trend: string;
  }>;
}

// 仪表盘概览响应
export interface DashboardOverviewResponse {
  overview: {
    totalTickets: number;
    pendingTickets: number;
    inProgressTickets: number;
    resolvedToday: number;
    avgResponseTime: number;
    avgResolutionTime: number;
  };
  recentActivities: Array<{
    id: number;
    type: string;
    description: string;
    user: string;
    timestamp: string;
    ticketId?: number; // 动态字段，用于工单相关活动
  }>;
  kpiMetrics: Array<{
    id: string;
    title: string;
    value: number;
    unit: string;
    color: string;
    trend: string;
    changePercent: number;
  }>;
  ticketTrend: Array<{
    date: string;
    created: number;
    resolved: number;
    pending: number;
  }>;
  slaData: {
    complianceRate: number;
    responseTimeCompliance: number;
    resolutionTimeCompliance: number;
    atRiskTickets: number;
    breachedTickets: number;
  };
  satisfactionData: {
    averageRating: number;
    totalRatings: number;
    ratingDistribution: Array<{
      rating: number;
      count: number;
    }>;
  };
}

// 仪表盘服务
class DashboardService {
  private readonly baseUrl = '/api/v1/dashboard';

  // 获取仪表盘概览
  async getOverview(): Promise<DashboardOverviewResponse> {
    return httpClient.get<DashboardOverviewResponse>(`${this.baseUrl}/overview`);
  }

  // 获取KPI指标
  async getKPIMetrics(): Promise<DashboardOverviewResponse['kpiMetrics']> {
    return httpClient.get<DashboardOverviewResponse['kpiMetrics']>(`${this.baseUrl}/kpi-metrics`);
  }

  // 获取工单趋势
  async getTicketTrend(): Promise<DashboardOverviewResponse['ticketTrend']> {
    return httpClient.get<DashboardOverviewResponse['ticketTrend']>(
      `${this.baseUrl}/ticket-trend`
    );
  }

  // 获取事件分布
  async getIncidentDistribution(): Promise<any> {
    return httpClient.get(`${this.baseUrl}/incident-distribution`);
  }

  // 获取SLA数据
  async getSLAData(): Promise<DashboardOverviewResponse['slaData']> {
    return httpClient.get<DashboardOverviewResponse['slaData']>(`${this.baseUrl}/sla-data`);
  }

  // 获取满意度数据
  async getSatisfactionData(): Promise<DashboardOverviewResponse['satisfactionData']> {
    return httpClient.get<DashboardOverviewResponse['satisfactionData']>(
      `${this.baseUrl}/satisfaction-data`
    );
  }

  // 获取快捷操作
  async getQuickActions(): Promise<any[]> {
    return httpClient.get<any[]>(`${this.baseUrl}/quick-actions`);
  }

  // 获取最近活动
  async getRecentActivities(): Promise<DashboardOverviewResponse['recentActivities']> {
    return httpClient.get<DashboardOverviewResponse['recentActivities']>(
      `${this.baseUrl}/recent-activities`
    );
  }
}

// 工单分析服务
class TicketAnalyticsService {
  private readonly baseUrl = '/api/v1/tickets';

  // 获取工单分析数据
  async getAnalytics(params?: {
    dateFrom?: string;
    dateTo?: string;
    groupBy?: string;
  }): Promise<TicketAnalyticsResponse> {
    return httpClient.get<TicketAnalyticsResponse>(`${this.baseUrl}/analytics`, params);
  }

  // 获取工单统计
  async getStats(): Promise<{
    total: number;
    open: number;
    inProgress: number;
    pending: number;
    resolved: number;
    closed: number;
  }> {
    return httpClient.get(`${this.baseUrl}/stats`);
  }

  // 导出分析报表
  async exportAnalytics(params: {
    dateFrom: string;
    dateTo: string;
    format: 'csv' | 'excel' | 'pdf';
    groupBy?: string;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'GET',
      url: `${this.baseUrl}/analytics/export`,
      params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export const dashboardService = new DashboardService();
export const ticketAnalyticsService = new TicketAnalyticsService();
