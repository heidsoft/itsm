/**
 * 工单流转相关类型定义
 */

import { TicketStatus, TicketUser } from './ticket';

/**
 * 工单流转操作类型
 */
export enum TicketWorkflowAction {
  ACCEPT = 'accept',           // 接单
  REJECT = 'reject',           // 驳回
  WITHDRAW = 'withdraw',       // 撤回
  FORWARD = 'forward',         // 转发
  CC = 'cc',                   // 抄送
  ESCALATE = 'escalate',       // 升级
  APPROVE = 'approve',         // 审批通过
  APPROVE_REJECT = 'approve_reject', // 审批拒绝
  DELEGATE = 'delegate',       // 委派
  RESOLVE = 'resolve',         // 解决
  CLOSE = 'close',             // 关闭
  REOPEN = 'reopen',           // 重开
}

/**
 * 工单流转记录
 */
export interface TicketWorkflowRecord {
  id: number;
  ticketId: number;
  action: TicketWorkflowAction;
  fromStatus?: TicketStatus;
  toStatus?: TicketStatus;
  operator: TicketUser;
  fromUser?: TicketUser;
  toUser?: TicketUser;
  comment?: string;
  reason?: string;
  attachments?: Array<{
    id: number;
    filename: string;
    url: string;
  }>;
  metadata?: Record<string, any>;
  createdAt: string;
}

/**
 * 工单当前状态
 */
export interface TicketWorkflowState {
  ticketId: number;
  currentStatus: TicketStatus;
  currentAssignee?: TicketUser;
  approvalStatus?: ApprovalStatus;
  currentApprovalLevel?: number;
  totalApprovalLevels?: number;
  pendingApprovers?: TicketUser[];
  completedApprovals?: ApprovalRecord[];
  canAccept: boolean;
  canReject: boolean;
  canWithdraw: boolean;
  canForward: boolean;
  canCC: boolean;
  canApprove: boolean;
  canResolve: boolean;
  canClose: boolean;
  availableActions: TicketWorkflowAction[];
}

/**
 * 审批状态
 */
export enum ApprovalStatus {
  PENDING = 'pending',           // 待审批
  IN_PROGRESS = 'in_progress',   // 审批中
  APPROVED = 'approved',         // 已通过
  REJECTED = 'rejected',         // 已拒绝
  CANCELLED = 'cancelled',       // 已取消
}

/**
 * 审批记录
 */
export interface ApprovalRecord {
  id: number;
  ticketId: number;
  level: number;
  levelName: string;
  approver: TicketUser;
  status: ApprovalStatus;
  action?: 'approve' | 'reject' | 'delegate';
  comment?: string;
  attachments?: Array<{
    id: number;
    filename: string;
    url: string;
  }>;
  delegateTo?: TicketUser;
  processedAt?: string;
  createdAt: string;
}

/**
 * 接单请求
 */
export interface AcceptTicketRequest {
  ticketId: number;
  comment?: string;
}

/**
 * 驳回请求
 */
export interface RejectTicketRequest {
  ticketId: number;
  reason: string;
  comment?: string;
  returnToStatus?: TicketStatus;
}

/**
 * 撤回请求
 */
export interface WithdrawTicketRequest {
  ticketId: number;
  reason: string;
}

/**
 * 转发请求
 */
export interface ForwardTicketRequest {
  ticketId: number;
  toUserId: number;
  comment?: string;
  transferOwnership: boolean; // 是否转移所有权
}

/**
 * 抄送请求
 */
export interface CCTicketRequest {
  ticketId: number;
  ccUsers: number[];
  comment?: string;
}

/**
 * 审批请求
 */
export interface ApproveTicketRequest {
  ticketId: number;
  approvalId: number;
  action: 'approve' | 'reject' | 'delegate';
  comment?: string;
  delegateToUserId?: number;
  attachments?: File[];
}

/**
 * 解决工单请求
 */
export interface ResolveTicketRequest {
  ticketId: number;
  resolution: string;
  resolutionCategory?: string;
  workNotes?: string;
  attachments?: File[];
}

/**
 * 关闭工单请求
 */
export interface CloseTicketRequest {
  ticketId: number;
  closeReason?: string;
  closeNotes?: string;
}

/**
 * 重开工单请求
 */
export interface ReopenTicketRequest {
  ticketId: number;
  reason: string;
}

/**
 * 抄送人
 */
export interface TicketCC {
  id: number;
  ticketId: number;
  user: TicketUser;
  addedBy: TicketUser;
  addedAt: string;
  isActive: boolean;
}

/**
 * 工单流转统计
 */
export interface TicketWorkflowStats {
  totalTransitions: number;
  averageTransitionTime: number; // 平均流转时间（小时）
  byAction: Record<TicketWorkflowAction, number>;
  byStatus: Record<TicketStatus, number>;
  approvalStats: {
    totalApprovals: number;
    approvedCount: number;
    rejectedCount: number;
    averageApprovalTime: number; // 平均审批时间（小时）
    approvalRate: number; // 审批通过率（%）
  };
}

/**
 * 工单操作权限
 */
export interface TicketActionPermissions {
  canAccept: boolean;
  canReject: boolean;
  canWithdraw: boolean;
  canForward: boolean;
  canCC: boolean;
  canApprove: boolean;
  canResolve: boolean;
  canClose: boolean;
  canReopen: boolean;
  canEdit: boolean;
  canDelete: boolean;
  canComment: boolean;
  canViewInternal: boolean;
}

