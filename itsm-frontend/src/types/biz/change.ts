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
  approvalType?: string;
  threshold?: number;
  createdAt?: string;
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
  items: Change[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 统计响应只有一份形状：`@/lib/api/change-api` 的 `ChangeStatsResponse`，
// 它逐字段镜像后端 `dto.ChangeStatsResponse`。这里只做 re-export，
// 不再维护副本 —— 2026-10-03 之前本文件、`lib/api/change-api.ts` 和
// `lib/services/change-service.ts` 各有一份互不一致的统计类型：
// 只有中间那份声明了后端从不返回的 `implementing`（后端字段是 `inProgress`），
// 报表页因此把「实施中」恒显示为 0，并用 total 的 30/50/20 伪造类型分布。
export type { ChangeStatsResponse as ChangeStats } from '@/lib/api/change-api';
export type { ChangeTypeCount } from '@/lib/api/change-api';
