// 仪表盘相关类型定义

export interface KPIMetric {
  id: string;
  title: string;
  value: number | string;
  unit: string;
  color: string;
  icon?: React.ReactNode;
  trend?: 'up' | 'down' | 'stable';
  change?: number;
  changeType?: 'increase' | 'decrease' | 'stable';
  description?: string;
  target?: number; // 目标值
  alert?: 'success' | 'warning' | 'error' | null; // 警告状态
}

// 工单趋势数据（扩展版）
export interface TicketTrendData {
  date: string;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
  // 新增：趋势分析
  newTickets?: number;        // 新建工单数
  completedTickets?: number;  // 完成工单数
  pendingTickets?: number;    // 待处理工单数（累计）
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

// 新增：工单类型分布
export interface TicketTypeDistribution {
  type: string;           // 类型名称（事件、请求、问题、变更）
  count: number;          // 数量
  percentage: number;     // 占比
  color: string;          // 颜色
}

// 新增：响应时间分布
export interface ResponseTimeDistribution {
  timeRange: string;      // 时间段（0-1h, 1-4h, 4-8h, >8h）
  count: number;          // 工单数
  percentage: number;     // 占比
  avgTime?: number;       // 该时段平均时间（小时）
}

// 新增：团队工作负载
export interface TeamWorkload {
  assignee: string;           // 处理人
  ticketCount: number;        // 工单数
  avgResponseTime: number;    // 平均响应时间（小时）
  completionRate: number;     // 完成率（%）
  activeTickets?: number;     // 进行中工单
}

// 新增：优先级分布
export interface PriorityDistribution {
  priority: string;       // 优先级（紧急、高、中、低）
  count: number;          // 数量
  percentage: number;     // 占比
  color: string;          // 颜色
}

// 新增：高峰时段数据
export interface PeakHourData {
  hour: string;          // 小时（0-23）
  count: number;         // 该时段创建的工单数
  avgResponseTime?: number; // 该时段平均响应时间
}

// Dashboard总数据结构（扩展版 - 保持向后兼容）
export interface DashboardData {
  // 原有数据（保持兼容）
  kpiMetrics: KPIMetric[];
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData: SLAData[];
  satisfactionData: SatisfactionData[];
  quickActions: QuickAction[];
  recentActivities: RecentActivity[];
  
  // 新增：工单分析维度
  typeDistribution?: TicketTypeDistribution[];        // 工单类型分布
  responseTimeDistribution?: ResponseTimeDistribution[]; // 响应时间分布
  teamWorkload?: TeamWorkload[];                      // 团队工作负载
  priorityDistribution?: PriorityDistribution[];      // 优先级分布
  peakHours?: PeakHourData[];                        // 高峰时段
  
  // 元数据
  metadata?: {
    lastUpdated: string;
    dateRange: { start: string; end: string };
    totalTickets: number;
  };
}

