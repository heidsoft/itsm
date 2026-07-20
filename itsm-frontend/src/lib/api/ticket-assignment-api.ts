/**
 * 工单智能分配API
 * 提供自动分配、分配推荐、分配规则管理功能
 */

import { httpClient } from './http-client';

export interface AssignRecommendation {
  userId: number;
  userName: string;
  userEmail?: string;
  userAvatar?: string;
  score: number;
  reason: string;
  factors: {
    skillMatch?: number;
    workload?: number;
    historySuccess?: number;
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
  ticketId: number;
  assignedTo?: number;
  assignmentType: 'rule' | 'smart' | 'manual';
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
    assigneeId?: number;
    priority?: string;
    status?: string;
    notifyUsers?: number[];
    escalationLevel?: number;
  };
}

export interface AssignmentRule {
  id: number;
  name: string;
  description?: string;
  priority: number;
  conditions: ConditionConfig[];
  actions: ActionConfig;
  isActive: boolean;
  executionCount: number;
  lastExecutedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAssignmentRuleRequest {
  name: string;
  description?: string;
  priority: number;
  conditions: ConditionConfig[];
  actions: ActionConfig;
  isActive?: boolean;
}

export interface UpdateAssignmentRuleRequest {
  name?: string;
  description?: string;
  priority?: number;
  conditions?: ConditionConfig[];
  actions?: ActionConfig;
  isActive?: boolean;
}

export interface TestAssignmentRuleRequest {
  ruleId: number;
  ticketId: number;
}

export interface TestAssignmentRuleResponse {
  matched: boolean;
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
    userId: item.userId ?? 0,
    userName: item.userName ?? item.name ?? item.username ?? `用户${item.userId ?? ''}`.trim(),
    userEmail: item.userEmail,
    userAvatar: item.userAvatar,
    score: item.score ?? 0,
    reason: item.reason ?? '',
    factors: item.factors ?? {
      workload: item.workload,
      skillMatch: item.skills?.length ? 100 : undefined,
    },
  };
}

function normalizeAutoAssign(response: AutoAssignResponse): AutoAssignResponse {
  const assignedTo = response.assignedTo;
  const ticketId = response.ticketId ?? 0;
  const assignmentType = response.assignmentType ?? 'smart';

  return {
    ...response,
    ticketId,
    assignedTo,
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
    isActive: rawRule.isActive ?? rawRule.isActive ?? false,
    executionCount: rawRule.executionCount ?? rawRule.executionCount ?? 0,
    lastExecutedAt: rawRule.lastExecutedAt ?? rawRule.lastExecutedAt,
    createdAt: rawRule.createdAt ?? rawRule.createdAt ?? '',
    updatedAt: rawRule.updatedAt ?? rawRule.updatedAt ?? '',
  };
}

function toBackendRulePayload<T extends CreateAssignmentRuleRequest | UpdateAssignmentRuleRequest>(
  data: T
): T {
  const payload = { ...data } as T;
  if ('is_active' in payload && payload.isActive !== undefined) {
    payload.isActive = payload.isActive;
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
        ruleId: data.ruleId ?? data.ruleId,
        ticketId: data.ticketId ?? data.ticketId,
      }
    );
    return {
      ...response,
      assignedTo: response.assignedTo,
    };
  }
}
