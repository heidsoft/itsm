/**
 * CMDB 资产管理类型定义
 */

import type { CIStatus } from '@/constants/cmdb';

// 配置项实体
export interface ConfigurationItem {
  id: number;
  name: string;
  description: string;
  type: string; // 冗余字段或分类
  status: CIStatus;
  environment?: string;
  criticality?: string;
  assetTag?: string;
  location?: string;
  serialNumber?: string;
  model?: string;
  vendor?: string;
  ciTypeId: number;
  tenantId: number;
  assignedTo?: string;
  ownedBy?: string;
  discoverySource?: string;
  source?: string;
  cloudProvider?: string;
  cloudAccountId?: string;
  cloudRegion?: string;
  cloudZone?: string;
  cloudResourceId?: string;
  cloudResourceType?: string;
  cloudResourceRefId?: number;
  cloudMetadata?: Record<string, any>;
  cloudTags?: Record<string, any>;
  cloudMetrics?: Record<string, any>;
  cloudSyncTime?: string;
  cloudSyncStatus?: string;
  attributes?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

// CI 类型实体
export interface CIType {
  id: number;
  name: string;
  description: string;
  icon?: string;
  color?: string;
  attributeSchema?: string;
  isActive: boolean;
  tenantId: number;
}

export interface CloudService {
  id: number;
  parentId?: number;
  provider: string;
  category?: string;
  serviceCode: string;
  serviceName: string;
  resourceTypeCode: string;
  resourceTypeName: string;
  apiVersion?: string;
  attributeSchema?: Record<string, any>;
  isSystem?: boolean;
  isActive: boolean;
  tenantId: number;
}

export interface CloudAccount {
  id: number;
  provider: string;
  accountId: string;
  accountName: string;
  credentialRef?: string;
  regionWhitelist?: string[];
  isActive: boolean;
  tenantId: number;
}

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
  metadata?: Record<string, any>;
  firstSeenAt?: string;
  lastSeenAt?: string;
  lifecycleState?: string;
  tenantId: number;
}

export interface RelationshipType {
  id: number;
  name: string;
  directional: boolean;
  reverseName?: string;
  description?: string;
  tenantId: number;
}

export interface DiscoverySource {
  id: string;
  name: string;
  sourceType: string;
  provider?: string;
  isActive: boolean;
  description?: string;
  tenantId: number;
}

export interface DiscoveryJob {
  id: number;
  sourceId: string;
  status: string;
  startedAt?: string;
  finishedAt?: string;
  summary?: Record<string, any>;
  tenantId: number;
}

export interface DiscoveryResult {
  id: number;
  jobId: number;
  ciId?: number;
  action: string;
  resourceType?: string;
  resourceId?: string;
  diff?: Record<string, any>;
  status: string;
  tenantId: number;
}

// CI 关系实体
export interface CIRelationship {
  id: number;
  sourceCiId: number;
  targetCiId: number;
  relationshipTypeId: number;
  description?: string;
  tenantId: number;
}

// 统计信息
export interface CMDBStats {
  totalCount: number;
  activeCount: number;
  inactiveCount: number;
  maintenanceCount: number;
  typeDistribution: Record<string, number>;
}

export interface ReconciliationSummary {
  resourceTotal: number;
  boundResourceCount: number;
  unboundResourceCount: number;
  orphanCICount: number;
  unlinkedCICount: number;
}

export interface ReconciliationResponse {
  summary: ReconciliationSummary;
  unboundResources: CloudResource[];
  orphanCIs: ConfigurationItem[];
  unlinkedCIs: ConfigurationItem[];
}

// 列表响应
export interface CIListResponse {
  items: ConfigurationItem[];
  total: number;
  page: number;
  size: number;
}

// 查询参数
export interface CIQuery {
  page?: number;
  pageSize?: number;
  status?: string;
  ciTypeId?: number;
  search?: string;
}
