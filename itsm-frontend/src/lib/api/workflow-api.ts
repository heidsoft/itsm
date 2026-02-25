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

export class WorkflowApi {
  // ==================== 工作流定义管理 ====================

  /**
   * 获取工作流列表
   */
  static async getWorkflows(
    query?: WorkflowQuery
  ): Promise<{
    workflows: WorkflowDefinition[];
    total: number;
  }> {
    const params: Record<string, string | number> = {};
    if (query?.page) params.page = query.page;
    if (query?.pageSize) params.page_size = query.pageSize;

    // 修正: 确保路径与后端一致，后端可能是 /api/v1/bpmn/process-definitions
    const res = await httpClient.get<Array<{
      id: number;
      key: string;
      name: string;
      description?: string;
      version: number;
      status: string;
      created_at: string;
      updated_at: string;
    }>>('/api/v1/bpmn/process-definitions', params);

    const list = Array.isArray(res) ? res : [];
    const workflows: WorkflowDefinition[] = list.map((item) => ({
      id: String(item.id || ''),
      code: item.key || '',
      name: item.name || '',
      type: WorkflowType.TICKET,
      version: item.version || 1,
      status: item.status ? (item.status as WorkflowStatus) : WorkflowStatus.DRAFT,
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
      createdAt: item.created_at ? new Date(item.created_at) : new Date(),
      updatedAt: item.updated_at ? new Date(item.updated_at) : new Date(),
      description: item.description,
    })) as WorkflowDefinition[];
    return { workflows, total: list.length };
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
      version: number;
      status: string;
      created_at: string;
      updated_at: string;
    }>(`/api/v1/bpmn/process-definitions/${id}`);

