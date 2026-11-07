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
  // comparable=false（或缺失）表示后端没有可比的上期基线，此时 change 恒为 0，
  // 卡片不得渲染"相比上期"。缺失按不可比处理。
  comparable?: boolean;
  // scope 是工单类 KPI 的行级统计范围：tenant = 全租户，own = 本人创建或受理。
  scope?: 'tenant' | 'own';
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
  // 趋势分析
  newTickets?: number; // 新建工单数
  completedTickets?: number; // 完成工单数
  pendingTickets?: number; // 待处理工单数（累计）
}

export interface IncidentDistributionData {
  category: string;
  count: number;
  color: string;
}

/**
 * SLA 合规快照，字段与后端 service.SLAComplianceData 一一对应。
 *
 * 注意这是**一个对象**，不是按服务拆分的数组：/dashboard/overview 与
 * /dashboard/sla-data 返回的都是同一份聚合快照。历史类型写成 `SLAData[]`
 * （service/target/actual 三字段），图表再用 Array.isArray 过滤，于是对象被
 * 判为"没有数据"，SLA 面板在生产上长期显示"暂无数据"——契约写错比字段缺失
 * 更难发现，因为类型检查全绿。
 */
export interface SLAComplianceData {
  /** 整体合规率（%）：(总工单 - 有未解决违规的工单) / 总工单 */
  complianceRate: number;
  /** 响应时限达成率（%），按 violation_type=response_* 分桶 */
  responseTimeCompliance: number;
  /** 解决时限达成率（%），按 violation_type=resolution_* 分桶 */
  resolutionTimeCompliance: number;
  /** 24 小时内将到期的工单数 */
  atRiskTickets: number;
  /** 存在未解决违规的工单数 */
  breachedTickets: number;
  /** 统计窗口（近 30 天）内的工单总数 */
  totalTickets: number;
  /** 未违约工单数 */
  compliantTickets: number;
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

// 工单类型分布
export interface TicketTypeDistribution {
  type: string; // 类型名称（事件、请求、问题、变更）
  count: number; // 数量
  percentage: number; // 占比
  color: string; // 颜色
}

// 响应时间分布
export interface ResponseTimeDistribution {
  timeRange: string; // 时间段（0-1h, 1-4h, 4-8h, >8h）
  count: number; // 工单数
  percentage: number; // 占比
  avgTime?: number; // 该时段平均时间（小时）
}

// 团队工作负载
export interface TeamWorkload {
  assignee: string; // 处理人
  ticketCount: number; // 工单数
  avgResponseTime: number; // 平均响应时间（小时）
  completionRate: number; // 完成率（%）
  activeTickets?: number; // 进行中工单
}

// 优先级分布
export interface PriorityDistribution {
  priority: string; // 优先级（紧急、高、中、低）
  count: number; // 数量
  percentage: number; // 占比
  color: string; // 颜色
}

// 高峰时段数据
export interface PeakHourData {
  hour: string; // 小时（0-23）
  count: number; // 该时段创建的工单数
  avgResponseTime?: number; // 该时段平均响应时间
}

// Dashboard总数据结构（扩展版 - 保持向后兼容）
export interface DashboardData {
  // 原有数据（保持兼容）
  kpiMetrics: KPIMetric[];
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData?: SLAComplianceData | null;
  satisfactionData: SatisfactionData[];
  quickActions: QuickAction[];
  recentActivities: RecentActivity[];

  // 工单分析维度
  typeDistribution?: TicketTypeDistribution[]; // 工单类型分布
  responseTimeDistribution?: ResponseTimeDistribution[]; // 响应时间分布
  teamWorkload?: TeamWorkload[]; // 团队工作负载
  priorityDistribution?: PriorityDistribution[]; // 优先级分布
  peakHours?: PeakHourData[]; // 高峰时段

  // 元数据
  metadata?: {
    lastUpdated: string;
    dateRange: { start: string; end: string };
    totalTickets: number;
  };
}

// 管理者统一服务质量视图（GET /api/v1/reports/service-quality）。
// 逐字段对齐后端 service.ServiceQualityReport；ticketStats/problemStats/
// changeStats 是可选装配块——后端对应域未装配时块缺省（显式可见降级），
// 前端不得用空对象兜底伪装成「全零」。
export interface ServiceQualityWindow {
  days: number;
  from: string;
  to: string;
}

export interface AnalyticsTicketStatsBlock {
  total: number;
  statusGroups: Array<{ status: string; count: number }>;
  priorityGroups: Array<{ priority: string; count: number }>;
  trend30d: Array<{ date: string; count: number }>;
  generatedAt: string;
}

export interface ServiceQualityReport {
  metricVersion: string;
  metricVersions: Record<string, string>;
  generatedAt: string;
  window: ServiceQualityWindow;
  /** KPI 行级口径：tenant=全租户，own=本人 */
  scope: 'tenant' | 'own';
  kpi: KPIMetric[];
  sla: SLAComplianceData | null;
  incidentDistribution: IncidentDistributionData[];
  satisfaction: SatisfactionData[];
  ticketTrend: TicketTrendData[];
  ticketStats?: AnalyticsTicketStatsBlock;
  problemStats?: import('@/lib/api/problem-api').ProblemStatsResponse;
  changeStats?: import('@/lib/api/change-api').ChangeStatsResponse;
}
