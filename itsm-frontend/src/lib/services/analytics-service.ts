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
    totalTickets: number;
    compliantTickets: number;
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
  // 后端实际路由：GET /api/v1/analytics/tickets (tenant 顶层)
  // 原路由 /api/v1/tickets/analytics 在 router.go 中没有注册，
  // GET /api/v1/tickets/:id 会把 `analytics` 当成 ID 直接吃下，返回「无效的工单ID」。
  // 这里改成调用 AnalyticsController.GetTicketAnalytics，
  // 响应字段为 total / statusGroups / priorityGroups / trend30d / generatedAt。
  private readonly baseUrl = '/api/v1/analytics';

  // 真实后端响应（与 controller.AnalyticsController.GetTicketAnalytics 对齐）
  async getAnalytics(params?: {
    dateFrom?: string;
    dateTo?: string;
    groupBy?: string;
  }): Promise<TicketAnalyticsResponse> {
    const raw = await httpClient.get<{
      total?: number;
      statusGroups?: Array<{ status: string; count: number }>;
      priorityGroups?: Array<{ priority: string; count: number }>;
      trend30d?: Array<{ date: string; count: number }>;
      generatedAt?: string;
    }>(`${this.baseUrl}/tickets`, {
      // 后端 GetTicketStats 暂不消费 dateFrom/dateTo，但保留参数以便将来支持
      ...(params?.dateFrom ? { dateFrom: params.dateFrom } : {}),
      ...(params?.dateTo ? { dateTo: params.dateTo } : {}),
    });

    const priorityColorMap: Record<string, string> = {
      urgent: '#722ed1',
      critical: '#722ed1',
      p0: '#722ed1',
      high: '#ff4d4f',
      p1: '#ff4d4f',
      medium: '#faad14',
      p2: '#faad14',
      low: '#52c41a',
      p3: '#52c41a',
      p4: '#52c41a',
    };
    const statusColorMap: Record<string, string> = {
      new: '#1890ff',
      open: '#1890ff',
      in_progress: '#faad14',
      pending: '#faad14',
      resolved: '#52c41a',
      closed: '#52c41a',
      cancelled: '#bfbfbf',
    };

    const statusDistribution = (raw.statusGroups ?? []).map((s) => ({
      name: s.status,
      value: s.count,
      color: statusColorMap[s.status] ?? '#1890ff',
    }));

    const priorityDistribution = (raw.priorityGroups ?? []).map((p) => ({
      name: p.priority,
      value: p.count,
      color: priorityColorMap[p.priority] ?? '#1890ff',
    }));

    // 后端 trend30d 只按天给出 count（新建工单数量），把当日 count 同时作为 created，
    // resolved/open 留作 0 以保持图表可渲染；趋势序列按日期升序排列。
    const trend = (raw.trend30d ?? [])
      .slice()
      .sort((a, b) => (a.date < b.date ? -1 : a.date > b.date ? 1 : 0))
      .map((d) => ({
        date: d.date,
        created: d.count,
        resolved: 0,
        open: 0,
      }));

    return {
      totalTickets: raw.total ?? 0,
      openTickets: statusDistribution
        .filter((s) => !['resolved', 'closed', 'cancelled'].includes(s.name))
        .reduce((acc, s) => acc + s.value, 0),
      resolvedTickets: statusDistribution
        .filter((s) => s.name === 'resolved' || s.name === 'closed')
        .reduce((acc, s) => acc + s.value, 0),
      closedTickets: statusDistribution
        .filter((s) => s.name === 'closed')
        .reduce((acc, s) => acc + s.value, 0),
      overdueTickets: 0,
      dailyTrend: trend,
      statusDistribution,
      priorityDistribution,
      typeDistribution: [],
      processingTimeStats: {
        avgProcessingTime: 0,
        avgResolutionTime: 0,
        slaComplianceRate: 0,
      },
      teamPerformance: [],
      hotCategories: [],
    };
  }

  // 获取工单统计：与 router.go line 549 对齐 → GET /api/v1/tickets/stats
  async getStats(): Promise<{
    total: number;
    open: number;
    inProgress: number;
    pending: number;
    resolved: number;
    closed: number;
  }> {
    const response = await httpClient.get<{
      total?: number;
      open?: number;
      inProgress?: number;
      pending?: number;
      resolved?: number;
      closed?: number;
    }>('/api/v1/tickets/stats');

    return {
      total: response.total ?? 0,
      open: response.open ?? 0,
      inProgress: response.inProgress ?? 0,
      pending: response.pending ?? 0,
      resolved: response.resolved ?? 0,
      closed: response.closed ?? 0,
    };
  }

  // 导出分析报表：与 router.go line 766 对齐 → POST /api/v1/tickets/analytics/export
  async exportAnalytics(params: {
    dateFrom: string;
    dateTo: string;
    format: 'csv' | 'excel' | 'pdf';
    groupBy?: string;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/tickets/analytics/export',
      params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export const dashboardService = new DashboardService();
export const ticketAnalyticsService = new TicketAnalyticsService();
