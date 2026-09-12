/**
 * CMDB API 服务 - 统一使用生产路由 /api/v1/cmdb
 *
 * 注意：云资源/云账号/云服务/发现/对账等子资源在后端被挂在
 * `/api/v1/cmdb/*` 下（不是 `/api/v1/configuration-items/*`），
 * 所有 CMDB 资源统一走 `/api/v1/cmdb/*`，避免依赖已弃用的兼容别名。
 */

import { httpClient } from './http-client';
import type {
  CIType,
  CloudService,
  CloudAccount,
  CloudResource,
  ConfigurationItem,
} from '@/types/biz/cmdb';
import type { TopologyGraph, ImpactAnalysisResponse } from './cmdb-relationship';

export interface CIRelationship {
  id: number;
  type: string;
  description?: string;
  parentId: number;
  childId: number;
  createdAt: string;
}

export interface CreateCIRequest {
  name: string;
  ciTypeId: number;
  status: string;
  environment?: string;
  criticality?: string;
  assetTag?: string;
  serialNumber?: string;
  model?: string;
  vendor?: string;
  location?: string;
  assignedTo?: string;
  ownedBy?: string;
  discoverySource?: string;
  source?: string;
  description?: string;
  attributes?: Record<string, unknown>;
  cloudProvider?: string;
  cloudAccountId?: string;
  cloudRegion?: string;
  cloudZone?: string;
  cloudResourceId?: string;
  cloudResourceType?: string;
  cloudSyncStatus?: string;
  cloudResourceRefId?: number;
  cloudMetadata?: Record<string, unknown>;
}

export interface GetCIListRequest {
  ciType?: string;
  ciTypeId?: number;
  search?: string;
  status?: string;
  environment?: string;
  page?: number;
  size?: number;
}

export interface GetCIListResponse {
  items: ConfigurationItem[];
  total: number;
}

export type CMDBCapabilityState = 'disabled' | 'unconfigured' | 'unready' | 'ready';

export interface CMDBRuntimeCapability {
  key: string;
  state: CMDBCapabilityState;
  buildCapability: boolean;
  deploymentReadiness: boolean;
  tenantReadiness: boolean;
  actorPermission: boolean;
  missingRequirements: string[];
}

export interface CMDBCapabilitiesResponse {
  items: CMDBRuntimeCapability[];
}

const CMDB_BASE = '/api/v1/cmdb';
// 必须是字符串字面量：派生模板常量无法被 api-contract 扫描器静态解析。
const CIS_BASE = '/api/v1/cmdb/cis';

export class CMDBApi {
  static async getCapabilities(): Promise<CMDBCapabilitiesResponse> {
    return httpClient.get(`${CMDB_BASE}/capabilities`);
  }

  // ==================== CI CRUD ====================

  static async getCIs(query?: GetCIListRequest): Promise<GetCIListResponse> {
    return httpClient.get(CIS_BASE, query);
  }

  static async getCI(id: string | number): Promise<ConfigurationItem> {
    return httpClient.get(`${CIS_BASE}/${id}`);
  }

  static async createCI(request: CreateCIRequest): Promise<ConfigurationItem> {
    return httpClient.post(CIS_BASE, request);
  }

  static async updateCI(
    id: string | number,
    request: Partial<CreateCIRequest> & Record<string, any>
  ): Promise<ConfigurationItem> {
    return httpClient.put(`${CIS_BASE}/${id}`, request);
  }

  static async deleteCI(id: string | number): Promise<void> {
    return httpClient.delete(`${CIS_BASE}/${id}`);
  }

  // ==================== Stats & Types ====================

  static async getCMDBStats(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.get(`${CIS_BASE}/stats`, params);
  }

  static async getCITypes(): Promise<CIType[]> {
    const all: CIType[] = [];
    const size = 200;
    for (let page = 1; ; page += 1) {
      const response = await httpClient.get<
        CIType[] | { items: CIType[]; total?: number; page?: number; size?: number }
      >(`${CMDB_BASE}/ci-types`, { page, size });
      if (Array.isArray(response)) return response;
      const items = response.items ?? [];
      all.push(...items);
      if (items.length < size || (response.total !== undefined && all.length >= response.total)) {
        return all;
      }
    }
  }

  static async getCMDBTypes(): Promise<CIType[]> {
    return this.getCITypes();
  }

  static async createCITypes(data: {
    name: string;
    description?: string;
    icon?: string;
    color?: string;
    attributeSchema?: string;
    parentTypeId?: number;
    isActive?: boolean;
  }): Promise<CIType> {
    return httpClient.post(`${CMDB_BASE}/ci-types`, data);
  }

  static async updateCITypes(
    id: number,
    data: {
      name: string;
      description?: string;
      icon?: string;
      color?: string;
      attributeSchema?: string;
      parentTypeId?: number;
      clearParent?: boolean;
      isActive?: boolean;
    }
  ): Promise<CIType> {
    return httpClient.put(`${CMDB_BASE}/ci-types/${id}`, data);
  }

