/**
 * 审批链类型定义
 */

import { BaseEntity } from './common';

export interface ApprovalChain extends BaseEntity {
  name: string;
  description?: string;
  isActive: boolean;
  tenantId: number;
  steps: ApprovalStep[];
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalStep extends BaseEntity {
  chainId: number;
  stepOrder: number;
  stepName: string;
  approverType: 'user' | 'role' | 'group';
  approverId: number;
  approverName: string;
  isRequired: boolean;
  timeoutHours?: number;
  conditions?: ApprovalCondition[];
}

export interface ApprovalCondition {
  field: string;
  operator: 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte' | 'in' | 'not_in';
  value: unknown;
}

export interface ApprovalChainStats {
  total: number;
  active: number;
  inactive: number;
  totalSteps: number;
  avgStepsPerChain: number;
}

export interface ApprovalChainFilters {
  status?: ('active' | 'inactive')[];
  name?: string;
  createdBy?: number[];
  dateRange?: {
    field: 'created' | 'updated';
    start: string;
    end: string;
  };
}

export interface CreateApprovalChainRequest {
  name: string;
  description?: string;
  isActive: boolean;
  steps: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[];
}

export interface UpdateApprovalChainRequest {
  name?: string;
  description?: string;
  isActive?: boolean;
  steps?: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[];
}
