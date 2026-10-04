/**
 * 工单智能分配API
 * 提供自动分配、分配推荐、分配规则管理功能
 */

import { httpClient } from './http-client';

export interface AutoAssignResponse {
  ticketId: number;
  assignedTo?: number;
  // 后端实测 emits: auto / rule / ticket_type_rule / manual（service/ticket_assignment_*.go）
  assignmentType: 'auto' | 'rule' | 'ticket_type_rule' | 'manual';
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

// 与后端 dto.AssignmentRecommendation 逐字段一致，禁止自行发明 userName/factors。
export interface AssignRecommendation {
  userId: number;
  username: string;
  name: string;
  email: string;
  score: number;
  reason: string;
  workload: number;
  skills?: string[];
  categories?: number[];
}

// 该端点实测不分页（后端 handler 用 Total: len(items)），契约就是诚实的两键。
export interface AssignRecommendationListResponse {
  items: AssignRecommendation[];
  total: number;
}

// 该端点实测不分页（后端 handler 用 Total: len(items)），契约就是诚实的两键。
export interface ListAssignmentRulesResponse {
  items: AssignmentRule[];
  total: number;
}

function normalizeRule(rule: AssignmentRule): AssignmentRule {
  // 后端 AssignmentRuleResponse 里 isActive/executionCount/createdAt/updatedAt 是非指针
  // 字段，响应里必然存在，无需兜底；只有 conditions/actions 可能为 nil，lastExecutedAt
  // 是 *string,omitempty。
  return {
    ...rule,
    actions: rule.actions ?? { type: 'user', value: 0 },
    conditions: rule.conditions ?? [],
  };
}

function toBackendRulePayload<T extends CreateAssignmentRuleRequest | UpdateAssignmentRuleRequest>(
  data: T
): T {
  // 规整规则创建/更新请求：所有键已经是 camelCase，body 直接透传。
  return { ...data };
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
  static async getRecommendations(ticketId: number): Promise<AssignRecommendationListResponse> {
    return httpClient.get<AssignRecommendationListResponse>(
      `/api/v1/tickets/assign-recommendations/${ticketId}`
    );
  }

  /**
   * 获取分配规则列表
   */
  static async listRules(): Promise<ListAssignmentRulesResponse> {
    const response = await httpClient.get<ListAssignmentRulesResponse>('/api/v1/tickets/assignment-rules');
    return {
      items: response.items.map(normalizeRule),
      total: response.total,
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
    return httpClient.post<TestAssignmentRuleResponse>('/api/v1/tickets/assignment-rules/test', {
      ruleId: data.ruleId,
      ticketId: data.ticketId,
    });
  }
}
