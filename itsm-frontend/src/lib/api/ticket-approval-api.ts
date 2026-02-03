/**
 * 工单审批流程 API 服务
 */

import { httpClient } from './http-client';

export interface ApprovalWorkflow {
  id: number;
  name: string;
  description: string;
  ticketType?: string;
  priority?: string;
  nodes: ApprovalNode[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalNode {
  id: string;
  level: number;
  name: string;
  approverType: 'user' | 'role' | 'department' | 'dynamic';
  approverIds: number[];
  approverNames: string[];
  approvalMode: 'sequential' | 'parallel' | 'any' | 'all';
  minimumApprovals?: number;
  timeoutHours?: number;
  conditions?: Array<{
    field: string;
    operator: 'equals' | 'not_equals' | 'greater_than' | 'less_than' | 'contains';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    value: any;
  }>;
  allowReject: boolean;
  allowDelegate: boolean;
  rejectAction: 'end' | 'return' | 'custom';
  returnToLevel?: number;
}

export interface ApprovalRecord {
  id: number;
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  workflowId: number;
  workflowName: string;
  currentLevel: number;
  totalLevels: number;
  approverId: number;
  approverName: string;
  status: 'pending' | 'approved' | 'rejected' | 'delegated' | 'timeout';
  action?: string;
  comment?: string;
  createdAt: string;
  processedAt?: string;
}

export class TicketApprovalApi {
  // 获取审批工作流列表
  static async getWorkflows(params?: {
    ticketType?: string;
    priority?: string;
    isActive?: boolean;
    page?: number;
    pageSize?: number;
  }): Promise<{
    items: ApprovalWorkflow[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    // Convert params to snake_case for query string
    const queryParams: Record<string, any> = {};
    if (params) {
      if (params.ticketType) queryParams.ticket_type = params.ticketType;
      if (params.priority) queryParams.priority = params.priority;
      if (params.isActive !== undefined) queryParams.is_active = params.isActive;
      if (params.page) queryParams.page = params.page;
      if (params.pageSize) queryParams.page_size = params.pageSize;
    }
    return httpClient.get('/api/tickets/approval/workflows', queryParams);
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
    ticketId?: number;
    workflowId?: number;
    status?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    items: ApprovalRecord[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    // Convert params to snake_case
    const queryParams: Record<string, any> = {};
    if (params) {
      if (params.ticketId) queryParams.ticket_id = params.ticketId;
      if (params.workflowId) queryParams.workflow_id = params.workflowId;
      if (params.status) queryParams.status = params.status;
      if (params.page) queryParams.page = params.page;
      if (params.pageSize) queryParams.page_size = params.pageSize;
    }
    return httpClient.get('/api/tickets/approval/records', queryParams);
  }

  // 提交审批
  static async submitApproval(data: {
    ticketId: number;
    approvalId: number;
    action: 'approve' | 'reject' | 'delegate';
    comment: string;
    delegateToUserId?: number;
  }): Promise<void> {
    return httpClient.post('/api/tickets/approval/submit', data);
  }
}

