/**
 * 工单智能分配API
 * 提供自动分配、分配推荐、分配规则管理功能
 */

import { httpClient } from './http-client';

export interface AssignRecommendation {
  user_id: number;
  user_name: string;
  user_email?: string;
  user_avatar?: string;
  score: number;
  reason: string;
  factors: {
    skill_match?: number;
    workload?: number;
    history_success?: number;
    availability?: number;
  };
}

export interface AutoAssignResponse {
  ticket_id: number;
  assigned_to?: number;
  assignment_type: 'rule' | 'smart' | 'manual';
  reason: string;
  score?: number;
}

// 条件配置类型
export interface ConditionConfig {
  field: string;
  operator: 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'greater_than' | 'less_than' | 'in' | 'not_in';
  value: string | number | boolean | string[] | number[];
}

// 动作配置类型
export interface ActionConfig {
  type: 'assign' | 'set_priority' | 'set_status' | 'notify' | 'escalate';
  params: {
    assignee_id?: number;
    priority?: string;
    status?: string;
    notify_users?: number[];
    escalation_level?: number;
  };
}

export interface AssignmentRule {
  id: number;
  name: string;
  description?: string;
  priority: number;
  conditions: ConditionConfig[];
  actions: ActionConfig[];
  is_active: boolean;
  execution_count: number;
  last_executed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAssignmentRuleRequest {
  name: string;
  description?: string;
  priority: number;
  conditions: ConditionConfig[];
  actions: ActionConfig[];
  is_active: boolean;
}

export interface UpdateAssignmentRuleRequest {
  name?: string;
  description?: string;
  priority?: number;
  conditions?: ConditionConfig[];
  actions?: ActionConfig[];
  is_active?: boolean;
}

export interface TestAssignmentRuleRequest {
  rule_id: number;
  ticket_id: number;
}

export interface TestAssignmentRuleResponse {
  matched: boolean;
  assigned_to?: number;
  reason?: string;
  score?: number;
}

export interface GetAssignRecommendationsResponse {
  recommendations: AssignRecommendation[];
  total: number;
}

export interface ListAssignmentRulesResponse {
  rules: AssignmentRule[];
  total: number;
}

export class TicketAssignmentApi {
  /**
   * 自动分配工单
   */
  static async autoAssign(ticketId: number): Promise<AutoAssignResponse> {
    return httpClient.post<AutoAssignResponse>(`/api/v1/tickets/${ticketId}/auto-assign`);
  }

  /**
   * 获取分配推荐
   */
  static async getRecommendations(ticketId: number): Promise<GetAssignRecommendationsResponse> {
    return httpClient.get<GetAssignRecommendationsResponse>(`/api/v1/tickets/assign-recommendations/${ticketId}`);
  }

  /**
   * 获取分配规则列表
   */
  static async listRules(): Promise<ListAssignmentRulesResponse> {
    return httpClient.get<ListAssignmentRulesResponse>('/api/v1/tickets/assignment-rules');
  }

  /**
   * 获取分配规则详情
   */
  static async getRule(ruleId: number): Promise<AssignmentRule> {
    return httpClient.get<AssignmentRule>(`/api/v1/tickets/assignment-rules/${ruleId}`);
  }

  /**
   * 创建分配规则
   */
  static async createRule(data: CreateAssignmentRuleRequest): Promise<AssignmentRule> {
    return httpClient.post<AssignmentRule>('/api/v1/tickets/assignment-rules', data);
  }

  /**
   * 更新分配规则
   */
  static async updateRule(ruleId: number, data: UpdateAssignmentRuleRequest): Promise<AssignmentRule> {
    return httpClient.put<AssignmentRule>(`/api/v1/tickets/assignment-rules/${ruleId}`, data);
  }

  /**
   * 删除分配规则
   */
  static async deleteRule(ruleId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/assignment-rules/${ruleId}`);
  }

  /**
   * 测试分配规则
   */
  static async testRule(data: TestAssignmentRuleRequest): Promise<TestAssignmentRuleResponse> {
    return httpClient.post<TestAssignmentRuleResponse>('/api/v1/tickets/assignment-rules/test', data);
  }
}

