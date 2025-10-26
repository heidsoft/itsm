'use client';

// KPI指标数据类型
export interface KPIMetric {
  id: string;
  title: string;
  value: number;
  unit?: string;
  change?: number;
  changeType?: 'increase' | 'decrease' | 'neutral';
  trend?: 'up' | 'down' | 'stable';
  icon?: React.ReactNode;
  color?: string;
  description?: string;
}

// 图表数据类型
export interface ChartData {
  labels: string[];
  datasets: {
    label: string;
    data: number[];
    backgroundColor?: string | string[];
    borderColor?: string | string[];
    borderWidth?: number;
  }[];
}

// 工单趋势数据类型
export interface TicketTrendData {
  date: string;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
}

// 事件分布数据类型
export interface IncidentDistributionData {
  category: string;
  count: number;
  percentage: number;
  color: string;
}

// SLA达成率数据类型
export interface SLAData {
  service: string;
  target: number;
  actual: number;
  status: 'met' | 'warning' | 'breach';
}

// 用户满意度数据类型
export interface SatisfactionData {
  month: string;
  rating: number;
  responses: number;
}

// 最近活动数据类型
export interface RecentActivity {
  id: string;
  type: 'ticket' | 'incident' | 'change' | 'problem';
  title: string;
  description: string;
  user: string;
  timestamp: string;
  status: string;
  priority?: 'low' | 'medium' | 'high' | 'urgent';
}

// 快速操作类型
export interface QuickAction {
  id: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  color: string;
  path: string;
  permission?: string;
}

// 仪表盘数据类型
export interface DashboardData {
  kpiMetrics: KPIMetric[];
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData: SLAData[];
  satisfactionData: SatisfactionData[];
  recentActivities: RecentActivity[];
  quickActions: QuickAction[];
}

// 仪表盘状态类型
export interface DashboardState {
  data: DashboardData | null;
  loading: boolean;
  error: string | null;
  lastUpdated: string | null;
  autoRefresh: boolean;
  refreshInterval: number;
}

// 仪表盘配置类型
export interface DashboardConfig {
  refreshInterval: number;
  autoRefresh: boolean;
  chartColors: {
    primary: string;
    secondary: string;
    success: string;
    warning: string;
    error: string;
  };
  permissions: {
    viewKPIs: boolean;
    viewCharts: boolean;
    viewActivities: boolean;
    viewQuickActions: boolean;
  };
}

// 图表配置类型
export interface ChartConfig {
  responsive: boolean;
  maintainAspectRatio: boolean;
  plugins: {
    legend: {
      position: 'top' | 'bottom' | 'left' | 'right';
    };
    tooltip: {
      enabled: boolean;
    };
  };
  scales?: {
    y?: {
      beginAtZero: boolean;
    };
    x?: {
      beginAtZero: boolean;
    };
  };
}

// WebSocket消息类型
export interface DashboardWebSocketMessage {
  type: 'dashboard_update' | 'kpi_update' | 'activity_update';
  data: Partial<DashboardData>;
  timestamp: string;
}

// 仪表盘Hook返回类型
export interface UseDashboardDataReturn {
  // 数据
  data: DashboardData | null;
  loading: boolean;
  error: string | null;
  lastUpdated: string | null;
  
  // 状态
  autoRefresh: boolean;
  refreshInterval: number;
  
  // 操作
  refresh: () => Promise<void>;
  setAutoRefresh: (enabled: boolean) => void;
  setRefreshInterval: (interval: number) => void;
  
  // 实时更新
  isConnected: boolean;
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error';
}
