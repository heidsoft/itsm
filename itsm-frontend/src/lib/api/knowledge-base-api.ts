/**
 * 知识库 API 服务
 *
 * 端点路径规范：
 * - 所有路径统一为 /api/v1/knowledge/articles 风格
 * - 分类/标签/搜索/推荐/统计挂在 /api/v1/knowledge/* 命名空间下
 * - 历史 /api/v1/knowledge-articles 由后端 router 兼容转发，但前端不再使用
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
  KnowledgeBaseStats,
  ArticleAnalytics,
  CreateArticleRequest,
  UpdateArticleRequest,
  PublishArticleRequest,
  ReviewArticleRequest,
  ArticleQuery,
  BatchArticleOperation,
  ImageUploadResult,
  KnowledgeReviewRecord,
} from '@/types/knowledge-base';

// 统一路径前缀：文章路径、命名空间路径分离，避免散落的字符串拼接
const ARTICLES_PREFIX = '/api/v1/knowledge/articles';
const KNOWLEDGE_PREFIX = '/api/v1/knowledge';

export class KnowledgeBaseApi {
  // ==================== 文章管理 ====================

  static async getArticles(query?: ArticleQuery): Promise<{
    items: KnowledgeArticle[];
    total: number;
  }> {
    return httpClient.get(`${ARTICLES_PREFIX}`, query);
  }

  static async getArticle(id: string): Promise<KnowledgeArticle> {
    return httpClient.get(`${ARTICLES_PREFIX}/${id}`);
  }

  static async getArticleBySlug(slug: string): Promise<KnowledgeArticle> {
    return httpClient.get(`${ARTICLES_PREFIX}/slug/${slug}`);
  }

  static async createArticle(request: CreateArticleRequest): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}`, request);
  }

  static async updateArticle(id: string, request: UpdateArticleRequest): Promise<KnowledgeArticle> {
    return httpClient.put(`${ARTICLES_PREFIX}/${id}`, request);
  }

  static async deleteArticle(id: string): Promise<void> {
    return httpClient.delete(`${ARTICLES_PREFIX}/${id}`);
  }

  static async publishArticle(
    id: number | string,
    request?: PublishArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/publish`, request);
  }

  static async unpublishArticle(id: number | string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/unpublish`);
  }

  static async cloneArticle(id: string, title: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/clone`, {
      title,
    });
  }

  static async batchOperation(
    operation: BatchArticleOperation
  ): Promise<{ success: number; failed: number }> {
    return httpClient.post(`${ARTICLES_PREFIX}/batch`, operation);
  }

  // ==================== 评审状态机 ====================
  //
  // 状态取值与后端 ent schema KnowledgeArticle.review_status 枚举严格对应：
  //   draft 草稿 / under_review 评审中 / approved 已通过 / rejected 已驳回
  //
  // 注意区分两个概念：
  //   reviewStatus  是「谁批准的」过程事实（评审链）
  //   isPublished   是「是否对外可见」的布尔投影（检索面）
  // 后端在 approve 时同步把 isPublished 置真、reject 时置假，
  // 但两者语义不同，前端不要拿 isPublished 反推评审状态。

  /** 作者提交送审 */
  static async submitReview(id: number | string, comment?: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/review/submit`, { comment });
  }

  /** 评审人通过 */
  static async approveReview(id: number | string, comment?: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/review/approve`, { comment });
  }

  /** 评审人驳回（comment 必填，后端会拒绝空意见） */
  static async rejectReview(id: number | string, comment: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/review/reject`, { comment });
  }

  /** 作者撤回送审 */
  static async withdrawReview(id: number | string, comment?: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/review/withdraw`, { comment });
  }

  /** 已通过内容被修改后重新送审 */
  static async republishReview(id: number | string, comment?: string): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${id}/review/republish`, { comment });
  }

  /** 取文章的评审历史（时间倒序） */
  static async getReviewHistory(
    id: number | string,
    params?: { page?: number; pageSize?: number }
  ): Promise<{
    items: KnowledgeReviewRecord[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  }> {
    return httpClient.get(`${ARTICLES_PREFIX}/${id}/reviews`, params);
  }

  // ==================== 版本控制 ====================

  static async getArticleVersions(articleId: string): Promise<ArticleVersion[]> {
    return httpClient.get(`${ARTICLES_PREFIX}/${articleId}/versions`);
  }

  static async restoreVersion(articleId: string, version: number): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/versions/${version}/restore`);
  }

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
    return httpClient.get(`${ARTICLES_PREFIX}/${articleId}/versions/compare`, {
      from: fromVersion,
      to: toVersion,
    });
  }

  // ==================== 分类和标签 ====================

  static async getCategories(): Promise<KnowledgeCategory[]> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/categories`);
  }

  static async createCategory(
    category: Omit<KnowledgeCategory, 'id' | 'level' | 'articleCount' | 'createdAt' | 'updatedAt'>
  ): Promise<KnowledgeCategory> {
    return httpClient.post(`${KNOWLEDGE_PREFIX}/categories`, category);
  }

  static async updateCategory(
    id: string,
    category: Partial<KnowledgeCategory>
  ): Promise<KnowledgeCategory> {
    return httpClient.put(`${KNOWLEDGE_PREFIX}/categories/${id}`, category);
  }

  static async deleteCategory(id: string): Promise<void> {
    return httpClient.delete(`${KNOWLEDGE_PREFIX}/categories/${id}`);
  }

  static async getTags(): Promise<KnowledgeTag[]> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/tags`);
  }

  static async createTag(
    tag: Omit<KnowledgeTag, 'id' | 'articleCount' | 'createdAt'>
  ): Promise<KnowledgeTag> {
    return httpClient.post(`${KNOWLEDGE_PREFIX}/tags`, tag);
  }

  // ==================== 评论和反馈 ====================

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
    return httpClient.get(`${ARTICLES_PREFIX}/${articleId}/comments`, params);
  }

  static async addComment(
    articleId: string,
    content: string,
    parentId?: string
  ): Promise<ArticleComment> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/comments`, {
      content,
      parentId,
    });
  }

  static async deleteComment(commentId: string): Promise<void> {
    return httpClient.delete(`${ARTICLES_PREFIX}/comments/${commentId}`);
  }

  static async markCommentHelpful(commentId: string, helpful: boolean): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/comments/${commentId}/helpful`, {
      helpful,
    });
  }

  static async submitFeedback(
    articleId: string,
    feedback: Omit<ArticleFeedback, 'id' | 'userId' | 'userName' | 'createdAt'>
  ): Promise<ArticleFeedback> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/feedback`, feedback);
  }

  // ==================== 互动 ====================

  static async likeArticle(articleId: string): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/like`);
  }

  static async unlikeArticle(articleId: string): Promise<void> {
    return httpClient.delete(`${ARTICLES_PREFIX}/${articleId}/like`);
  }

  static async bookmarkArticle(articleId: string): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/bookmark`);
  }

  static async unbookmarkArticle(articleId: string): Promise<void> {
    return httpClient.delete(`${ARTICLES_PREFIX}/${articleId}/bookmark`);
  }

  static async shareArticle(articleId: string, platform: string): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/share`, {
      platform,
    });
  }

  static async recordView(articleId: string): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/view`);
  }

  // ==================== 搜索和推荐 ====================

  static async search(request: KnowledgeSearchRequest): Promise<KnowledgeSearchResult> {
    return httpClient.post(`${KNOWLEDGE_PREFIX}/search`, request);
  }

  static async getRecommendations(articleId?: string, limit = 5): Promise<ArticleRecommendation[]> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/recommendations`, {
      articleId,
      limit,
    });
  }

  static async getPopularArticles(params?: {
    categoryId?: string;
    period?: 'day' | 'week' | 'month';
    limit?: number;
  }): Promise<KnowledgeArticle[]> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/popular`, params);
  }

  static async getRecentArticles(params?: {
    categoryId?: string;
    limit?: number;
  }): Promise<KnowledgeArticle[]> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/recent`, params);
  }

  // ==================== 富文本编辑器 ====================

  static async uploadImage(file: File): Promise<ImageUploadResult> {
    const formData = new FormData();
    formData.append('image', file);
    return httpClient.post(`${ARTICLES_PREFIX}/upload/image`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  }

  static async autoSaveDraft(articleId: string, content: string): Promise<void> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/autosave`, {
      content,
    });
  }

  // ==================== 审核 ====================

  static async submitForReview(articleId: string, reviewerId?: number): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/review`, {
      reviewerId,
    });
  }

  static async reviewArticle(
    articleId: string,
    request: ReviewArticleRequest
  ): Promise<KnowledgeArticle> {
    return httpClient.post(`${ARTICLES_PREFIX}/${articleId}/review/decision`, request);
  }

  // ==================== 统计和分析 ====================

  static async getStats(): Promise<KnowledgeBaseStats> {
    return httpClient.get(`${KNOWLEDGE_PREFIX}/stats`);
  }

  static async getArticleAnalytics(
    articleId: string,
    params?: {
      startDate?: string;
      endDate?: string;
    }
  ): Promise<ArticleAnalytics> {
    return httpClient.get(`${ARTICLES_PREFIX}/${articleId}/analytics`, params);
  }

  static async exportKnowledgeBase(params: {
    format: 'pdf' | 'html' | 'markdown';
    categoryId?: string;
    includeComments?: boolean;
  }): Promise<Blob> {
    const response = await httpClient.request({
      method: 'POST',
      url: `${ARTICLES_PREFIX}/export`,
      data: params,
      responseType: 'blob',
    });
    return response as Blob;
  }
}

export default KnowledgeBaseApi;
export const KnowledgeBaseAPI = KnowledgeBaseApi;
