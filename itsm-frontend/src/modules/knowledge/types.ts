/**
 * 知识库类型定义
 */

export interface KnowledgeArticle {
    id: number;
    title: string;
    content: string;
    category: string;
    tags: string[];
    status: string;
    author: string;
    views?: number;
    tenant_id: number;
    created_at: string;
    updated_at: string;
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
