/**
 * 工作流模板管理 API
 *
 * 负责工作流模板的查询、创建、保存等功能
 */

import { httpClient } from './http-client';
import type { WorkflowTemplate, WorkflowDefinition } from '@/types/workflow';
import { WorkflowStatus } from '@/types/workflow';

export class WorkflowTemplateApi {
  /**
   * 获取工作流模板列表
   */
  static async getTemplates(params?: {
    category?: string;
    search?: string;
  }): Promise<WorkflowTemplate[]> {
    return [];
  }

  /**
   * 获取单个工作流模板
   */
  static async getTemplate(id: string): Promise<WorkflowTemplate> {
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
  static async createFromTemplate(templateId: string, name: string): Promise<WorkflowDefinition> {
    return WorkflowDefinitionApi.createWorkflow({
      code: `workflow_${Date.now()}`,
      name,
      description: `从模板 ${templateId} 创建`,
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
}

// 需要引用 WorkflowDefinitionApi
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const WorkflowDefinitionApi: any = {
  createWorkflow: async (request: any) => {
    // 这里需要实际导入，但为避免循环依赖，暂时留空
    throw new Error('WorkflowDefinitionApi not initialized');
  },
};

export default WorkflowTemplateApi;
