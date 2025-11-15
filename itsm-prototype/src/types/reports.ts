/**
 * 报表系统类型定义
 * 支持报表设计、数据可视化、调度和导出
 */

// ==================== 报表基础类型 ====================

/**
 * 报表类型
 */
export enum ReportType {
  TABLE = 'table',               // 表格报表
  CHART = 'chart',               // 图表报表
  DASHBOARD = 'dashboard',       // 仪表盘
  PIVOT = 'pivot',               // 透视表
  MATRIX = 'matrix',             // 矩阵报表
}

/**
 * 报表状态
 */
export enum ReportStatus {
  DRAFT = 'draft',
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  ARCHIVED = 'archived',
}

/**
 * 报表定义
 */
export interface ReportDefinition {
  id: string;
  name: string;
  type: ReportType;
  status: ReportStatus;
  description?: string;
  
  // 分类
  category?: string;
  tags: string[];
  
  // 数据源
  dataSource: ReportDataSource;
  
  // 可视化配置
  visualization: ReportVisualization;
  
  // 过滤器
  filters: ReportFilter[];
  
  // 排序
  sorting?: ReportSorting[];
  
  // 分组
  grouping?: ReportGrouping[];
  
  // 权限
  visibility: 'public' | 'private' | 'shared';
  sharedWith?: number[];         // 分享给的用户ID
  
  // 调度
  schedule?: ReportSchedule;
  
  // 统计
  executionCount: number;
  lastExecutedAt?: Date;
  avgExecutionTime?: number;     // 平均执行时间（秒）
  
  // 元数据
  createdBy: number;
  createdByName: string;
  updatedBy?: number;
  updatedByName?: string;
  createdAt: Date;
  updatedAt: Date;
}

// ==================== 数据源 ====================

/**
 * 数据源类型
 */
export enum DataSourceType {
  QUERY = 'query',               // SQL查询
  API = 'api',                   // API接口
  PREDEFINED = 'predefined',     // 预定义数据集
  AGGREGATE = 'aggregate',       // 聚合数据
}

/**
 * 数据源配置
 */
export interface ReportDataSource {
  type: DataSourceType;
  
  // SQL查询
  query?: string;
  
  // API接口
  apiEndpoint?: string;
  apiMethod?: 'GET' | 'POST';
  apiParams?: Record<string, any>;
  
  // 预定义数据集
  datasetId?: string;
  
  // 聚合配置
  aggregation?: {
    collection: string;          // 数据集合
    metrics: AggregationMetric[];
    dimensions: string[];
    timeRange?: {
      start: string;
      end: string;
      granularity?: 'hour' | 'day' | 'week' | 'month';
    };
  };
  
  // 数据刷新
  refreshInterval?: number;      // 刷新间隔（分钟）
  cacheEnabled?: boolean;
  cacheDuration?: number;        // 缓存时长（分钟）
}

/**
 * 聚合指标
 */
export interface AggregationMetric {
  name: string;
  field: string;
  function: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'distinct';
  alias?: string;
}

// ==================== 可视化 ====================

/**
 * 图表类型
 */
export enum ChartType {
  LINE = 'line',
  BAR = 'bar',
  COLUMN = 'column',
  PIE = 'pie',
  DONUT = 'donut',
  AREA = 'area',
  SCATTER = 'scatter',
  GAUGE = 'gauge',
  FUNNEL = 'funnel',
  HEATMAP = 'heatmap',
  TREEMAP = 'treemap',
  RADAR = 'radar',
}

/**
 * 可视化配置
 */
export interface ReportVisualization {
  // 表格配置
  table?: {
    columns: TableColumn[];
    pagination?: {
      enabled: boolean;
      pageSize: number;
    };
    exportEnabled?: boolean;
  };
  
  // 图表配置
  chart?: {
    type: ChartType;
    xAxis: ChartAxis;
    yAxis: ChartAxis;
    series: ChartSeries[];
    legend?: ChartLegend;
    tooltip?: ChartTooltip;
    theme?: string;
  };
  
  // 仪表盘配置
  dashboard?: {
    layout: DashboardLayout;
    widgets: DashboardWidget[];
  };
  
