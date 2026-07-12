/**
 * 工作流 API 服务
 */

import { httpClient } from './http-client';
import type {
  WorkflowDefinition,
  WorkflowInstance,
  WorkflowTemplate,
  WorkflowStats,
  NodeInstance,
  ValidationResult,
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  StartWorkflowRequest,
  CompleteNodeRequest,
  WorkflowQuery,
  WorkflowExport,
  NodeExecutionStats,
  InstanceStats,
} from '@/types/workflow';
import { WorkflowStatus, WorkflowType, WorkflowInstanceStatus } from '@/types/workflow';

// Re-export commonly used workflow types for page imports
export type {
  WorkflowDefinition,
  WorkflowInstance,
  WorkflowTemplate,
  WorkflowStats,
  NodeInstance,
  ValidationResult,
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  StartWorkflowRequest,
  CompleteNodeRequest,
  WorkflowQuery,
  WorkflowExport,
  NodeExecutionStats,
  InstanceStats,
};

// Backward-compatible alias used by some pages
export type WorkflowTask = NodeInstance;

// 后端 BPMN 任务原始结构（兼容多种字段命名）
interface BpmnTaskRaw {
  id?: string | number;
  taskId?: string;
  ID?: string;
  processInstanceId?: string;
  instanceId?: string;
  taskDefinitionKey?: string;
  taskName?: string;
  taskType?: string;
  status?: string;
  assignee?: string;
  createdTime?: string;
  createdAt?: string;
  dueDate?: string;
  taskVariables?: Record<string, unknown>;
  variables?: Record<string, unknown>;
  retryCount?: number;
}

export class WorkflowApi {
  private static escapeCSV(value: unknown): string {
    const text =
      value instanceof Date
        ? value.toISOString()
        : value === undefined || value === null
          ? ''
          : String(value);
    return `"${text.replace(/"/g, '""')}"`;
  }

  // ==================== 工作流定义管理 ====================

  /**
   * 获取工作流列表
   */
  static async getWorkflows(query?: WorkflowQuery): Promise<{
    workflows: WorkflowDefinition[];
    total: number;
  }> {
    const params: Record<string, string | number> = {};
    if (query?.page) params.page = query.page;
    if (query?.pageSize) params.pageSize = query.pageSize;

    // 修正: 确保路径与后端一致，后端可能是 /api/v1/bpmn/process-definitions
    const res = await httpClient.get<
      | Array<{ id: number; key: string; name: string; description?: string; version: number; status: string; createdAt: string; updatedAt: string; }>
      | {
          data?: Array<{
            id: number;
            key: string;
            name: string;
            description?: string;
            version: number | string;
            status?: string;
            isActive?: boolean;
            category?: string;
            bpmnXml?: string;
            createdAt: string;
            updatedAt: string;
          }>;
          items?: Array<{
            id: number;
            key: string;
            name: string;
            description?: string;
            version: number | string;
            status?: string;
            isActive?: boolean;
            category?: string;
            bpmnXml?: string;
            createdAt: string;
            updatedAt: string;
          }>;
          list?: Array<{
            id: number;
            key: string;
            name: string;
            description?: string;
            version: number | string;
            status?: string;
            isActive?: boolean;
            category?: string;
            bpmnXml?: string;
            createdAt: string;
            updatedAt: string;
          }>;
          total?: number;
          pagination?: { total: number };
        }
    >('/api/v1/bpmn/process-definitions', params);

    const list = Array.isArray(res) ? res : (res.data || res.items || res.list || []);
    const total = Array.isArray(res)
      ? list.length
      : (res.total ?? res.pagination?.total ?? list.length);
    const workflows: WorkflowDefinition[] = list.map((item: {
      id: number;
      key: string;
      name: string;
      description?: string;
      version: number | string;
      status?: string;
      isActive?: boolean;
      category?: string;
      bpmnXml?: string;
      createdAt: string;
      updatedAt: string;
    }) => ({
      id: String(item.id || ''),
      code: item.key || '',
      name: item.name || '',
      type: WorkflowType.TICKET,
      version: Number(item.version) || 1,
      status: item.status
        ? (item.status as WorkflowStatus)
        : item.isActive
          ? WorkflowStatus.ACTIVE
          : WorkflowStatus.DRAFT,
      nodes: [],
      connections: [],
      variables: [],
      triggers: [],
      settings: {
        allowParallelInstances: true,
        enableVersionControl: true,
        enableAuditLog: true,
      },
      createdBy: 0,
      createdByName: '',
      createdAt: item.createdAt ? new Date(item.createdAt) : new Date(),
      updatedAt: item.updatedAt ? new Date(item.updatedAt) : new Date(),
      description: item.description,
      category: item.category,
      bpmnXml: item.bpmnXml,
    })) as WorkflowDefinition[];
    return { workflows, total };
  }

