/**
 * 仪表盘相关类型定义
 */

export type ChartType = 'line' | 'bar' | 'pie' | 'doughnut' | 'area' | 'scatter' | 'gauge';
export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '90d' | '1y' | 'custom';
export type MetricType = 'count' | 'percentage' | 'average' | 'sum' | 'min' | 'max';

export interface DashboardWidget {
  id: string;
  type: 'metric' | 'chart' | 'table' | 'list' | 'progress' | 'status';
  title: string;
  description?: string;
  position: {
    x: number;
    y: number;
    w: number;
    h: number;
  };
  config: WidgetConfig;
  dataSource: string;
  refreshInterval?: number; // seconds
  isVisible: boolean;
  permissions?: string[];
}

export interface WidgetConfig {
  // 通用配置
  showTitle?: boolean;
  showBorder?: boolean;
  backgroundColor?: string;
  
  // 图表配置
  chartType?: ChartType;
  xAxis?: string;
  yAxis?: string;
  groupBy?: string;
  aggregation?: MetricType;
  
  // 表格配置
  columns?: TableColumn[];
  pageSize?: number;
  showPagination?: boolean;
  
  // 指标配置
  metric?: string;
  unit?: string;
  format?: string;
  threshold?: {
    warning: number;
    critical: number;
  };
  
  // 样式配置
  colors?: string[];
  theme?: 'light' | 'dark';
}

export interface TableColumn {
  key: string;
  title: string;
  dataIndex: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  filterable?: boolean;
  render?: string; // 渲染函数名
}

export interface Dashboard {
  id: number;
  name: string;
  description?: string;
  isDefault: boolean;
  isPublic: boolean;
  layout: DashboardLayout;
  widgets: DashboardWidget[];
  filters: DashboardFilter[];
  permissions: string[];
  
  // 元数据
  createdBy: number;
  updatedBy: number;
  createdAt: string;
  updatedAt: string;
  
  // 共享设置
  shareSettings: {
    isShared: boolean;
    shareToken?: string;
    expiresAt?: string;
    allowedUsers?: number[];
    allowedRoles?: string[];
  };
}

export interface DashboardLayout {
  cols: number;
  rows: number;
  margin: [number, number];
  containerPadding: [number, number];
  rowHeight: number;
  isDraggable: boolean;
  isResizable: boolean;
}

export interface DashboardFilter {
  id: string;
  name: string;
  type: 'select' | 'multiselect' | 'date' | 'daterange' | 'text' | 'number';
  field: string;
  options?: FilterOption[];
  defaultValue?: unknown;
  isRequired: boolean;
  isVisible: boolean;
}

export interface FilterOption {
  label: string;
  value: unknown;
  color?: string;
}

// 仪表盘数据
export interface DashboardData {
  widgets: Record<string, WidgetData>;
  lastUpdated: string;
  timeRange: TimeRange;
  filters: Record<string, unknown>;
}

export interface WidgetData {
  data: unknown;
  loading: boolean;
  error?: string;
  lastUpdated: string;
}

// 统计数据
export interface TicketStats {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
  
  // 按优先级
  byPriority: Record<string, number>;
  
  // 按状态
  byStatus: Record<string, number>;
  
  // 按类型
  byType: Record<string, number>;
  
  // 按分配人
  byAssignee: Record<string, number>;
  
  // 按部门
  byDepartment: Record<string, number>;
  
  // 时间统计
  avgResolutionTime: number;
  avgResponseTime: number;
  slaCompliance: number;
  
  // 趋势数据
  trend: {
    period: string;
    created: number;
    resolved: number;
    backlog: number;
  }[];
}

export interface UserStats {
  total: number;
  active: number;
  online: number;
  
  // 按角色
  byRole: Record<string, number>;
  
  // 按部门
  byDepartment: Record<string, number>;
  
  // 活动统计
  loginToday: number;
  activeThisWeek: number;
  newThisMonth: number;
}

export interface SystemStats {
  uptime: number;
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  
  // 性能指标
  avgResponseTime: number;
  requestsPerSecond: number;
  errorRate: number;
  
  // 数据库统计
  dbConnections: number;
  dbSize: number;
  
  // 缓存统计
  cacheHitRate: number;
  cacheSize: number;
}

// 图表数据
export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
}

export interface ChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string | string[];
  borderColor?: string | string[];
  borderWidth?: number;
  fill?: boolean;
}

// 实时数据
export interface RealtimeData {
  type: 'ticket_created' | 'ticket_updated' | 'user_login' | 'system_alert';
  data: unknown;
  timestamp: string;
}

// 报告相关
export interface Report {
  id: number;
  name: string;
  description?: string;
  type: 'ticket' | 'user' | 'system' | 'custom';
  template: ReportTemplate;
  schedule?: ReportSchedule;
  recipients: string[];
  isActive: boolean;
  
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

export interface ReportTemplate {
  title: string;
  sections: ReportSection[];
  filters: DashboardFilter[];
  timeRange: TimeRange;
  format: 'pdf' | 'excel' | 'csv' | 'html';
}

export interface ReportSection {
  id: string;
  title: string;
  type: 'chart' | 'table' | 'metric' | 'text';
  config: WidgetConfig;
  dataSource: string;
}

export interface ReportSchedule {
  frequency: 'daily' | 'weekly' | 'monthly' | 'quarterly';
  time: string; // HH:mm
  dayOfWeek?: number; // 0-6, 0 = Sunday
  dayOfMonth?: number; // 1-31
  timezone: string;
  isActive: boolean;
}

// API 请求/响应
export interface CreateDashboardRequest {
  name: string;
  description?: string;
  isDefault?: boolean;
  isPublic?: boolean;
  layout: DashboardLayout;
  widgets: Omit<DashboardWidget, 'id'>[];
  filters: Omit<DashboardFilter, 'id'>[];
}

export interface UpdateDashboardRequest {
  name?: string;
  description?: string;
  isDefault?: boolean;
  isPublic?: boolean;
  layout?: DashboardLayout;
  widgets?: DashboardWidget[];
  filters?: DashboardFilter[];
}

export interface DashboardListResponse {
  dashboards: Dashboard[];
  total: number;
  page: number;
  pageSize: number;
}

export interface WidgetDataRequest {
  widgetId: string;
  timeRange: TimeRange;
  filters: Record<string, unknown>;
  startDate?: string;
  endDate?: string;
}

// 导出相关
export interface ExportDashboardRequest {
  dashboardId: number;
  format: 'pdf' | 'png' | 'jpg';
  timeRange: TimeRange;
  filters: Record<string, unknown>;
}

export interface ExportReportRequest {
  reportId: number;
  timeRange?: TimeRange;
  filters?: Record<string, unknown>;
  startDate?: string;
  endDate?: string;
}

// 模板相关
export interface DashboardTemplate {
  id: number;
  name: string;
  description?: string;
  category: string;
  tags: string[];
  thumbnail?: string;
  layout: DashboardLayout;
  widgets: Omit<DashboardWidget, 'id'>[];
  filters: Omit<DashboardFilter, 'id'>[];
  isPublic: boolean;
  downloadCount: number;
  rating: number;
  
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateDashboardFromTemplateRequest {
  templateId: number;
  name: string;
  description?: string;
  customizations?: {
    widgets?: Partial<DashboardWidget>[];
    filters?: Partial<DashboardFilter>[];
  };
}