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
