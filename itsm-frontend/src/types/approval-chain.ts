/**
 * 审批链类型定义
 */

import type { BaseEntity } from './common';

export interface ApprovalChain extends BaseEntity {
  name: string;
  description?: string;
  entityType?: string;
  status?: string;
  isActive: boolean;
  tenantId: number;
  steps: ApprovalStep[];
  createdAt: string;
  updatedAt: string;
}

export interface ApprovalStep extends BaseEntity {
  chainId: number;
  /** 审批层级：同层多个审批人共享同一 level，由 approvalType 决定或签/会签语义 */
  level: number;
  stepOrder: number;
  stepName: string;
  approverType: 'user' | 'role';
  approverId: number;
  approverName: string;
  isRequired: boolean;
  /** 或签 serial：任一人通过即可；会签 parallel：需 threshold 人通过 */
  approvalType?: ApprovalType;
  /** 会签通过人数；为空时后端按该层审批人总数处理（全员通过） */
  threshold?: number;
  /** 无审批人且本层必需时的兜底策略 */
  fallbackAction?: FallbackAction;
  fallbackApproverId?: number;
  fallbackRole?: string;
  /** 动态适配条件：优先级白名单（任一匹配即适用本层） */
  conditionPriorities?: string[];
  /** 适用金额闭区间，0 表示该侧无限制 */
  conditionAmountMin?: number;
  conditionAmountMax?: number;
}

/**
 * 与后端 approval_chain_evaluation.go 一致：
 * serial 逐层串行（层内阈值 1）；parallel/all 为会签（阈值默认=本层审批人数）；or 为或签（阈值 1）。
 */
export type ApprovalType = 'serial' | 'parallel' | 'all' | 'or';

export type FallbackAction = 'block' | 'auto_approve' | 'escalate' | 'auto_reject';

export const APPROVAL_TYPE_LABELS: Record<ApprovalType, string> = {
  serial: '顺序审批',
  parallel: '并行会签（达到阈值通过）',
  all: '全员通过（会签）',
  or: '任一通过（或签）',
};

/** 会签类审批方式需要配置阈值；serial/or 的阈值在后端恒为 1 */
export const QUORUM_APPROVAL_TYPES: ApprovalType[] = ['parallel', 'all'];

export const ENTITY_TYPE_LABELS: Record<string, string> = {
  ticket: '工单',
  incident: '事件',
  problem: '问题',
  change: '变更',
  service_request: '服务请求',
};

/** 未指定时按默认工单展示；未知取值原样回显，避免把其它实体类型误标成工单 */
export const entityTypeLabel = (entityType?: string): string =>
  entityType ? ENTITY_TYPE_LABELS[entityType] ?? entityType : ENTITY_TYPE_LABELS.ticket;

export const FALLBACK_ACTION_LABELS: Record<FallbackAction, string> = {
  block: '阻断（无人可审时报错）',
  escalate: '升级到指定审批人/角色',
  auto_approve: '自动通过',
  auto_reject: '自动拒绝',
};

/** 与 ITSM 实体 priority 实际取值一致（后端按不区分大小写比较） */
export const APPROVAL_CONDITION_PRIORITIES = ['low', 'medium', 'high', 'critical'] as const;

/** 优先级展示文案；配置值与后端比较仍使用英文枚举 */
export const PRIORITY_LABELS: Record<string, string> = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '紧急',
};

export interface ApprovalChainStats {
  total: number;
  active: number;
  inactive: number;
  totalSteps: number;
  avgStepsPerChain: number;
}

/** 只承载后端真实支持的筛选条件：name 为子串匹配，entityType/status 为精确匹配 */
export interface ApprovalChainFilters {
  name?: string;
  entityType?: string;
  status?: 'active' | 'inactive';
}

export interface CreateApprovalChainRequest {
  name: string;
  description?: string;
  entityType?: string;
  isActive: boolean;
  steps: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[];
}

export interface UpdateApprovalChainRequest {
  name?: string;
  description?: string;
  entityType?: string;
  isActive?: boolean;
  steps?: Omit<ApprovalStep, 'id' | 'chainId' | 'createdAt' | 'updatedAt'>[];
}
