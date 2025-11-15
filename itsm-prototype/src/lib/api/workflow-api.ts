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
    return httpClient.get('/api/v1/workflows', query);
  }

  /**
   * 获取单个工作流
   */
  static async getWorkflow(id: string): Promise<WorkflowDefinition> {
    return httpClient.get(`/api/v1/workflows/${id}`);
  }

  /**
   * 创建工作流
   */
  static async createWorkflow(
    request: CreateWorkflowRequest
  ): Promise<WorkflowDefinition> {
    return httpClient.post('/api/v1/workflows', request);
  }

  /**
   * 更新工作流
   */
  static async updateWorkflow(
    id: string,
    request: UpdateWorkflowRequest
  ): Promise<WorkflowDefinition> {
    return httpClient.put(`/api/v1/workflows/${id}`, request);
  }

  /**
   * 删除工作流
   */
  static async deleteWorkflow(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/workflows/${id}`);
  }

  /**
   * 复制工作流
   */
  static async cloneWorkflow(
    id: string,
    name: string
  ): Promise<WorkflowDefinition> {
    return httpClient.post(`/api/v1/workflows/${id}/clone`, { name });
  }

  /**
   * 激活工作流
   */
  static async activateWorkflow(id: string): Promise<void> {
    return httpClient.post(`/api/v1/workflows/${id}/activate`);
  }

  /**
   * 停用工作流
   */
  static async deactivateWorkflow(id: string): Promise<void> {
    return httpClient.post(`/api/v1/workflows/${id}/deactivate`);
  }

  /**
   * 验证工作流
   */
  static async validateWorkflow(
    workflow: Partial<WorkflowDefinition>
  ): Promise<ValidationResult> {
    return httpClient.post('/api/v1/workflows/validate', workflow);
  }

  /**
   * 导出工作流
   */
  static async exportWorkflow(id: string): Promise<WorkflowExport> {
    return httpClient.get(`/api/v1/workflows/${id}/export`);
  }

  /**
   * 导入工作流
   */
  static async importWorkflow(
    data: WorkflowExport
  ): Promise<WorkflowDefinition> {
    return httpClient.post('/api/v1/workflows/import', data);
  }

  // ==================== 工作流版本管理 ====================

  /**
   * 获取工作流版本列表
   */
  static async getWorkflowVersions(
    workflowId: string
  ): Promise<WorkflowDefinition[]> {
    return httpClient.get(`/api/v1/workflows/${workflowId}/versions`);
  }

  /**
   * 发布新版本
   */
  static async publishVersion(
    workflowId: string
  ): Promise<WorkflowDefinition> {
    return httpClient.post(`/api/v1/workflows/${workflowId}/publish`);
  }

  /**
   * 回滚到指定版本
   */
  static async rollbackVersion(
    workflowId: string,
    version: number
  ): Promise<WorkflowDefinition> {
    return httpClient.post(`/api/v1/workflows/${workflowId}/rollback`, {
      version,
    });
  }

  // ==================== 工作流实例管理 ====================

  /**
   * 启动工作流实例
   */
  static async startWorkflow(
    request: StartWorkflowRequest
  ): Promise<WorkflowInstance> {
    return httpClient.post('/api/v1/workflow-instances/start', request);
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
    return httpClient.get('/api/v1/workflow-instances', params);
  }

  /**
   * 获取单个工作流实例
   */
  static async getInstance(instanceId: string): Promise<WorkflowInstance> {
    return httpClient.get(`/api/v1/workflow-instances/${instanceId}`);
  }

  /**
   * 取消工作流实例
   */
  static async cancelInstance(
    instanceId: string,
    reason?: string
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/workflow-instances/${instanceId}/cancel`,
      { reason }
    );
  }

  /**
   * 暂停工作流实例
   */
  static async suspendInstance(instanceId: string): Promise<void> {
    return httpClient.post(`/api/v1/workflow-instances/${instanceId}/suspend`);
  }

  /**
   * 恢复工作流实例
   */
  static async resumeInstance(instanceId: string): Promise<void> {
    return httpClient.post(`/api/v1/workflow-instances/${instanceId}/resume`);
  }

  /**
   * 重试失败的实例
   */
  static async retryInstance(instanceId: string): Promise<void> {
    return httpClient.post(`/api/v1/workflow-instances/${instanceId}/retry`);
  }

  // ==================== 节点实例管理 ====================

  /**
   * 获取节点实例列表
   */
  static async getNodeInstances(
    instanceId: string
  ): Promise<NodeInstance[]> {
    return httpClient.get(
      `/api/v1/workflow-instances/${instanceId}/nodes`
    );
  }

  /**
   * 完成节点任务
   */
  static async completeNode(
    request: CompleteNodeRequest
  ): Promise<void> {
    return httpClient.post(
      `/api/v1/workflow-instances/${request.instanceId}/nodes/${request.nodeId}/complete`,
      {
        output: request.output,
        comment: request.comment,
      }
    );
  }

  /**
   * 跳过节点
   */
  static async skipNode(instanceId: string, nodeId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/workflow-instances/${instanceId}/nodes/${nodeId}/skip`
    );
  }

  /**
   * 重新执行节点
   */
  static async retryNode(instanceId: string, nodeId: string): Promise<void> {
    return httpClient.post(
      `/api/v1/workflow-instances/${instanceId}/nodes/${nodeId}/retry`
    );
  }

  // ==================== 工作流模板 ====================

  /**
   * 获取工作流模板列表
   */
  static async getTemplates(params?: {
    category?: string;
    search?: string;
  }): Promise<WorkflowTemplate[]> {
    return httpClient.get('/api/v1/workflow-templates', params);
  }

  /**
   * 获取单个工作流模板
   */
  static async getTemplate(id: string): Promise<WorkflowTemplate> {
    return httpClient.get(`/api/v1/workflow-templates/${id}`);
  }

  /**
   * 从模板创建工作流
   */
  static async createFromTemplate(
    templateId: string,
    name: string
  ): Promise<WorkflowDefinition> {
    return httpClient.post(
      `/api/v1/workflow-templates/${templateId}/create`,
      { name }
    );
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
    return httpClient.post(`/api/v1/workflows/${workflowId}/save-as-template`, data);
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
    return httpClient.get(`/api/v1/workflows/${workflowId}/stats`, params);
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
    return httpClient.get(`/api/v1/workflows/${workflowId}/node-stats`, params);
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
    return httpClient.get(`/api/v1/workflows/${workflowId}/bottlenecks`);
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
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/workflows/export-report',
      data: params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default WorkflowApi;
export const WorkflowAPI = WorkflowApi;
