/**
 * 通用业务类型定义
 * 定义系统中通用的业务实体和枚举
 */

// 基础实体接口
export interface BaseEntity {
  id: number;
  createdAt: string;
  updatedAt: string;
  createdBy?: number;
  updatedBy?: number;
}

// 软删除实体
export interface SoftDeleteEntity extends BaseEntity {
  deletedAt?: string;
  deletedBy?: number;
}

// 租户相关
export interface Tenant extends BaseEntity {
  name: string;
  code: string;
  description?: string;
  isActive: boolean;
  settings?: Record<string, unknown>;
}

// 用户相关
export interface User extends BaseEntity {
  username: string;
  email: string;
  fullName: string;
  avatar?: string;
  phone?: string;
  department?: string;
  position?: string;
  isActive: boolean;
  lastLoginAt?: string;
  tenantId: number;
}

// 角色权限
export interface Role extends BaseEntity {
  name: string;
  code: string;
  description?: string;
  permissions: Permission[];
  isSystem: boolean;
}

export interface Permission extends BaseEntity {
  name: string;
  code: string;
  resource: string;
  action: string;
  description?: string;
}

// 状态枚举
export type EntityStatus = 'active' | 'inactive' | 'pending' | 'suspended';

// 优先级枚举
export type Priority = 'low' | 'medium' | 'high' | 'urgent' | 'critical';

// 操作类型
export type OperationType = 'create' | 'read' | 'update' | 'delete' | 'export' | 'import';

// 审核状态
export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'cancelled';

// 通知类型
export type NotificationType = 'info' | 'success' | 'warning' | 'error';

// 文件类型
export type FileType = 'image' | 'document' | 'video' | 'audio' | 'archive' | 'other';

// 时间范围
export interface DateRange {
  start: string;
  end: string;
}

// 选项类型
export interface Option<T = string> {
  label: string;
  value: T;
  disabled?: boolean;
  children?: Option<T>[];
}

// 表单字段类型
export interface FormField {
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'select' | 'multiselect' | 'date' | 'datetime' | 'number' | 'boolean' | 'file';
  required?: boolean;
  placeholder?: string;
  options?: Option[];
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
}

// 表格列配置
export interface TableColumn<T = unknown> {
  key: string;
  title: string;
  dataIndex?: keyof T;
  width?: number;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  filterable?: boolean;
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
}

// 筛选器配置
export interface FilterConfig {
  key: string;
  label: string;
  type: 'text' | 'select' | 'multiselect' | 'date' | 'daterange' | 'number';
  options?: Option[];
  placeholder?: string;
  defaultValue?: unknown;
}

// 排序配置
export interface SortConfig {
  field: string;
  order: 'asc' | 'desc';
}

// 搜索配置
export interface SearchConfig {
  fields: string[];
  placeholder?: string;
  debounceMs?: number;
}

// 操作按钮配置
export interface ActionButton {
  key: string;
  label: string;
  icon?: React.ReactNode;
  type?: 'primary' | 'default' | 'dashed' | 'link' | 'text';
  danger?: boolean;
  disabled?: boolean;
  onClick: (record: unknown) => void;
}

// 面包屑配置
export interface BreadcrumbItem {
  title: string;
  href?: string;
  icon?: React.ReactNode;
}

// 菜单项配置
export interface MenuItem {
  key: string;
  title: string;
  icon?: React.ReactNode;
  path?: string;
  children?: MenuItem[];
  permission?: string;
  badge?: number;
}

// 标签配置
export interface TagConfig {
  color: string;
  text: string;
  icon?: React.ReactNode;
}

// 统计卡片配置
export interface StatCardConfig {
  title: string;
  value: number | string;
  icon?: React.ReactNode;
  color?: string;
  trend?: {
    value: number;
    isPositive: boolean;
  };
  suffix?: string;
  prefix?: string;
}

// 图表配置
export interface ChartConfig {
  type: 'line' | 'bar' | 'pie' | 'area' | 'scatter';
  title: string;
  data: unknown[];
  options?: Record<string, unknown>;
  height?: number;
}

// 工作流状态
export type WorkflowStatus = 'draft' | 'active' | 'paused' | 'completed' | 'cancelled';

// 工作流节点类型
export type WorkflowNodeType = 'start' | 'end' | 'task' | 'gateway' | 'timer' | 'script';

// 工作流节点
export interface WorkflowNode {
  id: string;
  type: WorkflowNodeType;
  name: string;
  position: { x: number; y: number };
  config?: Record<string, unknown>;
}

// 工作流连接
export interface WorkflowConnection {
  id: string;
  source: string;
  target: string;
  condition?: string;
}

// 工作流定义
export interface WorkflowDefinition extends BaseEntity {
  name: string;
  description?: string;
  version: string;
  status: WorkflowStatus;
  nodes: WorkflowNode[];
  connections: WorkflowConnection[];
  variables?: Record<string, unknown>;
}