/**
 * 知识库 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type { KnowledgeArticle, ArticleListResponse, ArticleQuery } from './types';

const BASE_URL = '/api/v1/knowledge-articles';

export class KnowledgeApi {
    /**
     * 获取文章列表
     */
    static async getArticles(query?: ArticleQuery): Promise<ArticleListResponse> {
        return httpClient.get<ArticleListResponse>(BASE_URL, query);
    }

    /**
     * 获取文章详情
     */
    static async getArticle(id: string | number): Promise<KnowledgeArticle> {
        return httpClient.get<KnowledgeArticle>(`${BASE_URL}/${id}`);
    }

    /**
     * 创建文章
     */
    static async createArticle(data: Partial<KnowledgeArticle>): Promise<KnowledgeArticle> {
        return httpClient.post<KnowledgeArticle>(BASE_URL, data);
    }

    /**
     * 更新文章
     */
    static async updateArticle(id: string | number, data: Partial<KnowledgeArticle>): Promise<KnowledgeArticle> {
        return httpClient.put<KnowledgeArticle>(`${BASE_URL}/${id}`, data);
    }

    /**
     * 删除文章
     */
    static async deleteArticle(id: string | number): Promise<void> {
        return httpClient.delete<void>(`${BASE_URL}/${id}`);
    }

    /**
     * 获取分类列表
     */
    static async getCategories(): Promise<string[]> {
        return httpClient.get<string[]>(`${BASE_URL}/categories`);
    }

    /**
     * 搜索文章
     */
    static async searchArticles(keyword: string, limit?: number): Promise<KnowledgeArticle[]> {
        return httpClient.get<KnowledgeArticle[]>(`${BASE_URL}/search`, { keyword, limit });
    }

    /**
     * 提交文章审批
     */
    static async submitForApproval(id: string | number): Promise<KnowledgeArticle> {
        return httpClient.post<KnowledgeArticle>(`${BASE_URL}/${id}/submit`);
    }

    /**
     * 审批文章（通过/拒绝）
     */
    static async approveArticle(id: string | number, action: 'approve' | 'reject', comment?: string): Promise<KnowledgeArticle> {
        return httpClient.post<KnowledgeArticle>(`${BASE_URL}/${id}/approve`, {
            action,
            comment,
        });
    }

    /**
     * 获取文章阅读量
     */
    static async getArticleViews(id: string | number): Promise<{ views: number; last_viewed_at: string }> {
        return httpClient.get(`${BASE_URL}/${id}/views`);
    }

    /**
     * 批量删除文章
     */
    static async batchDelete(ids: number[]): Promise<{ deleted: number }> {
        return httpClient.post(`${BASE_URL}/batch`, { ids });
    }

    /**
     * 导出文章
     */
    static async exportArticles(format: 'pdf' | 'html' | 'markdown', query?: ArticleQuery): Promise<Blob> {
        return httpClient.request({
            method: 'GET',
            url: `${BASE_URL}/export`,
            params: { format, ...query },
            responseType: 'blob',
        }) as Promise<Blob>;
    }
}
