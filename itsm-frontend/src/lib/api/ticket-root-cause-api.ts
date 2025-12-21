/**
 * 工单根因分析 API 服务
 */

import { httpClient } from './http-client';

export interface RootCause {
  id: string;
  title: string;
  description: string;
  confidence: number;
  category: 'system' | 'network' | 'application' | 'database' | 'user' | 'other';
  evidence: Array<{
    type: 'log' | 'metric' | 'ticket' | 'event';
    content: string;
    timestamp: string;
    relevance: number;
  }>;
  related_tickets: Array<{
    id: number;
    number: string;
    title: string;
    similarity: number;
  }>;
  impact_scope: {
    affected_tickets: number;
    affected_users: number;
    affected_systems: string[];
  };
  recommendations: string[];
  status: 'identified' | 'confirmed' | 'resolved' | 'false_positive';
  created_at: string;
  updated_at: string;
}

export interface RootCauseAnalysisReport {
  ticket_id: number;
  ticket_number: string;
  ticket_title: string;
  analysis_date: string;
  root_causes: RootCause[];
  analysis_summary: string;
  confidence_score: number;
  analysis_method: 'automatic' | 'manual' | 'hybrid';
  generated_at: string;
}

export class TicketRootCauseApi {
  // 执行根因分析
  static async analyzeTicket(ticketId: number): Promise<RootCauseAnalysisReport> {
    return httpClient.post<RootCauseAnalysisReport>(`/api/tickets/${ticketId}/root-cause/analyze`, {});
  }

  // 获取根因分析报告
  static async getAnalysisReport(ticketId: number): Promise<RootCauseAnalysisReport | null> {
    return httpClient.get<RootCauseAnalysisReport | null>(`/api/tickets/${ticketId}/root-cause/report`);
  }

  // 确认根因
  static async confirmRootCause(ticketId: number, rootCauseId: string): Promise<void> {
    return httpClient.post(`/api/tickets/${ticketId}/root-cause/${rootCauseId}/confirm`, {});
  }

  // 标记根因为已解决
  static async resolveRootCause(ticketId: number, rootCauseId: string): Promise<void> {
    return httpClient.post(`/api/tickets/${ticketId}/root-cause/${rootCauseId}/resolve`, {});
  }
}

