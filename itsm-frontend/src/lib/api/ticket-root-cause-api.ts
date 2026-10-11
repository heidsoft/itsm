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
  relatedTickets: Array<{
    id: number;
    number: string;
    title: string;
    similarity: number;
  }>;
  impactScope: {
    affectedTickets: number;
    affectedUsers: number;
    affectedSystems: string[];
  };
  recommendations: string[];
  status: 'identified' | 'confirmed' | 'resolved' | 'false_positive';
  createdAt: string;
  updatedAt: string;
}

export interface RootCauseAnalysisReport {
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  analysisDate: string;
  rootCauses: RootCause[];
  analysisSummary: string;
  confidenceScore: number;
  analysisMethod: 'automatic' | 'manual' | 'hybrid';
  generatedAt: string;
}

export class TicketRootCauseApi {
  // 根因分析端点挂在 /api/v1/incidents 下；类名和形参的 ticket 是历史命名。
  static async analyzeTicket(ticketId: number): Promise<RootCauseAnalysisReport> {
    return httpClient.post<RootCauseAnalysisReport>(
      `/api/v1/incidents/${ticketId}/root-cause`,
      {}
    );
  }

  static async getAnalysisReport(ticketId: number): Promise<RootCauseAnalysisReport | null> {
    return httpClient.get<RootCauseAnalysisReport | null>(
      `/api/v1/incidents/${ticketId}/root-cause`
    );
  }

  static async confirmRootCause(ticketId: number, rootCauseId: string): Promise<void> {
    return httpClient.post(`/api/v1/incidents/${ticketId}/root-cause/${rootCauseId}/confirm`, {});
  }

  static async resolveRootCause(ticketId: number, rootCauseId: string): Promise<void> {
    return httpClient.post(`/api/v1/incidents/${ticketId}/root-cause/${rootCauseId}/resolve`, {});
  }
}
