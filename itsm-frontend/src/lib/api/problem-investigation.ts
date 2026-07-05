/**
 * 问题调查 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';

// 问题调查状态
export type InvestigationStatus =
  | 'not_started'
  | 'in_progress'
  | 'on_hold'
  | 'completed'
  | 'cancelled';

// 调查步骤状态
export type StepStatus = 'pending' | 'in_progress' | 'completed' | 'blocked' | 'cancelled';

// 解决方案类型
export type SolutionType = 'workaround' | 'fix' | 'prevention' | 'process';

// 解决方案状态
export type SolutionStatus =
  | 'proposed'
  | 'approved'
  | 'pending_implementation'
  | 'in_progress'
  | 'implemented'
  | 'rejected'
  | 'cancelled';

// 实施状态
export type ImplementationStatus =
  | 'not_started'
  | 'in_progress'
  | 'on_hold'
  | 'completed'
  | 'failed';

// 置信度级别
export type ConfidenceLevel = 'low' | 'medium' | 'high';

// 问题调查
export interface ProblemInvestigation {
  id: number;
  problemId: number;
  investigatorId: number;
  investigatorName?: string;
  status: InvestigationStatus;
  startDate?: string;
  estimatedCompletionDate?: string;
  actualCompletionDate?: string;
  investigationSummary?: string;
  createdAt: string;
  updatedAt: string;
}

// 调查步骤
export interface InvestigationStep {
  id: number;
  investigationId: number;
  stepNumber: number;
  stepTitle: string;
  stepDescription: string;
  status: StepStatus;
  assignedTo?: number;
  assignedToName?: string;
  startDate?: string;
  completionDate?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

// 根本原因分析
export interface RootCauseAnalysis {
  id: number;
  problemId: number;
  analystId: number;
  analystName?: string;
  analysisMethod: string;
  rootCauseDescription: string;
  contributingFactors?: string;
  evidence?: string;
  confidenceLevel: ConfidenceLevel;
  analysisDate: string;
  reviewedBy?: number;
  reviewedByName?: string;
  reviewDate?: string;
  createdAt: string;
  updatedAt: string;
}

// 问题解决方案
export interface ProblemSolution {
  id: number;
  problemId: number;
  solutionType: SolutionType;
  solutionDescription: string;
  proposedBy: number;
  proposedByName?: string;
  proposedDate: string;
  status: SolutionStatus;
  priority: string;
  estimatedEffortHours?: number;
  estimatedCost?: number;
  riskAssessment?: string;
  approvalStatus: string;
  approvedBy?: number;
  approvedByName?: string;
  approvalDate?: string;
  createdAt: string;
  updatedAt: string;
}

// 解决方案实施
export interface SolutionImplementation {
  id: number;
  solutionId: number;
  implementerId: number;
  implementerName?: string;
  implementationStatus: ImplementationStatus;
  startDate?: string;
  completionDate?: string;
  actualEffortHours?: number;
  actualCost?: number;
  implementationNotes?: string;
  challengesEncountered?: string;
  lessonsLearned?: string;
  createdAt: string;
  updatedAt: string;
}

// 问题关联
export interface ProblemRelationship {
  id: number;
  problemId: number;
  relatedType: string; // 'ticket' | 'change' | 'incident'
  relatedId: number;
  relatedTitle?: string;
  relationshipType: string;
  description?: string;
  createdAt: string;
}

// 问题知识库文章
export interface ProblemKnowledgeArticle {
  id: number;
  problemId: number;
  articleTitle: string;
  articleContent: string;
  articleType: string;
  authorId: number;
  authorName?: string;
  status: string;
  publishedDate?: string;
  tags: string[];
  viewCount: number;
  helpfulCount: number;
  createdAt: string;
  updatedAt: string;
}

// 问题调查摘要
export interface ProblemInvestigationSummary {
  investigation?: ProblemInvestigation;
  steps: InvestigationStep[];
  rootCauseAnalysis?: RootCauseAnalysis;
  solutions: ProblemSolution[];
  implementations: SolutionImplementation[];
  relationships: ProblemRelationship[];
  knowledgeArticles: ProblemKnowledgeArticle[];
}

// 创建问题调查请求
export interface CreateInvestigationRequest {
  problemId: number;
  investigatorId?: number;
  estimatedCompletionDate?: string;
  investigationSummary?: string;
}

// 更新问题调查请求
export interface UpdateInvestigationRequest {
  status?: InvestigationStatus;
  estimatedCompletionDate?: string;
  actualCompletionDate?: string;
  investigationSummary?: string;
}

// 创建调查步骤请求
export interface CreateStepRequest {
  investigationId: number;
  stepNumber: number;
  stepTitle: string;
  stepDescription: string;
  assignedTo?: number;
  notes?: string;
}

// 更新调查步骤请求
export interface UpdateStepRequest {
  stepTitle?: string;
  stepDescription?: string;
  status?: StepStatus;
  assignedTo?: number;
  startDate?: string;
  completionDate?: string;
  notes?: string;
}

// 创建根本原因分析请求
export interface CreateRootCauseRequest {
  problemId: number;
  analystId?: number;
  analysisMethod: string;
  rootCauseDescription: string;
  contributingFactors?: string;
  evidence?: string;
  confidenceLevel: ConfidenceLevel;
}

// 更新根本原因分析请求
export interface UpdateRootCauseRequest {
  analysisMethod?: string;
  rootCauseDescription?: string;
  contributingFactors?: string;
  evidence?: string;
  confidenceLevel?: ConfidenceLevel;
  reviewedBy?: number;
  reviewDate?: string;
}

// 创建解决方案请求
export interface CreateSolutionRequest {
  problemId: number;
  solutionType: SolutionType;
  solutionDescription: string;
  priority: string;
  proposedBy?: number;
  estimatedEffortHours?: number;
  estimatedCost?: number;
  riskAssessment?: string;
}

// 更新解决方案请求
export interface UpdateSolutionRequest {
  solutionType?: SolutionType;
  solutionDescription?: string;
  priority?: string;
  status?: SolutionStatus;
  estimatedEffortHours?: number;
  estimatedCost?: number;
  riskAssessment?: string;
}

// 创建知识库文章请求
export interface CreateKnowledgeArticleRequest {
  problemId: number;
  articleTitle: string;
  articleContent: string;
  articleType: string;
  tags?: string[];
}

// 问题调查 API
export const ProblemInvestigationAPI = {
  // 获取问题调查摘要
  async getSummary(problemId: number): Promise<ProblemInvestigationSummary> {
    return httpClient.get(`/api/v1/problem-investigation/problems/${problemId}/summary`);
  },

  // 创建问题调查
  async createInvestigation(data: CreateInvestigationRequest): Promise<ProblemInvestigation> {
    return httpClient.post('/api/v1/problem-investigation/investigations', data);
  },

  // 更新问题调查
  async updateInvestigation(
    id: number,
    data: UpdateInvestigationRequest
  ): Promise<ProblemInvestigation> {
    return httpClient.put(`/api/v1/problem-investigation/investigations/${id}`, data);
  },

  // 获取调查步骤列表
  async getSteps(investigationId: number): Promise<InvestigationStep[]> {
    return httpClient.get(`/api/v1/problem-investigation/investigations/${investigationId}/steps`);
  },

  // 创建调查步骤
  async createStep(data: CreateStepRequest): Promise<InvestigationStep> {
    return httpClient.post('/api/v1/problem-investigation/steps', data);
  },

  // 更新调查步骤
  async updateStep(id: number, data: UpdateStepRequest): Promise<InvestigationStep> {
    return httpClient.put(`/api/v1/problem-investigation/steps/${id}`, data);
  },

  // 创建根本原因分析
  async createRootCause(data: CreateRootCauseRequest): Promise<RootCauseAnalysis> {
    return httpClient.post('/api/v1/problem-investigation/root-cause-analysis', data);
  },

  // 更新根本原因分析
  async updateRootCause(id: number, data: UpdateRootCauseRequest): Promise<RootCauseAnalysis> {
    return httpClient.put(`/api/v1/problem-investigation/root-cause-analysis/${id}`, data);
  },

  // 获取解决方案列表
  async getSolutions(problemId: number): Promise<ProblemSolution[]> {
    return httpClient.get(`/api/v1/problem-investigation/problems/${problemId}/solutions`);
  },

  // 创建解决方案
  async createSolution(data: CreateSolutionRequest): Promise<ProblemSolution> {
    return httpClient.post('/api/v1/problem-investigation/solutions', data);
  },

  // 更新解决方案
  async updateSolution(id: number, data: UpdateSolutionRequest): Promise<ProblemSolution> {
    return httpClient.put(`/api/v1/problem-investigation/solutions/${id}`, data);
  },

  // 获取关联列表
  async getRelationships(problemId: number): Promise<ProblemRelationship[]> {
    return httpClient.get(`/api/v1/problems/${problemId}/relationships`);
  },

  // 创建关联
  async createRelationship(data: {
    problemId: number;
    relatedType: string;
    relatedId: number;
    relationshipType: string;
    description?: string;
  }): Promise<ProblemRelationship> {
    return httpClient.post('/api/v1/problem-relationships', data);
  },

  // 创建知识库文章
  async createKnowledgeArticle(
    data: CreateKnowledgeArticleRequest
  ): Promise<ProblemKnowledgeArticle> {
    return httpClient.post('/api/v1/problem-knowledge-articles', data);
  },

  // 获取知识库文章列表
  async getKnowledgeArticles(problemId: number): Promise<ProblemKnowledgeArticle[]> {
    return httpClient.get(`/api/v1/problem-knowledge-articles/problems/${problemId}`);
  },
};

export default ProblemInvestigationAPI;
