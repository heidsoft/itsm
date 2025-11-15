/**
 * 知识库 API 服务
 */

import { httpClient } from './http-client';
import type {
  KnowledgeArticle,
  KnowledgeCategory,
  KnowledgeTag,
  ArticleComment,
  ArticleFeedback,
  KnowledgeSearchRequest,
  KnowledgeSearchResult,
  ArticleRecommendation,
  ArticleVersion,
  CollaborationSession,
  KnowledgeBaseStats,
  ArticleAnalytics,
  CreateArticleRequest,
  UpdateArticleRequest,
  PublishArticleRequest,
  ReviewArticleRequest,
  ArticleQuery,
  BatchArticleOperation,
  ImageUploadResult,
} from '@/types/knowledge-base';

export class KnowledgeBaseApi {
  // ==================== 文章管理 ====================

  /**
   * 获取文章列表
   */
  static async getArticles(
    query?: ArticleQuery
  ): Promise<{
    articles: KnowledgeArticle[];
    total: number;
  }> {
    return httpClient.get('/api/v1/knowledge/articles', query);
  }

  /**
   * 获取单个文章
   */
  static async getArticle(id: string): Promise<KnowledgeArticle> {
    return httpClient.get(`/api/v1/knowledge/articles/${id}`);
  }

  /**
   * 通过slug获取文章
   */
  static async getArticleBySlug(slug: string): Promise<KnowledgeArticle> {
    return httpClient.get(`/api/v1/knowledge/articles/slug/${slug}`);
  }

  /**
   * 创建文章
   */
  static async createArticle(
    request: CreateArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.post('/api/v1/knowledge/articles', request);
  }

  /**
   * 更新文章
   */
  static async updateArticle(
    id: string,
    request: UpdateArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.put(`/api/v1/knowledge/articles/${id}`, request);
  }

