/**
 * 知识库类型定义
 */

export interface KnowledgeArticle {
  id: number | string;
  title: string;
  content: string;
  summary?: string;
  category: string;
  categoryId?: number;
  categoryName?: string;
  tags: string[];
  status: string;
  author: string;
  authorId?: number;
  authorName?: string;
  views?: number;
  helpfulCount?: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
  submittedAt?: string;
  // 来源追溯（2026-10-22 沉淀闭环）：文章由哪个问题沉淀而来；undefined=手工创建。
  // 后端 knowledge_articles.source_problem_id（裸 ID 引用，知识是独立资产）。
  // 详情页据此展示「来源问题」并回链问题详情——与问题详情页「知识沉淀」Tab 构成双向闭环。
  sourceProblemId?: number;
  // 评审阶段（权威字段，2026-10-22 发布门禁）：后端 review_status 列。
  // isPublished 只是它的可见性投影；发布按钮必须据此判断，否则会把
  // 未过评审的草稿当成可发布内容（后端 /publish 已返回 409/4090）。
  // 老数据缺失时按 isPublished 推断：已发布≈approved，其余≈draft。
  reviewStatus?: 'draft' | 'under_review' | 'approved' | 'rejected';
}

export interface ArticleQuery {
  page?: number;
  pageSize?: number;
  category?: string;
  search?: string;
  status?: string;
}

export interface ArticleListResponse {
  articles?: KnowledgeArticle[];
  items?: KnowledgeArticle[];
  total: number;
  page?: number;
  size?: number;
  pageSize?: number;
}

export type KnowledgeArticleStatus = 'draft' | 'pending_review' | 'approved' | 'rejected' | 'archived';

export interface ReviewArticleRequest {
  action: 'approve' | 'reject';
  comment?: string;
}

// Re-export ArticleStatus for compatibility with knowledge-base.ts
export const ArticleStatus = {
  DRAFT: 'draft',
  UNDER_REVIEW: 'under_review',
  PUBLISHED: 'published',
  ARCHIVED: 'archived',
} as const;

export type ArticleStatus = typeof ArticleStatus[keyof typeof ArticleStatus];
