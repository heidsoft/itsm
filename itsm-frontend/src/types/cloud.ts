/**
 * Cloud API Types - 对齐后端 dto/cloud_dto.go
 * 后端使用 camelCase，前端直接使用
 */

// ===================================
// Cloud Account Types
// ===================================

export interface CloudAccount {
  id: number;
  provider: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  accountId: string;
  accountName: string;
  credentialRef?: string;
  regionWhitelist?: string[];
  isActive: boolean;
  tenantId?: number;
  lifecycleStatus?: 'draft' | 'online' | 'maintenance' | 'offline' | 'scrapped';
  effectiveAt?: string;
  expireAt?: string;
  type?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCloudAccountRequest {
  provider: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  accountId: string;
  accountName: string;
  credentialRef?: string;
  regionWhitelist?: string[];
  isActive?: boolean;
}

export interface UpdateCloudAccountRequest {
  accountName?: string;
  credentialRef?: string;
  regionWhitelist?: string[];
  isActive?: boolean;
}

export interface ListCloudAccountsRequest {
  page?: number;
  pageSize?: number;
  provider?: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  isActive?: boolean;
  search?: string;
}

export interface CloudAccountListResponse {
  cloudAccounts: CloudAccount[];
  total: number;
  page: number;
  pageSize: number;
}

// ===================================
// Cloud Service Types
// ===================================

export interface CloudService {
  id: number;
  parentId?: number;
  provider: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  category?: string;
  serviceCode: string;
  serviceName: string;
  resourceTypeCode: string;
  resourceTypeName: string;
  apiVersion?: string;
  attributeSchema?: Record<string, unknown>;
  isSystem: boolean;
  isActive: boolean;
  tenantId?: number;
  lifecycleStatus?: 'draft' | 'online' | 'maintenance' | 'offline' | 'scrapped';
  effectiveAt?: string;
  expireAt?: string;
  type?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCloudServiceRequest {
  provider: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  category?: string;
  serviceCode: string;
  serviceName: string;
  resourceTypeCode: string;
  resourceTypeName: string;
  apiVersion?: string;
  attributeSchema?: Record<string, unknown>;
  isSystem?: boolean;
  isActive?: boolean;
  parentId?: number;
}

export interface UpdateCloudServiceRequest {
  category?: string;
  serviceName?: string;
  resourceTypeName?: string;
  apiVersion?: string;
  attributeSchema?: Record<string, unknown>;
  isActive?: boolean;
}

export interface ListCloudServicesRequest {
  page?: number;
  pageSize?: number;
  provider?: 'aliyun' | 'tencent' | 'huawei' | 'aws' | 'azure' | 'onprem';
  category?: string;
  isSystem?: boolean;
  isActive?: boolean;
  search?: string;
  parentId?: number;
}

export interface CloudServiceListResponse {
  cloudServices: CloudService[];
  total: number;
  page: number;
  pageSize: number;
}

// ===================================
// Cloud Resource Types
// ===================================

export interface CloudResource {
  id: number;
  cloudAccountId: number;
  serviceId: number;
  resourceId: string;
  resourceName?: string;
  region?: string;
  zone?: string;
  status?: string;
  tags?: Record<string, string>;
  metadata?: Record<string, unknown>;
  lifecycleState?: string;
  tenantId?: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCloudResourceRequest {
  cloudAccountId: number;
  serviceId: number;
  resourceId: string;
  resourceName?: string;
  region?: string;
  zone?: string;
  status?: string;
  tags?: Record<string, string>;
  metadata?: Record<string, unknown>;
  lifecycleState?: string;
}

export interface UpdateCloudResourceRequest {
  resourceName?: string;
  region?: string;
  zone?: string;
  status?: string;
  tags?: Record<string, string>;
  metadata?: Record<string, unknown>;
  lifecycleState?: string;
}

export interface ListCloudResourcesRequest {
  page?: number;
  pageSize?: number;
  cloudAccountId?: number;
  serviceId?: number;
  region?: string;
  status?: string;
  search?: string;
}

export interface CloudResourceListResponse {
  cloudResources: CloudResource[];
  total: number;
  page: number;
  pageSize: number;
}