  /**
   * 获取单个工作流
   */
  static async getWorkflow(id: string): Promise<WorkflowDefinition> {
    // Assuming id is the key for BPMN controller which uses key
    // 修正: 确保路径与后端一致
    const res = await httpClient.get<{
      id: number;
      key: string;
      name: string;
      description?: string;
      category?: string;
      type?: string;
      version: number;
      status: string;
      isActive?: boolean;
      bpmnXml?: string;
      approvalConfig?: Record<string, unknown>;
      slaConfig?: Record<string, unknown>;
      createdAt: string;
      updatedAt: string;
    }>(`/api/v1/bpmn/process-definitions/${id}`);

    const item = res;
    return {
      id: String(item.id || ''),
      code: item.key || '',
      name: item.name || '',
      type: WorkflowType.TICKET,
      version: item.version || 1,
      status: item.status
        ? (item.status as WorkflowStatus)
        : item.isActive
          ? WorkflowStatus.ACTIVE
          : WorkflowStatus.DRAFT,
      nodes: [],
      connections: [],
      variables: [],
      triggers: [],
      settings: {
        allowParallelInstances: true,
        enableVersionControl: true,
        enableAuditLog: true,
      },
      createdBy: 0,
      createdByName: '',
      createdAt: item.createdAt ? new Date(item.createdAt) : new Date(),
      updatedAt: item.updatedAt ? new Date(item.updatedAt) : new Date(),
      description: item.description,
      bpmnXml: item.bpmnXml,
      category: item.category || item.type || 'general',
      approvalConfig: item.approvalConfig,
      slaConfig: item.slaConfig,
    };
  }

  // ==================== Backward-compatible method aliases (legacy pages) ====================

  static async getProcessDefinition(key: string): Promise<WorkflowDefinition> {
    return WorkflowApi.getWorkflow(key);
  }

  static async getProcessVersions(key: string): Promise<WorkflowDefinition[]> {
    return WorkflowApi.getWorkflowVersions(key);
  }

  static async createProcessDefinition(request: CreateWorkflowRequest): Promise<WorkflowDefinition> {
    return WorkflowApi.createWorkflow(request);
  }

  static async updateProcessDefinition(
    key: string,
    request: UpdateWorkflowRequest,
    version?: string
  ): Promise<WorkflowDefinition> {
    return WorkflowApi.updateWorkflow(key, request, version);
  }

  static async deployProcessDefinition(key: string, version?: string): Promise<void> {
    // 激活工作流即视为部署
    await WorkflowApi.activateWorkflow(key, version);
  }

  static async createProcessVersion(key: string): Promise<WorkflowDefinition> {
    // 创建新版本（通过复制当前定义）
    const current = await WorkflowApi.getWorkflow(key);
    return WorkflowApi.createWorkflow({
      code: current.code,
      name: current.name,
      description: current.description,
       
      type: current.type,
       
      bpmnXml: current.bpmnXml,
    });
  }

  static async deployWorkflow(workflowId: string): Promise<void> {
    // 激活工作流即视为部署
    try {
      await WorkflowApi.activateWorkflow(workflowId);
    } catch (error) {
      console.error('部署工作流失败:', error);
      throw error;
    }
  }

  static async listWorkflowInstances(
    params?: {
      workflowId?: string;
      ticketId?: number;
      status?: string;
      page?: number;
      pageSize?: number;
    }
  ): Promise<{ instances: WorkflowInstance[]; total: number }> {
    return WorkflowApi.getInstances(params);
  }

  static async listWorkflowTasks(instanceId: string): Promise<WorkflowTask[]> {
    // 调用后端 BPMN API 获取指定流程实例的任务列表
    try {
      const tenantId = httpClient.getTenantId() || 1;
      const res = await httpClient.get<{ items?: any[]; list?: any[]; data?: any[] }>('/api/v1/bpmn/tasks', {
        processInstanceId: instanceId,
        page: 1,
        pageSize: 100,
      });
      
      // 解析响应数据
      let tasks: any[] = [];
      if (Array.isArray(res)) {
        tasks = res;
      } else if (res && typeof res === 'object') {
        tasks = res.items || res.list || res.data || [];
      }
      
      // 转换为 WorkflowTask 格式
      return tasks.map((task: any) => ({
        id: String(task.id || task.taskId || task.ID || ''),
        instanceId: instanceId,
        nodeId: task.taskId || task.taskDefinitionKey || '',
        nodeName: task.taskName || task.taskName || '',
        nodeType: task.taskType || task.taskType || 'user_task',
        status: (task.status || 'pending') as NodeInstance['status'],
        assignee: task.assignee ? parseInt(task.assignee, 10) : undefined,
        assigneeId: task.assignee ? parseInt(task.assignee, 10) : undefined,
        createdAt: task.createdTime || task.createdAt || new Date().toISOString(),
        dueDate: task.dueDate || task.dueDate || null,
        variables: task.taskVariables || task.variables || {},
        retryCount: task.retryCount || 0,
      }));
    } catch (error) {
      console.error('Failed to fetch workflow tasks:', error);
      return [];
    }
  }

