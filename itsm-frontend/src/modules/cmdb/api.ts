/**
 * CMDB 资产管理 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type {
    ConfigurationItem,
    CIType,
    CIRelationship,
    CMDBStats,
    CIListResponse,
    CIQuery,
    CloudService,
    CloudAccount,
    CloudResource,
    RelationshipType,
    DiscoverySource,
    DiscoveryJob,
    DiscoveryResult,
    ReconciliationResponse
} from './types';

const BASE_URL = '/api/v1/cmdb';

export class CMDBApi {
    /**
     * 获取配置项列表
     */
    static async getCIs(query?: CIQuery): Promise<CIListResponse> {
        return httpClient.get<CIListResponse>(`${BASE_URL}/cis`, query);
    }

    /**
     * 获取配置项详情
     */
    static async getCI(id: string | number): Promise<ConfigurationItem> {
        return httpClient.get<ConfigurationItem>(`${BASE_URL}/cis/${id}`);
    }

    /**
     * 创建配置项
     */
    static async createCI(data: Partial<ConfigurationItem>): Promise<ConfigurationItem> {
        return httpClient.post<ConfigurationItem>(`${BASE_URL}/cis`, data);
    }

    /**
     * 更新配置项
     */
    static async updateCI(id: string | number, data: Partial<ConfigurationItem>): Promise<ConfigurationItem> {
        return httpClient.put<ConfigurationItem>(`${BASE_URL}/cis/${id}`, data);
    }

    /**
     * 删除配置项
     */
    static async deleteCI(id: string | number): Promise<void> {
        return httpClient.delete<void>(`${BASE_URL}/cis/${id}`);
    }

    /**
     * 获取统计信息
     */
    static async getStats(): Promise<CMDBStats> {
        return httpClient.get<CMDBStats>(`${BASE_URL}/stats`);
    }

    /**
     * 获取 CI 类型列表
     */
    static async getTypes(): Promise<CIType[]> {
        return httpClient.get<CIType[]>(`${BASE_URL}/types`);
    }

    /**
     * 获取关系类型
     */
    static async getRelationshipTypes(): Promise<RelationshipType[]> {
        return httpClient.get<RelationshipType[]>(`${BASE_URL}/relationship-types`);
    }

    /**
     * CMDB 对账
     */
    static async getReconciliation(): Promise<ReconciliationResponse> {
        return httpClient.get<ReconciliationResponse>(`${BASE_URL}/reconciliation`);
    }

    /**
     * 云服务目录
     */
    static async getCloudServices(provider?: string): Promise<CloudService[]> {
        return httpClient.get<CloudService[]>(`${BASE_URL}/cloud-services`, { provider });
    }

    static async createCloudService(data: Partial<CloudService>): Promise<CloudService> {
        return httpClient.post<CloudService>(`${BASE_URL}/cloud-services`, data);
    }

    /**
     * 云账号
     */
    static async getCloudAccounts(provider?: string): Promise<CloudAccount[]> {
        return httpClient.get<CloudAccount[]>(`${BASE_URL}/cloud-accounts`, { provider });
    }

    static async createCloudAccount(data: Partial<CloudAccount>): Promise<CloudAccount> {
        return httpClient.post<CloudAccount>(`${BASE_URL}/cloud-accounts`, data);
    }

    /**
     * 云资源
     */
    static async getCloudResources(params?: {
        provider?: string;
        service_id?: number;
        region?: string;
    }): Promise<CloudResource[]> {
        return httpClient.get<CloudResource[]>(`${BASE_URL}/cloud-resources`, params);
    }

    /**
     * 发现源
     */
    static async getDiscoverySources(): Promise<DiscoverySource[]> {
        return httpClient.get<DiscoverySource[]>(`${BASE_URL}/discovery-sources`);
    }

    static async createDiscoverySource(data: Partial<DiscoverySource>): Promise<DiscoverySource> {
        return httpClient.post<DiscoverySource>(`${BASE_URL}/discovery-sources`, data);
    }

    /**
     * 发现任务
     */
    static async createDiscoveryJob(data: { source_id: string }): Promise<DiscoveryJob> {
        return httpClient.post<DiscoveryJob>(`${BASE_URL}/discovery/jobs`, data);
    }

    static async getDiscoveryResults(jobId?: number): Promise<DiscoveryResult[]> {
        return httpClient.get<DiscoveryResult[]>(`${BASE_URL}/discovery/results`, { job_id: jobId });
    }

    /**
     * 获取配置项关系
     */
    static async getCIRelationships(ciId: string | number): Promise<CIRelationship[]> {
        return httpClient.get<CIRelationship[]>(`${BASE_URL}/cis/${ciId}/relationships`);
    }

    /**
     * 创建配置项关系
     */
    static async createCIRelationship(data: {
        source_ci_id: number;
        target_ci_id: number;
        relationship_type: string;
    }): Promise<CIRelationship> {
        return httpClient.post<CIRelationship>(`${BASE_URL}/relationships`, data);
    }

    /**
     * 删除配置项关系
     */
    static async deleteCIRelationship(relationshipId: string | number): Promise<void> {
        return httpClient.delete(`${BASE_URL}/relationships/${relationshipId}`);
    }

    /**
     * 批量删除配置项
     */
    static async batchDeleteCIs(ids: number[]): Promise<{ deleted: number }> {
        return httpClient.post(`${BASE_URL}/cis/batch`, { ids });
    }

    /**
     * 导出配置项
     */
    static async exportCIs(format: 'csv' | 'excel' | 'json', query?: CIQuery): Promise<Blob> {
        return httpClient.request({
            method: 'GET',
            url: `${BASE_URL}/cis/export`,
            params: { format, ...query },
            responseType: 'blob',
        }) as Promise<Blob>;
    }
}
