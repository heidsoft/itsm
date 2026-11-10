/**
 * 变更管理相关常量定义
 * @deprecated 请使用 @/constants/taxonomy 中的统一分类系统
 */

import {
  ChangeType,
  ChangeStatus,
  ChangeTypeConfig,
  ChangeStatusConfig,
} from '@/constants/taxonomy';

// 保持向后兼容的导出
export { ChangeType, ChangeStatus };

// 变更类型配置 (legacy)
export const ChangeTypeLabels: Record<ChangeType, string> = {
  [ChangeType.NORMAL]: 'Normal Change',
  [ChangeType.STANDARD]: 'Standard Change',
  [ChangeType.EMERGENCY]: 'Emergency Change',
};

// 变更优先级 (复用 TicketPriority)
export enum ChangePriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export const ChangePriorityLabels: Record<ChangePriority, string> = {
  [ChangePriority.LOW]: 'Low',
  [ChangePriority.MEDIUM]: 'Medium',
  [ChangePriority.HIGH]: 'High',
  [ChangePriority.CRITICAL]: 'Critical',
};

// 影响范围
export enum ChangeImpact {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
}

export const ChangeImpactLabels: Record<ChangeImpact, string> = {
  [ChangeImpact.LOW]: 'Low',
  [ChangeImpact.MEDIUM]: 'Medium',
  [ChangeImpact.HIGH]: 'High',
};

// 风险等级
export enum ChangeRisk {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
}

export const ChangeRiskLabels: Record<ChangeRisk, string> = {
  [ChangeRisk.LOW]: 'Low',
  [ChangeRisk.MEDIUM]: 'Medium',
  [ChangeRisk.HIGH]: 'High',
};

// 向后兼容的映射
export const ChangeStatusLabels: Record<ChangeStatus, string> = {
  [ChangeStatus.DRAFT]: 'Draft',
  [ChangeStatus.PENDING]: 'Pending Approval',
  [ChangeStatus.APPROVED]: 'Approved',
  [ChangeStatus.REJECTED]: 'Rejected',
  [ChangeStatus.SCHEDULED]: 'Scheduled',
  [ChangeStatus.IN_PROGRESS]: 'In Progress',
  [ChangeStatus.COMPLETED]: 'Completed',
  [ChangeStatus.FAILED]: 'Implementation failed',
  [ChangeStatus.ROLLED_BACK]: 'Rolled Back',
  [ChangeStatus.CANCELLED]: 'Cancelled',
  [ChangeStatus.CLOSED]: 'Closed',
};
