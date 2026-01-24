/**
 * 问题管理 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type {
    Problem,
    CreateProblemRequest,
    UpdateProblemRequest,
    ProblemListResponse,
    ProblemQuery,
    ProblemStats
} from './types';

const BASE_URL = '/api/v1/problems';

export class ProblemApi {

    /**
     * 获取问题列表
     */
    static async getProblems(query?: ProblemQuery): Promise<ProblemListResponse> {
        return httpClient.get<ProblemListResponse>(BASE_URL, query);
    }

    /**
     * 获取问题统计
     */
    static async getStats(): Promise<ProblemStats> {
        return httpClient.get<ProblemStats>(`${BASE_URL}/stats`);
    }

    /**
     * 获取单个问题详情
     */
    static async getProblem(id: string | number): Promise<Problem> {
        // 后端详情接口返回的是 { problem: ... } 还是直接 Problem?
        // 根据 handler.go 中的 toDTO: return &dto.ProblemDetailResponse{Problem: ep}
        // 所以返回值解包一下
        const resp = await httpClient.get<{ problem: Problem }>(`${BASE_URL}/${id}`);
        return resp.problem;
    }

    /**
     * 创建问题
     */
    static async createProblem(data: CreateProblemRequest): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(BASE_URL, data);
        return resp.problem;
    }

    /**
     * 更新问题
     */
    static async updateProblem(id: string | number, data: UpdateProblemRequest): Promise<Problem> {
        const resp = await httpClient.put<{ problem: Problem }>(`${BASE_URL}/${id}`, data);
        return resp.problem;
    }

    /**
     * 删除问题
     */
    static async deleteProblem(id: string | number): Promise<void> {
        return httpClient.delete(`${BASE_URL}/${id}`);
    }

    /**
     * 指派问题
     */
    static async assignProblem(id: string | number, assigneeId: number): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(`${BASE_URL}/${id}/assign`, {
            assignee_id: assigneeId,
        });
        return resp.problem;
    }

    /**
     * 解决（完成）问题
     */
    static async resolveProblem(id: string | number, resolution: string, rootCause?: string): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(`${BASE_URL}/${id}/resolve`, {
            resolution,
            root_cause: rootCause,
        });
        return resp.problem;
    }

    /**
     * 关闭问题
     */
    static async closeProblem(id: string | number, notes?: string): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(`${BASE_URL}/${id}/close`, {
            close_notes: notes,
        });
        return resp.problem;
    }

    /**
     * 升级问题
     */
    static async escalateProblem(id: string | number, level: number, reason: string): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(`${BASE_URL}/${id}/escalate`, {
            escalation_level: level,
            reason,
        });
        return resp.problem;
    }

    /**
     * 关联事件到问题
     */
    static async linkIncident(id: string | number, incidentIds: number[]): Promise<Problem> {
        const resp = await httpClient.post<{ problem: Problem }>(`${BASE_URL}/${id}/link-incidents`, {
            incident_ids: incidentIds,
        });
        return resp.problem;
    }

    /**
     * 获取问题活动记录
     */
    static async getProblemActivity(id: string | number): Promise<Array<{
        id: number;
        action: string;
        description: string;
        user_id: number;
        created_at: string;
    }>> {
        return httpClient.get(`${BASE_URL}/${id}/activity`);
    }
}
