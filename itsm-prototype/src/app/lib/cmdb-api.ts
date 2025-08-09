export interface CIItem {
  id: number;
  name: string;
  ci_type_id: number;
  description?: string;
  status: string;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface ListCIsParams {
  tenant_id?: number; // 后端从JWT注入，这里可选
  ci_type_id?: number;
  status?: string;
  limit?: number;
  offset?: number;
}

export interface ListCIsResponse {
  cis: Array<{ id: number; name: string; ci_type_id: number; description?: string; status: string; tenant_id: number; created_at: string; updated_at: string; }>;
  total: number;
}

export interface CreateCIRequest {
  name: string;
  ci_type_id: number;
  description?: string;
  status: 'active' | 'inactive' | 'maintenance';
  attributes?: Record<string, unknown>;
}

export interface UpdateCIRequest {
  name?: string;
  description?: string;
  status?: 'active' | 'inactive' | 'maintenance';
  attributes?: Record<string, unknown>;
}

import { httpClient } from './http-client';

export class CMDBApi {
  static list(params?: ListCIsParams) {
    return httpClient.get<ListCIsResponse>('/api/v1/cmdb/items', params);
  }
  static get(id: number) {
    return httpClient.get<CIItem>(`/api/v1/cmdb/items/${id}`);
  }
  static create(data: CreateCIRequest) {
    return httpClient.post<CIItem>('/api/v1/cmdb/items', data);
  }
  static update(id: number, data: UpdateCIRequest) {
    return httpClient.put<CIItem>(`/api/v1/cmdb/items/${id}`, data);
  }
  static remove(id: number) {
    return httpClient.delete<void>(`/api/v1/cmdb/items/${id}`);
  }
}
