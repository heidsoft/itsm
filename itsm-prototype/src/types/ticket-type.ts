/**
 * 工单类型管理类型定义
 */

import { ApprovalNodeConfig } from './workflow';

/**
 * 工单类型状态
 */
export enum TicketTypeStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  DRAFT = 'draft',
}

/**
 * 自定义字段类型
 */
export enum CustomFieldType {
  TEXT = 'text',
  TEXTAREA = 'textarea',
  NUMBER = 'number',
  DATE = 'date',
  DATETIME = 'datetime',
  SELECT = 'select',
  MULTI_SELECT = 'multi_select',
  CHECKBOX = 'checkbox',
  RADIO = 'radio',
  FILE = 'file',
  USER_PICKER = 'user_picker',
  DEPARTMENT_PICKER = 'department_picker',
}

/**
 * 自定义字段定义
 */
export interface CustomFieldDefinition {
  id: string;
  name: string;
  label: string;
  type: CustomFieldType;
  required: boolean;
  description?: string;
  placeholder?: string;
  defaultValue?: any;
  options?: Array<{
    label: string;
    value: string | number;
  }>;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
  conditionalDisplay?: {
    field: string;
    operator: 'equals' | 'not_equals' | 'contains';
    value: any;
  };
  order: number;
}

/**
 * 工单类型定义
 */
export interface TicketTypeDefinition {
  id: number;
  code: string;
  name: string;
  description?: string;
  icon?: string;
  color?: string;
  status: TicketTypeStatus;
  
  // 表单配置
  customFields: CustomFieldDefinition[];
  
  // 审批流程
  approvalEnabled: boolean;
  approvalWorkflowId?: string;
  approvalChain?: ApprovalChainDefinition[];
  
  // SLA配置
  slaEnabled: boolean;
  defaultSlaId?: number;
  
  // 分配规则
  autoAssignEnabled: boolean;
  assignmentRules?: AssignmentRule[];
  
  // 通知配置
  notificationConfig?: NotificationConfig;
  
  // 权限配置
  permissionConfig?: PermissionConfig;
  
  // 元数据
  createdBy: number;
  createdByName: string;
  updatedBy?: number;
  updatedByName?: string;
  createdAt: string;
  updatedAt: string;
  tenantId: number;
  departmentId?: number; // 部门ID
  
  // 统计
  usageCount?: number;
}

/**
 * 审批链定义
 */
export interface ApprovalChainDefinition {
  id: string;
  level: number;
  name: string;
  approvers: Array<{
    type: 'user' | 'role' | 'department' | 'dynamic';
    value: number | string;
    name: string;
  }>;
  approvalType: 'any' | 'all' | 'majority';
  minimumApprovals?: number;
  allowReject: boolean;
  allowDelegate: boolean;
  rejectAction: 'end' | 'return' | 'custom';
  returnToLevel?: number;
  conditions?: Array<{
    field: string;
    operator: 'equals' | 'not_equals' | 'greater_than' | 'less_than';
    value: any;
  }>;
  timeout?: number; // 超时时间（小时）
  timeoutAction?: 'auto_approve' | 'auto_reject' | 'escalate';
}

/**
 * 分配规则
 */
export interface AssignmentRule {
  id: string;
  name: string;
  priority: number;
  conditions: Array<{
    field: string;
    operator: 'equals' | 'not_equals' | 'contains' | 'greater_than' | 'less_than';
    value: any;
  }>;
  assignTo: {
    type: 'user' | 'role' | 'department' | 'round_robin' | 'load_balance';
    value: number | string;
  };
  enabled: boolean;
}

/**
 * 通知配置
 */
export interface NotificationConfig {
  onCreate: {
    enabled: boolean;
    recipients: Array<'requester' | 'assignee' | 'approvers' | 'watchers'>;
    template?: string;
  };
  onUpdate: {
    enabled: boolean;
    recipients: Array<'requester' | 'assignee' | 'approvers' | 'watchers'>;
  };
  onApproval: {
    enabled: boolean;
    recipients: Array<'requester' | 'assignee' | 'previous_approvers' | 'next_approvers'>;
  };
  onReject: {
    enabled: boolean;
    recipients: Array<'requester' | 'assignee' | 'approvers'>;
  };
  onComplete: {
    enabled: boolean;
    recipients: Array<'requester' | 'assignee' | 'watchers'>;
  };
}

/**
 * 权限配置
 */
export interface PermissionConfig {
  canCreate: string[]; // 角色列表
  canView: string[];
  canEdit: string[];
  canDelete: string[];
  canApprove: string[];
}

/**
 * 创建工单类型请求
 */
export interface CreateTicketTypeRequest {
  code: string;
  name: string;
  description?: string;
  icon?: string;
  color?: string;
  customFields: CustomFieldDefinition[];
  approvalEnabled: boolean;
  approvalChain?: ApprovalChainDefinition[];
  slaEnabled: boolean;
  defaultSlaId?: number;
  autoAssignEnabled: boolean;
  assignmentRules?: AssignmentRule[];
  notificationConfig?: NotificationConfig;
  permissionConfig?: PermissionConfig;
  departmentId?: number;
}

/**
 * 工单类型列表响应
 */
export interface UpdateTicketTypeRequest {
  name?: string;
  description?: string;
  icon?: string;
  color?: string;
  status?: TicketTypeStatus;
  customFields?: CustomFieldDefinition[];
  approvalEnabled?: boolean;
  approvalChain?: ApprovalChainDefinition[];
  slaEnabled?: boolean;
  defaultSlaId?: number;
  autoAssignEnabled?: boolean;
  assignmentRules?: AssignmentRule[];
  notificationConfig?: NotificationConfig;
  permissionConfig?: PermissionConfig;
  departmentId?: number;
}

/**
 * 工单类型列表响应
 */
export interface TicketTypeListResponse {
  types: TicketTypeDefinition[];
  total: number;
  page: number;
  pageSize: number;
}

