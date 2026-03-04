/**
 * 工作流版本管理 API
 *
 * 负责工作流版本控制、发布、激活、回滚、比较等功能
 */

import { httpClient } from './http-client';
import type { WorkflowDefinition } from '@/types/workflow';
import { WorkflowType, WorkflowStatus } from '@/types/workflow';

export class WorkflowVersionApi {
  /**
   * 获取工作流版本列表
   */
  static async getWorkflowVersions(workflowId: string): Promise<WorkflowDefinition[]> {
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
      createdAt: item.created_at ? new Date(item.created_at) : new Date(),
      updatedAt: item.updated_at ? new Date(item.updated_at) : new Date(),
      description: item.description,
    })) as WorkflowDefinition[];
  }

  /**
   * 发布新版本
   */
  static async publishVersion(workflowId: string): Promise<WorkflowDefinition> {
    const versions = await WorkflowVersionApi.getWorkflowVersions(workflowId);
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
    elements_added: string[];
    elements_removed: string[];
    elements_modified: string[];
    connections_added: string[];
    connections_removed: string[];
    variables_changed: string[];
    is_identical: boolean;
  }> {
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
      is_identical: !(
        data.added_nodes?.length ||
        data.removed_nodes?.length ||
        data.modified_nodes?.length
      ),
    };
  }

  // ==================== 向后兼容的别名方法 ====================

  static async getProcessVersions(key: string): Promise<WorkflowDefinition[]> {
    return WorkflowVersionApi.getWorkflowVersions(key);
  }
}

export default WorkflowVersionApi;
