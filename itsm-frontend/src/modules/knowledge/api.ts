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
}
