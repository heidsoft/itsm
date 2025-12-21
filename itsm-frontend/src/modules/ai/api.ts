/**
 * AI API 客户端
 */

import { httpClient } from '@/lib/api/http-client';
import type { AIChatResponse, RCAAnalysis } from './types';

const BASE_URL = '/api/v1/ai';

export class AIApi {
    /**
     * 发起 AI 对话
     */
    static async chat(data: { query: string; limit?: number; conversation_id?: number }): Promise<AIChatResponse> {
        return httpClient.post<AIChatResponse>(`${BASE_URL}/chat`, data);
    }

    /**
     * 获取深度分析
     */
    static async getAnalytics(data: any): Promise<any> {
        return httpClient.post<any>(`${BASE_URL}/analytics`, data);
    }

    /**
     * 获取趋势预测
     */
    static async getPredictions(data: any): Promise<any> {
        return httpClient.post<any>(`${BASE_URL}/predictions`, data);
    }

    /**
     * 执行根因分析
     */
    static async analyzeTicket(ticketId: number): Promise<RCAAnalysis> {
        return httpClient.post<RCAAnalysis>(`${BASE_URL}/tickets/${ticketId}/analyze`, {});
    }

    /**
     * 提交反馈
     */
    static async saveFeedback(data: any): Promise<{ message: string }> {
        return httpClient.post<{ message: string }>(`${BASE_URL}/feedback`, data);
    }
}
