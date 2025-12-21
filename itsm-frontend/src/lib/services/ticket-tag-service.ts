import { httpClient } from '@/lib/api/http-client';

export interface TicketTag {
  id: number;
  name: string;
  color: string;
  description?: string;
  is_active: boolean;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface ListTagsResponse {
  tags: TicketTag[];
  total: number;
}

export interface ListTagsParams {
  page?: number;
  page_size?: number;
  is_active?: boolean;
}

class TicketTagService {
  private readonly baseUrl = '/api/v1/ticket-tags';

  async listTags(params: ListTagsParams = {}): Promise<ListTagsResponse> {
    return httpClient.get<ListTagsResponse>(this.baseUrl, params);
  }

  async getTag(id: number): Promise<TicketTag> {
    return httpClient.get<TicketTag>(`${this.baseUrl}/${id}`);
  }

  async createTag(data: Partial<TicketTag>): Promise<TicketTag> {
    return httpClient.post<TicketTag>(this.baseUrl, data);
  }

  async updateTag(id: number, data: Partial<TicketTag>): Promise<TicketTag> {
    return httpClient.put<TicketTag>(`${this.baseUrl}/${id}`, data);
  }

  async deleteTag(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }
}

export const ticketTagService = new TicketTagService();

