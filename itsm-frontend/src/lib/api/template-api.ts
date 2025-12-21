/**
 * 工单模板 API 服务
 * 提供完整的模板CRUD、使用统计、评分等功能
 */

import { httpClient } from './http-client';
import type {
  TicketTemplate,
  CreateTemplateRequest,
  UpdateTemplateRequest,
  TemplateListQuery,
  TemplateListResponse,
  CreateTicketFromTemplateRequest,
  TemplateUsageStats,
  TemplateRating,
  TemplateDuplicateRequest,
  TemplateExportFormat,
  TemplateImportRequest,
  TemplateValidationResult,
  TemplateCategory,
} from '@/types/template';
import type { Ticket } from '@/types/ticket';

export class TemplateApi {
  // ==================== 模板CRUD ====================

  /**
   * 获取模板列表
   */
  static async getTemplates(
    query?: TemplateListQuery
  ): Promise<TemplateListResponse> {
    return httpClient.get<TemplateListResponse>('/api/v1/templates', query);
  }

  /**
   * 获取模板详情
   */
  static async getTemplate(templateId: string): Promise<TicketTemplate> {
    return httpClient.get<TicketTemplate>(`/api/v1/templates/${templateId}`);
  }

  /**
   * 创建模板
   */
  static async createTemplate(
    data: CreateTemplateRequest
  ): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>('/api/v1/templates', data);
  }

  /**
   * 更新模板
   */
  static async updateTemplate(
    templateId: string,
    data: UpdateTemplateRequest
  ): Promise<TicketTemplate> {
    return httpClient.patch<TicketTemplate>(
      `/api/v1/templates/${templateId}`,
      data
    );
  }

  /**
   * 删除模板（软删除）
   */
  static async deleteTemplate(templateId: string): Promise<void> {
    return httpClient.delete(`/api/v1/templates/${templateId}`);
  }

  /**
   * 归档模板
   */
  static async archiveTemplate(templateId: string): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>(
      `/api/v1/templates/${templateId}/archive`
    );
  }

  /**
   * 取消归档模板
   */
  static async unarchiveTemplate(templateId: string): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>(
      `/api/v1/templates/${templateId}/unarchive`
    );
  }

  // ==================== 模板版本控制 ====================

  /**
   * 发布模板（从草稿到正式版本）
   */
  static async publishTemplate(
    templateId: string,
    changelog?: string
  ): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>(
      `/api/v1/templates/${templateId}/publish`,
      { changelog }
    );
  }

  /**
   * 创建模板草稿
   */
  static async createDraft(templateId: string): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>(
      `/api/v1/templates/${templateId}/draft`
    );
  }

  /**
   * 获取模板版本历史
   */
  static async getTemplateVersions(templateId: string): Promise<Array<any>> {
    return httpClient.get(`/api/v1/templates/${templateId}/versions`);
  }

  /**
   * 回滚到指定版本
   */
  static async rollbackToVersion(
    templateId: string,
    version: string
  ): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>(
      `/api/v1/templates/${templateId}/rollback`,
      { version }
    );
  }

  /**
   * 比较两个版本
   */
  static async compareVersions(
    templateId: string,
    versionA: string,
    versionB: string
  ): Promise<any> {
    return httpClient.get(`/api/v1/templates/${templateId}/compare`, {
      version_a: versionA,
      version_b: versionB,
    });
  }

  // ==================== 从模板创建工单 ====================

  /**
   * 使用模板创建工单
   */
  static async createTicketFromTemplate(
    data: CreateTicketFromTemplateRequest
  ): Promise<Ticket> {
    return httpClient.post<Ticket>('/api/v1/templates/create-ticket', data);
  }

  /**
   * 预览从模板创建的工单
   */
  static async previewTicketFromTemplate(
    data: CreateTicketFromTemplateRequest
  ): Promise<Partial<Ticket>> {
    return httpClient.post<Partial<Ticket>>(
      '/api/v1/templates/preview-ticket',
      data
    );
  }

  // ==================== 模板分类管理 ====================

  /**
   * 获取模板分类列表
   */
  static async getCategories(): Promise<TemplateCategory[]> {
    return httpClient.get<TemplateCategory[]>('/api/v1/template-categories');
  }

  /**
   * 创建模板分类
   */
  static async createCategory(
    data: Omit<TemplateCategory, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<TemplateCategory> {
    return httpClient.post<TemplateCategory>(
      '/api/v1/template-categories',
      data
    );
  }

  /**
   * 更新模板分类
   */
  static async updateCategory(
    categoryId: string,
    data: Partial<TemplateCategory>
  ): Promise<TemplateCategory> {
    return httpClient.patch<TemplateCategory>(
      `/api/v1/template-categories/${categoryId}`,
      data
    );
  }

  /**
   * 删除模板分类
   */
  static async deleteCategory(categoryId: string): Promise<void> {
    return httpClient.delete(`/api/v1/template-categories/${categoryId}`);
  }

  // ==================== 模板使用统计 ====================

  /**
   * 获取模板使用统计
   */
  static async getTemplateStats(
    templateId: string
  ): Promise<TemplateUsageStats> {
    return httpClient.get<TemplateUsageStats>(
      `/api/v1/templates/${templateId}/stats`
    );
  }

  /**
   * 记录模板使用
   */
  static async recordTemplateUsage(templateId: string): Promise<void> {
    return httpClient.post(`/api/v1/templates/${templateId}/use`);
  }

  /**
   * 获取最近使用的模板
   */
  static async getRecentTemplates(limit = 10): Promise<TicketTemplate[]> {
    return httpClient.get<TicketTemplate[]>('/api/v1/templates/recent', {
      limit,
    });
  }

  /**
   * 获取最受欢迎的模板
   */
  static async getPopularTemplates(limit = 10): Promise<TicketTemplate[]> {
    return httpClient.get<TicketTemplate[]>('/api/v1/templates/popular', {
      limit,
    });
  }

  /**
   * 获取推荐模板
   */
  static async getRecommendedTemplates(
    userId?: string
  ): Promise<TicketTemplate[]> {
    return httpClient.get<TicketTemplate[]>('/api/v1/templates/recommended', {
      user_id: userId,
    });
  }

  // ==================== 模板评分 ====================

  /**
   * 为模板评分
   */
  static async rateTemplate(
    templateId: string,
    rating: number,
    comment?: string
  ): Promise<TemplateRating> {
    return httpClient.post<TemplateRating>(
      `/api/v1/templates/${templateId}/rate`,
      { rating, comment }
    );
  }

  /**
   * 获取模板评分列表
   */
  static async getTemplateRatings(
    templateId: string
  ): Promise<TemplateRating[]> {
    return httpClient.get<TemplateRating[]>(
      `/api/v1/templates/${templateId}/ratings`
    );
  }

  /**
   * 获取用户对模板的评分
   */
  static async getUserRating(
    templateId: string,
    userId: string
  ): Promise<TemplateRating | null> {
    return httpClient.get<TemplateRating | null>(
      `/api/v1/templates/${templateId}/ratings/${userId}`
    );
  }

  // ==================== 模板复制和导入导出 ====================

  /**
   * 复制模板
   */
  static async duplicateTemplate(
    data: TemplateDuplicateRequest
  ): Promise<TicketTemplate> {
    return httpClient.post<TicketTemplate>('/api/v1/templates/duplicate', data);
  }

  /**
   * 导出模板
   */
  static async exportTemplate(
    templateId: string,
    format: TemplateExportFormat
  ): Promise<Blob> {
    const response = await httpClient.request({
      method: 'GET',
      url: `/api/v1/templates/${templateId}/export`,
      params: format,
      responseType: 'blob',
    });
    return response as Blob;
  }

  /**
   * 批量导出模板
   */
  static async exportTemplates(
    templateIds: string[],
    format: TemplateExportFormat
  ): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/templates/export/batch',
      data: { template_ids: templateIds, ...format },
      responseType: 'blob',
    });
    return response as Blob;
  }

  /**
   * 导入模板
   */
  static async importTemplate(
    data: TemplateImportRequest
  ): Promise<TicketTemplate | TemplateValidationResult> {
    const formData = new FormData();
    
    if (data.data instanceof File) {
      formData.append('file', data.data);
    } else {
      formData.append('data', data.data);
    }
    
    formData.append('format', data.format);
    
    if (data.overwriteExisting !== undefined) {
      formData.append('overwrite_existing', String(data.overwriteExisting));
    }
    
    if (data.validateOnly !== undefined) {
      formData.append('validate_only', String(data.validateOnly));
    }

    return httpClient.post('/api/v1/templates/import', formData);
  }

  // ==================== 模板验证 ====================

  /**
   * 验证模板配置
   */
  static async validateTemplate(
    data: CreateTemplateRequest | UpdateTemplateRequest
  ): Promise<TemplateValidationResult> {
    return httpClient.post<TemplateValidationResult>(
      '/api/v1/templates/validate',
      data
    );
  }

  /**
   * 检查模板名称是否可用
   */
  static async checkTemplateName(
    name: string,
    excludeId?: string
  ): Promise<{ available: boolean; suggestions?: string[] }> {
    return httpClient.get('/api/v1/templates/check-name', {
      name,
      exclude_id: excludeId,
    });
  }

  // ==================== 批量操作 ====================

  /**
   * 批量启用/禁用模板
   */
  static async batchToggleTemplates(
    templateIds: string[],
    isActive: boolean
  ): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/templates/batch/toggle', {
      template_ids: templateIds,
      is_active: isActive,
    });
  }

  /**
   * 批量删除模板
   */
  static async batchDeleteTemplates(
    templateIds: string[]
  ): Promise<{ success: number; failed: number }> {
    return httpClient.request({
      method: 'DELETE',
      url: '/api/v1/templates/batch',
      data: { template_ids: templateIds },
    });
  }

  /**
   * 批量归档模板
   */
  static async batchArchiveTemplates(
    templateIds: string[]
  ): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/templates/batch/archive', {
      template_ids: templateIds,
    });
  }

  /**
   * 批量更新模板分类
   */
  static async batchUpdateCategory(
    templateIds: string[],
    categoryId: string
  ): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/templates/batch/update-category', {
      template_ids: templateIds,
      category_id: categoryId,
    });
  }

  // ==================== 模板搜索和推荐 ====================

  /**
   * 搜索模板（支持全文搜索）
   */
  static async searchTemplates(
    query: string,
    filters?: Partial<TemplateListQuery>
  ): Promise<TemplateListResponse> {
    return httpClient.get<TemplateListResponse>('/api/v1/templates/search', {
      q: query,
      ...filters,
    });
  }

  /**
   * 智能推荐模板（基于用户行为）
   */
  static async getSmartRecommendations(
    context?: {
      ticketType?: string;
      category?: string;
      priority?: string;
      tags?: string[];
    }
  ): Promise<TicketTemplate[]> {
    return httpClient.post<TicketTemplate[]>(
      '/api/v1/templates/smart-recommend',
      context
    );
  }

  // ==================== 模板字段建议 ====================

  /**
   * 获取字段配置建议
   */
  static async getFieldSuggestions(
    categoryId?: string
  ): Promise<Array<{ name: string; type: string; label: string }>> {
    return httpClient.get('/api/v1/templates/field-suggestions', {
      category_id: categoryId,
    });
  }

  /**
   * 获取常用字段模板
   */
  static async getCommonFields(): Promise<Array<any>> {
    return httpClient.get('/api/v1/templates/common-fields');
  }

  // ==================== 模板预览和测试 ====================

  /**
   * 生成模板预览
   */
  static async generatePreview(
    templateId: string,
    sampleData?: Record<string, any>
  ): Promise<any> {
    return httpClient.post(`/api/v1/templates/${templateId}/preview`, {
      sample_data: sampleData,
    });
  }

  /**
   * 测试模板自动化规则
   */
  static async testAutomation(
    templateId: string,
    testData: Record<string, any>
  ): Promise<{
    autoAssign: any;
    autoNotify: any;
    autoTag: any;
    approvalWorkflow: any;
  }> {
    return httpClient.post(`/api/v1/templates/${templateId}/test-automation`, {
      test_data: testData,
    });
  }

  // ==================== 收藏和关注 ====================

  /**
   * 收藏模板
   */
  static async favoriteTemplate(templateId: string): Promise<void> {
    return httpClient.post(`/api/v1/templates/${templateId}/favorite`);
  }

  /**
   * 取消收藏模板
   */
  static async unfavoriteTemplate(templateId: string): Promise<void> {
    return httpClient.delete(`/api/v1/templates/${templateId}/favorite`);
  }

  /**
   * 获取用户收藏的模板
   */
  static async getFavoriteTemplates(): Promise<TicketTemplate[]> {
    return httpClient.get<TicketTemplate[]>('/api/v1/templates/favorites');
  }

  /**
   * 检查模板是否已收藏
   */
  static async isFavorite(templateId: string): Promise<boolean> {
    const result = await httpClient.get<{ is_favorite: boolean }>(
      `/api/v1/templates/${templateId}/favorite/status`
    );
    return result.is_favorite;
  }
}

// 导出默认实例和类
export default TemplateApi;

// 导出别名以支持不同的导入方式
export const TemplateAPI = TemplateApi;

