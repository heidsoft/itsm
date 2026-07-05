/**
 * 工作流统计和分析 API
 *
 * 负责工作流实例统计、任务统计、性能分析等功能
 */

import { httpClient } from './http-client';
import { WorkflowApi } from './workflow-api';
import type { WorkflowStats, NodeExecutionStats } from '@/types/workflow';

export class WorkflowStatsApi {
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

    const res = await httpClient.get<
      | {
          total: number;
          running: number;
          completed: number;
          suspended: number;
          terminated: number;
        }
      | {
          code: number;
          message: string;
          data: {
            total: number;
            running: number;
            completed: number;
            suspended: number;
            terminated: number;
          };
        }
    >('/api/v1/bpmn/stats/instances', query);

    const data = (res as any)?.data || res;
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

    const res = await httpClient.get<
      | {
          totalTasks: number;
          completedTasks: number;
          pendingTasks: number;
          overdueTasks: number;
          averageCompletion: number;
        }
      | {
          code: number;
          message: string;
          data: {
            totalTasks: number;
            completedTasks: number;
            pendingTasks: number;
            overdueTasks: number;
            averageCompletion: number;
          };
        }
    >('/api/v1/bpmn/stats/tasks', query);

    const data = (res as any)?.data || res;
    return data || {
      totalTasks: 0,
      completedTasks: 0,
      pendingTasks: 0,
      overdueTasks: 0,
      averageCompletion: 0,
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
    return WorkflowApi.exportReport(params);
  }
}

export default WorkflowStatsApi;
