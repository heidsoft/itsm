/**
 * 工作流节点实例管理 API
 * 
 * 负责节点实例的查询、完成、跳过、重试等操作
 */

import { httpClient } from './http-client';
import type { NodeInstance, CompleteNodeRequest } from '@/types/workflow';

export class WorkflowNodeApi {
  /**
   * 获取节点实例列表
   */
  static async getNodeInstances(instanceId: string): Promise<NodeInstance[]> {
    try {
      const tasksRes = await httpClient.get<
        Array<{
          id: string;
          task_id: string;
          name: string;
          status: string;
          assignee?: string;
          assignee_name?: string;
          created_time?: string;
          due_date?: string;
        }>
      >(`/api/v1/bpmn/tasks?processInstanceId=${instanceId}`);
      const list = tasksRes || [];
      return list.map(item => ({
        id: item.task_id || item.id || '',
        instanceId: instanceId,
        nodeId: item.task_id || '',
        nodeName: item.name || '',
        status:
          (item.status as 'pending' | 'running' | 'completed' | 'failed' | 'skipped') || 'pending',
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
  static async completeNode(request: CompleteNodeRequest): Promise<void> {
    await httpClient.put(`/api/v1/bpmn/tasks/${request.nodeId}/complete`, {
      variables: request.output,
    });
  }

  /**
   * 跳过节点
   */
  static async skipNode(instanceId: string, nodeId: string): Promise<void> {
    await WorkflowNodeApi.completeNode({ instanceId, nodeId, output: { _skipped: true } });
  }

  /**
   * 重新执行节点
   */
  static async retryNode(instanceId: string, nodeId: string): Promise<void> {
    await WorkflowNodeApi.completeNode({ instanceId, nodeId, output: { _retried: true } });
  }

  // ==================== 向后兼容的别名方法 ====================

  static async listWorkflowTasks(_instanceId: string): Promise<NodeInstance[]> {
    return [];
  }
}

export default WorkflowNodeApi;
