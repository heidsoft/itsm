/**
 * AI服务类 - 提供智能分析和推荐功能
 * 支持工单分类、处理人推荐、解决方案建议等AI功能
 */

import { httpClient } from './http-client';

// AI服务响应类型定义
export interface AIAnalysisResult {
  confidence: number;
  reasoning: string;
  suggestions: string[];
}

export interface TicketClassificationResult extends AIAnalysisResult {
  category: string;
  subcategory?: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  urgency: 'low' | 'medium' | 'high' | 'critical';
}

export interface AssigneeRecommendation extends AIAnalysisResult {
  userId: number;
  userName: string;
  department: string;
  expertise: string[];
  workload: number;
  availability: 'available' | 'busy' | 'unavailable';
}

export interface SolutionSuggestion extends AIAnalysisResult {
  solutionId: string;
  title: string;
  description: string;
  steps: string[];
  estimatedTime: number; // 预计解决时间（分钟）
  successRate: number; // 历史成功率
  relatedKnowledge: string[];
}

export interface IncidentAnalysis extends AIAnalysisResult {
  severity: 'low' | 'medium' | 'high' | 'critical';
  impactScope: string[];
  rootCauseHypothesis: string[];
  recommendedActions: string[];
  escalationRequired: boolean;
}

export interface SearchResult {
  id: number;
  title: string;
  description: string;
  type: string;
  relevance: number;
  createdAt: string;
}

export interface TrendAnalysis {
  period: string;
  trends: {
    category: string;
    trend: 'increasing' | 'decreasing' | 'stable';
    changePercentage: number;
    prediction: string;
  }[];
  insights: string[];
  recommendations: string[];
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

export interface AssigneeRecommendationRequest {
  ticketId?: number;
  category: string;
  priority: string;
  description: string;
  requiredSkills?: string[];
  departmentPreference?: string;
}

export interface SolutionSearchRequest {
  query: string;
  category?: string;
  priority?: string;
  context?: string;
  limit?: number;
}

export interface IncidentAnalysisRequest {
  title: string;
  description: string;
  symptoms: string[];
  affectedSystems: string[];
  timeOfOccurrence: string;
  userReports?: string[];
}

/**
 * AI服务类
 * 提供各种AI驱动的智能分析和推荐功能
 */
export class AIService {
  private static readonly API_BASE = '/api/ai';

  /**
   * 工单智能分类
   * 基于标题、描述等信息自动分类工单并确定优先级
   */
  static async classifyTicket(request: TicketAnalysisRequest): Promise<TicketClassificationResult> {
    try {
      console.log('AIService.classifyTicket called with:', request);
      
      const response = await httpClient.post<TicketClassificationResult>(
        `${this.API_BASE}/classify-ticket`,
        request
      );
      
      console.log('AIService.classifyTicket response:', response);
      return response;
    } catch (error) {
      console.error('AIService.classifyTicket error:', error);
      
      // 提供降级方案
      return this.getFallbackClassification(request);
    }
  }

  /**
   * 智能推荐处理人
   * 基于工单内容、处理人技能、工作负载等推荐最合适的处理人
   */
  static async recommendAssignee(request: AssigneeRecommendationRequest): Promise<AssigneeRecommendation[]> {
    try {
      console.log('AIService.recommendAssignee called with:', request);
      
      const response = await httpClient.post<AssigneeRecommendation[]>(
        `${this.API_BASE}/recommend-assignee`,
        request
      );
      
      console.log('AIService.recommendAssignee response:', response);
      return response;
    } catch (error) {
      console.error('AIService.recommendAssignee error:', error);
      
      // 提供降级方案
      return this.getFallbackAssigneeRecommendations(request);
    }
  }

  /**
   * 智能解决方案建议
   * 基于问题描述搜索相关的解决方案和知识库文章
   */
  static async suggestSolutions(request: SolutionSearchRequest): Promise<SolutionSuggestion[]> {
    try {
      console.log('AIService.suggestSolutions called with:', request);
      
      const response = await httpClient.post<SolutionSuggestion[]>(
        `${this.API_BASE}/suggest-solutions`,
        request
      );
      
      console.log('AIService.suggestSolutions response:', response);
      return response;
    } catch (error) {
      console.error('AIService.suggestSolutions error:', error);
      
      // 提供降级方案
      return this.getFallbackSolutions(request);
    }
  }

  /**
   * 事件智能分析
   * 分析事件的严重程度、影响范围和可能的根本原因
   */
  static async analyzeIncident(request: IncidentAnalysisRequest): Promise<IncidentAnalysis> {
    try {
      console.log('AIService.analyzeIncident called with:', request);
      
      const response = await httpClient.post<IncidentAnalysis>(
        `${this.API_BASE}/analyze-incident`,
        request
      );
      
      console.log('AIService.analyzeIncident response:', response);
      return response;
    } catch (error) {
      console.error('AIService.analyzeIncident error:', error);
      
      // 提供降级方案
      return this.getFallbackIncidentAnalysis(request);
    }
  }

  /**
   * 趋势分析
   * 分析工单和事件的趋势，提供预测和建议
   */
  static async analyzeTrends(period: string = '30d'): Promise<TrendAnalysis> {
    try {
      console.log('AIService.analyzeTrends called with period:', period);
      
      const response = await httpClient.get<TrendAnalysis>(
        `${this.API_BASE}/analyze-trends?period=${period}`
      );
      
      console.log('AIService.analyzeTrends response:', response);
      return response;
    } catch (error) {
      console.error('AIService.analyzeTrends error:', error);
      
      // 提供降级方案
      return this.getFallbackTrendAnalysis();
    }
  }

