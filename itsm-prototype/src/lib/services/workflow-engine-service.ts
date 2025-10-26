/**
 * 工作流引擎 - 符合ITIL 4.0标准
 * 
 * 核心功能：
 * 1. 可视化流程设计
 * 2. 动态流程执行
 * 3. 条件分支和并行处理
 * 4. 流程监控和优化
 * 5. 人工任务管理
 * 6. 系统任务集成
 * 7. 流程版本控制
 * 8. 流程性能分析
 */

import { ApiResponse, PaginatedResponse } from '@/types/api';

// 工作流状态枚举
export enum WorkflowStatus {
  DRAFT = 'draft',           // 草稿
  ACTIVE = 'active',         // 激活
  INACTIVE = 'inactive',     // 停用
  SUSPENDED = 'suspended',   // 暂停
  ARCHIVED = 'archived',     // 归档
}

// 工作流实例状态枚举
export enum WorkflowInstanceStatus {
  RUNNING = 'running',       // 运行中
  PAUSED = 'paused',         // 暂停
  COMPLETED = 'completed',   // 已完成
  CANCELLED = 'cancelled',   // 已取消
  ERROR = 'error',          // 错误
  TIMEOUT = 'timeout',      // 超时
}

// 节点类型枚举
export enum NodeType {
  START = 'start',           // 开始节点
  END = 'end',              // 结束节点
  TASK = 'task',            // 任务节点
  APPROVAL = 'approval',    // 审批节点
  CONDITION = 'condition',  // 条件节点
  PARALLEL = 'parallel',    // 并行节点
  MERGE = 'merge',          // 合并节点
  TIMER = 'timer',          // 定时器节点
  SCRIPT = 'script',        // 脚本节点
  SUBPROCESS = 'subprocess', // 子流程节点
}

// 任务状态枚举
export enum TaskStatus {
  PENDING = 'pending',       // 待处理
  IN_PROGRESS = 'in_progress', // 处理中
  COMPLETED = 'completed',   // 已完成
  REJECTED = 'rejected',     // 已拒绝
  SKIPPED = 'skipped',      // 已跳过
  TIMEOUT = 'timeout',      // 超时
}

// 工作流定义
export interface WorkflowDefinition {
  id: number;
  name: string;
  description?: string;
  version: number;
  category: string;
  status: WorkflowStatus;
  nodes: WorkflowNode[];
  connections: WorkflowConnection[];
  variables: WorkflowVariable[];
  settings: WorkflowSettings;
  isDefault: boolean;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

// 工作流节点
export interface WorkflowNode {
  id: string;
  name: string;
  type: NodeType;
  description?: string;
  position: { x: number; y: number };
  properties: NodeProperties;
  conditions?: NodeCondition[];
  actions?: NodeAction[];
  timeout?: number; // 超时时间（分钟）
  retryCount?: number; // 重试次数
  isRequired: boolean;
  canSkip: boolean;
}

// 节点属性
export interface NodeProperties {
  assignee?: string; // 指派人
  assigneeType?: 'user' | 'role' | 'group' | 'expression';
  formId?: number; // 表单ID
  script?: string; // 脚本内容
  timerExpression?: string; // 定时器表达式
  subprocessId?: number; // 子流程ID
  notification?: NotificationConfig;
  escalation?: EscalationConfig;
  [key: string]: any;
}

// 通知配置
export interface NotificationConfig {
  enabled: boolean;
  channels: ('email' | 'sms' | 'push' | 'webhook')[];
  template?: string;
  recipients?: string[];
}

// 升级配置
export interface EscalationConfig {
  enabled: boolean;
  timeout: number; // 升级超时时间（分钟）
  actions: EscalationAction[];
}

// 升级动作
export interface EscalationAction {
  type: 'reassign' | 'notify' | 'escalate' | 'auto_approve';
  target: string;
  parameters?: Record<string, any>;
}

// 节点条件
export interface NodeCondition {
  id: string;
  field: string;
  operator: 'equals' | 'not_equals' | 'contains' | 'greater_than' | 'less_than' | 'in' | 'not_in' | 'exists' | 'not_exists';
  value: any;
  logic: 'and' | 'or';
}

// 节点动作
export interface NodeAction {
  id: string;
  type: 'update_field' | 'send_notification' | 'call_api' | 'execute_script' | 'create_ticket' | 'update_ticket';
  parameters: Record<string, any>;
  condition?: NodeCondition;
}

// 工作流连接
export interface WorkflowConnection {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  condition?: NodeCondition;
  label?: string;
}

// 工作流变量
export interface WorkflowVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'object' | 'array';
  defaultValue?: any;
  required: boolean;
  description?: string;
}