  static async suspendWorkflow(instanceId: string): Promise<void> {
    return WorkflowApi.suspendInstance(instanceId);
  }

  /**
   * 获取「我的待办」BPMN 任务列表（基于当前登录用户 ID 过滤）
   *
   * 后端控制器会根据 JWT 中的 user_id 自动过滤包含：
   * - assignee = 当前用户
   * - candidate_users 包含当前用户 ID / username / email
   * - candidate_groups 包含当前用户所在的组
   *
   * @param params 可覆盖分页/状态过滤。
   */
  static async listMyTasks(
    params?: { status?: string; page?: number; pageSize?: number }
  ): Promise<{ items: WorkflowTask[]; total: number; page: number; size: number }> {
    const query: Record<string, unknown> = {
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
    };
    if (params?.status) {
      query.status = params.status;
    }
    const res = await httpClient.get<BpmnTaskRaw[] | { items?: BpmnTaskRaw[]; list?: BpmnTaskRaw[]; data?: BpmnTaskRaw[]; total?: number; page?: number; size?: number }>('/api/v1/bpmn/tasks', query);
    // 兼容多种返回结构：后端 ListResponse / items / list / data
    const raw: BpmnTaskRaw[] = Array.isArray(res)
      ? res
      : (res?.items || res?.list || res?.data || []);
    const items: WorkflowTask[] = raw.map((task) => ({
      id: String(task.id || task.taskId || task.ID || ''),
      instanceId: task.processInstanceId || task.instanceId || '',
      nodeId: task.taskId || task.taskDefinitionKey || '',
      nodeName: task.taskName || task.taskName || '',
      nodeType: task.taskType || task.taskType || 'user_task',
      status: (task.status || 'pending') as NodeInstance['status'],
      assignee: task.assignee ? parseInt(task.assignee, 10) : undefined,
      assigneeId: task.assignee ? parseInt(task.assignee, 10) : undefined,
      createdAt: task.createdTime || task.createdAt || new Date().toISOString(),
      dueDate: task.dueDate || task.dueDate || null,
      variables: task.taskVariables || task.variables || {},
      retryCount: task.retryCount || 0,
    }));
    const meta = Array.isArray(res) ? null : res;
    const total = Number(meta?.total ?? items.length);
    const page = Number(meta?.page ?? params?.page ?? 1);
    const size = Number(meta?.size ?? params?.pageSize ?? items.length);
    return { items, total, page, size };
  }

