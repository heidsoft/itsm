/**
 * 工作流系统类型定义
 * 支持可视化设计和执行的工作流引擎
 */

import type { TicketStatus, TicketPriority } from './ticket';

// ==================== 工作流基础类型 ====================

/**
 * 工作流状态
 */
export enum WorkflowStatus {
  DRAFT = 'draft',           // 草稿
  ACTIVE = 'active',         // 激活
  INACTIVE = 'inactive',     // 停用
  ARCHIVED = 'archived',     // 归档
}

/**
 * 工作流类型
 */
export enum WorkflowType {
  TICKET = 'ticket',         // 工单流程
  APPROVAL = 'approval',     // 审批流程
  ESCALATION = 'escalation', // 升级流程
  AUTOMATION = 'automation', // 自动化流程
}

/**
 * 工作流定义
 */
export interface WorkflowDefinition {
  id: string;
  name: string;
  code: string;
  type: WorkflowType;
  description?: string;
  version: number;
  status: WorkflowStatus;
  nodes: WorkflowNode[];
  connections: WorkflowConnection[];
  variables: WorkflowVariable[];
  triggers: WorkflowTrigger[];
  settings: WorkflowSettings;
  createdBy: number;
  createdByName: string;
  updatedBy?: number;
  updatedByName?: string;
  createdAt: Date;
  updatedAt: Date;
  departmentId?: number;
  bpmn_xml?: string;
}

// ==================== 工作流节点 ====================

/**
 * 节点类型
 */
export enum NodeType {
  START = 'start',               // 开始节点
  END = 'end',                   // 结束节点
  TASK = 'task',                 // 任务节点
  APPROVAL = 'approval',         // 审批节点
  CONDITION = 'condition',       // 条件节点
  PARALLEL = 'parallel',         // 并行网关
  EXCLUSIVE = 'exclusive',       // 互斥网关
  SCRIPT = 'script',             // 脚本节点
  NOTIFICATION = 'notification', // 通知节点
  TIMER = 'timer',               // 定时器节点
  SUBPROCESS = 'subprocess',     // 子流程节点
}

/**
 * 工作流节点
 */
export interface WorkflowNode {
  id: string;
  type: NodeType;
  name: string;
  description?: string;
  position: {
    x: number;
    y: number;
  };
  config: NodeConfig;
  style?: NodeStyle;
}

/**
 * 节点配置（基础）
 */
export interface BaseNodeConfig {
  timeout?: number;              // 超时时间（秒）
  retryOnFailure?: boolean;      // 失败重试
  maxRetries?: number;           // 最大重试次数
  onTimeout?: 'skip' | 'fail' | 'escalate';
}

/**
 * 任务节点配置
 */
export interface TaskNodeConfig extends BaseNodeConfig {
  assigneeType: 'user' | 'role' | 'group' | 'expression';
  assignees?: number[];          // 指定用户
  assigneeRole?: string;         // 角色
  assigneeGroup?: string;        // 组
  assigneeExpression?: string;   // 动态表达式
  formFields?: string[];         // 表单字段
  requiredFields?: string[];     // 必填字段
  autoComplete?: boolean;        // 自动完成
}

/**
 * 审批节点配置
 */
export interface ApprovalNodeConfig extends BaseNodeConfig {
  approvers: number[];
  approvalType: 'any' | 'all' | 'majority'; // 任一/全部/多数同意
  minimumApprovals?: number;     // 最少审批数
  allowReject: boolean;
  allowDelegate: boolean;
  rejectAction: 'end' | 'return' | 'custom';
  returnToNode?: string;         // 驳回到的节点
}

/**
 * 条件节点配置
 */
export interface ConditionNodeConfig extends BaseNodeConfig {
  conditions: {
    id: string;
    name: string;
    expression: string;
    targetNodeId: string;
  }[];
  defaultTargetNodeId?: string;  // 默认目标节点
}

/**
 * 脚本节点配置
 */
export interface ScriptNodeConfig extends BaseNodeConfig {
  language: 'javascript' | 'python';
  script: string;
  inputVariables?: string[];
  outputVariables?: string[];
}

/**
 * 通知节点配置
 */
