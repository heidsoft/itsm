/**
 * 工作流状态机验证工具
 * 用于验证状态转换的合法性，防止绕过工作流引擎
 */

import { TicketStatus } from '@/constants/taxonomy';

/**
 * 工单状态转换规则
 * 定义每个状态允许转换到的目标状态
 */
export const VALID_TICKET_TRANSITIONS: Record<TicketStatus, TicketStatus[]> = {
  [TicketStatus.NEW]: [TicketStatus.OPEN, TicketStatus.CANCELLED],
  [TicketStatus.OPEN]: [
    TicketStatus.IN_PROGRESS,
    TicketStatus.PENDING,
    TicketStatus.PENDING_APPROVAL,
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.IN_PROGRESS]: [
    TicketStatus.PENDING,
    TicketStatus.PENDING_APPROVAL,
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.PENDING_APPROVAL]: [
    TicketStatus.OPEN, // 审批通过后返回待处理
    TicketStatus.REJECTED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.PENDING]: [
    TicketStatus.IN_PROGRESS,
    TicketStatus.PENDING_APPROVAL,
    TicketStatus.RESOLVED,
    TicketStatus.CANCELLED,
  ],
  [TicketStatus.RESOLVED]: [TicketStatus.CLOSED, TicketStatus.OPEN], // 可重开
  [TicketStatus.CLOSED]: [], // 已关闭不能转换
  [TicketStatus.CANCELLED]: [], // 已取消不能转换
  [TicketStatus.REJECTED]: [TicketStatus.OPEN, TicketStatus.CANCELLED], // 被拒绝可重新打开
};

/**
 * 状态转换动作映射
 * 定义状态转换应该使用的工作流方法，而非直接更新状态
 */
export const STATUS_TRANSITION_ACTIONS: Record<string, string> = {
  // 解决
  'new:resolved': 'resolveTicket',
  'open:resolved': 'resolveTicket',
  'in_progress:resolved': 'resolveTicket',
  'pending:resolved': 'resolveTicket',

  // 关闭
  'resolved:closed': 'closeTicket',

  // 审批
  'pending_approval:approved': 'approveTicket',
  'pending_approval:rejected': 'rejectTicket',

  // 重开
  'resolved:open': 'reopenTicket',
  'rejected:open': 'reopenTicket',
};

/**
 * 验证状态转换是否合法
 */
export function isValidTransition(
  currentStatus: TicketStatus,
  targetStatus: TicketStatus
): boolean {
  const allowedTransitions = VALID_TICKET_TRANSITIONS[currentStatus];
  return allowedTransitions?.includes(targetStatus) ?? false;
}

/**
 * 获取状态转换应该使用的动作
 */
export function getTransitionAction(
  currentStatus: TicketStatus,
  targetStatus: TicketStatus
): string | null {
  return STATUS_TRANSITION_ACTIONS[`${currentStatus}:${targetStatus}`] || null;
}

/**
 * 获取允许的目标状态列表
 */
export function getAllowedTransitions(currentStatus: TicketStatus): TicketStatus[] {
  return VALID_TICKET_TRANSITIONS[currentStatus] || [];
}

/**
 * 状态转换错误
 */
export class InvalidStateTransitionError extends Error {
  constructor(
    public currentStatus: TicketStatus,
    public targetStatus: TicketStatus
  ) {
    super(
      `Invalid state transition: ${currentStatus} -> ${targetStatus}. ` +
        `Allowed transitions: ${getAllowedTransitions(currentStatus).join(', ') || 'none'}`
    );
    this.name = 'InvalidStateTransitionError';
  }
}

/**
 * 验证并抛出错误（用于严格模式）
 */
export function validateTransitionOrThrow(
  currentStatus: TicketStatus,
  targetStatus: TicketStatus
): void {
  if (!isValidTransition(currentStatus, targetStatus)) {
    throw new InvalidStateTransitionError(currentStatus, targetStatus);
  }
}

/**
 * 检查是否为终态
 */
export function isFinalStatus(status: TicketStatus): boolean {
  return (
    status === TicketStatus.CLOSED ||
    status === TicketStatus.CANCELLED
  );
}

/**
 * 检查是否为活动状态（可以进行操作）
 */
export function isActiveStatus(status: TicketStatus): boolean {
  return !isFinalStatus(status);
}

/**
 * 事件状态转换规则
 */
export const INCIDENT_STATUS_TRANSITIONS: Record<string, string[]> = {
  new: ['investigating', 'resolved', 'cancelled'],
  investigating: ['identified', 'monitoring', 'resolved', 'cancelled'],
  identified: ['monitoring', 'resolved', 'cancelled'],
  monitoring: ['resolved', 'cancelled'],
  resolved: ['closed'],
  closed: [],
  cancelled: [],
};

/**
 * 验证事件状态转换
 */
export function isValidIncidentTransition(
  currentStatus: string,
  targetStatus: string
): boolean {
  const allowed = INCIDENT_STATUS_TRANSITIONS[currentStatus];
  return allowed?.includes(targetStatus) ?? false;
}

/**
 * 变更状态转换规则
 */
export const CHANGE_STATUS_TRANSITIONS: Record<string, string[]> = {
  draft: ['pending', 'cancelled'],
  pending: ['approved', 'rejected', 'cancelled'],
  approved: ['in_progress', 'cancelled'],
  inProgress: ['completed', 'rolled_back', 'cancelled'],
  completed: [],
  rejected: ['draft', 'cancelled'],
  rolledBack: [],
  cancelled: [],
};

/**
 * 验证变更状态转换
 */
export function isValidChangeTransition(
  currentStatus: string,
  targetStatus: string
): boolean {
  const allowed = CHANGE_STATUS_TRANSITIONS[currentStatus];
  return allowed?.includes(targetStatus) ?? false;
}

export default {
  isValidTransition,
  getTransitionAction,
  getAllowedTransitions,
  validateTransitionOrThrow,
  isFinalStatus,
  isActiveStatus,
  isValidIncidentTransition,
  isValidChangeTransition,
  VALID_TICKET_TRANSITIONS,
  INCIDENT_STATUS_TRANSITIONS,
  CHANGE_STATUS_TRANSITIONS,
};
