/**
 * SLA API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type { SLADefinition, SLADefinitionListResponse, SLAAlertRule } from './types';

const BASE_URL = '/api/v1/sla';

export class SLAApi {
    /**
     * 获取 SLA 定义列表
     */
    static async getDefinitions(params?: { page?: number; size?: number }): Promise<SLADefinitionListResponse> {
        return httpClient.get<SLADefinitionListResponse>(`${BASE_URL}/definitions`, params);
    }

    /**
     * 获取 SLA 定义详情
     */
    static async getDefinition(id: string | number): Promise<SLADefinition> {
        return httpClient.get<SLADefinition>(`${BASE_URL}/definitions/${id}`);
    }

    /**
     * 创建 SLA 定义
     */
    static async createDefinition(data: Partial<SLADefinition>): Promise<SLADefinition> {
        return httpClient.post<SLADefinition>(`${BASE_URL}/definitions`, data);
    }

    /**
     * 更新 SLA 定义
     */
    static async updateDefinition(id: string | number, data: Partial<SLADefinition>): Promise<SLADefinition> {
        return httpClient.put<SLADefinition>(`${BASE_URL}/definitions/${id}`, data);
    }

    /**
     * 删除 SLA 定义
     */
    static async deleteDefinition(id: string | number): Promise<void> {
        return httpClient.delete<void>(`${BASE_URL}/definitions/${id}`);
    }

    /**
     * 获取预警规则列表
     */
    static async getAlertRules(params?: { sla_definition_id?: number }): Promise<SLAAlertRule[]> {
        return httpClient.get<SLAAlertRule[]>(`${BASE_URL}/alert-rules`, params);
    }

    /**
     * 创建预警规则
     */
    static async createAlertRule(data: Partial<SLAAlertRule>): Promise<SLAAlertRule> {
        return httpClient.post<SLAAlertRule>(`${BASE_URL}/alert-rules`, data);
    }
}
