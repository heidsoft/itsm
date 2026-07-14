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
  assetNumber: string;
  name: string;
  description?: string;
  type?: AssetType;
  category?: string;
  subcategory?: string;
  ciId?: number;
  assignedTo?: number;
  locationId?: number;
  serialNumber?: string;
  model?: string;
  manufacturer?: string;
  vendor?: string;
  purchaseDate?: string;
  purchasePrice?: number;
  warrantyExpiry?: string;
  supportExpiry?: string;
  location?: string;
  department?: string;
  parentAssetId?: number;
  specifications?: Record<string, string>;
  customFields?: Record<string, string>;
  tags?: string[];
}

// 资产响应接口
export interface Asset {
  id: number;
  assetNumber: string;
  name: string;
  description?: string;
  type: AssetType;
  status: AssetStatus;
  category?: string;
  subcategory?: string;
  tenantId: number;
  ciId?: number;
  ciName?: string;
  assignedTo?: number;
  assignedToName?: string;
  locationId?: number;
  serialNumber?: string;
  model?: string;
  manufacturer?: string;
  vendor?: string;
  purchaseDate?: string;
  purchasePrice?: number;
  warrantyExpiry?: string;
  supportExpiry?: string;
  location?: string;
  department?: string;
  parentAssetId?: number;
  parentAssetName?: string;
  specifications?: Record<string, string>;
  customFields?: Record<string, string>;
  tags?: string[];
  createdAt: string;
  updatedAt: string;
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
  inUse: number;
  maintenance: number;
  retired: number;
  disposed: number;
}

// 资产状态更新请求
export interface AssetStatusUpdateRequest {
  status: AssetStatus;
  assignedTo?: number;
  comment?: string;
}

// 资产分配请求
export interface AssetAssignRequest {
  assignedTo: number;
  comment?: string;
}

// 资产退役请求
export interface AssetRetireRequest {
  retireDate: string;
  retireReason: string;
  disposalMethod?: string;
  disposalValue?: number;
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
  licenseType?: LicenseType;
  licenseKey?: string;
  totalQuantity?: number;
  assetId?: number;
  purchaseDate?: string;
  purchasePrice?: number;
  expiryDate?: string;
  supportVendor?: string;
  supportContact?: string;
  renewalCost?: string;
  notes?: string;
  users?: number[];
  tags?: string[];
}

// 许可证响应接口
export interface License {
  id: number;
  licenseKey?: string;
  name: string;
  description?: string;
  vendor?: string;
  licenseType: LicenseType;
  totalQuantity: number;
  usedQuantity: number;
  availableQuantity: number;
  tenantId: number;
  assetId?: number;
  assetName?: string;
  purchaseDate?: string;
  purchasePrice?: number;
  expiryDate?: string;
  supportVendor?: string;
  supportContact?: string;
  renewalCost?: string;
  status: LicenseStatus;
  notes?: string;
  users?: number[];
  userNames?: string[];
  tags?: string[];
  createdAt: string;
  updatedAt: string;
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
  expiringSoon: number;
  depleted: number;
  complianceRate: number;
}

// 许可证分配请求
export interface LicenseAssignRequest {
  userIds: number[];
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
      assignedTo: assignedTo,
    });
  }

  // 获取资产统计
  static async getAssetStats(): Promise<AssetStatsResponse> {
    return httpClient.get<AssetStatsResponse>('/api/v1/assets/stats');
  }

  // 分配资产
  static async assignAsset(id: number, assignedTo: number): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}/assign`, { assignedTo: assignedTo });
  }

  // 退役资产
  static async retireAsset(id: number, reason: string): Promise<Asset> {
    return httpClient.put<Asset>(`/api/v1/assets/${id}/retire`, { retireReason: reason });
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
    return httpClient.put<License>(`/api/v1/licenses/${id}/assign`, { userIds: userIds });
  }
}
