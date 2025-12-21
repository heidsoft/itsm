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
}
