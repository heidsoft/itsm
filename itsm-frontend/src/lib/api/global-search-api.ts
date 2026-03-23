import { httpClient } from './http-client';

export interface SearchResult {
  id: number;
  type: 'ticket' | 'incident' | 'problem' | 'change' | 'knowledge';
  title: string;
  description?: string;
  status?: string;
  ticketNumber?: string;
}

export interface GlobalSearchResponse {
  results: SearchResult[];
  total: number;
}

export async function globalSearch(keyword: string): Promise<GlobalSearchResponse> {
  return httpClient.get<GlobalSearchResponse>('/api/v1/global-search', { keyword });
}
