/**
 * 工单审批流程 API 服务
 */

import { httpClient } from './http-client';

export interface ApprovalWorkflow {
  id: number;
  name: string;
  description: string;
  ticket_type?: string;
  priority?: string;
  nodes: ApprovalNode[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ApprovalNode {
  id: string;
  level: number;
  name: string;
  approver_type: 'user' | 'role' | 'department' | 'dynamic';
  approver_ids: number[];
  approver_names: string[];
  approval_mode: 'sequential' | 'parallel' | 'any' | 'all';
  minimum_approvals?: number;
  timeout_hours?: number;
  conditions?: Array<{
    field: string;
    operator: 'equals' | 'not_equals' | 'greater_than' | 'less_than' | 'contains';
    value: any;
  }>;
  allow_reject: boolean;
  allow_delegate: boolean;
  reject_action: 'end' | 'return' | 'custom';
  return_to_level?: number;
}

export interface ApprovalRecord {
  id: number;
  ticket_id: number;
  ticket_number: string;
  ticket_title: string;
  workflow_id: number;
  workflow_name: string;
  current_level: number;
  total_levels: number;
  approver_id: number;
  approver_name: string;
  status: 'pending' | 'approved' | 'rejected' | 'delegated' | 'timeout';
  action?: string;
  comment?: string;
  created_at: string;
  processed_at?: string;
}

export class TicketApprovalApi {
  // 获取审批工作流列表
  static async getWorkflows(params?: {
    ticket_type?: string;
    priority?: string;
    is_active?: boolean;
    page?: number;
    page_size?: number;
  }): Promise<{
    items: ApprovalWorkflow[];
    total: number;
    page: number;
    page_size: number;
  }> {
    return httpClient.get('/api/tickets/approval/workflows', params);
  }

  // 创建审批工作流
  static async createWorkflow(data: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    return httpClient.post<ApprovalWorkflow>('/api/tickets/approval/workflows', data);
  }

  // 更新审批工作流
  static async updateWorkflow(id: number, data: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    return httpClient.put<ApprovalWorkflow>(`/api/tickets/approval/workflows/${id}`, data);
  }

  // 删除审批工作流
  static async deleteWorkflow(id: number): Promise<void> {
    return httpClient.delete(`/api/tickets/approval/workflows/${id}`);
  }

  // 获取审批记录
  static async getApprovalRecords(params?: {
    ticket_id?: number;
    workflow_id?: number;
    status?: string;
    page?: number;
    page_size?: number;
  }): Promise<{
    items: ApprovalRecord[];
    total: number;
    page: number;
    page_size: number;
  }> {
    return httpClient.get('/api/tickets/approval/records', params);
  }

  // 提交审批
  static async submitApproval(data: {
    ticket_id: number;
    approval_id: number;
    action: 'approve' | 'reject' | 'delegate';
    comment: string;
    delegate_to_user_id?: number;
  }): Promise<void> {
    return httpClient.post('/api/tickets/approval/submit', data);
  }
}

