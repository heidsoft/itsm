import { BaseService, type PaginatedResponse, type PaginationParams, type BaseEntity } from "./api-service";

// 工单模板相关类型定义
export interface TicketTemplate extends BaseEntity {
  id: string;
  name: string;
  description: string;
  category: string;
  priority: string;
  form_fields: Record<string, any>;
  workflow_steps: WorkflowStep[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
  tenant_id: string;
}

export interface WorkflowStep {
  id: string;
  name: string;
  type: "approval" | "assignment" | "notification" | "automation";
  assignee_group?: string;
  assignee_user?: string;
  due_time?: number;
  conditions?: string;
  order: number;
}

export interface FormField {
  id: string;
  name: string;
  label: string;
  type: "text" | "textarea" | "select" | "number" | "date" | "checkbox" | "radio";
  required: boolean;
  default_value?: string;
  options?: string[];
  placeholder?: string;
  validation_rules?: string;
  order: number;
}

export interface CreateTemplateRequest {
  name: string;
  description: string;
  category: string;
  priority: string;
  form_fields: Record<string, any>;
  workflow_steps: WorkflowStep[];
  is_active: boolean;
}

export interface UpdateTemplateRequest {
  name?: string;
  description?: string;
  category?: string;
  priority?: string;
  form_fields?: Record<string, any>;
  workflow_steps?: WorkflowStep[];
  is_active?: boolean;
}

export interface TemplateFilterParams extends PaginationParams {
  category?: string;
  is_active?: boolean;
  keyword?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// 工单模板服务类
export class TicketTemplateService extends BaseService {
  
  /**
   * 获取工单模板列表
   * 对应后端: GET /api/v1/templates
   */
  async getTemplates(params: TemplateFilterParams): Promise<PaginatedResponse<TicketTemplate>> {
    try {
      const queryParams = new URLSearchParams();
      
      // 分页参数
      if (params.page) queryParams.append('page', params.page.toString());
      if (params.size) queryParams.append('size', params.size.toString());
      if (params.sort_by) queryParams.append('sort_by', params.sort_by);
      if (params.sort_order) queryParams.append('sort_order', params.sort_order);
      
      // 过滤参数
      if (params.category) queryParams.append('category', params.category);
      if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString());
      if (params.keyword) queryParams.append('keyword', params.keyword);
      
      const url = this.getUrl(`/templates?${queryParams.toString()}`);
      return await this.client.get<PaginatedResponse<TicketTemplate>>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取单个工单模板
   * 对应后端: GET /api/v1/templates/:id
   */
  async getTemplate(id: string): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/templates/${id}`);
      return await this.client.get<TicketTemplate>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 创建工单模板
   * 对应后端: POST /api/v1/templates
   */
  async createTemplate(data: CreateTemplateRequest): Promise<TicketTemplate> {
    try {
      const url = this.getUrl('/templates');
      return await this.client.post<TicketTemplate>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 更新工单模板
   * 对应后端: PUT /api/v1/templates/:id
   */
  async updateTemplate(id: string, data: UpdateTemplateRequest): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/templates/${id}`);
      return await this.client.put<TicketTemplate>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 删除工单模板
   * 对应后端: DELETE /api/v1/templates/:id
   */
  async deleteTemplate(id: string): Promise<void> {
    try {
      const url = this.getUrl(`/templates/${id}`);
      await this.client.delete(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 复制工单模板
   * 对应后端: POST /api/v1/templates/:id/copy
   */
  async copyTemplate(id: string, newName?: string): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/templates/${id}/copy`);
      const data = newName ? { name: newName } : {};
      return await this.client.post<TicketTemplate>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取模板统计信息
   * 对应后端: GET /api/v1/templates/stats
   */
  async getTemplateStats(params?: {
    tenant_id?: string;
    date_from?: string;
    date_to?: string;
  }): Promise<{
    total: number;
    active: number;
    inactive: number;
    categories: number;
    total_usage: number;
  }> {
    try {
      const queryParams = new URLSearchParams();
      if (params?.tenant_id) queryParams.append('tenant_id', params.tenant_id);
      if (params?.date_from) queryParams.append('date_from', params.date_from);
      if (params?.date_to) queryParams.append('date_to', params.date_to);
      
      const url = this.getUrl(`/templates/stats?${queryParams.toString()}`);
      return await this.client.get(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 导出工单模板
   * 对应后端: GET /api/v1/templates/export
   */
  async exportTemplates(params: TemplateFilterParams, format: 'csv' | 'excel' = 'csv'): Promise<Blob> {
    try {
      const queryParams = new URLSearchParams();
      
      // 过滤参数
      if (params.category) queryParams.append('category', params.category);
      if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString());
      if (params.keyword) queryParams.append('keyword', params.keyword);
      if (params.sort_by) queryParams.append('sort_by', params.sort_by);
      if (params.sort_order) queryParams.append('sort_order', params.sort_order);
      
      queryParams.append('format', format);
      
      const url = this.getUrl(`/templates/export?${queryParams.toString()}`);
      const response = await fetch(url, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
        },
      });
      
      if (!response.ok) {
        throw new Error('导出失败');
      }
      
      return await response.blob();
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 导入工单模板
   * 对应后端: POST /api/v1/templates/import
   */
  async importTemplates(file: File, options?: {
    validate?: boolean;
    overwrite?: boolean;
  }): Promise<{
    success: number;
    failed: number;
    errors: string[];
  }> {
    try {
      const formData = new FormData();
      formData.append('file', file);
      
      if (options?.validate !== undefined) {
        formData.append('validate', options.validate.toString());
      }
      if (options?.overwrite !== undefined) {
        formData.append('overwrite', options.overwrite.toString());
      }
      
      const url = this.getUrl('/templates/import');
      return await this.client.post(url, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 批量操作工单模板
   * 对应后端: POST /api/v1/templates/batch
   */
  async batchOperation(operation: 'activate' | 'deactivate' | 'delete', templateIds: string[]): Promise<{
    success: number;
    failed: number;
    errors: string[];
  }> {
    try {
      const url = this.getUrl('/templates/batch');
      return await this.client.post(url, {
        operation,
        template_ids: templateIds,
      });
    } catch (error) {
      this.handleError(error);
    }
  }
}

// 创建服务实例
export const ticketTemplateService = new TicketTemplateService();
