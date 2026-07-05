import { httpClient } from '@/lib/api/http-client';

export interface TicketTemplate {
  id: number;
  name: string;
  description: string;
  category: string;
  fields: TemplateField[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface TemplateField {
  name: string;
  label: string;
  type: 'text' | 'textarea' | 'select' | 'number' | 'date' | 'checkbox';
  required: boolean;
  options?: string[];
  defaultValue?: string;
  validationRules?: string[];
}

export interface CreateTemplateRequest {
  name: string;
  description: string;
  category: string;
  fields: TemplateField[];
}

export interface UpdateTemplateRequest {
  name?: string;
  description?: string;
  category?: string;
  fields?: TemplateField[];
  isActive?: boolean;
}

export interface ListTemplatesParams {
  category?: string;
  search?: string;
  isActive?: boolean;
  page?: number;
  pageSize?: number;
}

export interface TemplateListResponse {
  templates: TicketTemplate[];
  total: number;
  page: number;
  pageSize: number;
}

class TicketTemplateService {
  // 获取模板列表
  async getTemplates(params: ListTemplatesParams = {}): Promise<TemplateListResponse> {
    try {
      // 将params对象转换为查询参数
      const queryParams: Record<string, unknown> = {};
      if (params.page) queryParams.page = params.page;
      if (params.pageSize) queryParams.pageSize = params.pageSize;
      if (params.category) queryParams.category = params.category;
      if (params.isActive !== undefined) queryParams.isActive = params.isActive;

      const response = await httpClient.get<TemplateListResponse>(
        '/api/v1/tickets/templates',
        queryParams
      );
      return response;
    } catch (error) {
      console.error('TicketTemplateService.getTemplates error:', error);
      throw error;
    }
  }

  // 获取单个模板
  async getTemplate(id: number): Promise<TicketTemplate> {
    try {
      const response = await httpClient.get<TicketTemplate>(`/api/v1/tickets/templates/${id}`);
      return response;
    } catch (error) {
      console.error('TicketTemplateService.getTemplate error:', error);
      throw error;
    }
  }

  // 创建模板
  async createTemplate(data: CreateTemplateRequest): Promise<TicketTemplate> {
    try {
      const response = await httpClient.post<TicketTemplate>('/api/v1/tickets/templates', data);
      return response;
    } catch (error) {
      console.error('TicketTemplateService.createTemplate error:', error);
      throw error;
    }
  }

  // 更新模板
  async updateTemplate(id: number, data: UpdateTemplateRequest): Promise<TicketTemplate> {
    try {
      const response = await httpClient.put<TicketTemplate>(
        `/api/v1/tickets/templates/${id}`,
        data
      );
      return response;
    } catch (error) {
      console.error('TicketTemplateService.updateTemplate error:', error);
      throw error;
    }
  }

  // 删除模板
  async deleteTemplate(id: number): Promise<void> {
    try {
      await httpClient.delete(`/api/v1/tickets/templates/${id}`);
    } catch (error) {
      console.error('TicketTemplateService.deleteTemplate error:', error);
      throw error;
    }
  }

  // 激活/停用模板
  async toggleTemplateStatus(id: number, isActive: boolean): Promise<TicketTemplate> {
    try {
      const response = await httpClient.patch<TicketTemplate>(
        `/api/v1/tickets/templates/${id}/status`,
        {
          isActive: isActive,
        }
      );
      return response;
    } catch (error) {
      console.error('TicketTemplateService.toggleTemplateStatus error:', error);
      throw error;
    }
  }

  // 复制模板
  async copyTemplate(id: number, newName: string): Promise<TicketTemplate> {
    try {
      const response = await httpClient.post<TicketTemplate>(
        `/api/v1/tickets/templates/${id}/copy`,
        {
          name: newName,
        }
      );
      return response;
    } catch (error) {
      console.error('TicketTemplateService.copyTemplate error:', error);
      throw error;
    }
  }

  // 获取模板分类
  async getTemplateCategories(): Promise<string[]> {
    try {
      const response = await httpClient.get<string[]>('/api/v1/tickets/templates/categories');
      return response;
    } catch (error) {
      console.error('TicketTemplateService.getTemplateCategories error:', error);
      throw error;
    }
  }

  // 健康检查
  async healthCheck(): Promise<boolean> {
    try {
      await httpClient.get('/api/v1/health');
      return true;
    } catch (error) {
      console.warn('TicketTemplateService health check failed:', error);
      return false;
    }
  }
}

export const ticketTemplateService = new TicketTemplateService();
