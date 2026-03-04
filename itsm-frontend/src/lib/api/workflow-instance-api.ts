/**
 * 工作流实例管理 API
 *
 * 负责工作流实例的启动、查询、取消、暂停、恢复等操作
 */

import { httpClient } from './http-client';
import type { WorkflowInstance, StartWorkflowRequest } from '@/types/workflow';
import { WorkflowInstanceStatus } from '@/types/workflow';

export class WorkflowInstanceApi {
  /**
   * 启动工作流实例
   */
  static async startWorkflow(request: StartWorkflowRequest): Promise<WorkflowInstance> {
    const payload = {
      process_definition_key: request.workflowId,
      business_key: `BIZ-${Date.now()}`,
      variables: request.variables,
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
    const query: Record<string, string | number> = {};
    if (params?.workflowId) query.process_definition_key = params.workflowId;
    if (params?.status) query.status = params.status;
    if (params?.page) query.page = params.page;
    if (params?.pageSize) query.page_size = params.pageSize;

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

    const list = Array.isArray(res) ? res : res?.data || [];
    const instances: WorkflowInstance[] = list.map(item => ({
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
      total: (res as any)?.pagination?.total || (res as any)?.total || list.length,
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
    await WorkflowInstanceApi.resumeInstance(instanceId);
  }

  // ==================== 向后兼容的别名方法 ====================

  static async listWorkflowInstances(
    params?: unknown
  ): Promise<{ instances: WorkflowInstance[]; total: number }> {
    return WorkflowInstanceApi.getInstances(params as any);
  }

  static async suspendWorkflow(instanceId: string): Promise<void> {
    return WorkflowInstanceApi.suspendInstance(instanceId);
  }

  static async resumeWorkflow(instanceId: string): Promise<void> {
    return WorkflowInstanceApi.resumeInstance(instanceId);
  }

  static async terminateWorkflow(instanceId: string, reason?: string): Promise<void> {
    return WorkflowInstanceApi.cancelInstance(instanceId, reason);
  }
}

export default WorkflowInstanceApi;
