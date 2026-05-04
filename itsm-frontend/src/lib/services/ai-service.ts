/**
 * AI服务类 - 提供智能分析和推荐功能
 * 支持工单分类、处理人推荐、解决方案建议等AI功能
 */

import { httpClient } from '@/lib/api/http-client';
import { logger } from '@/lib/env';

// AI服务响应类型定义
export interface AIAnalysisResult {
  confidence: number;
  reasoning: string;
  suggestions?: string[];
}

export interface TicketClassificationResult extends AIAnalysisResult {
  category: string;
  subcategory?: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  urgency: 'low' | 'medium' | 'high' | 'critical' | string;
}

export interface SolutionSuggestion extends AIAnalysisResult {
  solutionId: string;
  title: string;
  description: string;
  steps?: string[];
  estimatedTime?: number; // 预计解决时间（分钟）
  successRate?: number; // 历史成功率
  relatedKnowledge?: string[];
}

export interface SearchResult {
  id: number;
  title: string;
  description: string;
  type: string;
  status?: string;
  number?: string;
}

// AI服务请求参数类型
export interface TicketAnalysisRequest {
  title: string;
  description: string;
  attachments?: string[];
  userContext?: {
    department: string;
    role: string;
    location: string;
  };
}

export interface SolutionSearchRequest {
  query: string;
  category?: string;
  priority?: string;
  context?: string;
  limit?: number;
}

/**
 * AI服务类
 * 提供各种AI驱动的智能分析和推荐功能
 */
export class AIService {
  private static readonly API_BASE = '/api/v1/ai';

  /**
   * 工单智能分类
   * 基于标题、描述等信息自动分类工单并确定优先级
   */
  static async classifyTicket(request: TicketAnalysisRequest): Promise<TicketClassificationResult> {
    try {
      logger.debug('AIService.classifyTicket', { title: request.title });

      const res = await httpClient.post<{
        title: string;
        description: string;
        suggestions?: {
          category?: string;
          priority?: string;
          confidence?: number;
          reasoning?: string;
          urgency?: string;
        };
      }>(`${this.API_BASE}/triage`, {
        title: request.title,
        description: request.description,
      });

      const s = res?.suggestions || {};
      const priority = (s.priority || 'medium') as TicketClassificationResult['priority'];
      const urgency = (s.urgency || priority) as TicketClassificationResult['urgency'];
      const result: TicketClassificationResult = {
        category: s.category || 'general',
        priority,
        urgency,
        confidence: typeof s.confidence === 'number' ? s.confidence : 0,
        reasoning: s.reasoning || '',
        suggestions: [],
      };

      logger.debug('AIService.classifyTicket complete', { category: result.category });
      return result;
    } catch (error) {
      logger.error('AIService.classifyTicket failed', error);
      throw error instanceof Error ? error : new Error('AI分析失败，请稍后重试');
    }
  }

  /**
   * 智能解决方案建议
   * 基于问题描述搜索相关的解决方案和知识库文章
   */
  static async suggestSolutions(request: SolutionSearchRequest): Promise<SolutionSuggestion[]> {
    try {
      logger.debug('AIService.suggestSolutions', { query: request.query?.substring(0, 50) });

      const limit = request.limit && request.limit > 0 ? request.limit : 5;
      const answers = await httpClient.post<any[]>(`${this.API_BASE}/knowledge/search`, {
        query: request.query,
        limit,
        type: 'kb',
      });

      const list = Array.isArray(answers) ? answers : [];
      const mapped = list.map((item, idx) => {
        const title = String(item?.title || item?.source || `知识项 ${idx + 1}`);
        const snippet = String(item?.snippet || item?.content || item?.text || '');
        const score = typeof item?.score === 'number' ? item.score : undefined;

        const suggestion: SolutionSuggestion = {
          solutionId: String(item?.id ?? `${Date.now()}_${idx}`),
          title,
          description: snippet,
          steps: [],
          relatedKnowledge: item?.source ? [String(item.source)] : [],
          confidence: typeof score === 'number' ? score : 0,
          reasoning: '来自知识库检索结果',
        };

        return suggestion;
      });

      logger.debug('AIService.suggestSolutions complete', { count: mapped.length });
      return mapped;
    } catch (error) {
      logger.error('AIService.suggestSolutions failed', error);
      throw error instanceof Error ? error : new Error('知识搜索失败，请稍后重试');
    }
  }

  /**
   * 智能搜索
   * 基于自然语言查询搜索相关工单、知识库等
   */
  static async intelligentSearch(
    query: string,
    filters?: {
      type?: 'tickets' | 'knowledge' | 'incidents' | 'all';
      dateRange?: { start: string; end: string };
      category?: string;
    }
  ): Promise<{
    tickets: SearchResult[];
    knowledge: SearchResult[];
    incidents: SearchResult[];
    suggestions: string[];
  }> {
    try {
      logger.debug('AIService.intelligentSearch', {
        query: query.substring(0, 50),
        type: filters?.type,
      });

      const res = await httpClient.get<{ results: Array<any>; total: number }>(
        `/api/v1/global-search?keyword=${encodeURIComponent(query)}`
      );
      const results = Array.isArray(res?.results) ? res.results : [];

      const normalize = (r: any): SearchResult => ({
        id: Number(r?.id || 0),
        type: String(r?.type || ''),
        title: String(r?.title || ''),
        description: String(r?.description || ''),
        status: r?.status ? String(r.status) : undefined,
        number: r?.ticketNumber ? String(r.ticketNumber) : r?.number ? String(r.number) : undefined,
      });

      const tickets = results.filter(r => r?.type === 'ticket').map(normalize);
      const incidents = results.filter(r => r?.type === 'incident').map(normalize);
      const knowledge = results.filter(r => r?.type === 'knowledge').map(normalize);

      const response = {
        tickets,
        knowledge,
        incidents,
        suggestions: [],
      };

      logger.debug('AIService.intelligentSearch complete', {
        ticketsCount: response.tickets.length,
        knowledgeCount: response.knowledge.length,
      });
      return response;
    } catch (error) {
      logger.error('AIService.intelligentSearch failed', error);
      throw error instanceof Error ? error : new Error('搜索失败，请稍后重试');
    }
  }
}

// 导出AI服务实例
export const aiService = AIService;
