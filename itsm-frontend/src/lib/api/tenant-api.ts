import { httpClient } from './http-client';
import {
  Tenant,
  TenantListResponse,
  CreateTenantRequest,
  UpdateTenantRequest,
  GetTenantsParams,
} from './api-config';

export class TenantAPI {
  // 获取租户列表
  static async getTenants(params?: GetTenantsParams): Promise<TenantListResponse> {
    return httpClient.get<TenantListResponse>('/api/tenants', params);
  }

  // 获取单个租户
  static async getTenant(id: number): Promise<Tenant> {
    return httpClient.get<Tenant>(`/api/tenants/${id}`);
  }

  // 创建租户
  static async createTenant(data: CreateTenantRequest): Promise<Tenant> {
    return httpClient.post<Tenant>('/api/tenants', data);
  }

  // 更新租户
  static async updateTenant(id: number, data: UpdateTenantRequest): Promise<Tenant> {
    return httpClient.put<Tenant>(`/api/tenants/${id}`, data);
  }

  // 删除租户
  static async deleteTenant(id: number): Promise<void> {
    return httpClient.delete<void>(`/api/tenants/${id}`);
  }

  // 获取当前用户的租户信息
  static async getCurrentTenant(): Promise<Tenant> {
    return httpClient.get<Tenant>('/api/tenants/current');
  }

  // 切换租户（如果用户属于多个租户）
  static async switchTenant(tenantId: number): Promise<void> {
    return httpClient.post<void>('/api/tenants/switch', { tenant_id: tenantId });
  }
}