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
    CIQuery
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
}
