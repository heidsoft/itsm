/**
 * 变更管理 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type {
    Change,
    CreateChangeRequest,
    ChangeListResponse,
    ChangeQuery,
    ChangeStats,
    ApprovalRecord,
    RiskAssessment
} from './types';

const BASE_URL = '/api/v1/changes';

export class ChangeApi {

    /**
     * 获取变更列表
     */
    static async getChanges(query?: ChangeQuery): Promise<ChangeListResponse> {
        return httpClient.get<ChangeListResponse>(BASE_URL, query);
    }

    /**
     * 获取变更统计
     */
    static async getStats(): Promise<ChangeStats> {
        return httpClient.get<ChangeStats>(`${BASE_URL}/stats`);
    }

    /**
     * 获取变更详情
     */
    static async getChange(id: string | number): Promise<Change> {
        return httpClient.get<Change>(`${BASE_URL}/${id}`);
    }

    /**
     * 创建变更
     */
    static async createChange(data: CreateChangeRequest): Promise<Change> {
        return httpClient.post<Change>(BASE_URL, data);
    }

    /**
     * 更新变更
     */
    static async updateChange(id: string | number, data: Partial<Change>): Promise<Change> {
        return httpClient.put<Change>(`${BASE_URL}/${id}`, data);
    }

    /**
     * 提交审批
     */
    static async submitApproval(changeId: number, data: { approver_id: number; comment?: string }): Promise<ApprovalRecord> {
        return httpClient.post<ApprovalRecord>(`${BASE_URL}/${changeId}/approvals`, data);
    }

    /**
     * 获取审批摘要 (历史与链)
     */
    static async getApprovalSummary(changeId: number): Promise<{ history: ApprovalRecord[], chain: any[] }> {
        return httpClient.get<any>(`${BASE_URL}/${changeId}/approval-summary`);
    }

    /**
     * 获取风险评估
     */
    static async getRiskAssessment(changeId: number): Promise<RiskAssessment> {
        return httpClient.get<RiskAssessment>(`${BASE_URL}/${changeId}/risk-assessment`);
    }

    /**
     * 删除变更
     */
    static async deleteChange(id: string | number): Promise<void> {
        return httpClient.delete(`${BASE_URL}/${id}`);
    }

    /**
     * 指派变更
     */
    static async assignChange(id: string | number, assigneeId: number): Promise<Change> {
        return httpClient.post<Change>(`${BASE_URL}/${id}/assign`, {
            assignee_id: assigneeId,
        });
    }

    /**
     * 审批变更（通过/拒绝）
     */
    static async approveChange(id: string | number, action: 'approve' | 'reject', comment?: string): Promise<Change> {
        return httpClient.post<Change>(`${BASE_URL}/${id}/approve`, {
            action,
            comment,
        });
    }

    /**
     * 取消变更
     */
    static async cancelChange(id: string | number, reason: string): Promise<Change> {
        return httpClient.post<Change>(`${BASE_URL}/${id}/cancel`, {
            reason,
        });
    }

    /**
     * 获取变更活动记录
     */
    static async getChangeActivity(id: string | number): Promise<Array<{
        id: number;
        action: string;
        description: string;
        user_id: number;
        created_at: string;
    }>> {
        return httpClient.get(`${BASE_URL}/${id}/activity`);
    }
}
