import { httpClient } from './http-client';
import type { User } from './user-api';

export interface Group {
  id: number;
  name: string;
  description?: string;
  tenantId: number;
  members?: User[];
  createdAt: string;
  updatedAt: string;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string;
}

export interface GetGroupsParams {
  page?: number;
  pageSize?: number;
  tenantId?: number;
  search?: string;
}

export interface PagedGroupsResponse {
  groups: Group[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPage: number;
  };
}

export interface GroupMembersResponse {
  members: User[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPage: number;
  };
}

export class GroupAPI {
  private static baseURL = '/api/v1/groups';

  static async getGroups(params: GetGroupsParams = {}): Promise<PagedGroupsResponse> {
    return httpClient.get<PagedGroupsResponse>(this.baseURL, params);
  }

  static async getGroup(id: number): Promise<Group> {
    return httpClient.get<Group>(`${this.baseURL}/${id}`);
  }

  static async createGroup(data: CreateGroupRequest): Promise<Group> {
    return httpClient.post<Group>(this.baseURL, data);
  }

  static async updateGroup(id: number, data: UpdateGroupRequest): Promise<Group> {
    return httpClient.put<Group>(`${this.baseURL}/${id}`, data);
  }

  static async deleteGroup(id: number): Promise<void> {
    await httpClient.delete<void>(`${this.baseURL}/${id}`);
  }

  static async getMembers(
    id: number,
    params: { page?: number; pageSize?: number } = {}
  ): Promise<GroupMembersResponse> {
    return httpClient.get<GroupMembersResponse>(`${this.baseURL}/${id}/members`, params);
  }

  static async addMember(id: number, userId: number): Promise<void> {
    await httpClient.post<void>(`${this.baseURL}/${id}/members`, { userId });
  }

  static async removeMember(id: number, userId: number): Promise<void> {
    await httpClient.delete<void>(`${this.baseURL}/${id}/members`, { userId });
  }
}
