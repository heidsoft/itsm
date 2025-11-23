/**
 * 工作流引擎服务 - 集成后端BPMN引擎
 */

import { PaginatedResponse } from '@/types/api';

// 工作流状态枚举
export enum WorkflowStatus {
  DRAFT = 'draft',
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  SUSPENDED = 'suspended',
  ARCHIVED = 'archived',
}

// 工作流实例状态枚举
export enum WorkflowInstanceStatus {
  RUNNING = 'running',
  PAUSED = 'paused',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
  ERROR = 'error',
  TIMEOUT = 'timeout',
}

// 工作流定义
export interface WorkflowDefinition {
  id: number;
  key: string; // 对应后端 process_definition_key
  name: string;
  description?: string;
  version: string;
  category: string;
  bpmn_xml: string;
  isActive: boolean;
  isLatest: boolean;
  processVariables?: Record<string, any>;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// 工作流实例
export interface WorkflowInstance {
  id: number;
  processInstanceId: string;
  processDefinitionKey: string;
  processDefinitionId: string;
  businessKey: string;
  status: string;
  startTime: string;
  endTime?: string;
  variables: Record<string, any>;
  currentActivityId?: string;
  currentActivityName?: string;
}

// 工作流任务
export interface WorkflowTask {
  id: number;
  taskId: string;
  processInstanceId: string;
  processDefinitionKey: string;
  taskDefinitionKey: string;
  taskName: string;
  assignee?: string;
  candidateUsers?: string[];
  candidateGroups?: string[];
  status: string;
  createdTime: string;
  dueDate?: string;
  taskVariables?: Record<string, any>;
}

// 工作流服务类
export class WorkflowEngineService {
  private readonly baseUrl = '/api/v1/bpmn';

  // 工作流定义管理
  async getWorkflowDefinitions(params?: any): Promise<PaginatedResponse<WorkflowDefinition>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.pageSize) query.append('page_size', params.pageSize.toString());
    if (params?.key) query.append('key', params.key);
    if (params?.category) query.append('category', params.category);

    const response = await fetch(`${this.baseUrl}/process-definitions?${query.toString()}`);
    if (!response.ok) throw new Error('Failed to fetch workflow definitions');
    return response.json();
  }

  async getWorkflowDefinition(key: string, version?: string): Promise<WorkflowDefinition> {
    const query = version ? `?version=${version}` : '';
    const response = await fetch(`${this.baseUrl}/process-definitions/${key}${query}`);
    if (!response.ok) throw new Error('Failed to fetch workflow definition');
    const res = await response.json();
    return res.data;
  }

  async createWorkflowDefinition(data: {
    key: string;
    name: string;
    description?: string;
    category?: string;
    bpmn_xml: string;
    process_variables?: Record<string, any>;
  }): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/process-definitions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create workflow definition');
    const res = await response.json();
    return res.data;
  }

  async updateWorkflowDefinition(key: string, version: string, data: Partial<WorkflowDefinition>): Promise<WorkflowDefinition> {
    const response = await fetch(`${this.baseUrl}/process-definitions/${key}?version=${version}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to update workflow definition');
    const res = await response.json();
    return res.data;
  }

  async setWorkflowDefinitionActive(key: string, version: string, active: boolean): Promise<void> {
    const response = await fetch(`${this.baseUrl}/process-definitions/${key}/active?version=${version}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ active }),
    });
    if (!response.ok) throw new Error('Failed to set workflow definition status');
  }

  // 工作流实例管理
  async getWorkflowInstances(params?: any): Promise<PaginatedResponse<WorkflowInstance>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.pageSize) query.append('page_size', params.pageSize.toString());
    if (params?.processDefinitionKey) query.append('process_definition_key', params.processDefinitionKey);
    if (params?.status) query.append('status', params.status);

    const response = await fetch(`${this.baseUrl}/process-instances?${query.toString()}`);
    if (!response.ok) throw new Error('Failed to fetch workflow instances');
    return response.json();
  }

  async startWorkflowInstance(processDefinitionKey: string, businessKey: string, variables?: Record<string, any>): Promise<WorkflowInstance> {
    const response = await fetch(`${this.baseUrl}/process-instances`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        process_definition_key: processDefinitionKey,
        business_key: businessKey,
        variables: variables || {},
      }),
    });
    if (!response.ok) throw new Error('Failed to start workflow instance');
    const res = await response.json();
    return res.data;
  }

  async getWorkflowInstance(id: string): Promise<WorkflowInstance> {
    const response = await fetch(`${this.baseUrl}/process-instances/${id}`);
    if (!response.ok) throw new Error('Failed to fetch workflow instance');
    const res = await response.json();
    return res.data;
  }

  // 任务管理
  async getUserTasks(params?: any): Promise<PaginatedResponse<WorkflowTask>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.assignee) query.append('assignee', params.assignee);
    if (params?.status) query.append('status', params.status);

    const response = await fetch(`${this.baseUrl}/tasks?${query.toString()}`);
    if (!response.ok) throw new Error('Failed to fetch user tasks');
    return response.json();
  }

  async completeTask(taskId: string, variables?: Record<string, any>): Promise<void> {
    const response = await fetch(`${this.baseUrl}/tasks/${taskId}/complete`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ variables }),
    });
    if (!response.ok) throw new Error('Failed to complete task');
  }

  async assignTask(taskId: string, assignee: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/tasks/${taskId}/assign`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assignee }),
    });
    if (!response.ok) throw new Error('Failed to assign task');
  }
}

export const workflowEngineService = new WorkflowEngineService();
export default WorkflowEngineService;