  /**
   * 智能搜索
   * 基于自然语言查询搜索相关工单、知识库等
   */
  static async intelligentSearch(query: string, filters?: {
    type?: 'tickets' | 'knowledge' | 'incidents' | 'all';
    dateRange?: { start: string; end: string };
    category?: string;
  }): Promise<{
    tickets: SearchResult[];
    knowledge: SearchResult[];
    incidents: SearchResult[];
    suggestions: string[];
  }> {
    try {
      console.log('AIService.intelligentSearch called with:', { query, filters });
      
      const response = await httpClient.post<{
        tickets: SearchResult[];
        knowledge: SearchResult[];
        incidents: SearchResult[];
        suggestions: string[];
      }>(`${this.API_BASE}/intelligent-search`, {
        query,
        filters
      });
      
      console.log('AIService.intelligentSearch response:', response);
      return response;
    } catch (error) {
      console.error('AIService.intelligentSearch error:', error);
      
      // 提供降级方案
      return {
        tickets: [],
        knowledge: [],
        incidents: [],
        suggestions: [`搜索"${query}"的相关内容`, '尝试使用更具体的关键词', '检查拼写是否正确']
      };
    }
  }

  // 降级方案实现
  private static getFallbackClassification(_request: TicketAnalysisRequest): TicketClassificationResult {
    // 基于关键词的简单分类逻辑
    const title = _request.title.toLowerCase();
    const description = _request.description.toLowerCase();
    const content = `${title} ${description}`;

    let category = 'general';
    let priority: 'low' | 'medium' | 'high' | 'critical' = 'medium';

    // 简单的关键词匹配
    if (content.includes('网络') || content.includes('连接') || content.includes('网速')) {
      category = 'network';
    } else if (content.includes('软件') || content.includes('应用') || content.includes('系统')) {
      category = 'software';
    } else if (content.includes('硬件') || content.includes('设备') || content.includes('电脑')) {
      category = 'hardware';
    } else if (content.includes('账号') || content.includes('密码') || content.includes('权限')) {
      category = 'access';
    }

    // 优先级判断
    if (content.includes('紧急') || content.includes('严重') || content.includes('无法工作')) {
      priority = 'high';
    } else if (content.includes('影响') || content.includes('重要')) {
      priority = 'medium';
    }

    return {
      category,
      priority,
      urgency: priority,
      confidence: 0.6,
      reasoning: '基于关键词匹配的基础分类',
      suggestions: [
        '建议提供更详细的问题描述',
        '如有截图或错误信息请一并提供',
        '说明问题的影响范围和紧急程度'
      ]
    };
  }

  private static getFallbackAssigneeRecommendations(_request: AssigneeRecommendationRequest): AssigneeRecommendation[] {
    // 模拟推荐数据
    const mockRecommendations: AssigneeRecommendation[] = [
      {
        userId: 1,
        userName: '张工程师',
        department: 'IT支持部',
        expertise: ['网络问题', '系统维护'],
        workload: 75,
        availability: 'available',
        confidence: 0.7,
        reasoning: '具有相关技能且当前工作负载适中',
        suggestions: ['建议优先分配给该工程师']
      },
      {
        userId: 2,
        userName: '李技术员',
        department: 'IT支持部',
        expertise: ['软件问题', '用户支持'],
        workload: 60,
        availability: 'available',
        confidence: 0.6,
        reasoning: '工作负载较轻，可以及时处理',
        suggestions: ['适合处理一般性技术问题']
      }
    ];

    return mockRecommendations;
  }

  private static getFallbackSolutions(_request: SolutionSearchRequest): SolutionSuggestion[] {
    // 模拟解决方案数据
    const mockSolutions: SolutionSuggestion[] = [
      {
        solutionId: 'sol_001',
        title: '常见网络连接问题解决方案',
        description: '针对网络连接问题的标准处理流程',
        steps: [
          '检查网络连接状态',
          '重启网络适配器',
          '检查IP配置',
          '联系网络管理员'
        ],
        estimatedTime: 30,
        successRate: 0.85,
        relatedKnowledge: ['网络故障排除指南', 'IP配置手册'],
        confidence: 0.8,
        reasoning: '基于历史相似问题的解决方案',
        suggestions: ['建议按步骤逐一排查']
      }
    ];

    return mockSolutions;
  }

  private static getFallbackIncidentAnalysis(_request: IncidentAnalysisRequest): IncidentAnalysis {
    return {
      severity: 'medium',
      impactScope: ['用户工作站', '网络连接'],
      rootCauseHypothesis: [
        '网络设备故障',
        '配置错误',
        '软件冲突'
      ],
      recommendedActions: [
        '立即检查网络设备状态',
        '收集更多用户反馈',
        '准备回滚方案'
      ],
      escalationRequired: false,
      confidence: 0.6,
      reasoning: '基于事件描述的初步分析',
      suggestions: [
        '建议收集更多技术细节',
        '监控系统状态变化',
        '准备通知相关用户'
      ]
    };
  }

  private static getFallbackTrendAnalysis(): TrendAnalysis {
    return {
      period: '30d',
      trends: [
        {
          category: '网络问题',
          trend: 'increasing',
          changePercentage: 15,
          prediction: '预计下月将继续增长'
        },
        {
          category: '软件问题',
          trend: 'stable',
          changePercentage: 2,
          prediction: '保持稳定水平'
        }
      ],
      insights: [
        '网络相关问题呈上升趋势',
        '软件问题处理效率有所提升',
        '用户满意度整体稳定'
      ],
      recommendations: [
        '加强网络基础设施监控',
        '提供网络使用培训',
        '优化问题处理流程'
      ]
    };
  }
}

// 导出AI服务实例
export const aiService = AIService;