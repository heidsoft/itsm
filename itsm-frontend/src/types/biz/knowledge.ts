/**
 * 知识库类型定义
 */

export interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  summary?: string;
  category: string;
  category_id?: number;
  categoryId?: number;
  category_name?: string;
  categoryName?: string;
  tags: string[];
  status: string;
  author: string;
  author_id?: number;
  authorId?: number;
  views?: number;
  helpful_count?: number;
  helpfulCount?: number;
  tenant_id: number;
  tenantId?: number;
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;
}

export interface ArticleQuery {
  page?: number;
  page_size?: number;
  category?: string;
  search?: string;
  status?: string;
}

export interface ArticleListResponse {
  articles: KnowledgeArticle[];
  total: number;
  page: number;
  size: number;
}

export type KnowledgeArticleStatus = 'draft' | 'pending_review' | 'approved' | 'rejected' | 'archived';

export interface ReviewArticleRequest {
  action: 'approve' | 'reject';
  comment?: string;
}