export interface NotificationNodeConfig extends BaseNodeConfig {
  notificationType: 'email' | 'sms' | 'push' | 'webhook';
  recipients: {
    type: 'user' | 'role' | 'expression';
    value: string | number[];
  }[];
  template: string;
  subject?: string;
  priority?: 'low' | 'normal' | 'high';
}

/**
 * 定时器节点配置
 */
export interface TimerNodeConfig extends BaseNodeConfig {
  timerType: 'duration' | 'date' | 'cron';
  duration?: number;             // 持续时间（秒）
  date?: string;                 // 特定日期
  cronExpression?: string;       // Cron表达式
}

/**
 * 子流程节点配置
 */
export interface SubprocessNodeConfig extends BaseNodeConfig {
  workflowId: string;
  passVariables?: boolean;       // 传递变量
  variableMapping?: Record<string, string>;
  waitForCompletion?: boolean;   // 等待完成
}

/**
 * 节点配置联合类型
 */
export type NodeConfig =
  | TaskNodeConfig
  | ApprovalNodeConfig
  | ConditionNodeConfig
  | ScriptNodeConfig
  | NotificationNodeConfig
  | TimerNodeConfig
  | SubprocessNodeConfig
  | BaseNodeConfig;

/**
 * 节点样式
 */
export interface NodeStyle {
  width?: number;
  height?: number;
  backgroundColor?: string;
  borderColor?: string;
  borderWidth?: number;
  borderRadius?: number;
  fontSize?: number;
  fontColor?: string;
  icon?: string;
}

// ==================== 工作流连接 ====================

/**
 * 连接类型
 */
export enum ConnectionType {
  SEQUENCE = 'sequence',         // 顺序流
  CONDITIONAL = 'conditional',   // 条件流
  DEFAULT = 'default',           // 默认流
}

/**
 * 工作流连接
 */
export interface WorkflowConnection {
  id: string;
  type: ConnectionType;
  name?: string;
  sourceNodeId: string;
  targetNodeId: string;
  condition?: string;            // 条件表达式
  style?: ConnectionStyle;
}

/**
 * 连接样式
 */
export interface ConnectionStyle {
  color?: string;
  width?: number;
  lineType?: 'solid' | 'dashed' | 'dotted';
  animated?: boolean;
}

// ==================== 工作流变量 ====================

/**
 * 变量类型
 */
export enum VariableType {
  STRING = 'string',
  NUMBER = 'number',
  BOOLEAN = 'boolean',
  DATE = 'date',
  ARRAY = 'array',
  OBJECT = 'object',
}

/**
 * 工作流变量
 */
export interface WorkflowVariable {
  name: string;
  type: VariableType;
  defaultValue?: any;
  description?: string;
  required?: boolean;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    enum?: any[];
  };
}

// ==================== 工作流触发器 ====================

/**
 * 触发器类型
 */
export enum TriggerType {
  MANUAL = 'manual',             // 手动触发
  STATUS_CHANGE = 'status_change', // 状态变更
  FIELD_CHANGE = 'field_change', // 字段变更
  TIME_BASED = 'time_based',     // 基于时间
  WEBHOOK = 'webhook',           // Webhook
}

/**
 * 工作流触发器
 */
export interface WorkflowTrigger {
  id: string;
  type: TriggerType;
  enabled: boolean;
  config: TriggerConfig;
}

/**
 * 触发器配置
 */
export type TriggerConfig =
  | {
      type: 'status_change';
      fromStatus?: TicketStatus;
      toStatus: TicketStatus;
    }
  | {
      type: 'field_change';
      field: string;
      operator: 'equals' | 'not_equals' | 'contains';
      value: any;
    }
  | {
      type: 'time_based';
      schedule: string; // Cron表达式
    }
  | {
      type: 'webhook';
      url: string;
      secret?: string;
    };

// ==================== 工作流设置 ====================

/**
 * 工作流设置
 */
export interface WorkflowSettings {
  allowParallelInstances: boolean; // 允许并行实例
  maxParallelInstances?: number;
  enableVersionControl: boolean;
  enableAuditLog: boolean;
  notifyOnStart?: boolean;
  notifyOnComplete?: boolean;
  notifyOnError?: boolean;
  timeoutAction?: 'cancel' | 'continue';
  defaultTimeout?: number;         // 默认超时（秒）
}

