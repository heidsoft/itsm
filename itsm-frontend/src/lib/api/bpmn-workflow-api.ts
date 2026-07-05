/**
 * BPMN 流程设计器 API（完整版）
 *
 * 对应后端 controller/bpmn_workflow_controller.go
 * 路由前缀：/api/v1/bpmn
 *
 * 包含：
 * - 流程定义管理（CRUD、导入导出、克隆、激活）
 * - 流程实例管理（启动、暂停、恢复、终止）
 * - 任务管理（签收、完成、转派、取消）
 * - 会签管理（创建会签、投票、状态）
 * - 统计（实例统计、任务统计）
 * - 版本管理（创建、激活、回滚、比较）
 * - 变更日志
 */

import { httpClient } from './http-client';

// ============================================================
// 类型定义
// ============================================================

// 流程定义
export interface ProcessDefinition {
  id: number;
  key: string;
  name: string;
  description?: string;
  category?: string;
  version: number;
  status: 'draft' | 'active' | 'suspended' | 'archived';
  xml: string;
  deploymentId?: number;
  deploymentTime?: string;
  tenantId?: number;
  createdBy?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface ProcessDefinitionListResponse {
  items: ProcessDefinition[];
  total: number;
  page: number;
  pageSize: number;
}

// 流程定义请求
export interface CreateProcessDefinitionRequest {
  key: string;
  name: string;
  description?: string;
  category?: string;
  xml: string;
}

export interface UpdateProcessDefinitionRequest {
  name?: string;
  description?: string;
  category?: string;
  xml?: string;
}

export interface CloneProcessDefinitionRequest {
  newKey: string;
  newName: string;
  description?: string;
}

// 流程实例
export interface ProcessInstance {
  id: string;
  processDefinitionKey: string;
  processDefinitionVersion?: number;
  businessKey?: string;
  status: 'created' | 'running' | 'completed' | 'terminated' | 'suspended';
  startTime?: string;
  endTime?: string;
  duration?: number;
  startUserId?: number;
  tenantId?: number;
  variables?: Record<string, unknown>;
}

export interface ProcessInstanceListResponse {
  items: ProcessInstance[];
  total: number;
  page: number;
  pageSize: number;
}

export interface StartProcessRequest {
  processDefinitionKey: string;
  businessKey?: string;
  variables?: Record<string, unknown>;
}

// 任务
export interface UserTask {
  id: string;
  name: string;
  taskDefinitionKey: string;
  processInstanceId: string;
  processDefinitionKey?: string;
  assignee?: number;
  assigneeName?: string;
  candidateUsers?: number[];
  candidateGroups?: string[];
  dueDate?: string;
  priority?: number;
  status?: string;
  createdTime?: string;
  claimedTime?: string;
  completedTime?: string;
}

export interface UserTaskListResponse {
  items: UserTask[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ClaimTaskRequest {
  userId: number;
}

export interface CompleteTaskRequest {
  variables?: Record<string, unknown>;
  comment?: string;
}

export interface AssignTaskRequest {
  assignee: number;
}

// 会签
export interface CounterSignTask {
  id: string;
  mainTaskId: string;
  userId: number;
  userName?: string;
  status: 'pending' | 'approved' | 'rejected';
  comment?: string;
  createdTime?: string;
  completedTime?: string;
}

export interface CounterSignRequest {
  users: number[];
  type: 'sequential' | 'parallel';
  voteType: 'agree' | 'disagree' | 'abstain';
}

export interface CounterSignStatusResponse {
  mainTaskId: string;
  type: string;
  total: number;
  approved: number;
  rejected: number;
  abstained: number;
  completed: boolean;
  createdTime?: string;
  completedTime?: string;
}

export interface VoteRequest {
  vote: 'agree' | 'disagree' | 'abstain';
  comment?: string;
}

// 版本
export interface ProcessVersion {
  key: string;
  version: number;
  name: string;
  description?: string;
  xml: string;
  status: string;
  isActivated: boolean;
  createdBy?: number;
  createdAt?: string;
}

export interface ProcessVersionListResponse {
  items: ProcessVersion[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateVersionRequest {
  key: string;
  name?: string;
  description?: string;
  xml: string;
}

export interface VersionCompareResponse {
  key: string;
  version1: number;
  version2: number;
  diff: {
    added: string[];
    removed: string[];
    modified: string[];
  };
}

// 统计
export interface InstanceStats {
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  terminatedInstances: number;
  suspendedInstances: number;
  avgDurationMinutes?: number;
  completionRate?: number;
}

export interface TaskStats {
  totalTasks: number;
  pendingTasks: number;
  completedTasks: number;
  overdueTasks: number;
  avgCompletionTimeMinutes?: number;
  completionRate?: number;
}

// 变更日志
export interface VersionChangeLog {
  id: number;
  processDefinitionKey: string;
  version: number;
  changeType: string;
  changedFields: string[];
  changeDetails?: string;
  createdBy?: number;
  createdAt?: string;
}

export interface ChangeLogListResponse {
  items: VersionChangeLog[];
  total: number;
  page: number;
  pageSize: number;
}

// ============================================================
// API 类
// ============================================================

export class BPMNWorkflowApi {
  private static readonly baseUrl = '/api/v1/bpmn';

  // ==================== 流程定义管理 ====================

  /**
   * 创建流程定义
   */
  static async createProcessDefinition(
    data: CreateProcessDefinitionRequest
  ): Promise<ProcessDefinition> {
    const res = await httpClient.post<{ data?: ProcessDefinition } & ProcessDefinition>(
      `${this.baseUrl}/process-definitions`,
      data
    );
    return (res as { data?: ProcessDefinition }).data ?? (res as ProcessDefinition);
  }

  /**
   * 获取流程定义列表
   */
  static async listProcessDefinitions(params?: {
    page?: number;
    pageSize?: number;
    category?: string;
    status?: string;
    keyword?: string;
  }): Promise<ProcessDefinitionListResponse> {
    const query: Record<string, string> = {};
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.pageSize = String(params.pageSize);
    if (params?.category) query.category = params.category;
    if (params?.status) query.status = params.status;
    if (params?.keyword) query.keyword = params.keyword;

    const res = await httpClient.get<
      { data?: ProcessDefinitionListResponse } & ProcessDefinitionListResponse
    >(`${this.baseUrl}/process-definitions`, query);
    return (
      (res as { data?: ProcessDefinitionListResponse }).data ??
      (res as ProcessDefinitionListResponse)
    );
  }

  /**
   * 获取单个流程定义
   */
  static async getProcessDefinition(key: string): Promise<ProcessDefinition> {
    const res = await httpClient.get<{ data?: ProcessDefinition } & ProcessDefinition>(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}`
    );
    return (res as { data?: ProcessDefinition }).data ?? (res as ProcessDefinition);
  }

  /**
   * 更新流程定义
   */
  static async updateProcessDefinition(
    key: string,
    data: UpdateProcessDefinitionRequest
  ): Promise<ProcessDefinition> {
    const res = await httpClient.put<{ data?: ProcessDefinition } & ProcessDefinition>(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}`,
      data
    );
    return (res as { data?: ProcessDefinition }).data ?? (res as ProcessDefinition);
  }

  /**
   * 删除流程定义
   */
  static async deleteProcessDefinition(key: string): Promise<void> {
    await httpClient.delete(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}`
    );
  }

  /**
   * 导出流程定义 XML
   */
  static async exportProcessDefinition(key: string): Promise<string> {
    const res = await httpClient.get<string>(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}/export`
    );
    return typeof res === 'string' ? res : (res as unknown as string);
  }

  /**
   * 克隆流程定义
   */
  static async cloneProcessDefinition(
    key: string,
    data: CloneProcessDefinitionRequest
  ): Promise<ProcessDefinition> {
    const res = await httpClient.post<{ data?: ProcessDefinition } & ProcessDefinition>(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}/clone`,
      data
    );
    return (res as { data?: ProcessDefinition }).data ?? (res as ProcessDefinition);
  }

  /**
   * 设置流程定义激活状态
   */
  static async setProcessDefinitionActive(
    key: string,
    active: boolean
  ): Promise<ProcessDefinition> {
    const res = await httpClient.put<{ data?: ProcessDefinition } & ProcessDefinition>(
      `${this.baseUrl}/process-definitions/${encodeURIComponent(key)}/active`,
      { active }
    );
    return (res as { data?: ProcessDefinition }).data ?? (res as ProcessDefinition);
  }

  // ==================== 流程实例管理 ====================

  /**
   * 启动流程实例
   */
  static async startProcess(data: StartProcessRequest): Promise<ProcessInstance> {
    const res = await httpClient.post<{ data?: ProcessInstance } & ProcessInstance>(
      `${this.baseUrl}/process-instances`,
      data
    );
    return (res as { data?: ProcessInstance }).data ?? (res as ProcessInstance);
  }

  /**
   * 获取流程实例列表
   */
  static async listProcessInstances(params?: {
    page?: number;
    pageSize?: number;
    processDefinitionKey?: string;
    status?: string;
    startTimeFrom?: string;
    startTimeTo?: string;
  }): Promise<ProcessInstanceListResponse> {
    const query: Record<string, string> = {};
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.pageSize = String(params.pageSize);
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.status) query.status = params.status;
    if (params?.startTimeFrom) query.startTimeFrom = params.startTimeFrom;
    if (params?.startTimeTo) query.startTimeTo = params.startTimeTo;

    const res = await httpClient.get<
      { data?: ProcessInstanceListResponse } & ProcessInstanceListResponse
    >(`${this.baseUrl}/process-instances`, query);
    return (
      (res as { data?: ProcessInstanceListResponse }).data ??
      (res as ProcessInstanceListResponse)
    );
  }

  /**
   * 获取单个流程实例
   */
  static async getProcessInstance(id: string): Promise<ProcessInstance> {
    const res = await httpClient.get<{ data?: ProcessInstance } & ProcessInstance>(
      `${this.baseUrl}/process-instances/${encodeURIComponent(id)}`
    );
    return (res as { data?: ProcessInstance }).data ?? (res as ProcessInstance);
  }

  /**
   * 设置流程实例变量
   */
  static async setProcessInstanceVariables(
    id: string,
    variables: Record<string, unknown>
  ): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/process-instances/${encodeURIComponent(id)}/variables`,
      { variables }
    );
  }

  /**
   * 暂停流程实例
   */
  static async suspendProcess(id: string): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/process-instances/${encodeURIComponent(id)}/suspend`,
      {}
    );
  }

  /**
   * 恢复流程实例
   */
  static async resumeProcess(id: string): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/process-instances/${encodeURIComponent(id)}/resume`,
      {}
    );
  }

