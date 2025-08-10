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

export interface FormField {
  id: string;
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'select' | 'number' | 'date' | 'checkbox' | 'radio';
  required: boolean;
  default_value?: string;
  options?: string[];
  placeholder?: string;
  validation_rules?: string;
  order: number;
}

export interface WorkflowStep {
  id: string;
  name: string;
  type: 'approval' | 'assignment' | 'notification' | 'automation';
  assignee_group?: string;
  assignee_user?: string;
  due_time?: number;
  conditions?: string;
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
}

// 工单模板服务类
export class TicketTemplateService extends BaseService {
  
  /**
   * 获取工单模板列表
   * 对应后端: GET /api/v1/ticket-templates
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
      
      const url = this.getUrl(`/ticket-templates?${queryParams.toString()}`);
      return await this.client.get<PaginatedResponse<TicketTemplate>>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取单个工单模板
   * 对应后端: GET /api/v1/ticket-templates/:id
   */
  async getTemplate(id: string): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/ticket-templates/${id}`);
      return await this.client.get<TicketTemplate>(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 创建工单模板
   * 对应后端: POST /api/v1/ticket-templates
   */
  async createTemplate(data: CreateTemplateRequest): Promise<TicketTemplate> {
    try {
      const url = this.getUrl('/ticket-templates');
      return await this.client.post<TicketTemplate>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 更新工单模板
   * 对应后端: PUT /api/v1/ticket-templates/:id
   */
  async updateTemplate(id: string, data: UpdateTemplateRequest): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/ticket-templates/${id}`);
      return await this.client.put<TicketTemplate>(url, data);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 删除工单模板
   * 对应后端: DELETE /api/v1/ticket-templates/:id
   */
  async deleteTemplate(id: string): Promise<void> {
    try {
      const url = this.getUrl(`/ticket-templates/${id}`);
      return await this.client.delete(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 复制工单模板
   * 对应后端: POST /api/v1/ticket-templates/:id/copy
   */
  async copyTemplate(id: string, newName: string): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/ticket-templates/${id}/copy`);
      return await this.client.post<TicketTemplate>(url, { name: newName });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 启用/禁用工单模板
   * 对应后端: PATCH /api/v1/ticket-templates/:id/status
   */
  async updateTemplateStatus(id: string, isActive: boolean): Promise<TicketTemplate> {
    try {
      const url = this.getUrl(`/ticket-templates/${id}/status`);
      return await this.client.patch<TicketTemplate>(url, { is_active: isActive });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 获取模板统计信息
   * 对应后端: GET /api/v1/ticket-templates/stats
   */
  async getTemplateStats(): Promise<{
    total: number;
    active: number;
    inactive: number;
    by_category: Record<string, number>;
  }> {
    try {
      const url = this.getUrl('/ticket-templates/stats');
      return await this.client.get(url);
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 批量操作模板
   * 对应后端: POST /api/v1/ticket-templates/batch
   */
  async batchOperation(operation: 'enable' | 'disable' | 'delete', ids: string[]): Promise<{
    success: number;
    failed: number;
    errors: string[];
  }> {
    try {
      const url = this.getUrl('/ticket-templates/batch');
      return await this.client.post(url, { operation, ids });
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 导出模板
   * 对应后端: GET /api/v1/ticket-templates/export
   */
  async exportTemplates(format: 'json' | 'csv' | 'excel', filters?: TemplateFilterParams): Promise<Blob> {
    try {
      const queryParams = new URLSearchParams();
      queryParams.append('format', format);
      
      if (filters) {
        if (filters.category) queryParams.append('category', filters.category);
        if (filters.is_active !== undefined) queryParams.append('is_active', filters.is_active.toString());
        if (filters.keyword) queryParams.append('keyword', filters.keyword);
      }
      
      const url = this.getUrl(`/ticket-templates/export?${queryParams.toString()}`);
      const response = await this.client.get(url, { responseType: 'blob' });
      return response;
    } catch (error) {
      this.handleError(error);
    }
  }

  /**
   * 导入模板
   * 对应后端: POST /api/v1/ticket-templates/import
   */
  async importTemplates(file: File, options?: {
    overwrite?: boolean;
    validate_only?: boolean;
  }): Promise<{
    imported: number;
    skipped: number;
    errors: string[];
  }> {
    try {
      const formData = new FormData();
      formData.append('file', file);
      
      if (options) {
        if (options.overwrite !== undefined) formData.append('overwrite', options.overwrite.toString());
        if (options.validate_only !== undefined) formData.append('validate_only', options.validate_only.toString());
      }
      
      const url = this.getUrl('/ticket-templates/import');
      return await this.client.post(url, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
    } catch (error) {
      this.handleError(error);
    }
  }
}

// 创建服务实例
export const ticketTemplateService = new TicketTemplateService();