// ==================== 工作流实例 ====================

/**
 * 工作流实例状态
 */
export enum WorkflowInstanceStatus {
  RUNNING = 'running',
  COMPLETED = 'completed',
  FAILED = 'failed',
  CANCELLED = 'cancelled',
  SUSPENDED = 'suspended',
}

/**
 * 工作流实例
 */
export interface WorkflowInstance {
  id: string;
  workflowId: string;
  workflowName: string;
  version: number;
  status: WorkflowInstanceStatus;
  ticketId?: number;
  currentNodeId?: string;
  variables: Record<string, any>;
  startTime: Date;
  endTime?: Date;
  duration?: number;             // 执行时长（秒）
  error?: string;
  startedBy: number;
  startedByName: string;
}

/**
 * 节点实例
 */
export interface NodeInstance {
  id: string;
  instanceId: string;
  nodeId: string;
  nodeName: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  assignee?: number;
  assigneeName?: string;
  startTime?: Date;
  endTime?: Date;
  duration?: number;
  input?: Record<string, any>;
  output?: Record<string, any>;
  error?: string;
  retryCount: number;
}

// ==================== 工作流模板 ====================

/**
 * 工作流模板
 */
export interface WorkflowTemplate {
  id: string;
  name: string;
  category: string;
  description?: string;
  thumbnail?: string;
  definition: Omit<WorkflowDefinition, 'id' | 'createdBy' | 'createdByName' | 'createdAt' | 'updatedAt'>;
  isPublic: boolean;
  usageCount: number;
  rating?: number;
  tags: string[];
  createdAt: Date;
}

// ==================== 工作流验证 ====================

/**
 * 验证错误
 */
export interface ValidationError {
  nodeId?: string;
  connectionId?: string;
  type: 'error' | 'warning';
  message: string;
  field?: string;
}

/**
 * 验证结果
 */
export interface ValidationResult {
  isValid: boolean;
  errors: ValidationError[];
  warnings: ValidationError[];
}

// ==================== 工作流统计 ====================

/**
 * 工作流统计
 */
export interface WorkflowStats {
  workflowId: string;
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  failedInstances: number;
  avgDuration: number;           // 平均时长（秒）
  successRate: number;           // 成功率（%）
  bottlenecks: Array<{
    nodeId: string;
    nodeName: string;
    avgDuration: number;
    count: number;
  }>;
}

/**
 * 节点执行统计
 */
export interface NodeExecutionStats {
  nodeId: string;
  nodeName: string;
  executionCount: number;
  successCount: number;
  failureCount: number;
  avgDuration: number;
  minDuration: number;
  maxDuration: number;
}

// ==================== API 请求/响应 ====================

/**
 * 创建工作流请求
 */
export interface CreateWorkflowRequest {
  name: string;
  code: string;
  type: WorkflowType;
  description?: string;
  nodes?: WorkflowNode[];
  connections?: WorkflowConnection[];
  variables?: WorkflowVariable[];
  settings?: Partial<WorkflowSettings>;
  departmentId?: number;
  bpmn_xml?: string;
}

/**
 * 更新工作流请求
 */
export interface UpdateWorkflowRequest {
  name?: string;
  description?: string;
  nodes?: WorkflowNode[];
  connections?: WorkflowConnection[];
  variables?: WorkflowVariable[];
  triggers?: WorkflowTrigger[];
  settings?: Partial<WorkflowSettings>;
  departmentId?: number;
}

/**
 * 启动工作流请求
 */
export interface StartWorkflowRequest {
  workflowId: string;
  ticketId?: number;
  variables?: Record<string, any>;
}

/**
 * 完成节点请求
 */
export interface CompleteNodeRequest {
  instanceId: string;
  nodeId: string;
  output?: Record<string, any>;
  comment?: string;
}

/**
 * 工作流查询
 */
export interface WorkflowQuery {
  type?: WorkflowType;
  status?: WorkflowStatus;
  search?: string;
  page?: number;
  pageSize?: number;
}

/**
 * 工作流导出格式
 */
export interface WorkflowExport {
  version: string;
  workflow: WorkflowDefinition;
  exportedAt: Date;
  exportedBy: string;
}

export default WorkflowDefinition;

