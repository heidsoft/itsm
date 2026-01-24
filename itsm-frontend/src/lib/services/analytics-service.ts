import { httpClient } from '@/lib/api/http-client';

// 工单分析响应
export interface TicketAnalyticsResponse {
  total_tickets: number;
  open_tickets: number;
  resolved_tickets: number;
  closed_tickets: number;
  overdue_tickets: number;

  // 趋势数据
  daily_trend: Array<{
    date: string;
    created: number;
    resolved: number;
    open: number;
  }>;

  // 状态分布
  status_distribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 优先级分布
  priority_distribution: Array<{
    name: string;
    value: number;
    color: string;
  }>;

  // 类型分布
  type_distribution: Array<{
    name: string;
    value: number;
  }>;

  // 处理时间统计
  processing_time_stats: {
    avg_processing_time: number;
    avg_resolution_time: number;
    sla_compliance_rate: number;
  };

  // 团队表现
  team_performance: Array<{
    assignee_name: string;
    total_handled: number;
    resolved: number;
    avg_response_time: number;
    avg_resolution_time: number;
  }>;

  // 热门类别
  hot_categories: Array<{
    category: string;
    count: number;
    trend: string;
  }>;
}

// 仪表盘概览响应
export interface DashboardOverviewResponse {
  overview: {
    total_tickets: number;
    pending_tickets: number;
    in_progress_tickets: number;
    resolved_today: number;
    avg_response_time: number;
    avg_resolution_time: number;
  };
  recent_activities: Array<{
    id: number;
    type: string;
    description: string;
    user: string;
    timestamp: string;
  }>;
  kpi_metrics: Array<{
    id: string;
    title: string;
    value: number;
    unit: string;
    color: string;
    trend: string;
    change_percent: number;
  }>;
  ticket_trend: Array<{
    date: string;
    created: number;
    resolved: number;
    pending: number;
  }>;
  sla_data: {
    compliance_rate: number;
    response_time_compliance: number;
    resolution_time_compliance: number;
    at_risk_tickets: number;
    breached_tickets: number;
  };
  satisfaction_data: {
    average_rating: number;
    total_ratings: number;
    rating_distribution: Array<{
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
  async getKPIMetrics(): Promise<DashboardOverviewResponse['kpi_metrics']> {
    return httpClient.get<DashboardOverviewResponse['kpi_metrics']>(`${this.baseUrl}/kpi-metrics`);
  }

  // 获取工单趋势
  async getTicketTrend(): Promise<DashboardOverviewResponse['ticket_trend']> {
    return httpClient.get<DashboardOverviewResponse['ticket_trend']>(`${this.baseUrl}/ticket-trend`);
  }

  // 获取事件分布
  async getIncidentDistribution(): Promise<any> {
    return httpClient.get(`${this.baseUrl}/incident-distribution`);
  }

  // 获取SLA数据
  async getSLAData(): Promise<DashboardOverviewResponse['sla_data']> {
    return httpClient.get<DashboardOverviewResponse['sla_data']>(`${this.baseUrl}/sla-data`);
  }

  // 获取满意度数据
  async getSatisfactionData(): Promise<DashboardOverviewResponse['satisfaction_data']> {
    return httpClient.get<DashboardOverviewResponse['satisfaction_data']>(`${this.baseUrl}/satisfaction-data`);
  }

  // 获取快捷操作
  async getQuickActions(): Promise<any[]> {
    return httpClient.get<any[]>(`${this.baseUrl}/quick-actions`);
  }

  // 获取最近活动
  async getRecentActivities(): Promise<DashboardOverviewResponse['recent_activities']> {
    return httpClient.get<DashboardOverviewResponse['recent_activities']>(`${this.baseUrl}/recent-activities`);
  }
}

// 工单分析服务
class TicketAnalyticsService {
  private readonly baseUrl = '/api/v1/tickets';

  // 获取工单分析数据
  async getAnalytics(params?: {
    date_from?: string;
    date_to?: string;
    group_by?: string;
  }): Promise<TicketAnalyticsResponse> {
    return httpClient.get<TicketAnalyticsResponse>(`${this.baseUrl}/analytics`, params);
  }

  // 获取工单统计
  async getStats(): Promise<{
    total: number;
    open: number;
    in_progress: number;
    pending: number;
    resolved: number;
    closed: number;
  }> {
    return httpClient.get(`${this.baseUrl}/stats`);
  }
}

export const dashboardService = new DashboardService();
export const ticketAnalyticsService = new TicketAnalyticsService();