    const item = res;
    return {
      id: String(item.id || ''),
      code: item.key || '',
      name: item.name || '',
      type: WorkflowType.TICKET,
      version: item.version || 1,
      status: item.status ? (item.status as WorkflowStatus) : WorkflowStatus.DRAFT,
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
      createdAt: item.created_at ? new Date(item.created_at) : new Date(),
      updatedAt: item.updated_at ? new Date(item.updated_at) : new Date(),
      description: item.description,
    };
  }

  // ==================== Backward-compatible method aliases (legacy pages) ====================

  static async getProcessDefinition(key: string): Promise<WorkflowDefinition> {
    return WorkflowApi.getWorkflow(key);
  }

  static async getProcessVersions(key: string): Promise<WorkflowDefinition[]> {
    return WorkflowApi.getWorkflowVersions(key);
  }

  static async createProcessDefinition(request: unknown): Promise<WorkflowDefinition> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return WorkflowApi.createWorkflow(request as any);
  }

  static async updateProcessDefinition(key: string, request: unknown, version?: string): Promise<WorkflowDefinition> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return WorkflowApi.updateWorkflow(key, request as any, version);
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
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      type: current.type as any,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      bpmn_xml: (current as any).bpmn_xml,
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

  static async listWorkflowInstances(params?: unknown): Promise<{ instances: WorkflowInstance[]; total: number }> {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return WorkflowApi.getInstances(params as any);
  }

  static async listWorkflowTasks(_instanceId: string): Promise<WorkflowTask[]> {
    // tasks API not implemented; return empty list for now
    return [];
  }

  static async suspendWorkflow(instanceId: string): Promise<void> {
    return WorkflowApi.suspendInstance(instanceId);
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
  static async createWorkflow(
    request: CreateWorkflowRequest
  ): Promise<WorkflowDefinition> {
    // 从 httpClient 获取当前租户ID
    const tenantId = httpClient.getTenantId() || 1;

    const payload = {
        key: request.code || `process_${Date.now()}`,
        name: request.name,
        description: request.description,
        category: request.type, // Map type to category
        bpmn_xml: (request as any).bpmn_xml,
        tenant_id: tenantId,
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
        bpmn_xml: (request as any).bpmn_xml,
    };
    // Backend expects key in path. Assuming id passed here is key.
    // Also backend needs version parameter. We default to 1.0.0 or need to fetch it.
    const ver = version || '1.0.0';
    const response = await httpClient.put<{
      code: number;
      message: string;
      data: WorkflowDefinition;
    }>(`/api/v1/bpmn/process-definitions/${id}?version=${ver}`, payload);

    if (response?.code !== 0) {
      throw new Error(response?.message || '更新工作流失败');
    }
    return response?.data;
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
  static async cloneWorkflow(
    id: string,
    name: string
  ): Promise<WorkflowDefinition> {
    // 后端暂不支持，通过获取原工作流并创建新实例来实现
    const original = await WorkflowApi.getWorkflow(id);
    return WorkflowApi.createWorkflow({
      code: `${original.code}_copy_${Date.now()}`,
      name: name || `${original.name} (副本)`,
      description: original.description,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      type: original.type as any,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      bpmn_xml: (original as any).bpmn_xml,
    });
  }

  /**
   * 激活工作流
   */
  static async activateWorkflow(id: string, version?: string): Promise<void> {
    const ver = version || '1.0.0';
    const response = await httpClient.put<{
      code: number;
      message: string;
    }>(`/api/v1/bpmn/process-definitions/${id}/active?version=${ver}`, { active: true });

    if (response?.code !== 0) {
      throw new Error(response?.message || '激活工作流失败');
    }
  }

  /**
   * 停用工作流
   */
  static async deactivateWorkflow(id: string, version?: string): Promise<void> {
    const ver = version || '1.0.0';
    await httpClient.put(`/api/v1/bpmn/process-definitions/${id}/active?version=${ver}`, { active: false });
  }

  /**
   * 验证工作流
   */
  static async validateWorkflow(
    workflow: Partial<WorkflowDefinition>
  ): Promise<ValidationResult> {
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
        bpmn_xml: (workflow as any).bpmn_xml || '',
      },
    };
  }

  /**
   * 导入工作流
   */
  static async importWorkflow(
    data: WorkflowExport
  ): Promise<WorkflowDefinition> {
    // 后端暂不支持导入，使用创建接口
    if (!data.workflow) {
      throw new Error('无效的导入数据');
    }
    return WorkflowApi.createWorkflow({
      code: data.workflow.code || `imported_${Date.now()}`,
      name: data.workflow.name,
      description: data.workflow.description,
      type: (data.workflow as any).type || 'ticket',
      bpmn_xml: data.workflow.bpmn_xml || (data.workflow as any).bpmn_xml,
    });
  }

  // ==================== 工作流版本管理 ====================

  /**
   * 获取工作流版本列表
   */
  static async getWorkflowVersions(
    workflowId: string
  ): Promise<WorkflowDefinition[]> {
    interface ProcessDefinitionResponse {
      id?: number;
      key?: string;
      name?: string;
      description?: string;
      version?: number;
      status?: string;
      created_at?: string;
      updated_at?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
    }

     // 使用新的版本 API
     const res = await httpClient.get<Array<{
      id: number;
      key: string;
      name: string;
      description?: string;
      version: number;
      status: string;
      created_at: string;
      updated_at: string;
     }>>(`/api/v1/bpmn/versions?process_key=${workflowId}`);

    const list = Array.isArray(res) ? res : [];
    return list.map((item) => ({
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
      createdAt: item.created_at ? new Date(item.created_at) : new Date(),
      updatedAt: item.updated_at ? new Date(item.updated_at) : new Date(),
      description: item.description,
    })) as WorkflowDefinition[];
  }

  /**
   * 发布新版本
   */
  static async publishVersion(
    workflowId: string
  ): Promise<WorkflowDefinition> {
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
  static async activateVersion(
    processKey: string,
    version: number
  ): Promise<void> {
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
  static async deleteVersion(
    processKey: string,
    version: number
  ): Promise<void> {
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
    elements_added: string[];
    elements_removed: string[];
    elements_modified: string[];
    connections_added: string[];
    connections_removed: string[];
    variables_changed: string[];
    is_identical: boolean;
  }> {
    interface CompareResponse {
      added_nodes?: string[];
      removed_nodes?: string[];
      modified_nodes?: string[];
    }

    interface VersionComparisonResponse {
      elements_added: string[];
      elements_removed: string[];
      elements_modified: string[];
      connections_added: string[];
      connections_removed: string[];
      variables_changed: string[];
      is_identical: boolean;
    }

    const res = await httpClient.get<{
      added_nodes?: string[];
      removed_nodes?: string[];
      modified_nodes?: string[];
    }>(`/api/v1/bpmn/versions/${processKey}/compare`, {
      base_version: baseVersion,
      target_version: targetVersion,
    });
    const data = res;
    if (!data) {
      return {
        elements_added: [],
        elements_removed: [],
        elements_modified: [],
        connections_added: [],
        connections_removed: [],
        variables_changed: [],
        is_identical: true,
      };
    }
    // 适配后端返回格式
    return {
      elements_added: data.added_nodes || [],
      elements_removed: data.removed_nodes || [],
      elements_modified: data.modified_nodes || [],
      connections_added: [],
      connections_removed: [],
      variables_changed: [],
      is_identical: !(data.added_nodes?.length || data.removed_nodes?.length || data.modified_nodes?.length),
    };
  }

  // ==================== 工作流实例管理 ====================

  /**
   * 启动工作流实例
   */
  static async startWorkflow(
    request: StartWorkflowRequest
  ): Promise<WorkflowInstance> {
    const payload = {
        process_definition_key: request.workflowId, // Assuming workflowId is the Key
        business_key: `BIZ-${Date.now()}`, // Auto generate or from request
        variables: request.variables
    };
    const item = await httpClient.post<{
      id: string;
      process_instance_id: string;
      process_definition_key: string;
      business_key: string;
      status: string;
      start_time: string;
      end_time?: string;
    }>('/api/v1/bpmn/process-instances', payload);
    return {
      id: item.process_instance_id || item.id || '',
      workflowId: item.process_definition_key || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item.start_time ? new Date(item.start_time) : new Date(),
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
      instance_id?: string;
      process_definition_key?: string;
      business_key?: string;
      status?: string;
      start_time?: string;
      end_time?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
      instances?: T[];
      pagination?: {
        page: number;
        page_size: number;
        total: number;
      };
      total?: number;
    }

    const query: Record<string, string | number> = {};
    if (params?.workflowId) query.process_definition_key = params.workflowId;
    if (params?.status) query.status = params.status;
    if (params?.page) query.page = params.page;
    if (params?.pageSize) query.page_size = params.pageSize;

    // 修正: 确保路径与后端一致
    const res = await httpClient.get<{
      data?: Array<{
        id: string;
        instance_id: string;
        process_definition_key: string;
        business_key: string;
        status: string;
        start_time: string;
        end_time?: string;
      }>;
      pagination?: { total: number };
      total?: number;
    }>('/api/v1/bpmn/process-instances', query);
    // httpClient.get returns responseData.data directly, which is an array
    const list = Array.isArray(res) ? res : (res?.data || []);
    const instances: WorkflowInstance[] = list.map((item) => ({
      id: item.instance_id || item.id || '',
      workflowId: item.process_definition_key || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item.start_time ? new Date(item.start_time) : new Date(),
      endTime: item.end_time ? new Date(item.end_time) : undefined,
      startedBy: 0,
      startedByName: '',
    }));
    return {
        instances,
        total: (res as any)?.pagination?.total || (res as any)?.total || list.length
    };
  }

  /**
   * 获取单个工作流实例
   */
  static async getInstance(instanceId: string): Promise<WorkflowInstance> {
    const item = await httpClient.get<{
      id: string;
      instance_id: string;
      process_definition_key: string;
      business_key: string;
      status: string;
      start_time: string;
      end_time?: string;
    }>(`/api/v1/bpmn/process-instances/${instanceId}`);
    return {
      id: item.instance_id || item.id || '',
      workflowId: item.process_definition_key || '',
      workflowName: '',
      version: 1,
      status: (item.status as WorkflowInstanceStatus) || WorkflowInstanceStatus.RUNNING,
      variables: {},
      startTime: item?.start_time ? new Date(item.start_time) : new Date(),
      endTime: item?.end_time ? new Date(item.end_time) : undefined,
      startedBy: 0,
      startedByName: '',
    };
  }

  /**
   * 取消工作流实例
   */
  static async cancelInstance(
    instanceId: string,
    reason?: string
  ): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/terminate`, { reason });
  }

  /**
   * 暂停工作流实例
   */
  static async suspendInstance(instanceId: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-instances/${instanceId}/suspend`, { reason: "User requested" });
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
  static async getNodeInstances(
    instanceId: string
  ): Promise<NodeInstance[]> {
    interface TaskResponse {
      id?: string;
      task_id?: string;
      name?: string;
      status?: string;
      assignee?: string;
      assignee_name?: string;
      created_time?: string;
      due_date?: string;
      [key: string]: unknown;
    }

    interface PaginatedResponse<T> {
      data?: T[];
      items?: T[];
    }

    // 后端暂无专用节点实例API，尝试从任务列表获取
    try {
      const instance = await WorkflowApi.getInstance(instanceId);
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const _unused = instance;
      const tasksRes = await httpClient.get<Array<{
        id: string;
        task_id: string;
        name: string;
        status: string;
        assignee?: string;
        assignee_name?: string;
        created_time?: string;
        due_date?: string;
      }>>(`/api/v1/bpmn/tasks?processInstanceId=${instanceId}`);
      const list = tasksRes || [];
      return list.map((item) => ({
        id: item.task_id || item.id || '',
        instanceId: instanceId,
        nodeId: item.task_id || '',
        nodeName: item.name || '',
        status: (item.status as 'pending' | 'running' | 'completed' | 'failed' | 'skipped') || 'pending',
        assignee: item.assignee ? parseInt(item.assignee) : undefined,
        assigneeName: item.assignee_name,
        startTime: item.created_time ? new Date(item.created_time) : undefined,
        retryCount: 0,
      }));
    } catch {
      return [];
    }
  }

  /**
   * 完成节点任务
   */
  static async completeNode(
    request: CompleteNodeRequest
  ): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/tasks/${request.nodeId}/complete`, {
        variables: request.output
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
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        type: 'ticket' as any,
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
  static async createFromTemplate(
    templateId: string,
    name: string
  ): Promise<WorkflowDefinition> {
    // 后端暂无模板创建接口，使用基础创建
    return WorkflowApi.createWorkflow({
      code: `workflow_${Date.now()}`,
      name,
      description: `从模板 ${templateId} 创建`,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      type: 'ticket' as any,
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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        type: 'ticket' as any,
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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const _unused = workflowId;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const _unused2 = params;
    return {
        workflowId,
        totalInstances: 0,
        runningInstances: 0,
        completedInstances: 0,
        failedInstances: 0,
        avgDuration: 0,
        successRate: 0,
        bottlenecks: []
    };
  }

  /**
   * 获取实例统计
   */
  static async getInstanceStats(params?: {
    process_definition_key?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<{
    total: number;
    running: number;
    completed: number;
    suspended: number;
    terminated: number;
  }> {
    const query: Record<string, string> = {};
    if (params?.process_definition_key) query.process_definition_key = params.process_definition_key;
    if (params?.status) query.status = params.status;
    if (params?.start_date) query.start_date = params.start_date;
    if (params?.end_date) query.end_date = params.end_date;

    const res = await httpClient.get<{
      code: number;
      message: string;
      data: {
        total: number;
        running: number;
        completed: number;
        suspended: number;
        terminated: number;
      };
    }>('/api/v1/bpmn/stats/instances', query);

    return res?.data || {
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
    process_definition_key?: string;
    assignee?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<{
    total_tasks: number;
    completed_tasks: number;
    pending_tasks: number;
    overdue_tasks: number;
    average_completion: number;
  }> {
    const query: Record<string, string> = {};
    if (params?.process_definition_key) query.process_definition_key = params.process_definition_key;
    if (params?.assignee) query.assignee = params.assignee;
    if (params?.status) query.status = params.status;
    if (params?.start_date) query.start_date = params.start_date;
    if (params?.end_date) query.end_date = params.end_date;

    const res = await httpClient.get<{
      code: number;
      message: string;
      data: {
        total_tasks: number;
        completed_tasks: number;
        pending_tasks: number;
        overdue_tasks: number;
        average_completion: number;
      };
    }>('/api/v1/bpmn/stats/tasks', query);

    return res?.data || {
      total_tasks: 0,
      completed_tasks: 0,
      pending_tasks: 0,
      overdue_tasks: 0,
      average_completion: 0,
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
  ): Promise<Array<{
    task_id: string;
    assignee: string;
    status: string;
  }>> {
    const res = await httpClient.post<{
      code: number;
      message: string;
      data: Array<{
        task_id: string;
        assignee: string;
        status: string;
      }>;
    }>(`/api/v1/bpmn/tasks/${taskId}/counter-sign`, {
      approval_type: approvalType,
      approvers,
      threshold: threshold || approvers.length,
    });

    return res?.data || [];
  }

  /**
   * 获取会签状态
   */
  static async getCounterSignStatus(taskId: string): Promise<{
    parent_task_id: string;
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
        parent_task_id: string;
        total: number;
        completed: number;
        approved: number;
        rejected: number;
        pending: number;
        status: 'pending' | 'approved' | 'rejected';
      };
    }>(`/api/v1/bpmn/tasks/${taskId}/counter-sign-status`);

    return res?.data || {
      parent_task_id: taskId,
      total: 0,
      completed: 0,
      approved: 0,
      rejected: 0,
      pending: 0,
      status: 'pending',
    };
  }

  /**
   * 投票（完成会签任务）
   */
  static async vote(
    taskId: string,
    approved: boolean,
    comment?: string
  ): Promise<void> {
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
  static async getBottleneckAnalysis(
    workflowId: string
  ): Promise<{
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
    // 后端暂不支持报告导出，抛出明确错误
    throw new Error('报告导出功能暂未实现，请稍后重试');
  }
}

export default WorkflowApi;
export const WorkflowAPI = WorkflowApi;