  static async deleteCITypes(id: number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/ci-types/${id}`);
  }

  // ==================== Topology & Impact ====================

  static async getCITopology(id: number, depth = 3): Promise<TopologyGraph> {
    return httpClient.get(`${CIS_BASE}/${id}/topology`, { depth });
  }

  static async getCIImpactAnalysis(id: number): Promise<ImpactAnalysisResponse> {
    return httpClient.get(`${CIS_BASE}/${id}/impact-analysis`);
  }

  /**
   * P0-1：本体自描述端点（version/ciTypes/relationshipTypes/enums/aiTools）
   * 单一接口取代散落的硬编码元数据；fail-soft 用于前端词汇表初始化。
   */
  static async getOntology(): Promise<Record<string, unknown>> {
    return httpClient.get(`${CMDB_BASE}/ontology`);
  }

  /**
   * P0-2：关系词表（来自 /cmdb/relationship-types，单一源 13 种 + reverse）
   * 配合 relationship-vocabulary.ts 的 loadRelationshipVocabulary() 做本地缓存。
   */
  static async getRelationshipTypes(): Promise<{
    types: Array<{
      type: string;
      name: string;
      description: string;
      direction: string;
      icon?: string;
      reverse?: string;
    }>;
  }> {
    return httpClient.get(`${CMDB_BASE}/relationship-types`);
  }

  static async analyzeImpact(request: {
    ciId: string;
    analysisType?: string;
    maxDepth?: number;
  }): Promise<ImpactAnalysisResponse> {
    return httpClient.get(`${CIS_BASE}/${request.ciId}/impact-analysis`, {
      maxDepth: request.maxDepth,
    });
  }

  static async getCIChangeHistory(
    id: number,
    params?: { page?: number; pageSize?: number }
  ): Promise<{
    items?: Array<Record<string, unknown>>;
    data?: Array<Record<string, unknown>>;
    total?: number;
  }> {
    return httpClient.get(`${CIS_BASE}/${id}/history`, params);
  }

  // ==================== Relationships ====================

  static async createCIRelationship(data: {
    parentId: number;
    childId: number;
    type: string;
    description?: string;
  }): Promise<CIRelationship> {
    return httpClient.post(`${CMDB_BASE}/relationships`, data);
  }

  static async createRelationship(request: {
    sourceCiId: number;
    targetCiId: number;
    type: string;
    description?: string;
  }): Promise<CIRelationship> {
    return this.createCIRelationship({
      parentId: request.sourceCiId,
      childId: request.targetCiId,
      type: request.type,
      description: request.description,
    });
  }

  static async getCIRelationships(
    ciId: string | number,
    params?: {
      direction?: 'incoming' | 'outgoing' | 'both';
      types?: string[];
    }
  ): Promise<CIRelationship[]> {
    return httpClient.get(`${CIS_BASE}/${ciId}/relationships`, params);
  }

  static async deleteRelationship(id: string): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/relationships/${id}`);
  }

  // ==================== Reconciliation ====================

  static async getReconciliationResults(
    params?: Record<string, unknown>
  ): Promise<Record<string, unknown>> {
    return httpClient.get(`${CMDB_BASE}/reconciliation`, params);
  }

  // ==================== Cloud ====================

  static async getCloudServices(provider?: string): Promise<CloudService[]> {
    return httpClient.get(`${CMDB_BASE}/cloud-services`, provider ? { provider } : undefined);
  }

  static async createCloudService(data: Record<string, unknown>): Promise<CloudService> {
    return httpClient.post(`${CMDB_BASE}/cloud-services`, data);
  }

  static async updateCloudService(
    id: string | number,
    data: Record<string, unknown>
  ): Promise<CloudService> {
    return httpClient.put(`${CMDB_BASE}/cloud-services/${id}`, data);
  }

  static async deleteCloudService(id: string | number): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/cloud-services/${id}`);
  }

  static async getCloudAccounts(): Promise<CloudAccount[]> {
    return httpClient.get(`${CMDB_BASE}/cloud-accounts`);
  }

  static async createCloudAccount(data: Record<string, unknown>): Promise<CloudAccount> {
    return httpClient.post(`${CMDB_BASE}/cloud-accounts`, data);
  }

  static async deleteCloudAccount(id: string): Promise<void> {
    return httpClient.delete(`${CMDB_BASE}/cloud-accounts/${id}`);
  }

  static async updateCloudAccount(
    id: string | number,
    data: Record<string, unknown>
  ): Promise<CloudAccount> {
    return httpClient.put(`${CMDB_BASE}/cloud-accounts/${id}`, data);
  }

  static async getCloudResources(params?: Record<string, unknown>): Promise<CloudResource[]> {
    return httpClient.get(`${CMDB_BASE}/cloud-resources`, params);
  }

  // ==================== Discovery ====================

  static async getDiscoveryRules(): Promise<Array<Record<string, unknown>>> {
    return httpClient.get(`${CMDB_BASE}/discovery/sources`);
  }

  static async getDiscoverySources(): Promise<Array<Record<string, unknown>>> {
    return httpClient.get(`${CMDB_BASE}/discovery/sources`);
  }

  static async getDiscoveryHistory(ruleId?: string): Promise<Array<Record<string, unknown>>> {
    return httpClient.get(`${CMDB_BASE}/discovery/results`, ruleId ? { jobId: ruleId } : undefined);
  }

  static async runDiscoveryRule(ruleId: string): Promise<void> {
    return httpClient.post(`${CMDB_BASE}/discovery/jobs`, { sourceId: ruleId });
  }

  // ==================== Search ====================

  static async searchCIs(query: {
    keyword?: string;
    ciType?: string;
    status?: string;
  }): Promise<{ items: ConfigurationItem[]; total: number }> {
    const result = await this.getCIs(query);
    return {
      items: result.items ?? [],
      total: result.total,
    };
  }

  // ==================== Batch ====================

  static async batchCreateCIs(requests: CreateCIRequest[]): Promise<ConfigurationItem[]> {
    const results: ConfigurationItem[] = [];
    for (const request of requests) {
      try {
        results.push(await this.createCI(request));
      } catch (error) {
        console.error('批量创建CI失败:', error);
      }
    }
    return results;
  }
}

export default CMDBApi;