// 工作流设置
export interface WorkflowSettings {
  timeout?: number; // 全局超时时间（分钟）
  retryPolicy?: RetryPolicy;
  errorHandling?: ErrorHandlingConfig;
  logging?: LoggingConfig;
  performance?: PerformanceConfig;
}

// 重试策略
export interface RetryPolicy {
  enabled: boolean;
  maxRetries: number;
  retryDelay: number; // 重试延迟（分钟）
  backoffMultiplier: number; // 退避乘数
}

// 错误处理配置
export interface ErrorHandlingConfig {
  strategy: 'continue' | 'stop' | 'retry' | 'escalate';
  notificationEnabled: boolean;
  notificationRecipients: string[];
}

// 日志配置
export interface LoggingConfig {
  enabled: boolean;
  level: 'debug' | 'info' | 'warn' | 'error';
  retentionDays: number;
}

// 性能配置
export interface PerformanceConfig {
  maxConcurrentInstances: number;
  maxExecutionTime: number; // 最大执行时间（分钟）
  resourceLimits: Record<string, number>;
}

// 工作流实例
export interface WorkflowInstance {
  id: number;
  workflowDefinitionId: number;
  workflowDefinition?: WorkflowDefinition;
  ticketId?: number;
  ticketNumber?: string;
  status: WorkflowInstanceStatus;
  currentNodes: string[]; // 当前活动节点ID列表
  variables: Record<string, any>;
  startTime: string;
  endTime?: string;
  duration?: number; // 持续时间（分钟）
  errorMessage?: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

// 工作流任务
export interface WorkflowTask {
  id: number;
  workflowInstanceId: number;
  nodeId: string;
  nodeName: string;
  assignee: number;
  assigneeName?: string;
  status: TaskStatus;
  formData?: Record<string, any>;
  comments?: string;
  attachments?: string[];
  startTime: string;
  endTime?: string;
  duration?: number; // 持续时间（分钟）
  timeoutAt?: string;
  createdAt: string;
  updatedAt: string;
}

// 工作流历史
export interface WorkflowHistory {
  id: number;
  workflowInstanceId: number;
  nodeId: string;
  action: string;
  actor: number;
  actorName?: string;
  details: Record<string, any>;
  timestamp: string;
}

// 工作流统计
export interface WorkflowStats {
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  failedInstances: number;
  averageExecutionTime: number;
  successRate: number;
  nodePerformance: Array<{
    nodeId: string;
    nodeName: string;
    averageExecutionTime: number;
    successRate: number;
    failureRate: number;
  }>;
}

// 工作流服务类
export class WorkflowEngineService {
  private readonly baseUrl = '/api/v1/workflow';

  // 工作流定义管理
  async getWorkflowDefinitions(params?: any): Promise<PaginatedResponse<WorkflowDefinition>> {
    const response = await fetch(`${this.baseUrl}/definitions?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch workflow definitions');
    return response.json();
  }

  async getWorkflowDefinition(id: number): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`);
    if (!response.ok) throw new Error('Failed to fetch workflow definition');
    return response.json();
  }

  async createWorkflowDefinition(data: Omit<WorkflowDefinition, 'id' | 'createdAt' | 'updatedAt'>): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/definitions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create workflow definition');
    return response.json();
  }

