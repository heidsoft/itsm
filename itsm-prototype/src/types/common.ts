// 通用类型定义，替换常用的any类型

// 表格行数据类型
export type TableRecord = Record<string, unknown>;

// 表单值类型
export type FormValues = Record<string, unknown>;

// 配置项类型
export type ConfigItem = {
  key: string;
  value: string | number | boolean;
  type: 'string' | 'number' | 'boolean' | 'select';
  label: string;
  description?: string;
  options?: Array<{ label: string; value: string | number }>;
  required?: boolean;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
};

// 用户信息类型
export type UserInfo = {
  id: number;
  username: string;
  email: string;
  fullName: string;
  avatar?: string;
  role: string;
  department?: string;
  status: 'active' | 'inactive' | 'suspended';
  lastLogin?: string;
  createdAt: string;
  updatedAt: string;
};

// 分页参数类型
export type PaginationParams = {
  page: number;
  pageSize: number;
  total?: number;
};

// 筛选参数类型
export type FilterParams = {
  keyword?: string;
  status?: string;
  priority?: string;
  category?: string;
  assignee?: string;
  dateRange?: [string, string];
  tags?: string[];
};

// 排序参数类型
export type SortParams = {
  field: string;
  order: 'ascend' | 'descend';
};

// 操作结果类型
export type OperationResult<T = unknown> = {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
  code?: number;
};

// 表格列配置类型
export type TableColumnConfig<T = TableRecord> = {
  title: string;
  dataIndex: string;
  key: string;
  width?: number | string;
  fixed?: 'left' | 'right';
  sorter?: boolean | ((a: T, b: T) => number);
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
  filters?: Array<{ text: string; value: string }>;
  onFilter?: (value: string, record: T) => boolean;
  ellipsis?: boolean;
  copyable?: boolean;
  editable?: boolean;
};

// 工作流节点类型
export type WorkflowNode = {
  id: string;
  name: string;
  type: 'start' | 'task' | 'gateway' | 'end';
  position: { x: number; y: number };
  size: { width: number; height: number };
  properties: Record<string, unknown>;
  connections: Array<{
    id: string;
    source: string;
    target: string;
    condition?: string;
  }>;
};

// 通知模板类型
export type NotificationTemplate = {
  id: number;
  name: string;
  description: string;
  type: 'info' | 'success' | 'warning' | 'error';
  channels: Array<'email' | 'sms' | 'in_app' | 'webhook'>;
  subject?: string;
  content: string;
  variables: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

// 报表参数类型
export type ReportParameter = {
  name: string;
  type: 'string' | 'number' | 'date' | 'select' | 'boolean';
  label: string;
  required: boolean;
  defaultValue?: unknown;
  options?: Array<{ label: string; value: unknown }>;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
};

// 通用状态类型
export type StatusType = 'active' | 'inactive' | 'pending' | 'completed' | 'failed' | 'cancelled';

// 优先级类型
export type PriorityType = 'low' | 'medium' | 'high' | 'critical';

// 影响范围类型
export type ImpactType = 'low' | 'medium' | 'high' | 'critical';

// 紧急程度类型
export type UrgencyType = 'low' | 'medium' | 'high' | 'critical';

// 时间范围类型
export type TimeRange = {
  start: string;
  end: string;
  label?: string;
};

// 统计数据类型
export type StatisticsData = {
  total: number;
  completed: number;
  pending: number;
  failed: number;
  successRate: number;
  averageTime: number;
  trend: 'up' | 'down' | 'stable';
};

// 文件上传类型
export type FileUpload = {
  id: string;
  name: string;
  size: number;
  type: string;
  url: string;
  status: 'uploading' | 'success' | 'error';
  progress?: number;
  error?: string;
};

// 评论类型
export type Comment = {
  id: number;
  content: string;
  author: UserInfo;
  createdAt: string;
  updatedAt?: string;
  attachments?: FileUpload[];
  isPrivate?: boolean;
};

// 活动日志类型
export type ActivityLog = {
  id: number;
  action: string;
  description: string;
  operator: UserInfo;
  target: string;
  targetId: string;
  changes?: Record<string, { old: unknown; new: unknown }>;
  timestamp: string;
  ipAddress?: string;
  userAgent?: string;
};
