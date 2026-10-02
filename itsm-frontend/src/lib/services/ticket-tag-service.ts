import { httpClient } from '@/lib/api/http-client';

export interface TicketTag {
  id: number;
  name: string;
  color: string;
  description?: string;
  isActive: boolean;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// 后端 data 信封固定为 { items, total }，不存在第二套列表键。
export interface ListTagsResponse {
  items: TicketTag[];
  total: number;
}

// 可写字段。tenantId 只能由后端从认证上下文取，不接受客户端自报。
export interface TicketTagInput {
  name: string;
  color?: string;
  description?: string;
  isActive?: boolean;
}

export interface ListTagsParams {
  page?: number;
  pageSize?: number;
  isActive?: boolean;
}

class TicketTagService {
  private readonly baseUrl = '/api/v1/ticket-tags';

  async listTags(params: ListTagsParams = {}): Promise<ListTagsResponse> {
    return httpClient.get<ListTagsResponse>(this.baseUrl, params);
  }

  async getTag(id: number): Promise<TicketTag> {
    return httpClient.get<TicketTag>(`${this.baseUrl}/${id}`);
  }

  async createTag(data: TicketTagInput): Promise<TicketTag> {
    return httpClient.post<TicketTag>(this.baseUrl, data);
  }

  async updateTag(id: number, data: Partial<TicketTagInput>): Promise<TicketTag> {
    return httpClient.put<TicketTag>(`${this.baseUrl}/${id}`, data);
  }

  async deleteTag(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }
}

export const ticketTagService = new TicketTagService();
