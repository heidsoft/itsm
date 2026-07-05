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
      bpmnXml: item.bpmnXml || (item as any).bpmnXml,
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
    elementsAdded: string[];
    elementsRemoved: string[];
    elementsModified: string[];
    connectionsAdded: string[];
    connectionsRemoved: string[];
    variablesChanged: string[];
    isIdentical: boolean;
  }> {
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

  // ==================== 向后兼容的别名方法 ====================

  static async getProcessVersions(key: string): Promise<WorkflowDefinition[]> {
    return WorkflowVersionApi.getWorkflowVersions(key);
  }
}

export default WorkflowVersionApi;