  // 透视表配置
  pivot?: {
    rows: string[];
    columns: string[];
    values: PivotValue[];
    aggregations?: Record<string, string>;
  };
}

/**
 * 表格列配置
 */
export interface TableColumn {
  key: string;
  title: string;
  dataType: 'string' | 'number' | 'date' | 'boolean';
  width?: number;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  filterable?: boolean;
  formatter?: string;            // 格式化函数
  render?: string;               // 自定义渲染
}

/**
 * 图表坐标轴
 */
export interface ChartAxis {
  field: string;
  label?: string;
  type?: 'category' | 'value' | 'time';
  format?: string;
  min?: number;
  max?: number;
}

/**
 * 图表系列
 */
export interface ChartSeries {
  name: string;
  field: string;
  type?: ChartType;
  color?: string;
  stack?: string;                // 堆叠组
  smooth?: boolean;
}

/**
 * 图表图例
 */
export interface ChartLegend {
  show: boolean;
  position?: 'top' | 'right' | 'bottom' | 'left';
  orient?: 'horizontal' | 'vertical';
}

/**
 * 图表提示框
 */
export interface ChartTooltip {
  show: boolean;
  trigger?: 'item' | 'axis';
  formatter?: string;
}

/**
 * 仪表盘布局
 */
export interface DashboardLayout {
  type: 'grid' | 'free';
  columns?: number;
  rowHeight?: number;
  gap?: number;
}

/**
 * 仪表盘小部件
 */
export interface DashboardWidget {
  id: string;
  type: 'kpi' | 'chart' | 'table' | 'text';
  title?: string;
  position: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  config: any;
}

/**
 * 透视表值
 */
export interface PivotValue {
  field: string;
  aggregation: 'sum' | 'avg' | 'count' | 'min' | 'max';
  format?: string;
}

// ==================== 过滤器 ====================

/**
 * 过滤器类型
 */
export enum FilterType {
  TEXT = 'text',
  NUMBER = 'number',
  DATE = 'date',
  SELECT = 'select',
  MULTISELECT = 'multiselect',
  DATERANGE = 'daterange',
  BOOLEAN = 'boolean',
}

/**
 * 过滤器操作符
 */
export enum FilterOperator {
  EQUALS = 'equals',
  NOT_EQUALS = 'not_equals',
  CONTAINS = 'contains',
  NOT_CONTAINS = 'not_contains',
  STARTS_WITH = 'starts_with',
  ENDS_WITH = 'ends_with',
  GREATER_THAN = 'greater_than',
  LESS_THAN = 'less_than',
  BETWEEN = 'between',
  IN = 'in',
  NOT_IN = 'not_in',
  IS_NULL = 'is_null',
  IS_NOT_NULL = 'is_not_null',
}

/**
 * 报表过滤器
 */
export interface ReportFilter {
  id: string;
  field: string;
  label: string;
  type: FilterType;
  operator: FilterOperator;
  value?: any;
  defaultValue?: any;
  required?: boolean;
  options?: Array<{
    label: string;
    value: any;
  }>;
  visible?: boolean;             // 是否在UI中显示
}

/**
 * 排序配置
 */
export interface ReportSorting {
  field: string;
  order: 'asc' | 'desc';
}

/**
 * 分组配置
 */
export interface ReportGrouping {
  field: string;
  label?: string;
  aggregation?: 'sum' | 'avg' | 'count';
}

// ==================== 报表执行 ====================

/**
 * 报表执行结果
 */
export interface ReportExecutionResult {
  id: string;
  reportId: string;
  reportName: string;
  
  status: 'running' | 'completed' | 'failed';
  
  // 结果数据
  data?: {
    columns: string[];
    rows: any[][];
    total: number;
  };
  
  // 执行信息
  startTime: Date;
  endTime?: Date;
  duration?: number;             // 执行时长（秒）
  
  // 错误信息
  error?: string;
  
  // 执行参数
  filters?: Record<string, any>;
  
  executedBy: number;
  executedByName: string;
}

/**
 * 报表调度
 */
export interface ReportSchedule {
  enabled: boolean;
  frequency: 'once' | 'daily' | 'weekly' | 'monthly' | 'cron';
  
