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
  problem_id: number;
  investigator_id: number;
  investigator_name?: string;
  status: InvestigationStatus;
  start_date?: string;
  estimated_completion_date?: string;
  actual_completion_date?: string;
  investigation_summary?: string;
  created_at: string;
  updated_at: string;
}

// 调查步骤
export interface InvestigationStep {
  id: number;
  investigation_id: number;
  step_number: number;
  step_title: string;
  step_description: string;
  status: StepStatus;
  assigned_to?: number;
  assigned_to_name?: string;
  start_date?: string;
  completion_date?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

// 根本原因分析
export interface RootCauseAnalysis {
  id: number;
  problem_id: number;
  analyst_id: number;
  analyst_name?: string;
  analysis_method: string;
  root_cause_description: string;
  contributing_factors?: string;
  evidence?: string;
  confidence_level: ConfidenceLevel;
  analysis_date: string;
  reviewed_by?: number;
  reviewed_by_name?: string;
  review_date?: string;
  created_at: string;
  updated_at: string;
}

// 问题解决方案
export interface ProblemSolution {
  id: number;
  problem_id: number;
  solution_type: SolutionType;
  solution_description: string;
  proposed_by: number;
  proposed_by_name?: string;
  proposed_date: string;
  status: SolutionStatus;
  priority: string;
  estimated_effort_hours?: number;
  estimated_cost?: number;
  risk_assessment?: string;
  approval_status: string;
  approved_by?: number;
  approved_by_name?: string;
  approval_date?: string;
  created_at: string;
  updated_at: string;
}

// 解决方案实施
export interface SolutionImplementation {
  id: number;
  solution_id: number;
  implementer_id: number;
  implementer_name?: string;
  implementation_status: ImplementationStatus;
  start_date?: string;
  completion_date?: string;
  actual_effort_hours?: number;
  actual_cost?: number;
  implementation_notes?: string;
  challenges_encountered?: string;
  lessons_learned?: string;
  created_at: string;
  updated_at: string;
}

// 问题关联
export interface ProblemRelationship {
  id: number;
  problem_id: number;
  related_type: string; // 'ticket' | 'change' | 'incident'
  related_id: number;
  related_title?: string;
  relationship_type: string;
  description?: string;
  created_at: string;
}

// 问题知识库文章
export interface ProblemKnowledgeArticle {
  id: number;
  problem_id: number;
  article_title: string;
  article_content: string;
  article_type: string;
  author_id: number;
  author_name?: string;
  status: string;
  published_date?: string;
  tags: string[];
  view_count: number;
  helpful_count: number;
  created_at: string;
  updated_at: string;
}

// 问题调查摘要
export interface ProblemInvestigationSummary {
  investigation?: ProblemInvestigation;
  steps: InvestigationStep[];
  root_cause_analysis?: RootCauseAnalysis;
  solutions: ProblemSolution[];
  implementations: SolutionImplementation[];
  relationships: ProblemRelationship[];
  knowledge_articles: ProblemKnowledgeArticle[];
}

// 创建问题调查请求
export interface CreateInvestigationRequest {
  problem_id: number;
  investigator_id?: number;
  estimated_completion_date?: string;
  investigation_summary?: string;
}

// 更新问题调查请求
export interface UpdateInvestigationRequest {
  status?: InvestigationStatus;
  estimated_completion_date?: string;
  actual_completion_date?: string;
  investigation_summary?: string;
}

// 创建调查步骤请求
export interface CreateStepRequest {
  investigation_id: number;
  step_number: number;
  step_title: string;
  step_description: string;
  assigned_to?: number;
  notes?: string;
}

// 更新调查步骤请求
export interface UpdateStepRequest {
  step_title?: string;
  step_description?: string;
  status?: StepStatus;
  assigned_to?: number;
  start_date?: string;
  completion_date?: string;
  notes?: string;
}

// 创建根本原因分析请求
export interface CreateRootCauseRequest {
  problem_id: number;
  analyst_id?: number;
  analysis_method: string;
  root_cause_description: string;
  contributing_factors?: string;
  evidence?: string;
  confidence_level: ConfidenceLevel;
}

// 更新根本原因分析请求
export interface UpdateRootCauseRequest {
  analysis_method?: string;
  root_cause_description?: string;
  contributing_factors?: string;
  evidence?: string;
  confidence_level?: ConfidenceLevel;
  reviewed_by?: number;
  review_date?: string;
}

// 创建解决方案请求
export interface CreateSolutionRequest {
  problem_id: number;
  solution_type: SolutionType;
  solution_description: string;
  priority: string;
  proposed_by?: number;
  estimated_effort_hours?: number;
  estimated_cost?: number;
  risk_assessment?: string;
}

// 更新解决方案请求
export interface UpdateSolutionRequest {
  solution_type?: SolutionType;
  solution_description?: string;
  priority?: string;
  status?: SolutionStatus;
  estimated_effort_hours?: number;
  estimated_cost?: number;
  risk_assessment?: string;
}

// 创建知识库文章请求
export interface CreateKnowledgeArticleRequest {
  problem_id: number;
  article_title: string;
  article_content: string;
  article_type: string;
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
    problem_id: number;
    related_type: string;
    related_id: number;
    relationship_type: string;
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
