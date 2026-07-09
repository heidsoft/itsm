/**
 * 工作流引擎服务 - 集成后端BPMN引擎
 */

import type { PaginatedResponse } from '@/types/api';
import { httpClient } from '@/lib/api/http-client';

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
  bpmnXml: string;
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

// 查询参数接口
export interface WorkflowDefinitionQueryParams {
  page?: number;
  pageSize?: number;
  key?: string;
  category?: string;
}

export interface WorkflowInstanceQueryParams {
  page?: number;
  pageSize?: number;
  processDefinitionKey?: string;
  status?: string;
}

export interface UserTaskQueryParams {
  page?: number;
  assignee?: string;
  status?: string;
}

// 工作流服务类
// 注意：后端 BPMN 路由为 /api/v1/bpmn/process-definitions, /api/v1/bpmn/process-instances, /api/v1/workflow/tasks
export class WorkflowEngineService {
  private readonly baseUrl = '/api/v1/bpmn';

  // 工作流定义管理
  async getWorkflowDefinitions(params?: WorkflowDefinitionQueryParams): Promise<PaginatedResponse<WorkflowDefinition>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.pageSize) query.append('pageSize', params.pageSize.toString());
    if (params?.key) query.append('key', params.key);
    if (params?.category) query.append('category', params.category);
    const endpoint = query.toString() ? `${this.baseUrl}/process-definitions?${query.toString()}` : `${this.baseUrl}/process-definitions`;
    return httpClient.get<PaginatedResponse<WorkflowDefinition>>(endpoint);
  }

  async getWorkflowDefinition(key: string, version?: string): Promise<WorkflowDefinition> {
    const query = version ? `?version=${version}` : '';
    return httpClient.get<WorkflowDefinition>(`${this.baseUrl}/process-definitions/${key}${query}`);
  }

  async createWorkflowDefinition(data: {
    key: string;
    name: string;
    description?: string;
    category?: string;
    bpmnXml: string;
    processVariables?: Record<string, any>;
  }): Promise<WorkflowDefinition> {
    return httpClient.post<WorkflowDefinition>(`${this.baseUrl}/process-definitions`, data);
  }

  async updateWorkflowDefinition(
    key: string,
    version: string,
    data: Partial<WorkflowDefinition>
  ): Promise<WorkflowDefinition> {
    return httpClient.put<WorkflowDefinition>(`${this.baseUrl}/process-definitions/${key}?version=${version}`, data);
  }

  async setWorkflowDefinitionActive(key: string, version: string, active: boolean): Promise<void> {
    return httpClient.put<void>(`${this.baseUrl}/process-definitions/${key}/active?version=${version}`, { active });
  }

  // 工作流实例管理
  async getWorkflowInstances(params?: WorkflowInstanceQueryParams): Promise<PaginatedResponse<WorkflowInstance>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.pageSize) query.append('pageSize', params.pageSize.toString());
    if (params?.processDefinitionKey) query.append('processDefinitionKey', params.processDefinitionKey);
    if (params?.status) query.append('status', params.status);
    const endpoint = query.toString() ? `${this.baseUrl}/process-instances?${query.toString()}` : `${this.baseUrl}/process-instances`;
    return httpClient.get<PaginatedResponse<WorkflowInstance>>(endpoint);
  }

  async startWorkflowInstance(
    processDefinitionKey: string,
    businessKey: string,
    variables?: Record<string, any>
  ): Promise<WorkflowInstance> {
    return httpClient.post<WorkflowInstance>(`${this.baseUrl}/process-instances`, {
      processDefinitionKey,
      businessKey,
      variables: variables || {},
    });
  }

  async getWorkflowInstance(id: string): Promise<WorkflowInstance> {
    return httpClient.get<WorkflowInstance>(`${this.baseUrl}/process-instances/${id}`);
  }

  // 任务管理 - 后端路由为 /api/v1/workflow/tasks
  async getUserTasks(params?: UserTaskQueryParams): Promise<PaginatedResponse<WorkflowTask>> {
    const query = new URLSearchParams();
    if (params?.page) query.append('page', params.page.toString());
    if (params?.assignee) query.append('assignee', params.assignee);
    if (params?.status) query.append('status', params.status);
    const endpoint = query.toString() ? `/api/v1/workflow/tasks?${query.toString()}` : '/api/v1/workflow/tasks';
    return httpClient.get<PaginatedResponse<WorkflowTask>>(endpoint);
  }

  async completeTask(taskId: string, variables?: Record<string, any>): Promise<void> {
    return httpClient.put<void>(`/api/v1/workflow/tasks/${taskId}/complete`, { variables });
  }

  async assignTask(taskId: string, assignee: string): Promise<void> {
    return httpClient.put<void>(`/api/v1/workflow/tasks/${taskId}/claim`, { assignee });
  }
}

export const workflowEngineService = new WorkflowEngineService();
export default WorkflowEngineService;
