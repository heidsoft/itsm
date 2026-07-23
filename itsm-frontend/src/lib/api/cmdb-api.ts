/**
 * CMDB API 服务 - 统一使用 /api/v1/configuration-items
 *
 * 注意：云资源/云账号/云服务/发现/对账等子资源在后端被挂在
 * `/api/v1/cmdb/*` 下（不是 `/api/v1/configuration-items/*`），
 * 所以这些子资源需要走 CMDB_BASE。CI 本体、CI 类型、关系仍走
 * `${BASE}`（与后端 /configuration-items 一致）。
 */

import { httpClient } from './http-client';
import type { CIType, CloudService, CloudAccount, CloudResource, ConfigurationItem } from '@/types/biz/cmdb';
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
  offset?: number;
  limit?: number;
	page?: number;
	size?: number;
}

export interface GetCIListResponse {
	items: ConfigurationItem[];
  total: number;
}

const BASE = '/api/v1/configuration-items';
// CI 资源以外的子模块（云资源、云服务、云账号、发现、对账）后端挂在 `/api/v1/cmdb/*` 下
const CMDB_BASE = '/api/v1/cmdb';

export class CMDBApi {
  // ==================== CI CRUD ====================

  static async getCIs(query?: GetCIListRequest): Promise<GetCIListResponse> {
	const { offset, limit, ...filters } = query ?? {};
	const size = filters.size ?? limit;
	const page = filters.page ?? (size && offset !== undefined ? Math.floor(offset / size) + 1 : undefined);
	return httpClient.get(BASE, { ...filters, page, size });
  }

  static async getCI(id: string | number): Promise<ConfigurationItem> {
    return httpClient.get(`${BASE}/${id}`);
  }

  static async createCI(request: CreateCIRequest): Promise<ConfigurationItem> {
    return httpClient.post(BASE, request);
  }

  static async updateCI(id: string | number, request: Partial<CreateCIRequest> & Record<string, any>): Promise<ConfigurationItem> {
    return httpClient.put(`${BASE}/${id}`, request);
  }

  static async deleteCI(id: string | number): Promise<void> {
    return httpClient.delete(`${BASE}/${id}`);
  }

  // ==================== Stats & Types ====================

  static async getCMDBStats(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/stats`, params);
  }

  static async getCITypes(): Promise<CIType[]> {
	const response = await httpClient.get<CIType[] | { items: CIType[] }>(`${BASE}/types`);
	return Array.isArray(response) ? response : response.items ?? [];
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
    isActive?: boolean;
  }): Promise<CIType> {
    return httpClient.post(`${BASE}/types`, data);
  }

  static async updateCITypes(id: number, data: {
    name: string;
    description?: string;
    icon?: string;
    color?: string;
    attributeSchema?: string;
    isActive?: boolean;
  }): Promise<CIType> {
    return httpClient.put(`${BASE}/types/${id}`, data);
  }

  static async deleteCITypes(id: number): Promise<void> {
    return httpClient.delete(`${BASE}/types/${id}`);
  }

  // ==================== Topology & Impact ====================

  static async getCITopology(id: number, depth = 3): Promise<TopologyGraph> {
    return httpClient.get(`${BASE}/${id}/topology`, { depth });
  }

  static async getCIImpactAnalysis(id: number): Promise<ImpactAnalysisResponse> {
    return httpClient.get(`${BASE}/${id}/impact-analysis`);
  }

  static async analyzeImpact(request: { ciId: string; analysisType?: string; maxDepth?: number }): Promise<ImpactAnalysisResponse> {
	return httpClient.get(`${BASE}/${request.ciId}/impact-analysis`, { maxDepth: request.maxDepth });
  }

  static async getCIChangeHistory(id: number, params?: { page?: number; pageSize?: number }): Promise<{ items?: Array<Record<string, unknown>>; data?: Array<Record<string, unknown>>; total?: number }> {
    return httpClient.get(`${BASE}/${id}/change-history`, params);
  }

  // ==================== Relationships ====================

  static async createCIRelationship(data: {
    parentId: number;
    childId: number;
    type: string;
    description?: string;
  }): Promise<CIRelationship> {
    return httpClient.post(`${BASE}/relationships`, data);
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

  static async getCIRelationships(ciId: string | number, params?: {
    direction?: 'incoming' | 'outgoing' | 'both';
    types?: string[];
  }): Promise<CIRelationship[]> {
	return httpClient.get(`${BASE}/${ciId}/relationships`, params);
  }

  static async deleteRelationship(id: string): Promise<void> {
    return httpClient.delete(`${BASE}/relationships/${id}`);
  }

  // ==================== Reconciliation ====================

  static async getReconciliationResults(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.get(`${CMDB_BASE}/reconciliation`, params);
  }

  // ==================== Cloud ====================

  static async getCloudServices(provider?: string): Promise<CloudService[]> {
    return httpClient.get(`${CMDB_BASE}/cloud-services`, provider ? { provider } : undefined);
  }

  static async createCloudService(data: Record<string, unknown>): Promise<CloudService> {
    return httpClient.post(`${CMDB_BASE}/cloud-services`, data);
  }

  static async updateCloudService(id: string | number, data: Record<string, unknown>): Promise<CloudService> {
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

	static async updateCloudAccount(id: string | number, data: Record<string, unknown>): Promise<CloudAccount> {
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
