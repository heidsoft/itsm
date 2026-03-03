/**
 * 资产管理 API 服务
 */

import { httpClient } from './http-client';

// 资产状态类型
export type AssetStatus = 'available' | 'in-use' | 'maintenance' | 'retired' | 'disposed';

// 资产类型
export type AssetType = 'hardware' | 'software' | 'cloud' | 'license';

// 资产请求接口
export interface AssetRequest {
  asset_number: string;
  name: string;
  description?: string;
  type?: AssetType;
  category?: string;
  subcategory?: string;
  ci_id?: number;
  assigned_to?: number;
  location_id?: number;
  serial_number?: string;
  model?: string;
  manufacturer?: string;
  vendor?: string;
  purchase_date?: string;
  purchase_price?: number;
  warranty_expiry?: string;
  support_expiry?: string;
  location?: string;
  department?: string;
  parent_asset_id?: number;
  specifications?: Record<string, string>;
  custom_fields?: Record<string, string>;
  tags?: string[];
}

// 资产响应接口
export interface Asset {
  id: number;
  asset_number: string;
  name: string;
  description?: string;
  type: AssetType;
  status: AssetStatus;
  category?: string;
  subcategory?: string;
  tenant_id: number;
  ci_id?: number;
  ci_name?: string;
  assigned_to?: number;
  assigned_to_name?: string;
  location_id?: number;
  serial_number?: string;
  model?: string;
  manufacturer?: string;
  vendor?: string;
  purchase_date?: string;
  purchase_price?: number;
  warranty_expiry?: string;
  support_expiry?: string;
  location?: string;
  department?: string;
  parent_asset_id?: number;
  parent_asset_name?: string;
  specifications?: Record<string, string>;
  custom_fields?: Record<string, string>;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

// 资产列表响应
export interface AssetListResponse {
  total: number;
  assets: Asset[];
}

// 资产统计响应
export interface AssetStatsResponse {
  total: number;
  available: number;
  in_use: number;
  maintenance: number;
  retired: number;
  disposed: number;
}

// 资产状态更新请求
export interface AssetStatusUpdateRequest {
  status: AssetStatus;
  assigned_to?: number;
  comment?: string;
}

// 资产分配请求
export interface AssetAssignRequest {
  assigned_to: number;
  comment?: string;
}

// 资产退役请求
export interface AssetRetireRequest {
  retire_date: string;
  retire_reason: string;
  disposal_method?: string;
  disposal_value?: number;
  comment?: string;
}

// 许可证状态类型
export type LicenseStatus = 'active' | 'expired' | 'expiring-soon' | 'depleted';

// 许可证类型
export type LicenseType = 'perpetual' | 'subscription' | 'per-user' | 'per-seat' | 'site';

// 许可证请求接口
export interface LicenseRequest {
  name: string;
  description?: string;
  vendor?: string;
  license_type?: LicenseType;
  total_quantity?: number;
  asset_id?: number;
  purchase_date?: string;
  purchase_price?: number;
  expiry_date?: string;
  support_vendor?: string;
  support_contact?: string;
  renewal_cost?: string;
  notes?: string;
  users?: number[];
  tags?: string[];
}

// 许可证响应接口
export interface License {
  id: number;
  license_key?: string;
  name: string;
  description?: string;
  vendor?: string;
  license_type: LicenseType;
  total_quantity: number;
  used_quantity: number;
  available_quantity: number;
  tenant_id: number;
  asset_id?: number;
  asset_name?: string;
  purchase_date?: string;
  purchase_price?: number;
  expiry_date?: string;
  support_vendor?: string;
  support_contact?: string;
  renewal_cost?: string;
  status: LicenseStatus;
  notes?: string;
  users?: number[];
  user_names?: string[];
  tags?: string[];
  created_at: string;
  updated_at: string;
}

// 许可证列表响应
export interface LicenseListResponse {
  total: number;
  licenses: License[];
}

// 许可证统计响应
export interface LicenseStatsResponse {
  total: number;
  active: number;
  expired: number;
  expiring_soon: number;
  depleted: number;
  compliance_rate: number;
}

// 许可证分配请求
export interface LicenseAssignRequest {
  user_ids: number[];
  comment?: string;
}

// 资产管理API类
export class AssetApi {
  // 获取资产列表
  static async getAssets(params?: {
    page?: number;
    pageSize?: number;
    type?: AssetType;
    status?: AssetStatus;
    category?: string;
  }): Promise<AssetListResponse> {
    return httpClient.get<AssetListResponse>('/api/v1/assets', params);
  }

  // 获取单个资产
  static async getAsset(id: number): Promise<Asset> {
    return httpClient.get<Asset>(`/api/v1/assets/${id}`);
  }

  // 创建资产
  static async createAsset(data: AssetRequest): Promise<Asset> {
    return httpClient.post<Asset>('/api/v1/assets', data);
  }

  // 更新资产
  static async updateAsset(id: number, data: Partial<AssetRequest>): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}`, data);
  }

  // 删除资产
  static async deleteAsset(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/assets/${id}`);
  }

  // 更新资产状态
  static async updateAssetStatus(
    id: number,
    status: AssetStatus,
    assignedTo?: number
  ): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}/status`, {
      status,
      assigned_to: assignedTo,
    });
  }

  // 获取资产统计
  static async getAssetStats(): Promise<AssetStatsResponse> {
    return httpClient.get<AssetStatsResponse>('/api/v1/assets/stats');
  }

  // 分配资产
  static async assignAsset(id: number, assignedTo: number): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}/assign`, { assigned_to: assignedTo });
  }

  // 退役资产
  static async retireAsset(id: number, reason: string): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}/retire`, { retire_reason: reason });
  }

  // 获取许可证列表
  static async getLicenses(params?: {
    page?: number;
    pageSize?: number;
    type?: LicenseType;
    status?: LicenseStatus;
  }): Promise<LicenseListResponse> {
    return httpClient.get<LicenseListResponse>('/api/v1/licenses', params);
  }

  // 获取单个许可证
  static async getLicense(id: number): Promise<License> {
    return httpClient.get<License>(`/api/v1/licenses/${id}`);
  }

  // 创建许可证
  static async createLicense(data: LicenseRequest): Promise<License> {
    return httpClient.post<License>('/api/v1/licenses', data);
  }

  // 更新许可证
  static async updateLicense(id: number, data: Partial<LicenseRequest>): Promise<License> {
    return httpClient.put<License>(`/api/v1/licenses/${id}`, data);
  }

  // 删除许可证
  static async deleteLicense(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/licenses/${id}`);
  }

  // 获取许可证统计
  static async getLicenseStats(): Promise<LicenseStatsResponse> {
    return httpClient.get<LicenseStatsResponse>('/api/v1/licenses/stats');
  }

  // 分配许可证给用户
  static async assignLicenseUsers(id: number, userIds: number[]): Promise<License> {
    return httpClient.put<License>(`/api/v1/licenses/${id}/assign`, { user_ids: userIds });
  }
}
