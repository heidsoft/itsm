/**
 * AI 类型定义
 */

export interface AIConversation {
    id: number;
    title: string;
    user_id: number;
    tenant_id: number;
    created_at: string;
}

export interface AIMessage {
    id: number;
    conversation_id: number;
    role: string;
    content: string;
    request_id?: string;
    created_at: string;
}

export interface RCAAnalysis {
    id: number;
    ticket_id: number;
    ticket_number: string;
    ticket_title: string;
    analysis_date: string;
    root_causes: any[];
    analysis_summary: string;
    confidence_score: number;
    analysis_method: string;
    created_at: string;
}

export interface AIChatResponse {
    answers: any[];
    conversation_id: number;
}
