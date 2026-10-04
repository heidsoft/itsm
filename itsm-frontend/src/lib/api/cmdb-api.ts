/**
 * CMDB API 服务 - 统一使用生产路由 /api/v1/cmdb
 *
 * 注意：云资源/云账号/云服务/发现/对账等子资源在后端被挂在
 * `/api/v1/cmdb/*` 下（不是 `/api/v1/configuration-items/*`），
 * 所有 CMDB 资源统一走 `/api/v1/cmdb/*`，避免依赖已弃用的兼容别名。
 */

import { httpClient } from './http-client';
import type { PaginationResponse } from './types';
import type {
  CIType,
  CIHistoryItem,
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
  pageSize?: number;
}

// 与后端 dto.CIListResponse 逐字段一致；不要再声明第二套键名。
export type GetCIListResponse = PaginationResponse<ConfigurationItem>;

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

// 与后端 dto.ListCloudResourcesRequest + CloudResourceListResponse 逐字段一致。
// 刻意没有 offset/limit/accountId/service_id：后端只认这份查询串。
export interface GetCloudResourcesRequest {
  provider?: string;
  cloudAccountId?: number;
  serviceId?: number;
  region?: string;
  status?: string;
  search?: string;
  page?: number;
  pageSize?: number;
}

export type CloudResourceListResponse = PaginationResponse<CloudResource>;

// 与后端 dto.CloudServiceListResponse / CloudAccountListResponse 一致：
// 这两个端点不分页，所以契约里就没有 page/pageSize/totalPages，前端不得假造。
export interface CloudServiceListResponse {
  items: CloudService[];
  total: number;
}

export interface CloudAccountListResponse {
  items: CloudAccount[];
  total: number;
}

const CMDB_BASE = '/api/v1/cmdb';
// 必须是字符串字面量：派生模板常量无法被 api-contract 扫描器静态解析。
const CIS_BASE = '/api/v1/cmdb/cis';
// 后端 common.MaxPageSize：请求更大的页长不会报错，而是被单点回落成默认 20。
// 需要「尽量取全」的数据源只能翻页，不能靠放大页长实现。
const MAX_PAGE_SIZE = 100;

export class CMDBApi {
  static async getCapabilities(): Promise<CMDBCapabilitiesResponse> {
    return httpClient.get(`${CMDB_BASE}/capabilities`);
  }

  // ==================== CI CRUD ====================

  static async getCIs(query?: GetCIListRequest): Promise<GetCIListResponse> {
    return httpClient.get(CIS_BASE, query);
  }

  private static async pageThrough<T>(
    fetchPage: (page: number, pageSize: number) => Promise<PaginationResponse<T>>,
    maxRecords = Number.POSITIVE_INFINITY,
  ): Promise<T[]> {
    const all: T[] = [];
    for (let page = 1; ; page += 1) {
      const response = await fetchPage(page, MAX_PAGE_SIZE);
      all.push(...response.items);
      // 终止条件用响应里的生效页长，不用请求值：后端回落时两者可能不同。
      if (
        response.items.length < response.pageSize ||
        all.length >= Math.min(maxRecords, response.total)
      ) {
        break;
      }
    }
    return all;
  }

  /** 翻页读满 maxRecords 或读到末页；用于下拉/穿梭框这类需要候选集的数据源。 */
  static async getAllCIs(
    params: Omit<GetCIListRequest, 'page' | 'pageSize'> = {},
    maxRecords = 200,
  ): Promise<ConfigurationItem[]> {
    return this.pageThrough(
      (page, pageSize) => httpClient.get<GetCIListResponse>(CIS_BASE, { ...params, page, pageSize }),
      maxRecords,
    );
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
    return this.pageThrough(
      (page, pageSize) =>
        httpClient.get<PaginationResponse<CIType>>(`${CMDB_BASE}/ci-types`, { page, pageSize }),
    );
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
  ): Promise<PaginationResponse<CIHistoryItem>> {
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
  //
  // 云账号/云服务/云资源只有 /api/v1/cmdb/cloud-* 这一套表面。
  // /api/v1/cloud/* 是同一用例的第二套实现（零前端调用方），2026-10-04 已删除。
  // 云账号与云服务是 CI 表单/云资源页的选择器数据源，后端整份返回，契约只有 {items,total}；
  // 云资源是真实分页列表，契约为 {items,total,page,pageSize,totalPages}。

  static async getCloudServices(
    provider?: string
  ): Promise<CloudServiceListResponse> {
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

  static async getCloudAccounts(): Promise<CloudAccountListResponse> {
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

  static async getCloudResources(
    params?: GetCloudResourcesRequest
  ): Promise<CloudResourceListResponse> {
    return httpClient.get(`${CMDB_BASE}/cloud-resources`, params);
  }

  /** 翻页读满 maxRecords 或读到末页；仪表盘这类聚合数据源不能靠放大页长取全。 */
  static async getAllCloudResources(
    params: Omit<GetCloudResourcesRequest, 'page' | 'pageSize'> = {},
    maxRecords = 200,
  ): Promise<CloudResource[]> {
    return this.pageThrough(
      (page, pageSize) =>
        httpClient.get<CloudResourceListResponse>(`${CMDB_BASE}/cloud-resources`, {
          ...params,
          page,
          pageSize,
        }),
      maxRecords,
    );
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
