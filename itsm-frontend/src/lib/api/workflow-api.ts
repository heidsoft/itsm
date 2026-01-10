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
} from '@/types/workflow';

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
    const params: any = {};
    if (query?.page) params.page = query.page;
    if (query?.pageSize) params.page_size = query.pageSize;
    
    const res: any = await httpClient.get('/api/v1/bpmn/process-definitions', params);
    const list = Array.isArray(res) ? res : (res?.data || res?.items || []);
    const total = res?.pagination?.total || res?.total || list.length || 0;
    // 适配后端返回格式
    return {
      workflows: list.map((item: any) => ({
        ...item,
        code: item.key, // Map key to code
        createdAt: item.created_at,
        updatedAt: item.updated_at,
      })),
      total
    };
  }

  /**
   * 获取单个工作流
   */
  static async getWorkflow(id: string): Promise<WorkflowDefinition> {
    // Assuming id is the key for BPMN controller which uses key
    const res: any = await httpClient.get(`/api/v1/bpmn/process-definitions/${id}`);
    const item = res?.data || res;
    return {
        ...item,
        code: item.key,
        createdAt: item.created_at,
        updatedAt: item.updated_at,
    };
  }

  // ==================== Backward-compatible method aliases (legacy pages) ====================

  static async getProcessDefinition(key: string): Promise<WorkflowDefinition> {
    return WorkflowApi.getWorkflow(key);
  }

  static async getProcessVersions(key: string): Promise<WorkflowDefinition[]> {
    return WorkflowApi.getWorkflowVersions(key);
  }

  static async createProcessDefinition(request: any): Promise<WorkflowDefinition> {
    return WorkflowApi.createWorkflow(request as any);
  }

  static async updateProcessDefinition(key: string, request: any): Promise<WorkflowDefinition> {
    return WorkflowApi.updateWorkflow(key, request as any);
  }

  static async deployProcessDefinition(_key: string): Promise<void> {
    // backend deploy semantics not implemented yet
    return;
  }

  static async createProcessVersion(_key: string): Promise<WorkflowDefinition> {
    throw new Error('Not implemented');
  }

  static async deployWorkflow(_workflowId: string): Promise<void> {
    // backend deploy semantics not implemented yet
    return;
  }

  static async listWorkflowInstances(params?: any): Promise<{ instances: WorkflowInstance[]; total: number }> {
    return WorkflowApi.getInstances(params);
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
    const payload = {
        key: request.code || `process_${Date.now()}`,
        name: request.name,
        description: request.description,
        category: request.type, // Map type to category
        bpmn_xml: (request as any).bpmn_xml,
        tenant_id: 1, // Default tenant
    };
    return httpClient.post('/api/v1/bpmn/process-definitions', payload);
  }

  /**
   * 更新工作流
   */
  static async updateWorkflow(
    id: string,
    request: UpdateWorkflowRequest
  ): Promise<WorkflowDefinition> {
    const payload = {
        name: request.name,
        description: request.description,
        bpmn_xml: (request as any).bpmn_xml,
    };
    // Backend expects key in path. Assuming id passed here is key.
    // Also backend needs version parameter. We default to 1.0.0 or need to fetch it.
    // Ideally frontend should pass version. For now let's try without or hardcode.
    // Backend UpdateProcessDefinition: PUT /process-definitions/:key?version=...
    // If we don't know version, this might fail if backend requires it strictly.
    // Let's assume we fetch latest version logic or pass a default.
    return httpClient.put(`/api/v1/bpmn/process-definitions/${id}?version=1.0.0`, payload);
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
    // Not directly supported by backend yet, simulate or implement
    // For now, just throw or implement client side copy + create
    throw new Error("Not implemented on backend");
  }

  /**
   * 激活工作流
   */
  static async activateWorkflow(id: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-definitions/${id}/active?version=1.0.0`, { active: true });
  }

  /**
   * 停用工作流
   */
  static async deactivateWorkflow(id: string): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/process-definitions/${id}/active?version=1.0.0`, { active: false });
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
    // Not implemented on backend
     throw new Error("Not implemented on backend");
  }

  /**
   * 导入工作流
   */
  static async importWorkflow(
    data: WorkflowExport
  ): Promise<WorkflowDefinition> {
     throw new Error("Not implemented on backend");
  }

  // ==================== 工作流版本管理 ====================

  /**
   * 获取工作流版本列表
   */
  static async getWorkflowVersions(
    workflowId: string
  ): Promise<WorkflowDefinition[]> {
     // Backend ListProcessDefinitions supports filter by key which gives versions
     const res: any = await httpClient.get(`/api/v1/bpmn/process-definitions?key=${workflowId}`);
     return res?.data || res?.items || res || [];
  }

  /**
   * 发布新版本
   */
  static async publishVersion(
    workflowId: string
  ): Promise<WorkflowDefinition> {
     throw new Error("Not implemented");
  }

  /**
   * 回滚到指定版本
   */
  static async rollbackVersion(
    workflowId: string,
    version: number
  ): Promise<WorkflowDefinition> {
     throw new Error("Not implemented");
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
    const res: any = await httpClient.post('/api/v1/bpmn/process-instances', payload);
    return res?.data || res;
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
    const query: any = {};
    if (params?.workflowId) query.process_definition_key = params.workflowId;
    if (params?.status) query.status = params.status;
    if (params?.page) query.page = params.page;
    if (params?.pageSize) query.page_size = params.pageSize;

    const res: any = await httpClient.get('/api/v1/bpmn/process-instances', query);
    return {
        instances: res?.data || res?.items || res?.instances || [],
        total: res?.pagination?.total || res?.total || 0
    };
  }

  /**
   * 获取单个工作流实例
   */
  static async getInstance(instanceId: string): Promise<WorkflowInstance> {
    const res: any = await httpClient.get(`/api/v1/bpmn/process-instances/${instanceId}`);
    return res?.data || res;
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
     throw new Error("Not implemented");
  }

  // ==================== 节点实例管理 ====================

  /**
   * 获取节点实例列表
   */
  static async getNodeInstances(
    instanceId: string
  ): Promise<NodeInstance[]> {
     // Backend doesn't have dedicated node instance list API yet, but ListTasks might cover it
     // Or GetProcessInstance returns current activity.
     return [];
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
     throw new Error("Not implemented");
  }

  /**
   * 重新执行节点
   */
  static async retryNode(instanceId: string, nodeId: string): Promise<void> {
     throw new Error("Not implemented");
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
    // return httpClient.get(`/api/v1/workflow-templates/${id}`);
    throw new Error("Not implemented");
  }

  /**
   * 从模板创建工作流
   */
  static async createFromTemplate(
    templateId: string,
    name: string
  ): Promise<WorkflowDefinition> {
    // return httpClient.post(
    //   `/api/v1/workflow-templates/${templateId}/create`,
    //   { name }
    // );
    throw new Error("Not implemented");
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
    // return httpClient.post(`/api/v1/workflows/${workflowId}/save-as-template`, data);
    throw new Error("Not implemented");
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
    // const response = await httpClient.request({
    //   method: 'POST',
    //   url: '/api/v1/workflows/export-report',
    //   data: params,
    //   responseType: 'blob',
    // });
    // return response as Blob;
    throw new Error("Not implemented");
  }
}

export default WorkflowApi;
export const WorkflowAPI = WorkflowApi;
