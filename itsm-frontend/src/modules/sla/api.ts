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

    /**
     * 获取 SLA 监控仪表盘数据
     */
    static async getDashboardData(params?: {
        start_date?: string;
        end_date?: string;
        group_by?: 'day' | 'week' | 'month';
    }): Promise<{
        total_tickets: number;
        compliant_tickets: number;
        breached_tickets: number;
        compliance_rate: number;
        avg_response_time: number;
        avg_resolution_time: number;
        trends: Array<{
            date: string;
            compliance_rate: number;
            total: number;
            breached: number;
        }>;
    }> {
        return httpClient.get(`${BASE_URL}/dashboard`, params);
    }

    /**
     * 获取 SLA 违规记录
     */
    static async getBreaches(params?: {
        page?: number;
        size?: number;
        start_date?: string;
        end_date?: string;
    }): Promise<{
        items: Array<{
            id: number;
            ticket_id: number;
            ticket_number: string;
            sla_name: string;
            breach_type: 'response' | 'resolution';
            breached_at: string;
            exceeded_by_minutes: number;
        }>;
        total: number;
    }> {
        return httpClient.get(`${BASE_URL}/breaches`, params);
    }

    /**
     * 获取待处理 SLA 预警列表
     */
    static async getPendingAlerts(): Promise<Array<{
        id: number;
        ticket_id: number;
        ticket_number: string;
        sla_name: string;
        alert_type: 'warning' | 'breach';
        remaining_minutes: number;
        created_at: string;
    }>> {
        return httpClient.get(`${BASE_URL}/pending-alerts`);
    }

    /**
     * 获取 SLA 统计报告
     */
    static async getSLAReport(slaId: string | number, params?: {
        start_date: string;
        end_date: string;
    }): Promise<{
        sla: SLADefinition;
        total_tickets: number;
        compliant_tickets: number;
        breached_tickets: number;
        compliance_rate: number;
        avg_response_time: number;
        avg_resolution_time: number;
        response_breaches: number;
        resolution_breaches: number;
    }> {
        return httpClient.get(`${BASE_URL}/reports/${slaId}`, params);
    }

    /**
     * 删除预警规则
     */
    static async deleteAlertRule(id: string | number): Promise<void> {
        return httpClient.delete(`${BASE_URL}/alert-rules/${id}`);
    }

    /**
     * 更新预警规则
     */
    static async updateAlertRule(id: string | number, data: Partial<SLAAlertRule>): Promise<SLAAlertRule> {
        return httpClient.put<SLAAlertRule>(`${BASE_URL}/alert-rules/${id}`, data);
    }
}
