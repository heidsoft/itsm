/**
 * 变更管理类型定义
 */

import type {
  ChangeStatus,
  ChangeType,
  ChangePriority,
  ChangeImpact,
  ChangeRisk,
} from '@/constants/change';

// 变更实体
export interface Change {
  id: number;
  title: string;
  description: string;
  justification: string;
  type: ChangeType;
  status: ChangeStatus;
  priority: ChangePriority;
  impactScope: ChangeImpact;
  riskLevel: ChangeRisk;
  assigneeId?: number;
  assigneeName?: string;
  createdBy: number;
  createdByName: string;
  tenantId: number;
  plannedStartDate?: string;
  plannedEndDate?: string;
  actualStartDate?: string;
  actualEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCis?: string[];
  relatedTickets?: string[];
  createdAt: string;
  updatedAt: string;
}

// 审批记录
export interface ApprovalRecord {
  id: number;
  changeId: number;
  approverId: number;
  approverName: string;
  status: ChangeStatus;
  comment?: string;
  approvedAt?: string;
  createdAt: string;
}

// 审批链/工作流项
export interface ApprovalChainItem {
  id: number;
  level: number;
  approverId: number;
  approverName: string;
  role: string;
  status: string;
  isRequired: boolean;
}

// 风险评估
export interface RiskAssessment {
  id: number;
  changeId: number;
  riskLevel: ChangeRisk;
  riskDescription: string;
  impactAnalysis: string;
  mitigationMeasures: string;
  contingencyPlan?: string;
  riskOwner: string;
  riskReviewDate?: string;
}

// 创建变更请求
export interface CreateChangeRequest {
  title: string;
  description: string;
  justification: string;
  type: ChangeType;
  priority: ChangePriority;
  impactScope: ChangeImpact;
  riskLevel: ChangeRisk;
  plannedStartDate?: string;
  plannedEndDate?: string;
  implementationPlan: string;
  rollbackPlan: string;
  affectedCis?: string[];
  relatedTickets?: string[];
}

// 列表查询参数
export interface ChangeQuery {
  page?: number;
  pageSize?: number;
  status?: string;
  type?: string;
  search?: string;
}

// 列表响应
export interface ChangeListResponse {
  changes: Change[];
  total: number;
  page: number;
  size: number;
}

// 统计响应
export interface ChangeStats {
  total: number;
  pending: number;
  approved: number;
  inProgress: number;
  completed: number;
  rolledBack: number;
  rejected: number;
  cancelled: number;
}
