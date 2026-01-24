/**
 * 事件管理 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type {
    Incident,
    CreateIncidentRequest,
    UpdateIncidentRequest,
    IncidentListResponse,
    IncidentQuery,
} from './types';

const BASE_URL = '/api/v1/incidents';

export class IncidentApi {

    /**
     * 获取事件列表
     */
    static async getIncidents(query?: IncidentQuery): Promise<IncidentListResponse> {
        const params: Record<string, any> = {
            page: query?.page ?? 1,
            size: query?.size ?? 10,
        };

        if (query?.status) params.status = query.status;
        if (query?.priority) params.priority = query.priority;
        if (query?.keyword) params.keyword = query.keyword;
        if (query?.scope) params.scope = query.scope;

        const resp = await httpClient.get<any>(BASE_URL, params);

        return {
            items: resp.items || [],
            total: resp.total || 0,
        };
    }

    /**
     * 获取单个事件详情
     */
    static async getIncident(id: string | number): Promise<Incident> {
        return httpClient.get<Incident>(`${BASE_URL}/${id}`);
    }

    /**
     * 创建事件
     */
    static async createIncident(data: CreateIncidentRequest): Promise<Incident> {
        return httpClient.post<Incident>(BASE_URL, data);
    }

    /**
     * 更新事件
     */
    static async updateIncident(id: string | number, data: UpdateIncidentRequest): Promise<Incident> {
        return httpClient.put<Incident>(`${BASE_URL}/${id}`, data);
    }

    /**
     * 升级事件
     */
    static async escalateIncident(id: string | number, level: number, reason: string): Promise<Incident> {
        return httpClient.post<Incident>(`${BASE_URL}/${id}/escalate`, {
            escalation_level: level,
            reason: reason,
        });
    }

    /**
     * 删除事件
     */
    static async deleteIncident(id: string | number): Promise<void> {
        return httpClient.delete(`${BASE_URL}/${id}`);
    }

    /**
     * 指派事件
     */
    static async assignIncident(id: string | number, assigneeId: number): Promise<Incident> {
        return httpClient.post<Incident>(`${BASE_URL}/${id}/assign`, {
            assignee_id: assigneeId,
        });
    }

    /**
     * 解决事件
     */
    static async resolveIncident(id: string | number, resolution: string): Promise<Incident> {
        return httpClient.post<Incident>(`${BASE_URL}/${id}/resolve`, {
            resolution,
        });
    }

    /**
     * 关闭事件
     */
    static async closeIncident(id: string | number, notes?: string): Promise<Incident> {
        return httpClient.post<Incident>(`${BASE_URL}/${id}/close`, {
            close_notes: notes,
        });
    }

    /**
     * 接受事件（指派给自己）
     */
    static async acceptIncident(id: string | number): Promise<Incident> {
        return httpClient.post<Incident>(`${BASE_URL}/${id}/accept`, {});
    }

    /**
     * 获取事件活动记录
     */
    static async getIncidentActivity(id: string | number): Promise<Array<{
        id: number;
        action: string;
        description: string;
        user_id: number;
        created_at: string;
    }>> {
        return httpClient.get(`${BASE_URL}/${id}/activity`);
    }
}
