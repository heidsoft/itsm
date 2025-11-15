// 仪表盘相关类型定义

export interface KPIMetric {
  id: string;
  title: string;
  value: number | string;
  unit?: string;
  color: string;
  icon?: React.ReactNode;
  trend?: 'up' | 'down' | 'stable';
  change?: number;
  changeType?: 'increase' | 'decrease' | 'stable';
  description?: string;
}

export interface TicketTrendData {
  date: string;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
}

export interface IncidentDistributionData {
  category: string;
  count: number;
  color: string;
}

export interface SLAData {
  service: string;
  target: number;
  actual: number;
}

export interface SatisfactionData {
  month: string;
  rating: number;
  responses: number;
}

export interface QuickAction {
  id: string;
  title: string;
  description: string;
  path: string;
  icon?: React.ReactNode;
  color: string;
  permission?: string;
}

export interface RecentActivity {
  id: string;
  type: 'ticket' | 'incident' | 'change' | 'problem';
  title: string;
  description: string;
  user: string;
  timestamp: string;
  priority?: 'urgent' | 'high' | 'medium' | 'low';
  status: string;
}

export interface DashboardData {
  kpiMetrics: KPIMetric[];
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData: SLAData[];
  satisfactionData: SatisfactionData[];
  quickActions: QuickAction[];
  recentActivities: RecentActivity[];
}

