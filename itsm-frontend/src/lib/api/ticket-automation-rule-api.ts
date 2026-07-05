/**
 * 工单自动化规则API
 * 提供自动化规则的创建、查询、更新、删除和测试功能
 */

import { httpClient } from './http-client';

export interface AutomationRule {
  id: number;
  name: string;
  description?: string;
  type?: string;
  priority: number;
  conditions: Array<{
    field: string;
    operator: string;
    value: unknown;
  }>;
  actions: Array<{
    type: string;
    [key: string]: unknown;
  }>;
  isActive: boolean;
  executionCount: number;
  lastExecutedAt?: string;
  createdBy: number;
  creator?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role: string;
  };
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAutomationRuleRequest {
  name: string;
  description?: string;
  type?: string;
  priority: number;
  conditions: Array<{
    field: string;
    operator: string;
    value: unknown;
  }>;
  actions: Array<{
    type: string;
    [key: string]: unknown;
  }>;
  isActive: boolean;
}

export interface UpdateAutomationRuleRequest {
  name?: string;
  description?: string;
  type?: string;
  priority?: number;
  conditions?: Array<{
    field: string;
    operator: string;
    value: unknown;
  }>;
  actions?: Array<{
    type: string;
    [key: string]: unknown;
  }>;
  isActive?: boolean;
}

export interface TestAutomationRuleRequest {
  ruleId: number;
  ticketId: number;
}

export interface TestAutomationRuleResponse {
  matched: boolean;
  actions?: string[];
  reason?: string;
  error?: string;
}

export interface ListAutomationRulesResponse {
  rules: AutomationRule[];
  total: number;
}

export class TicketAutomationRuleApi {
  /**
   * 获取自动化规则列表
   */
  static async listRules(): Promise<ListAutomationRulesResponse> {
    return httpClient.get<ListAutomationRulesResponse>('/api/v1/tickets/automation-rules');
  }

  /**
   * 获取自动化规则详情
   */
  static async getRule(ruleId: number): Promise<AutomationRule> {
    return httpClient.get<AutomationRule>(`/api/v1/tickets/automation-rules/${ruleId}`);
  }

  /**
   * 创建自动化规则
   */
  static async createRule(data: CreateAutomationRuleRequest): Promise<AutomationRule> {
    return httpClient.post<AutomationRule>('/api/v1/tickets/automation-rules', data);
  }

  /**
   * 更新自动化规则
   */
  static async updateRule(
    ruleId: number,
    data: UpdateAutomationRuleRequest
  ): Promise<AutomationRule> {
    return httpClient.put<AutomationRule>(`/api/v1/tickets/automation-rules/${ruleId}`, data);
  }

  /**
   * 删除自动化规则
   */
  static async deleteRule(ruleId: number): Promise<void> {
    return httpClient.delete(`/api/v1/tickets/automation-rules/${ruleId}`);
  }

  /**
   * 测试自动化规则
   */
  static async testRule(data: TestAutomationRuleRequest): Promise<TestAutomationRuleResponse> {
    return httpClient.post<TestAutomationRuleResponse>(
      `/api/v1/tickets/automation-rules/${data.ruleId}/test`,
      {
        ticketId: data.ticketId,
      }
    );
  }
}
