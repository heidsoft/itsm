import { httpClient } from './http-client';
import {
  ApiResponse,
  Tenant,
  TenantListResponse,
  CreateTenantRequest,
  UpdateTenantRequest,
  GetTenantsParams,
} from './api-config';

export class TenantAPI {
  // 获取租户列表
  static async getTenants(params?: GetTenantsParams): Promise<TenantListResponse> {
    const response = await httpClient.get<TenantListResponse>('/api/tenants', params);
    return response.data;
  }

  // 获取单个租户
  static async getTenant(id: number): Promise<Tenant> {
    const response = await httpClient.get<Tenant>(`/api/tenants/${id}`);
    return response.data;
  }

  // 创建租户
  static async createTenant(data: CreateTenantRequest): Promise<Tenant> {
    const response = await httpClient.post<Tenant>('/api/tenants', data);
    return response.data;
  }

  // 更新租户
  static async updateTenant(id: number, data: UpdateTenantRequest): Promise<Tenant> {
    const response = await httpClient.put<Tenant>(`/api/tenants/${id}`, data);
    return response.data;
  }

  // 删除租户
  static async deleteTenant(id: number): Promise<void> {
    await httpClient.delete(`/api/tenants/${id}`);
  }

  // 获取当前用户的租户信息
  static async getCurrentTenant(): Promise<Tenant> {
    const response = await httpClient.get<Tenant>('/api/tenants/current');
    return response.data;
  }

  // 切换租户（如果用户属于多个租户）
  static async switchTenant(tenantId: number): Promise<void> {
    await httpClient.post('/api/tenants/switch', { tenant_id: tenantId });
  }
}