  async updateWorkflowDefinition(id: number, data: Partial<WorkflowDefinition>): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to update workflow definition');
    return response.json();
  }

  async deleteWorkflowDefinition(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to delete workflow definition');
  }

  // 工作流实例管理
  async getWorkflowInstances(params?: any): Promise<PaginatedResponse<WorkflowInstance>> {
    const response = await fetch(`${this.baseUrl}/instances?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch workflow instances');
    return response.json();
  }

  async getWorkflowInstance(id: number): Promise<WorkflowInstance> {
    const response = await fetch(`${this.baseUrl}/instances/${id}`);
    if (!response.ok) throw new Error('Failed to fetch workflow instance');
    return response.json();
  }

  async startWorkflowInstance(workflowDefinitionId: number, variables?: Record<string, any>): Promise<WorkflowInstance> {
    const response = await fetch(`${this.baseUrl}/instances`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workflowDefinitionId,
        variables: variables || {},
      }),
    });
    if (!response.ok) throw new Error('Failed to start workflow instance');
    return response.json();
  }

  async pauseWorkflowInstance(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${id}/pause`, {
      method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to pause workflow instance');
  }

  async resumeWorkflowInstance(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${id}/resume`, {
      method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to resume workflow instance');
  }

  async cancelWorkflowInstance(id: number, reason?: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${id}/cancel`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });
    if (!response.ok) throw new Error('Failed to cancel workflow instance');
  }

  // 工作流任务管理
  async getWorkflowTasks(params?: any): Promise<PaginatedResponse<WorkflowTask>> {
    const response = await fetch(`${this.baseUrl}/tasks?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch workflow tasks');
    return response.json();
  }

  async getWorkflowTask(id: number): Promise<WorkflowTask> {
    const response = await fetch(`${this.baseUrl}/tasks/${id}`);
    if (!response.ok) throw new Error('Failed to fetch workflow task');
    return response.json();
  }

  async completeWorkflowTask(id: number, formData: Record<string, any>, comments?: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/tasks/${id}/complete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ formData, comments }),
    });
    if (!response.ok) throw new Error('Failed to complete workflow task');
  }

  async rejectWorkflowTask(id: number, reason: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/tasks/${id}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });
    if (!response.ok) throw new Error('Failed to reject workflow task');
  }

  async reassignWorkflowTask(id: number, newAssignee: number, reason?: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/tasks/${id}/reassign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ newAssignee, reason }),
    });
    if (!response.ok) throw new Error('Failed to reassign workflow task');
  }

  // 工作流历史
  async getWorkflowHistory(instanceId: number): Promise<WorkflowHistory[]> {
    const response = await fetch(`${this.baseUrl}/instances/${instanceId}/history`);
    if (!response.ok) throw new Error('Failed to fetch workflow history');
    return response.json();
  }

  // 工作流统计
  async getWorkflowStats(params?: any): Promise<WorkflowStats> {
    const response = await fetch(`${this.baseUrl}/stats?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch workflow stats');
    return response.json();
  }

  // 工作流执行引擎
  async executeWorkflowNode(instanceId: number, nodeId: string, data?: Record<string, any>): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${instanceId}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nodeId, data }),
    });
    if (!response.ok) throw new Error('Failed to execute workflow node');
  }

  // 条件评估
  async evaluateConditions(conditions: NodeCondition[], variables: Record<string, any>): Promise<boolean> {
    const response = await fetch(`${this.baseUrl}/evaluate-conditions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ conditions, variables }),
    });
    if (!response.ok) throw new Error('Failed to evaluate conditions');
    const result = await response.json();
    return result.result;
  }

  // 脚本执行
  async executeScript(script: string, variables: Record<string, any>): Promise<any> {
    const response = await fetch(`${this.baseUrl}/execute-script`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ script, variables }),
    });
    if (!response.ok) throw new Error('Failed to execute script');
    return response.json();
  }

  // 工作流验证
  async validateWorkflow(workflow: WorkflowDefinition): Promise<{ valid: boolean; errors: string[] }> {
    const response = await fetch(`${this.baseUrl}/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(workflow),
    });
    if (!response.ok) throw new Error('Failed to validate workflow');
    return response.json();
  }

  // 工作流模板
  async getWorkflowTemplates(): Promise<WorkflowDefinition[]> {
    const response = await fetch(`${this.baseUrl}/templates`);
    if (!response.ok) throw new Error('Failed to fetch workflow templates');
    return response.json();
  }

  async createWorkflowFromTemplate(templateId: number, name: string, description?: string): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/templates/${templateId}/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, description }),
    });
    if (!response.ok) throw new Error('Failed to create workflow from template');
    return response.json();
  }

  // 工作流性能分析
  async getWorkflowPerformance(workflowDefinitionId: number, period?: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/performance/${workflowDefinitionId}?${new URLSearchParams({ period })}`);
    if (!response.ok) throw new Error('Failed to fetch workflow performance');
    return response.json();
  }

  // 工作流优化建议
  async getOptimizationSuggestions(workflowDefinitionId: number): Promise<any[]> {
    const response = await fetch(`${this.baseUrl}/optimization/${workflowDefinitionId}`);
    if (!response.ok) throw new Error('Failed to fetch optimization suggestions');
    return response.json();
  }
}

// 导出单例实例
export const workflowEngineService = new WorkflowEngineService();
export default WorkflowEngineService;
