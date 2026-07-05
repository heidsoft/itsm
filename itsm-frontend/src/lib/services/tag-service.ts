import { httpClient } from '@/lib/api/http-client';

export interface Tag {
  id: number;
  name: string;
  code: string;
  description?: string;
  color?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateTagRequest {
  name: string;
  code: string;
  description?: string;
  color?: string;
}

class TagService {
  private readonly baseUrl = '/api/v1/tags';

  async listTags(): Promise<Tag[]> {
    return httpClient.get<Tag[]>(this.baseUrl);
  }

  async createTag(data: CreateTagRequest): Promise<Tag> {
    return httpClient.post<Tag>(this.baseUrl, data);
  }

  async updateTag(id: number, data: Partial<CreateTagRequest>): Promise<Tag> {
    return httpClient.put<Tag>(`${this.baseUrl}/${id}`, data);
  }

  async deleteTag(id: number): Promise<void> {
    return httpClient.delete(`${this.baseUrl}/${id}`);
  }

  async bindTag(tagId: number, entityType: string, entityId: number): Promise<void> {
    return httpClient.post<void>(`${this.baseUrl}/bind`, {
      tagId: tagId,
      entityType: entityType,
      entityId: entityId,
    });
  }
}

export const tagService = new TagService();