  // 定时配置
  time?: string;                 // HH:mm
  dayOfWeek?: number[];          // 0-6 (Sunday-Saturday)
  dayOfMonth?: number;           // 1-31
  cronExpression?: string;
  
  // 时区
  timezone?: string;
  
  // 开始/结束时间
  startDate?: Date;
  endDate?: Date;
  
  // 接收人
  recipients: ReportRecipient[];
  
  // 输出格式
  outputFormat: 'pdf' | 'excel' | 'csv' | 'html';
  
  // 最后执行
  lastRunTime?: Date;
  nextRunTime?: Date;
  lastRunStatus?: 'success' | 'failed';
}

/**
 * 报表接收人
 */
export interface ReportRecipient {
  type: 'user' | 'group' | 'email';
  id?: number;
  email?: string;
  name?: string;
}

// ==================== 报表模板 ====================

/**
 * 报表模板
 */
export interface ReportTemplate {
  id: string;
  name: string;
  category: string;
  description?: string;
  thumbnail?: string;
  
  // 模板配置
  config: Omit<ReportDefinition, 'id' | 'createdBy' | 'createdByName' | 'createdAt' | 'updatedAt'>;
  
  // 标签
  tags: string[];
  
  // 使用统计
  usageCount: number;
  
  isPublic: boolean;
  createdAt: Date;
}

// ==================== 报表分析 ====================

/**
 * 报表统计
 */
export interface ReportStats {
  totalReports: number;
  activeReports: number;
  scheduledReports: number;
  
  reportsByType: Record<ReportType, number>;
  reportsByCategory: Record<string, number>;
  
  topReports: Array<{
    report: ReportDefinition;
    executionCount: number;
    avgExecutionTime: number;
  }>;
  
  recentExecutions: ReportExecutionResult[];
  
  executionTrend: {
    date: string;
    count: number;
    avgDuration: number;
  }[];
}

/**
 * 报表性能分析
 */
export interface ReportPerformance {
  reportId: string;
  reportName: string;
  
  metrics: {
    totalExecutions: number;
    avgExecutionTime: number;
    minExecutionTime: number;
    maxExecutionTime: number;
    failureRate: number;         // 失败率（%）
  };
  
  executionHistory: {
    timestamp: Date;
    duration: number;
    status: 'success' | 'failed';
    rowCount?: number;
  }[];
  
  recommendations: string[];
}

// ==================== API请求/响应 ====================

/**
 * 创建报表请求
 */
export interface CreateReportRequest {
  name: string;
  type: ReportType;
  description?: string;
  category?: string;
  tags?: string[];
  dataSource: ReportDataSource;
  visualization: ReportVisualization;
  filters?: ReportFilter[];
  visibility?: 'public' | 'private' | 'shared';
  sharedWith?: number[];
}

/**
 * 更新报表请求
 */
export type UpdateReportRequest = Partial<CreateReportRequest>;

/**
 * 执行报表请求
 */
export interface ExecuteReportRequest {
  reportId: string;
  filters?: Record<string, any>;
  outputFormat?: 'json' | 'pdf' | 'excel' | 'csv';
}

/**
 * 报表查询
 */
export interface ReportQuery {
  type?: ReportType;
  category?: string;
  status?: ReportStatus;
  createdBy?: number;
  tags?: string[];
  search?: string;
  page?: number;
  pageSize?: number;
  sortBy?: 'name' | 'created' | 'updated' | 'executions';
  sortOrder?: 'asc' | 'desc';
}

/**
 * 数据集定义
 */
export interface DatasetDefinition {
  id: string;
  name: string;
  description?: string;
  source: 'tickets' | 'changes' | 'incidents' | 'problems' | 'assets' | 'custom';
  fields: DatasetField[];
  filters?: Record<string, any>;
  createdAt: Date;
}

/**
 * 数据集字段
 */
export interface DatasetField {
  name: string;
  label: string;
  dataType: 'string' | 'number' | 'date' | 'boolean';
  aggregatable: boolean;
  filterable: boolean;
  sortable: boolean;
}

export default ReportDefinition;

