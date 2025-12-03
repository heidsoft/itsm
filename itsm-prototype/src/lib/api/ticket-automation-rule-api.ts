/**
 * 工单自动化规则API
 * 提供自动化规则的创建、查询、更新、删除和测试功能
 */

import { httpClient } from './http-client';

export interface AutomationRule {
  id: number;
  name: string;
  description?: string;
  priority: number;
  conditions: Array<{
    field: string;
    operator: string;
    value: any;
  }>;
  actions: Array<{
    type: string;
    [key: string]: any;
  }>;
  is_active: boolean;
  execution_count: number;
  last_executed_at?: string;
  created_by: number;
  creator?: {
    id: number;
    username: string;
    name: string;
    email: string;
    role: string;
  };
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAutomationRuleRequest {
  name: string;
  description?: string;
  priority: number;
  conditions: Array<{
    field: string;
    operator: string;
    value: any;
  }>;
  actions: Array<{
    type: string;
    [key: string]: any;
  }>;
  is_active: boolean;
}

export interface UpdateAutomationRuleRequest {
  name?: string;
  description?: string;
  priority?: number;
  conditions?: Array<{
    field: string;
    operator: string;
    value: any;
  }>;
  actions?: Array<{
    type: string;
    [key: string]: any;
  }>;
  is_active?: boolean;
}

export interface TestAutomationRuleRequest {
  rule_id: number;
  ticket_id: number;
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
  static async updateRule(ruleId: number, data: UpdateAutomationRuleRequest): Promise<AutomationRule> {
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
    return httpClient.post<TestAutomationRuleResponse>(`/api/v1/tickets/automation-rules/${data.rule_id}/test`, {
      ticket_id: data.ticket_id,
    });
  }
}