  /**
   * 终止流程实例
   */
  static async terminateProcess(id: string): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/process-instances/${encodeURIComponent(id)}/terminate`,
      {}
    );
  }

  // ==================== 任务管理 ====================

  /**
   * 获取用户任务列表
   */
  static async listUserTasks(params?: {
    page?: number;
    pageSize?: number;
    processInstanceId?: string;
    processDefinitionKey?: string;
    assignee?: number;
    status?: string;
  }): Promise<UserTaskListResponse> {
    const query: Record<string, string> = {};
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.pageSize = String(params.pageSize);
    if (params?.processInstanceId)
      query.processInstanceId = params.processInstanceId;
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.assignee) query.assignee = String(params.assignee);
    if (params?.status) query.status = params.status;

    const res = await httpClient.get<{ data?: UserTaskListResponse } & UserTaskListResponse>(
      `${this.baseUrl}/tasks`,
      query
    );
    return (
      (res as { data?: UserTaskListResponse }).data ?? (res as UserTaskListResponse)
    );
  }

  /**
   * 获取单个任务
   */
  static async getTask(id: string): Promise<UserTask> {
    const res = await httpClient.get<{ data?: UserTask } & UserTask>(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}`
    );
    return (res as { data?: UserTask }).data ?? (res as UserTask);
  }

  /**
   * 签收任务
   */
  static async claimTask(id: string, data: ClaimTaskRequest): Promise<void> {
    await httpClient.post(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/claim`,
      data
    );
  }

  /**
   * 指派任务
   */
  static async assignTask(id: string, data: AssignTaskRequest): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/assign`,
      data
    );
  }

  /**
   * 完成任务
   */
  static async completeTask(id: string, data?: CompleteTaskRequest): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/complete`,
      data ?? {}
    );
  }

  /**
   * 取消任务
   */
  static async cancelTask(id: string): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/cancel`,
      {}
    );
  }

  /**
   * 设置任务变量
   */
  static async setTaskVariables(
    id: string,
    variables: Record<string, unknown>
  ): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/variables`,
      { variables }
    );
  }

  // ==================== 会签管理 ====================

  /**
   * 创建会签任务
   */
  static async createCounterSignTask(
    id: string,
    data: CounterSignRequest
  ): Promise<CounterSignTask[]> {
    const res = await httpClient.post<{ data?: CounterSignTask[] } & CounterSignTask[]>(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/counter-sign`,
      data
    );
    return (res as { data?: CounterSignTask[] }).data ?? (res as CounterSignTask[]);
  }

  /**
   * 获取会签状态
   */
  static async getCounterSignStatus(id: string): Promise<CounterSignStatusResponse> {
    const res = await httpClient.get<
      { data?: CounterSignStatusResponse } & CounterSignStatusResponse
    >(`${this.baseUrl}/tasks/${encodeURIComponent(id)}/counter-sign-status`);
    return (
      (res as { data?: CounterSignStatusResponse }).data ??
      (res as CounterSignStatusResponse)
    );
  }

  /**
   * 投票
   */
  static async vote(id: string, data: VoteRequest): Promise<void> {
    await httpClient.put(
      `${this.baseUrl}/tasks/${encodeURIComponent(id)}/vote`,
      data
    );
  }

  // ==================== 统计 ====================

  /**
   * 获取流程实例统计
   */
  static async getInstanceStats(params?: {
    processDefinitionKey?: string;
    startTime?: string;
    endTime?: string;
  }): Promise<InstanceStats> {
    const query: Record<string, string> = {};
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.startTime) query.startTime = params.startTime;
    if (params?.endTime) query.endTime = params.endTime;

    const res = await httpClient.get<{ data?: InstanceStats } & InstanceStats>(
      `${this.baseUrl}/stats/instances`,
      query
    );
    return (res as { data?: InstanceStats }).data ?? (res as InstanceStats);
  }

  /**
   * 获取任务统计
   */
  static async getTaskStats(params?: {
    processDefinitionKey?: string;
    startTime?: string;
    endTime?: string;
  }): Promise<TaskStats> {
    const query: Record<string, string> = {};
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.startTime) query.startTime = params.startTime;
    if (params?.endTime) query.endTime = params.endTime;

    const res = await httpClient.get<{ data?: TaskStats } & TaskStats>(
      `${this.baseUrl}/stats/tasks`,
      query
    );
    return (res as { data?: TaskStats }).data ?? (res as TaskStats);
  }

  // ==================== 版本管理 ====================

  /**
   * 获取版本列表
   */
  static async listVersions(key: string, params?: {
    page?: number;
    pageSize?: number;
  }): Promise<ProcessVersionListResponse> {
    const query: Record<string, string> = {};
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.pageSize = String(params.pageSize);

    const res = await httpClient.get<
      { data?: ProcessVersionListResponse } & ProcessVersionListResponse
    >(`${this.baseUrl}/versions`, { ...query, key });
    return (
      (res as { data?: ProcessVersionListResponse }).data ??
      (res as ProcessVersionListResponse)
    );
  }

  /**
   * 获取单个版本
   */
  static async getVersion(key: string, version: number): Promise<ProcessVersion> {
    const res = await httpClient.get<{ data?: ProcessVersion } & ProcessVersion>(
      `${this.baseUrl}/versions/${encodeURIComponent(key)}/${String(version)}`
    );
    return (res as { data?: ProcessVersion }).data ?? (res as ProcessVersion);
  }

  /**
   * 创建新版本
   */
  static async createVersion(data: CreateVersionRequest): Promise<ProcessVersion> {
    const res = await httpClient.post<{ data?: ProcessVersion } & ProcessVersion>(
      `${this.baseUrl}/versions`,
      data
    );
    return (res as { data?: ProcessVersion }).data ?? (res as ProcessVersion);
  }

  /**
   * 激活版本
   */
  static async activateVersion(key: string, version: number): Promise<ProcessVersion> {
    const res = await httpClient.put<{ data?: ProcessVersion } & ProcessVersion>(
      `${this.baseUrl}/versions/${encodeURIComponent(key)}/${String(version)}/activate`,
      {}
    );
    return (res as { data?: ProcessVersion }).data ?? (res as ProcessVersion);
  }

  /**
   * 回滚版本
   */
  static async rollbackVersion(key: string, version: number): Promise<ProcessVersion> {
    const res = await httpClient.put<{ data?: ProcessVersion } & ProcessVersion>(
      `${this.baseUrl}/versions/${encodeURIComponent(key)}/${String(version)}/rollback`,
      {}
    );
    return (res as { data?: ProcessVersion }).data ?? (res as ProcessVersion);
  }

  /**
   * 比较两个版本
   */
  static async compareVersions(
    key: string,
    version1: number,
    version2: number
  ): Promise<VersionCompareResponse> {
    const res = await httpClient.get<
      { data?: VersionCompareResponse } & VersionCompareResponse
    >(
      `${this.baseUrl}/versions/${encodeURIComponent(key)}/compare`,
      { version1: String(version1), version2: String(version2) }
    );
    return (
      (res as { data?: VersionCompareResponse }).data ?? (res as VersionCompareResponse)
    );
  }

  // ==================== 变更日志 ====================

  /**
   * 获取变更日志列表
   */
  static async getVersionChangeLogs(
    key: string,
    params?: {
      page?: number;
      pageSize?: number;
    }
  ): Promise<ChangeLogListResponse> {
    const query: Record<string, string> = { key };
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.pageSize = String(params.pageSize);

    const res = await httpClient.get<
      { data?: ChangeLogListResponse } & ChangeLogListResponse
    >(`${this.baseUrl}/process-definitions/${encodeURIComponent(key)}/changelogs`, query);
    return (
      (res as { data?: ChangeLogListResponse }).data ??
      (res as ChangeLogListResponse)
    );
  }

  /**
   * 获取单个变更日志详情
   */
  static async getVersionChangeLogById(id: number): Promise<VersionChangeLog> {
    const res = await httpClient.get<{ data?: VersionChangeLog } & VersionChangeLog>(
      `${this.baseUrl}/process-definitions/changelogs/${String(id)}`
    );
    return (res as { data?: VersionChangeLog }).data ?? (res as VersionChangeLog);
  }
}

export default BPMNWorkflowApi;
