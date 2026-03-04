/**
 * 工作流定义管理 API
 *
 * 负责工作流定义的 CRUD 操作、激活/停用、导入导出等功能
 */

import { httpClient } from './http-client';
import type {
  WorkflowDefinition,
  ValidationResult,
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  WorkflowExport,
  WorkflowQuery,
} from '@/types/workflow';
import { WorkflowStatus, WorkflowType } from '@/types/workflow';

export class WorkflowDefinitionApi {
  /**
   * 获取工作流列表
   */
  static async getWorkflows(query?: WorkflowQuery): Promise<{
    workflows: WorkflowDefinition[];
    total: number;
  }> {
    const params: Record<string, string | number> = {};
    if (query?.page) params.page = query.page;
    if (query?.pageSize) params.page_size = query.pageSize;

    const res = await httpClient.get<
      Array<{
        id: number;
        key: string;
        name: string;
        description?: string;
        version: number;
        status: string;
        created_at: string;
        updated_at: string;
      }>
    >('/api/v1/bpmn/process-definitions', params);

    const list = Array.isArray(res) ? res : [];
    const workflows: WorkflowDefinition[] = list.map(item => ({
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

  /**
   * 创建工作流
   */
  static async createWorkflow(request: CreateWorkflowRequest): Promise<WorkflowDefinition> {
    const tenantId = httpClient.getTenantId() || 1;

    const payload = {
      key: request.code || `process_${Date.now()}`,
      name: request.name,
      description: request.description,
      category: request.type,
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
    await httpClient.delete(`/api/v1/bpmn/process-definitions/${id}?version=1.0.0`);
  }

  /**
   * 复制工作流
   */
  static async cloneWorkflow(id: string, name: string): Promise<WorkflowDefinition> {
    const original = await WorkflowApi.getWorkflow(id);
    return WorkflowApi.createWorkflow({
      code: `${original.code}_copy_${Date.now()}`,
      name: name || `${original.name} (副本)`,
      description: original.description,
      type: original.type as any,
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
    await httpClient.put(`/api/v1/bpmn/process-definitions/${id}/active?version=${ver}`, {
      active: false,
    });
  }

  /**
   * 验证工作流
   */
  static async validateWorkflow(workflow: Partial<WorkflowDefinition>): Promise<ValidationResult> {
    return { isValid: true, errors: [], warnings: [] };
  }

  /**
   * 导出工作流
   */
  static async exportWorkflow(id: string): Promise<WorkflowExport> {
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
  static async importWorkflow(data: WorkflowExport): Promise<WorkflowDefinition> {
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

  // ==================== 向后兼容的别名方法 ====================

  static async getProcessDefinition(key: string): Promise<WorkflowDefinition> {
    return WorkflowDefinitionApi.getWorkflow(key);
  }

  static async createProcessDefinition(request: unknown): Promise<WorkflowDefinition> {
    return WorkflowDefinitionApi.createWorkflow(request as any);
  }

  static async updateProcessDefinition(
    key: string,
    request: unknown,
    version?: string
  ): Promise<WorkflowDefinition> {
    return WorkflowDefinitionApi.updateWorkflow(key, request as any, version);
  }

  static async deployProcessDefinition(key: string, version?: string): Promise<void> {
    await WorkflowDefinitionApi.activateWorkflow(key, version);
  }

  static async createProcessVersion(key: string): Promise<WorkflowDefinition> {
    const current = await WorkflowDefinitionApi.getWorkflow(key);
    return WorkflowDefinitionApi.createWorkflow({
      code: current.code,
      name: current.name,
      description: current.description,
      type: current.type as any,
      bpmn_xml: (current as any).bpmn_xml,
    });
  }

  static async deployWorkflow(workflowId: string): Promise<void> {
    try {
      await WorkflowDefinitionApi.activateWorkflow(workflowId);
    } catch (error) {
      console.error('部署工作流失败:', error);
      throw error;
    }
  }
}

// 需要引用主 WorkflowApi 来处理循环依赖
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const WorkflowApi: any = WorkflowDefinitionApi;

export default WorkflowDefinitionApi;
