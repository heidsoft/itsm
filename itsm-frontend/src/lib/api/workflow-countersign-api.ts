/**
 * 会签审批管理 API
 * 
 * 负责会签任务的创建、状态查询、投票等功能
 */

import { httpClient } from './http-client';

export class WorkflowCounterSignApi {
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
      task_id: string;
      assignee: string;
      status: string;
    }>
  > {
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

    return (
      res?.data || {
        parent_task_id: taskId,
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
}

export default WorkflowCounterSignApi;
