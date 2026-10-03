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
  parentTypeId?: number;
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
  hasCredential?: boolean;
  regionWhitelist?: string[];
  isActive: boolean;
  tenantId: number;
}

export interface CloudResource {
  id: number;
  cloudAccountId: number;
  serviceId: number;
  resourceId: string;
  identityVersion: number;
  provider?: string;
  partition?: string;
  canonicalAccountId?: string;
  resourceScope?: string;
  serviceCode?: string;
  resourceType?: string;
  identityHash?: string;
  sourceId?: string;
  sourceFingerprint?: string;
  missingCount: number;
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
  cloudAccountId?: number;
  serviceCodes?: string[];
  regions?: string[];
  schedule?: string;
  reconcilePolicy: 'manual' | 'discovered_wins' | 'cmdb_wins';
  staleThreshold: number;
  lastSuccessAt?: string;
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
  resourceIdentity?: string;
  identityVersion: number;
  resourceSnapshot?: Record<string, unknown>;
  beforeHash?: string;
  afterHash?: string;
  diff?: Record<string, any>;
  status: string;
  errorCode?: string;
  errorMessage?: string;
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

// CI 变更历史条目 —— 与后端 dto.CIHistoryResponse 逐字段一致
export interface CIHistoryItem {
  id: number;
  ciId: number;
  version: number;
  // 实测写入值：create / update / delete / revert / lifecycle_update；未知值原样显示，不做枚举断言
  operation: string;
  before?: Record<string, unknown>;
  after?: Record<string, unknown>;
  changedFields?: string[];
  operatorId: number;
  operatorName?: string;
  remark?: string;
  lifecycleStatus?: string;
  effectiveAt?: string;
  expireAt?: string;
  type?: string;
  createdAt: string;
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