  /**
   * 领取任务（claim）。
   */
  static async claimMyTask(taskId: string | number): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/tasks/${taskId}/claim`);
  }

  static async resumeWorkflow(instanceId: string): Promise<void> {
    return WorkflowApi.resumeInstance(instanceId);
  }

  static async terminateWorkflow(instanceId: string, reason?: string): Promise<void> {
    return WorkflowApi.cancelInstance(instanceId, reason);
  }

  /**
   * 创建工作流
   */
  static async createWorkflow(request: CreateWorkflowRequest): Promise<WorkflowDefinition> {
    // 从 httpClient 获取当前租户ID
    const tenantId = httpClient.getTenantId() || 1;

    const payload = {
      key: request.code || `process_${Date.now()}`,
      name: request.name,
      description: request.description,
      category: request.type, // Map type to category
      bpmnXml: request.bpmnXml,
      tenantId: tenantId,
    };
    return httpClient.post('/api/v1/bpmn/process-definitions', payload);
  }

  /**
   * 更新工作流
   */
  static async updateWorkflow(
    id: string,
    request: UpdateWorkflowRequest,
    version?: string
  ): Promise<WorkflowDefinition> {
    const payload = {
      name: request.name,
      description: request.description,
      bpmnXml: request.bpmnXml,
    };
    // Backend expects key in path. Assuming id passed here is key.
    // Also backend needs version parameter. We default to 1.0.0 or need to fetch it.
    const ver = version || '1.0.0';
	return httpClient.put<WorkflowDefinition>(
	  `/api/v1/bpmn/process-definitions/${encodeURIComponent(id)}?version=${encodeURIComponent(ver)}`,
	  payload
	);
  }

  /**
   * 删除工作流
   */
  static async deleteWorkflow(id: string): Promise<void> {
    // Backend requires version.
    await httpClient.delete(`/api/v1/bpmn/process-definitions/${id}?version=1.0.0`);
  }

  /**
   * 复制工作流
   */
  static async cloneWorkflow(id: string, name: string): Promise<WorkflowDefinition> {
    // 后端暂不支持，通过获取原工作流并创建新实例来实现
    const original = await WorkflowApi.getWorkflow(id);
    return WorkflowApi.createWorkflow({
      code: `${original.code}_copy_${Date.now()}`,
      name: name || `${original.name} (副本)`,
      description: original.description,
       
      type: original.type,
       
      bpmnXml: original.bpmnXml,
    });
  }

  /**
   * 激活工作流
   */
  static async activateWorkflow(id: string, version?: string): Promise<void> {
    const ver = version || '1.0.0';
    // httpClient.put 已经处理了错误情况（code !== 0 时会抛出异常）
    // 所以这里不需要再检查 response.code
    await httpClient.put<unknown>(`/api/v1/bpmn/process-definitions/${id}/active?version=${ver}`, { active: true });
  }

  /**
   * 停用工作流
   */
  static async deactivateWorkflow(id: string, version?: string): Promise<void> {
    const ver = version || '1.0.0';
    await httpClient.put(`/api/v1/bpmn/process-definitions/${id}/active?version=${ver}`, {
      active: false,
    });
  }

  /**
   * 验证工作流
   */
  static async validateWorkflow(workflow: Partial<WorkflowDefinition>): Promise<ValidationResult> {
    // Mock validation or call backend if supported
    return { isValid: true, errors: [], warnings: [] };
  }

  /**
   * 导出工作流
   */
  static async exportWorkflow(id: string): Promise<WorkflowExport> {
    // 后端暂不支持导出功能，返回空数据
    const workflow = await WorkflowApi.getWorkflow(id);
    return {
      version: '1.0',
      exportedAt: new Date(),
      exportedBy: 'system',
      workflow: {
        ...workflow,
        bpmnXml: workflow.bpmnXml || '',
      },
    };
  }

  /**
   * 导入工作流
   */
  static async importWorkflow(data: WorkflowExport): Promise<WorkflowDefinition> {
    // 后端暂不支持导入，使用创建接口
    if (!data.workflow) {
      throw new Error('无效的导入数据');
    }
    return WorkflowApi.createWorkflow({
      code: data.workflow.code || `imported_${Date.now()}`,
      name: data.workflow.name,
      description: data.workflow.description,
      type: data.workflow.type,
      bpmnXml: data.workflow.bpmnXml,
    });
  }

  // ==================== 工作流版本管理 ====================

  /**
   * 获取工作流版本列表
   */
  static async getWorkflowVersions(workflowId: string): Promise<WorkflowDefinition[]> {
    interface ProcessDefinitionResponse {
      id?: number;
      key?: string;
      name?: string;
      description?: string;
      bpmnXml?: string;
      version?: number;
      status?: string;
      createdAt?: string;
      updatedAt?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
    }

    // 使用新的版本 API
    const res = await httpClient.get<
      Array<{
        id: number;
        key: string;
        name: string;
        description?: string;
        bpmnXml?: string;
        version: number;
        status: string;
        createdAt: string;
        updatedAt: string;
      }>
    >(`/api/v1/bpmn/versions?process_key=${workflowId}`);

    const list = Array.isArray(res) ? res : [];
    return list.map(item => ({
      id: String(item.id || ''),
      code: item.key || '',
      name: item.name || '',
      type: WorkflowType.TICKET,
      version: item.version || 1,
      status: (item.status as WorkflowStatus) || WorkflowStatus.DRAFT,
      nodes: [],
      connections: [],
      variables: [],
      triggers: [],
      settings: {
        allowParallelInstances: true,
        enableVersionControl: true,
        enableAuditLog: true,
      },
      createdBy: 0,
      createdByName: '',
      createdAt: item.createdAt ? new Date(item.createdAt) : new Date(),
      updatedAt: item.updatedAt ? new Date(item.updatedAt) : new Date(),
      description: item.description,
      bpmnXml: item.bpmnXml,
    })) as WorkflowDefinition[];
  }

	static async createWorkflowVersion(data: {
	  processDefinitionKey: string;
	  name: string;
	  description?: string;
	  bpmnXml: string;
	  changeLog?: string;
	}): Promise<unknown> {
	  return httpClient.post('/api/v1/bpmn/versions', data);
	}

  /**
   * 发布新版本
   */
  static async publishVersion(workflowId: string): Promise<WorkflowDefinition> {
    // 后端暂不支持版本发布，获取最新版本返回
    const versions = await WorkflowApi.getWorkflowVersions(workflowId);
    if (versions.length > 0) {
      return versions[0];
    }
    throw new Error('未找到可发布的版本');
  }

  /**
   * 激活指定版本
   */
  static async activateVersion(processKey: string, version: number): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/versions/${processKey}/${version}/activate`, {});
  }

  /**
   * 回滚到指定版本
   */
  static async rollbackVersion(
    processKey: string,
    version: number,
    reason?: string
  ): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/versions/${processKey}/${version}/rollback`, {
      reason: reason || '回滚操作',
    });
  }

  /**
   * 删除指定版本
   */
  static async deleteVersion(processKey: string, version: number): Promise<void> {
    await httpClient.delete(`/api/v1/bpmn/process-definitions/${processKey}?version=${version}`);
  }

  /**
   * 比较两个版本
   */
  static async compareVersions(
    processKey: string,
    baseVersion: number,
    targetVersion: number
  ): Promise<{
    elementsAdded: string[];
    elementsRemoved: string[];
    elementsModified: string[];
    connectionsAdded: string[];
    connectionsRemoved: string[];
    variablesChanged: string[];
    isIdentical: boolean;
  }> {
    interface CompareResponse {
      addedNodes?: string[];
      removedNodes?: string[];
      modifiedNodes?: string[];
    }

    interface VersionComparisonResponse {
      elementsAdded: string[];
      elementsRemoved: string[];
      elementsModified: string[];
      connectionsAdded: string[];
      connectionsRemoved: string[];
      variablesChanged: string[];
      isIdentical: boolean;
    }

    const res = await httpClient.get<{
      addedNodes?: string[];
      removedNodes?: string[];
      modifiedNodes?: string[];
    }>(`/api/v1/bpmn/versions/${processKey}/compare`, {
      baseVersion: baseVersion,
      targetVersion: targetVersion,
    });
    const data = res;
    if (!data) {
      return {
        elementsAdded: [],
        elementsRemoved: [],
        elementsModified: [],
        connectionsAdded: [],
        connectionsRemoved: [],
        variablesChanged: [],
        isIdentical: true,
      };
    }
    // 适配后端返回格式
    return {
      elementsAdded: data.addedNodes || [],
      elementsRemoved: data.removedNodes || [],
      elementsModified: data.modifiedNodes || [],
      connectionsAdded: [],
      connectionsRemoved: [],
      variablesChanged: [],
      isIdentical: !(
        data.addedNodes?.length ||
        data.removedNodes?.length ||
        data.modifiedNodes?.length
      ),
    };
  }

  // ==================== 工作流实例管理 ====================

  /**
   * 启动工作流实例
   */
  static async startWorkflow(request: StartWorkflowRequest): Promise<WorkflowInstance> {
    const payload = {
      processDefinitionKey: request.workflowId, // Assuming workflowId is the Key
      businessKey: `BIZ-${Date.now()}`, // Auto generate or from request
      variables: request.variables,
    };
    const item = await httpClient.post<{
      id: string;
      processInstanceId: string;
      processDefinitionKey: string;
      businessKey: string;
      status: string;
      startTime: string;
      endTime?: string;
    }>('/api/v1/bpmn/process-instances', payload);
    return {
      id: item.processInstanceId || item.id || '',
      workflowId: item.processDefinitionKey || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item.startTime ? new Date(item.startTime) : new Date(),
      startedBy: 0,
      startedByName: '',
    };
  }

  /**
   * 获取工作流实例列表
   */
  static async getInstances(params?: {
    workflowId?: string;
    ticketId?: number;
    status?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    instances: WorkflowInstance[];
    total: number;
  }> {
    interface ProcessInstanceResponse {
      id?: string;
      instanceId?: string;
      processDefinitionKey?: string;
      businessKey?: string;
      status?: string;
      startTime?: string;
      endTime?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
      instances?: T[];
      pagination?: {
        page: number;
        pageSize: number;
        total: number;
      };
      total?: number;
    }

    const query: Record<string, string | number> = {};
    if (params?.workflowId) query.processDefinitionKey = params.workflowId;
    if (params?.status) query.status = params.status;
    if (params?.page) query.page = params.page;
    if (params?.pageSize) query.pageSize = params.pageSize;

    // 修正: 确保路径与后端一致
    const res = await httpClient.get<{
      data?: Array<{
        id: string;
        instanceId: string;
        processDefinitionKey: string;
        businessKey: string;
        status: string;
        startTime: string;
        endTime?: string;
      }>;
      pagination?: { total: number };
      total?: number;
    }>('/api/v1/bpmn/process-instances', query);
    // httpClient.get returns responseData.data directly, which is an array
    const list = Array.isArray(res) ? res : res?.data || [];
    const instances: WorkflowInstance[] = list.map(item => ({
      id: item.instanceId || item.id || '',
      workflowId: item.processDefinitionKey || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item.startTime ? new Date(item.startTime) : new Date(),
      endTime: item.endTime ? new Date(item.endTime) : undefined,
      startedBy: 0,
      startedByName: '',
    }));
    return {
      instances,
      total: res?.pagination?.total || res?.total || list.length,
    };
  }

  /**
   * 获取单个工作流实例
   */
  static async getInstance(instanceId: string): Promise<WorkflowInstance> {
    const item = await httpClient.get<{
      id: string;
      instanceId: string;
      processDefinitionKey: string;
      businessKey: string;
      status: string;
      startTime: string;
      endTime?: string;
    }>(`/api/v1/bpmn/process-instances/${instanceId}`);
    return {
      id: item.instanceId || item.id || '',
      workflowId: item.processDefinitionKey || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item?.startTime ? new Date(item.startTime) : new Date(),
      endTime: item?.endTime ? new Date(item.endTime) : undefined,
      startedBy: 0,
      startedByName: '',
    };
  }

  /**
   * 取消工作流实例
   */
  static async cancelInstance(instanceId: string, reason?: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/terminate`, { reason });
  }

  /**
   * 暂停工作流实例
   */
  static async suspendInstance(instanceId: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/suspend`, {
      reason: 'User requested',
    });
  }

  /**
   * 恢复工作流实例
   */
  static async resumeInstance(instanceId: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/resume`);
  }

  /**
   * 重试失败的实例
   */
  static async retryInstance(instanceId: string): Promise<void> {
    // 后端暂不支持重试，标记为已完成
    await WorkflowApi.resumeInstance(instanceId);
  }

  // ==================== 节点实例管理 ====================

  /**
   * 获取节点实例列表
   */
  static async getNodeInstances(instanceId: string): Promise<NodeInstance[]> {
    interface TaskResponse {
      id?: string;
      taskId?: string;
      name?: string;
      status?: string;
      assignee?: string;
      assigneeName?: string;
      createdTime?: string;
      dueDate?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
    }

    // 后端暂无专用节点实例API，尝试从任务列表获取
    try {
      const instance = await WorkflowApi.getInstance(instanceId);
       
      const _unused = instance;
      const tasksRes = await httpClient.get<
        Array<{
          id: string;
          taskId: string;
          name: string;
          status: string;
          assignee?: string;
          assigneeName?: string;
          createdTime?: string;
          dueDate?: string;
        }>
      >(`/api/v1/bpmn/tasks?processInstanceId=${instanceId}`);
      const list = tasksRes || [];
      return list.map(item => ({
        id: item.taskId || item.id || '',
        instanceId: instanceId,
        nodeId: item.taskId || '',
        nodeName: item.name || '',
        status:
          (item.status as 'pending' | 'running' | 'completed' | 'failed' | 'skipped') || 'pending',
        assignee: item.assignee ? parseInt(item.assignee) : undefined,
        assigneeName: item.assigneeName,
        startTime: item.createdTime ? new Date(item.createdTime) : undefined,
        retryCount: 0,
      }));
    } catch {
      return [];
    }
  }

  /**
   * 完成节点任务
   */
  static async completeNode(request: CompleteNodeRequest): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/tasks/${request.nodeId}/complete`, {
      variables: request.output,
    });
  }

  /**
   * 跳过节点
   */
  static async skipNode(instanceId: string, nodeId: string): Promise<void> {
    // 后端暂不支持跳过节点，直接完成节点
    await WorkflowApi.completeNode({ instanceId, nodeId, output: { _skipped: true } });
  }

  /**
   * 重新执行节点
   */
  static async retryNode(instanceId: string, nodeId: string): Promise<void> {
    // 后端暂不支持重新执行节点，创建新任务实例
    await WorkflowApi.completeNode({ instanceId, nodeId, output: { _retried: true } });
  }

  // ==================== 工作流模板 ====================

  /**
   * 获取工作流模板列表
   */
  static async getTemplates(params?: {
    category?: string;
    search?: string;
  }): Promise<WorkflowTemplate[]> {
    // return httpClient.get('/api/v1/workflow-templates', params);
    return [];
  }

  /**
   * 获取单个工作流模板
   */
  static async getTemplate(id: string): Promise<WorkflowTemplate> {
    // 后端暂无模板接口，返回空模板数据
    return {
      id,
      name: '未命名模板',
      description: '',
      category: 'general',
      isPublic: false,
      tags: [],
      definition: {
        code: '',
        name: '',
         
        type: WorkflowType.TICKET,
        version: 1,
        status: WorkflowStatus.DRAFT,
        nodes: [],
        connections: [],
        variables: [],
        triggers: [],
        settings: {
          allowParallelInstances: true,
          enableVersionControl: true,
          enableAuditLog: true,
        },
      },
      usageCount: 0,
      createdAt: new Date(),
    };
  }

  /**
   * 从模板创建工作流
   */
  static async createFromTemplate(templateId: string, name: string): Promise<WorkflowDefinition> {
    // 后端暂无模板创建接口，使用基础创建
    return WorkflowApi.createWorkflow({
      code: `workflow_${Date.now()}`,
      name,
      description: `从模板 ${templateId} 创建`,
       
      type: WorkflowType.TICKET,
    });
  }

  /**
   * 保存为模板
   */
  static async saveAsTemplate(
    workflowId: string,
    data: {
      name: string;
      category: string;
      description?: string;
      isPublic?: boolean;
      tags?: string[];
    }
  ): Promise<WorkflowTemplate> {
    // 后端暂无保存模板接口，返回模拟数据
     
    const _unused = workflowId;
    return {
      id: `template_${Date.now()}`,
      name: data.name,
      category: data.category,
      description: data.description,
      isPublic: data.isPublic ?? false,
      tags: data.tags ?? [],
      definition: {
        code: '',
        name: data.name,
         
        type: WorkflowType.TICKET,
        version: 1,
        status: WorkflowStatus.DRAFT,
        nodes: [],
        connections: [],
        variables: [],
        triggers: [],
        settings: {
          allowParallelInstances: true,
          enableVersionControl: true,
          enableAuditLog: true,
        },
      },
      usageCount: 0,
      createdAt: new Date(),
    };
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取工作流统计
   */
  static async getWorkflowStats(
    workflowId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<WorkflowStats> {
    // return httpClient.get(`/api/v1/workflows/${workflowId}/stats`, params);
    // Return mock for now
     
    const _unused = workflowId;
     
    const _unused2 = params;
    return {
      workflowId,
      totalInstances: 0,
      runningInstances: 0,
      completedInstances: 0,
      failedInstances: 0,
      avgDuration: 0,
      successRate: 0,
      bottlenecks: [],
    };
  }

  /**
   * 获取实例统计
   */
  static async getInstanceStats(params?: {
    processDefinitionKey?: string;
    status?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{
    total: number;
    running: number;
    completed: number;
    suspended: number;
    terminated: number;
  }> {
    const query: Record<string, string> = {};
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.status) query.status = params.status;
    if (params?.startDate) query.startDate = params.startDate;
    if (params?.endDate) query.endDate = params.endDate;

    const res = await httpClient.get<{
      total: number;
      running: number;
      completed: number;
      suspended: number;
      terminated: number;
    }>('/api/v1/bpmn/stats/instances', query);

    const data = res;
    return data || {
      total: 0,
      running: 0,
      completed: 0,
      suspended: 0,
      terminated: 0,
    };
  }

  /**
   * 获取任务统计
   */
  static async getTaskStats(params?: {
    processDefinitionKey?: string;
    assignee?: string;
    status?: string;
    startDate?: string;
    endDate?: string;
  }): Promise<{
    totalTasks: number;
    completedTasks: number;
    pendingTasks: number;
    overdueTasks: number;
    averageCompletion: number;
  }> {
    const query: Record<string, string> = {};
    if (params?.processDefinitionKey)
      query.processDefinitionKey = params.processDefinitionKey;
    if (params?.assignee) query.assignee = params.assignee;
    if (params?.status) query.status = params.status;
    if (params?.startDate) query.startDate = params.startDate;
    if (params?.endDate) query.endDate = params.endDate;

    const res = await httpClient.get<{
      totalTasks: number;
      completedTasks: number;
      pendingTasks: number;
      overdueTasks: number;
      averageCompletion: number;
    }>('/api/v1/bpmn/stats/tasks', query);

    const data = res;
    return data || {
      totalTasks: 0,
      completedTasks: 0,
      pendingTasks: 0,
      overdueTasks: 0,
      averageCompletion: 0,
    };
  }

  // ==================== 会签审批 ====================

  /**
   * 创建会签任务
   */
  static async createCounterSignTasks(
    taskId: string,
    approvers: string[],
    approvalType: 'serial' | 'parallel' = 'parallel',
    threshold?: number
  ): Promise<
    Array<{
      taskId: string;
      assignee: string;
      status: string;
    }>
  > {
    const res = await httpClient.post<{
      code: number;
      message: string;
      data: Array<{
        taskId: string;
        assignee: string;
        status: string;
      }>;
    }>(`/api/v1/bpmn/tasks/${taskId}/counter-sign`, {
      approvalType: approvalType,
      approvers,
      threshold: threshold || approvers.length,
    });

    return res?.data || [];
  }

  /**
   * 获取会签状态
   */
  static async getCounterSignStatus(taskId: string): Promise<{
    parentTaskId: string;
    total: number;
    completed: number;
    approved: number;
    rejected: number;
    pending: number;
    status: 'pending' | 'approved' | 'rejected';
  }> {
    const res = await httpClient.get<{
      code: number;
      message: string;
      data: {
        parentTaskId: string;
        total: number;
        completed: number;
        approved: number;
        rejected: number;
        pending: number;
        status: 'pending' | 'approved' | 'rejected';
      };
    }>(`/api/v1/bpmn/tasks/${taskId}/counter-sign-status`);

    return (
      res?.data || {
        parentTaskId: taskId,
        total: 0,
        completed: 0,
        approved: 0,
        rejected: 0,
        pending: 0,
        status: 'pending',
      }
    );
  }

  /**
   * 投票（完成会签任务）
   */
  static async vote(taskId: string, approved: boolean, comment?: string): Promise<void> {
    await httpClient.put<{
      code: number;
      message: string;
    }>(`/api/v1/bpmn/tasks/${taskId}/vote`, {
      approved,
      comment,
    });
  }

  /**
   * 获取节点执行统计
   */
  static async getNodeStats(
    workflowId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<NodeExecutionStats[]> {
    // return httpClient.get(`/api/v1/workflows/${workflowId}/node-stats`, params);
    return [];
  }

  /**
   * 获取性能瓶颈分析
   */
  static async getBottleneckAnalysis(workflowId: string): Promise<{
    bottlenecks: Array<{
      nodeId: string;
      nodeName: string;
      avgDuration: number;
      impact: 'high' | 'medium' | 'low';
      recommendations: string[];
    }>;
  }> {
    // return httpClient.get(`/api/v1/workflows/${workflowId}/bottlenecks`);
    return { bottlenecks: [] };
  }

  /**
   * 导出工作流报告
   */
  static async exportReport(params: {
    workflowId: string;
    format: 'excel' | 'pdf';
    startDate: string;
    endDate: string;
  }): Promise<Blob> {
    const [workflow, workflowStats, instanceStats, taskStats] = await Promise.all([
      WorkflowApi.getWorkflow(params.workflowId).catch(() => null),
      WorkflowApi.getWorkflowStats(params.workflowId, {
        startDate: params.startDate,
        endDate: params.endDate,
      }),
      WorkflowApi.getInstanceStats({
        processDefinitionKey: params.workflowId,
        startDate: params.startDate,
        endDate: params.endDate,
      }),
      WorkflowApi.getTaskStats({
        processDefinitionKey: params.workflowId,
        startDate: params.startDate,
        endDate: params.endDate,
      }),
    ]);

    const rows = [
      ['报告类型', '工作流报告'],
      ['工作流ID', params.workflowId],
      ['工作流名称', workflow?.name || params.workflowId],
      ['开始日期', params.startDate],
      ['结束日期', params.endDate],
      ['总实例数', workflowStats.totalInstances || instanceStats.total],
      ['运行中实例', workflowStats.runningInstances || instanceStats.running],
      ['已完成实例', workflowStats.completedInstances || instanceStats.completed],
      ['失败实例', workflowStats.failedInstances],
      ['暂停实例', instanceStats.suspended],
      ['终止实例', instanceStats.terminated],
      ['总任务数', taskStats.totalTasks],
      ['已完成任务', taskStats.completedTasks],
      ['待办任务', taskStats.pendingTasks],
      ['逾期任务', taskStats.overdueTasks],
      ['平均任务完成时间', taskStats.averageCompletion],
      ['成功率', workflowStats.successRate],
      ['平均耗时', workflowStats.avgDuration],
      ['导出格式请求', params.format],
      ['导出时间', new Date().toISOString()],
    ];
    const csv = rows.map(row => row.map(WorkflowApi.escapeCSV).join(',')).join('\n');
    return new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' });
  }
}

export default WorkflowApi;
export const WorkflowAPI = WorkflowApi;
