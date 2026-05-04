/**
 * CMDB API 服务 - 统一使用 /api/v1/configuration-items
 */

import { httpClient } from './http-client';
import type { CIType, CloudService, CloudAccount } from '@/types/biz/cmdb';

export interface ConfigurationItem {
  id: number;
  name: string;
  ci_type: string;
  ciType?: string;
  status: string;
  environment: string;
  criticality: string;
  asset_tag?: string;
  assetTag?: string;
  serial_number?: string;
  serialNumber?: string;
  location?: string;
  assigned_to?: string;
  assignedTo?: string;
  owned_by?: string;
  ownedBy?: string;
  discovery_source?: string;
  discoverySource?: string;
  attributes?: Record<string, unknown>;
  tenant_id: number;
  tenantId?: number;
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;
}

export interface CIRelationship {
  id: number;
  type: string;
  description?: string;
  parent_id: number;
  child_id: number;
  created_at: string;
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

export interface CITopology {
  ci: ConfigurationItem;
  children: CITopology[];
}

export interface GetCIListRequest {
  ci_type?: string;
  status?: string;
  environment?: string;
  offset?: number;
  limit?: number;
}

export interface GetCIListResponse {
  items?: ConfigurationItem[];
  cis?: ConfigurationItem[];
  total: number;
}

const BASE = '/api/v1/configuration-items';

export class CMDBApi {
  // ==================== CI CRUD ====================

  static async getCIs(query?: GetCIListRequest): Promise<GetCIListResponse> {
    return httpClient.get(BASE, query);
  }

  static async getConfigurationItems(params?: GetCIListRequest): Promise<{ data: ConfigurationItem[] }> {
    return httpClient.get(BASE, params);
  }

  static async getCI(id: string | number): Promise<ConfigurationItem> {
    return httpClient.get(`${BASE}/${id}`);
  }

  static async getConfigurationItem(id: number): Promise<ConfigurationItem> {
    return httpClient.get(`${BASE}/${id}`);
  }

  static async createCI(request: CreateCIRequest): Promise<ConfigurationItem> {
    return httpClient.post(BASE, request);
  }

  static async createConfigurationItem(data: CreateCIRequest): Promise<ConfigurationItem> {
    return httpClient.post(BASE, data);
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
    return httpClient.get(`${BASE}/types`);
  }

  static async getCMDBTypes(): Promise<CIType[]> {
    return this.getCITypes();
  }

  static async createCITypes(data: {
    name: string;
    description?: string;
    icon?: string;
    color?: string;
    attribute_schema?: string;
    is_active?: boolean;
  }): Promise<CIType> {
    return httpClient.post(`${BASE}/types`, data);
  }

  static async updateCITypes(id: number, data: {
    name: string;
    description?: string;
    icon?: string;
    color?: string;
    attribute_schema?: string;
    is_active?: boolean;
  }): Promise<CIType> {
    return httpClient.put(`${BASE}/types/${id}`, data);
  }

  static async deleteCITypes(id: number): Promise<void> {
    return httpClient.delete(`${BASE}/types/${id}`);
  }

  // ==================== Topology & Impact ====================

  static async getCITopology(id: number, depth = 3): Promise<CITopology> {
    return httpClient.get(`${BASE}/${id}/topology`, { depth });
  }

  static async getCIImpactAnalysis(id: number): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/${id}/impact-analysis`);
  }

  static async analyzeImpact(request: { ciId: string; analysisType?: string; maxDepth?: number }): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/${request.ciId}/impact-analysis`);
  }

  static async getCIHealth(id: string | number): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/${id}/health`);
  }

  static async getCIChangeHistory(id: number, params?: { page?: number; page_size?: number }): Promise<{ items?: Array<Record<string, unknown>>; data?: Array<Record<string, unknown>>; total?: number }> {
    return httpClient.get(`${BASE}/${id}/change-history`, params);
  }

  // ==================== Relationships ====================

  static async createCIRelationship(data: {
    parent_id: number;
    child_id: number;
    type: string;
    description?: string;
  }): Promise<CIRelationship> {
    return httpClient.post(`${BASE}/relationships`, data);
  }

  static async createRelationship(request: {
    source_ci_id: number;
    target_ci_id: number;
    type: string;
    description?: string;
  }): Promise<CIRelationship> {
    return this.createCIRelationship({
      parent_id: request.source_ci_id,
      child_id: request.target_ci_id,
      type: request.type,
      description: request.description,
    });
  }

  static async getCIRelationships(ciId: string | number, params?: {
    direction?: 'incoming' | 'outgoing' | 'both';
    types?: string[];
  }): Promise<CIRelationship[]> {
    try {
      const topology = await this.getCITopology(Number(ciId));
      return [];
    } catch {
      return [];
    }
  }

  static async deleteRelationship(id: string): Promise<void> {
    return httpClient.delete(`${BASE}/relationships/${id}`);
  }

  // ==================== Reconciliation ====================

  static async getReconciliationResults(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/reconciliation`, params);
  }

  static async runReconciliation(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.post(`${BASE}/reconciliation/run`, params);
  }

  // ==================== Cloud ====================

  static async getCloudServices(provider?: string): Promise<CloudService[]> {
    return httpClient.get(`${BASE}/cloud-services`, provider ? { provider } : undefined);
  }

  static async createCloudService(data: Record<string, unknown>): Promise<CloudService> {
    return httpClient.post(`${BASE}/cloud-services`, data);
  }

  static async getCloudAccounts(): Promise<CloudAccount[]> {
    return httpClient.get(`${BASE}/cloud-accounts`);
  }

  static async createCloudAccount(data: Record<string, unknown>): Promise<CloudAccount> {
    return httpClient.post(`${BASE}/cloud-accounts`, data);
  }

  static async deleteCloudAccount(id: string): Promise<void> {
    return httpClient.delete(`${BASE}/cloud-accounts/${id}`);
  }

  static async getCloudResources(params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    return httpClient.get(`${BASE}/cloud-resources`, params);
  }

  // ==================== Discovery ====================

  static async getDiscoveryRules(): Promise<Array<Record<string, unknown>>> {
    return httpClient.get(`${BASE}/discovery-sources`);
  }

  static async getDiscoveryHistory(ruleId?: string): Promise<Array<Record<string, unknown>>> {
    return httpClient.get(`${BASE}/discovery/results`, ruleId ? { source_id: ruleId } : undefined);
  }

  static async runDiscoveryRule(ruleId: string): Promise<void> {
    return httpClient.post(`${BASE}/discovery/jobs`, { source_id: ruleId });
  }

  // ==================== Search ====================

  static async searchCIs(query: {
    keyword?: string;
    ci_type?: string;
    status?: string;
  }): Promise<{ items: ConfigurationItem[]; total: number }> {
    const result = await this.getCIs(query);
    return {
      items: result.items ?? result.cis ?? [],
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

  // ==================== Deprecated aliases ====================

  /** @deprecated use getCITypes */
  static getTypes = CMDBApi.getCITypes;

  /** @deprecated use getReconciliationResults */
  static getReconciliation = CMDBApi.getReconciliationResults;

  static get cis() {
    return {
      list: (params?: GetCIListRequest) => CMDBApi.getCIs(params),
      items: (params?: GetCIListRequest) => CMDBApi.getCIs(params).then((r: GetCIListResponse) => r.items ?? r.cis ?? []),
    };
  }
}

export default CMDBApi;
export const CMDBAPI = CMDBApi;
