export interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  category: string;
  tags?: string[];
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface ListKnowledgeArticlesParams {
  page?: number;
  page_size?: number;
  category?: string;
  status?: string;
  search?: string;
}

export interface ListKnowledgeArticlesResponse {
  articles: KnowledgeArticle[];
  total: number;
  page: number;
  size: number;
}

export interface CreateKnowledgeArticleRequest {
  title: string;
  content: string;
  category: string;
  tags?: string[];
}

export interface UpdateKnowledgeArticleRequest {
  title?: string;
  content?: string;
  category?: string;
  tags?: string[];
}

import { httpClient } from './http-client';

export class KnowledgeApi {
  static list(params?: ListKnowledgeArticlesParams) {
    return httpClient.get<ListKnowledgeArticlesResponse>('/api/v1/knowledge-articles', params);
  }
  static get(id: number) {
    return httpClient.get<KnowledgeArticle>(`/api/v1/knowledge-articles/${id}`);
  }
  static create(data: CreateKnowledgeArticleRequest) {
    return httpClient.post<KnowledgeArticle>('/api/v1/knowledge-articles', data);
  }
  static update(id: number, data: UpdateKnowledgeArticleRequest) {
    return httpClient.put<KnowledgeArticle>(`/api/v1/knowledge-articles/${id}`, data);
  }
  static remove(id: number) {
    return httpClient.delete<void>(`/api/v1/knowledge-articles/${id}`);
  }
  static categories() {
    return httpClient.get<string[]>('/api/v1/knowledge-articles/categories');
  }
}
