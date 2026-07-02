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

type RawAssignRecommendation = Partial<AssignRecommendation> & {
  userId?: number;
  username?: string;
  name?: string;
  userName?: string;
  userEmail?: string;
  userAvatar?: string;
  workload?: number;
  skills?: string[];
};

export interface AutoAssignResponse {
  ticket_id: number;
  ticketId?: number;
  assigned_to?: number;
  assignedTo?: number;
  assignment_type: 'rule' | 'smart' | 'manual';
  assignmentType?: 'rule' | 'smart' | 'manual';
  reason: string;
  score?: number;
}

// 条件配置类型
export interface ConditionConfig {
  field: string;
  operator:
    | 'equals'
    | 'not_equals'
    | 'contains'
    | 'not_contains'
    | 'greater_than'
    | 'less_than'
    | 'in'
    | 'not_in';
  value: string | number | boolean | string[] | number[];
}

// 动作配置类型
export interface ActionConfig {
  type: 'user' | 'round_robin' | 'load_balance' | 'assign' | 'set_priority' | 'set_status' | 'notify' | 'escalate';
  value?: number | number[] | string | boolean;
  params?: {
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
  actions: ActionConfig;
  is_active: boolean;
  isActive?: boolean;
  execution_count: number;
  executionCount?: number;
  last_executed_at?: string;
  lastExecutedAt?: string;
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;
}

export interface CreateAssignmentRuleRequest {
  name: string;
  description?: string;
  priority: number;
  conditions: ConditionConfig[];
  actions: ActionConfig;
  is_active?: boolean;
  isActive?: boolean;
}

export interface UpdateAssignmentRuleRequest {
  name?: string;
  description?: string;
  priority?: number;
  conditions?: ConditionConfig[];
  actions?: ActionConfig;
  is_active?: boolean;
  isActive?: boolean;
}

export interface TestAssignmentRuleRequest {
  rule_id: number;
  ruleId?: number;
  ticket_id: number;
  ticketId?: number;
}

export interface TestAssignmentRuleResponse {
  matched: boolean;
  assigned_to?: number;
  assignedTo?: number;
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

function normalizeRecommendation(item: RawAssignRecommendation): AssignRecommendation {
  return {
    user_id: item.user_id ?? item.userId ?? 0,
    user_name: item.user_name ?? item.userName ?? item.name ?? item.username ?? `用户 ${item.userId ?? item.user_id ?? ''}`.trim(),
    user_email: item.user_email ?? item.userEmail,
    user_avatar: item.user_avatar ?? item.userAvatar,
    score: item.score ?? 0,
    reason: item.reason ?? '',
    factors: item.factors ?? {
      workload: item.workload,
      skill_match: item.skills?.length ? 100 : undefined,
    },
  };
}

function normalizeAutoAssign(response: AutoAssignResponse): AutoAssignResponse {
  const assignedTo = response.assigned_to ?? response.assignedTo;
  const ticketId = response.ticket_id ?? response.ticketId ?? 0;
  const assignmentType = response.assignment_type ?? response.assignmentType ?? 'smart';

  return {
    ...response,
    ticket_id: ticketId,
    ticketId,
    assigned_to: assignedTo,
    assignedTo,
    assignment_type: assignmentType,
    assignmentType,
  };
}

function normalizeRule(rule: AssignmentRule): AssignmentRule {
  const rawRule = rule as AssignmentRule & {
    isActive?: boolean;
    executionCount?: number;
    lastExecutedAt?: string;
    createdAt?: string;
    updatedAt?: string;
  };

  return {
    ...rawRule,
    actions: rawRule.actions || { type: 'user', value: 0 },
    conditions: rawRule.conditions || [],
    is_active: rawRule.is_active ?? rawRule.isActive ?? false,
    isActive: rawRule.isActive ?? rawRule.is_active ?? false,
    execution_count: rawRule.execution_count ?? rawRule.executionCount ?? 0,
    executionCount: rawRule.executionCount ?? rawRule.execution_count ?? 0,
    last_executed_at: rawRule.last_executed_at ?? rawRule.lastExecutedAt,
    lastExecutedAt: rawRule.lastExecutedAt ?? rawRule.last_executed_at,
    created_at: rawRule.created_at ?? rawRule.createdAt ?? '',
    createdAt: rawRule.createdAt ?? rawRule.created_at ?? '',
    updated_at: rawRule.updated_at ?? rawRule.updatedAt ?? '',
    updatedAt: rawRule.updatedAt ?? rawRule.updated_at ?? '',
  };
}

function toBackendRulePayload<T extends CreateAssignmentRuleRequest | UpdateAssignmentRuleRequest>(
  data: T
): T {
  const payload = { ...data } as T;
  if ('is_active' in payload && payload.is_active !== undefined) {
    payload.isActive = payload.is_active;
  }
  return payload;
}

export class TicketAssignmentApi {
  /**
   * 自动分配工单
   */
  static async autoAssign(ticketId: number): Promise<AutoAssignResponse> {
    const response = await httpClient.post<AutoAssignResponse>(`/api/v1/tickets/${ticketId}/auto-assign`);
    return normalizeAutoAssign(response);
  }

  /**
   * 获取分配推荐
   */
  static async getRecommendations(ticketId: number): Promise<GetAssignRecommendationsResponse> {
    const response = await httpClient.get<GetAssignRecommendationsResponse>(
      `/api/v1/tickets/assign-recommendations/${ticketId}`
    );
    return {
      ...response,
      recommendations: (response.recommendations || []).map(normalizeRecommendation),
      total: response.total ?? response.recommendations?.length ?? 0,
    };
  }

  /**
   * 获取分配规则列表
   */
  static async listRules(): Promise<ListAssignmentRulesResponse> {
    const response = await httpClient.get<ListAssignmentRulesResponse>('/api/v1/tickets/assignment-rules');
    return {
      ...response,
      rules: (response.rules || []).map(normalizeRule),
      total: response.total ?? response.rules?.length ?? 0,
    };
  }

  /**
   * 获取分配规则详情
   */
  static async getRule(ruleId: number): Promise<AssignmentRule> {
    const response = await httpClient.get<AssignmentRule>(`/api/v1/tickets/assignment-rules/${ruleId}`);
    return normalizeRule(response);
  }

  /**
   * 创建分配规则
   */
  static async createRule(data: CreateAssignmentRuleRequest): Promise<AssignmentRule> {
    const response = await httpClient.post<AssignmentRule>(
      '/api/v1/tickets/assignment-rules',
      toBackendRulePayload(data)
    );
    return normalizeRule(response);
  }

  /**
   * 更新分配规则
   */
  static async updateRule(
    ruleId: number,
    data: UpdateAssignmentRuleRequest
  ): Promise<AssignmentRule> {
    const response = await httpClient.put<AssignmentRule>(
      `/api/v1/tickets/assignment-rules/${ruleId}`,
      toBackendRulePayload(data)
    );
    return normalizeRule(response);
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
    const response = await httpClient.post<TestAssignmentRuleResponse>(
      '/api/v1/tickets/assignment-rules/test',
      {
        ruleId: data.rule_id ?? data.ruleId,
        ticketId: data.ticket_id ?? data.ticketId,
      }
    );
    return {
      ...response,
      assigned_to: response.assigned_to ?? response.assignedTo,
      assignedTo: response.assignedTo ?? response.assigned_to,
    };
  }
}