  /**
   * 删除文章
   */
  static async deleteArticle(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/knowledge/articles/${id}`);
  }

  /**
   * 发布文章
   */
  static async publishArticle(
    id: string,
    request?: PublishArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.post(`/api/v1/knowledge/articles/${id}/publish`, request);
  }

  /**
   * 归档文章
   */
  static async archiveArticle(id: string): Promise<KnowledgeArticle> {
    return httpClient.post(`/api/v1/knowledge/articles/${id}/archive`);
  }

  /**
   * 复制文章
   */
  static async cloneArticle(
    id: string,
    title: string
  ): Promise<KnowledgeArticle> {
    return httpClient.post(`/api/v1/knowledge/articles/${id}/clone`, {
      title,
    });
  }

  /**
   * 批量操作
   */
  static async batchOperation(
    operation: BatchArticleOperation
  ): Promise<{ success: number; failed: number }> {
    return httpClient.post('/api/v1/knowledge/articles/batch', operation);
  }

  // ==================== 版本控制 ====================

  /**
   * 获取文章版本历史
   */
  static async getArticleVersions(
    articleId: string
  ): Promise<ArticleVersion[]> {
    return httpClient.get(`/api/v1/knowledge/articles/${articleId}/versions`);
  }

  /**
   * 恢复到指定版本
   */
  static async restoreVersion(
    articleId: string,
    version: number
  ): Promise<KnowledgeArticle> {
    return httpClient.post(
      `/api/v1/knowledge/articles/${articleId}/versions/${version}/restore`
    );
  }

  /**
   * 比较版本
   */
  static async compareVersions(
    articleId: string,
    fromVersion: number,
    toVersion: number
  ): Promise<{
    diff: string;
    changes: Array<{
      type: 'added' | 'removed' | 'modified';
      content: string;
    }>;
  }> {
    return httpClient.get(
      `/api/v1/knowledge/articles/${articleId}/versions/compare`,
      { from: fromVersion, to: toVersion }
    );
  }

  // ==================== 分类和标签 ====================

  /**
   * 获取分类列表
   */
  static async getCategories(): Promise<KnowledgeCategory[]> {
    return httpClient.get('/api/v1/knowledge/categories');
  }

  /**
   * 创建分类
   */
  static async createCategory(
    category: Omit<KnowledgeCategory, 'id' | 'level' | 'articleCount' | 'createdAt' | 'updatedAt'>
  ): Promise<KnowledgeCategory> {
    return httpClient.post('/api/v1/knowledge/categories', category);
  }

  /**
   * 更新分类
   */
  static async updateCategory(
    id: string,
    category: Partial<KnowledgeCategory>
  ): Promise<KnowledgeCategory> {
    return httpClient.put(`/api/v1/knowledge/categories/${id}`, category);
  }

  /**
   * 删除分类
   */
  static async deleteCategory(id: string): Promise<void> {
    return httpClient.delete(`/api/v1/knowledge/categories/${id}`);
  }

  /**
   * 获取标签列表
   */
  static async getTags(): Promise<KnowledgeTag[]> {
    return httpClient.get('/api/v1/knowledge/tags');
  }

  /**
   * 创建标签
   */
  static async createTag(
    tag: Omit<KnowledgeTag, 'id' | 'articleCount' | 'createdAt'>
  ): Promise<KnowledgeTag> {
    return httpClient.post('/api/v1/knowledge/tags', tag);
  }

  // ==================== 评论和反馈 ====================

  /**
   * 获取文章评论
   */
  static async getComments(
    articleId: string,
    params?: {
      page?: number;
      pageSize?: number;
    }
  ): Promise<{
    comments: ArticleComment[];
    total: number;
  }> {
    return httpClient.get(
      `/api/v1/knowledge/articles/${articleId}/comments`,
      params
    );
  }

  /**
   * 添加评论
   */
  static async addComment(
    articleId: string,
    content: string,
    parentId?: string
  ): Promise<ArticleComment> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/comments`, {
      content,
      parentId,
    });
  }

  /**
   * 删除评论
   */
  static async deleteComment(commentId: string): Promise<void> {
    return httpClient.delete(`/api/v1/knowledge/comments/${commentId}`);
  }

  /**
   * 标记评论有用
   */
  static async markCommentHelpful(commentId: string, helpful: boolean): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/comments/${commentId}/helpful`, {
      helpful,
    });
  }

  /**
   * 提交反馈
   */
  static async submitFeedback(
    articleId: string,
    feedback: Omit<ArticleFeedback, 'id' | 'userId' | 'userName' | 'createdAt'>
  ): Promise<ArticleFeedback> {
    return httpClient.post(
      `/api/v1/knowledge/articles/${articleId}/feedback`,
      feedback
    );
  }

  // ==================== 互动 ====================

  /**
   * 点赞文章
   */
  static async likeArticle(articleId: string): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/like`);
  }

  /**
   * 取消点赞
   */
  static async unlikeArticle(articleId: string): Promise<void> {
    return httpClient.delete(`/api/v1/knowledge/articles/${articleId}/like`);
  }

  /**
   * 收藏文章
   */
  static async bookmarkArticle(articleId: string): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/bookmark`);
  }

  /**
   * 取消收藏
   */
  static async unbookmarkArticle(articleId: string): Promise<void> {
    return httpClient.delete(
      `/api/v1/knowledge/articles/${articleId}/bookmark`
    );
  }

  /**
   * 分享文章
   */
  static async shareArticle(articleId: string, platform: string): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/share`, {
      platform,
    });
  }

  /**
   * 记录浏览
   */
  static async recordView(articleId: string): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/view`);
  }

  // ==================== 搜索和推荐 ====================

  /**
   * 搜索文章
   */
  static async search(
    request: KnowledgeSearchRequest
  ): Promise<KnowledgeSearchResult> {
    return httpClient.post('/api/v1/knowledge/search', request);
  }

  /**
   * 获取推荐文章
   */
  static async getRecommendations(
    articleId?: string,
    limit = 5
  ): Promise<ArticleRecommendation[]> {
    return httpClient.get('/api/v1/knowledge/recommendations', {
      articleId,
      limit,
    });
  }

  /**
   * 获取热门文章
   */
  static async getPopularArticles(
    params?: {
      categoryId?: string;
      period?: 'day' | 'week' | 'month';
      limit?: number;
    }
  ): Promise<KnowledgeArticle[]> {
    return httpClient.get('/api/v1/knowledge/popular', params);
  }

  /**
   * 获取最新文章
   */
  static async getRecentArticles(
    params?: {
      categoryId?: string;
      limit?: number;
    }
  ): Promise<KnowledgeArticle[]> {
    return httpClient.get('/api/v1/knowledge/recent', params);
  }

  // ==================== 富文本编辑器 ====================

  /**
   * 上传图片
   */
  static async uploadImage(file: File): Promise<ImageUploadResult> {
    const formData = new FormData();
    formData.append('image', file);
    return httpClient.post('/api/v1/knowledge/upload/image', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  }

  /**
   * 自动保存草稿
   */
  static async autoSaveDraft(
    articleId: string,
    content: string
  ): Promise<void> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/autosave`, {
      content,
    });
  }

  // ==================== 审核 ====================

  /**
   * 提交审核
   */
  static async submitForReview(
    articleId: string,
    reviewerId?: number
  ): Promise<KnowledgeArticle> {
    return httpClient.post(`/api/v1/knowledge/articles/${articleId}/review`, {
      reviewerId,
    });
  }

  /**
   * 审核文章
   */
  static async reviewArticle(
    articleId: string,
    request: ReviewArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.post(
      `/api/v1/knowledge/articles/${articleId}/review/decision`,
      request
    );
  }

  // ==================== 统计和分析 ====================

  /**
   * 获取知识库统计
   */
  static async getStats(): Promise<KnowledgeBaseStats> {
    return httpClient.get('/api/v1/knowledge/stats');
  }

  /**
   * 获取文章分析
   */
  static async getArticleAnalytics(
    articleId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<ArticleAnalytics> {
    return httpClient.get(
      `/api/v1/knowledge/articles/${articleId}/analytics`,
      params
    );
  }

  /**
   * 导出知识库
   */
  static async exportKnowledgeBase(params: {
    format: 'pdf' | 'html' | 'markdown';
    categoryId?: string;
    includeComments?: boolean;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: '/api/v1/knowledge/export',
      data: params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default KnowledgeBaseApi;
export const KnowledgeBaseAPI = KnowledgeBaseApi;

