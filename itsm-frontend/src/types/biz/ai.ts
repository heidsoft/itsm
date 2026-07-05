/**
 * AI 类型定义
 */

export interface AIConversation {
  id: number;
  title: string;
  userId: number;
  tenantId: number;
  createdAt: string;
}

export interface AIMessage {
  id: number;
  conversationId: number;
  role: string;
  content: string;
  requestId?: string;
  createdAt: string;
}

export interface RCAAnalysis {
  id: number;
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  analysisDate: string;
  rootCauses: unknown[];
  analysisSummary: string;
  confidenceScore: number;
  analysisMethod: string;
  createdAt: string;
}

export interface AIChatResponse {
  answers: unknown[];
  conversationId: number;
}
