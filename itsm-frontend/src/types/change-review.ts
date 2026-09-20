/**
 * 变更评审组成员名册类型定义。
 * 后端仅负责名册管理；评审审批流转由审批链引擎（review:REVIEW / review:EREVIEW 步骤）驱动。
 */

export type ReviewBoardType = 'REVIEW' | 'EREVIEW';

export type ReviewMemberRole = 'member' | 'chair' | 'secretary';

export interface ReviewMember {
  id: number;
  userId: number;
  userName: string;
  email: string;
  /** REVIEW 或 EREVIEW */
  type: ReviewBoardType;
  /** member, chair, secretary */
  role: ReviewMemberRole;
  isActive: boolean;
  tenantId: number;
  createdAt: string;
}

export interface AddReviewMemberRequest {
  userId: number;
  type: ReviewBoardType;
  role?: ReviewMemberRole;
}

export interface UpdateReviewMemberRequest {
  role: ReviewMemberRole;
  isActive: boolean;
}

/** 审批链步骤 role 中以 review: 前缀标识评审步骤，例如 review:REVIEW / review:EREVIEW。 */
export const REVIEW_ROLE_PREFIX = 'review:';

export function isReviewRole(role?: string): boolean {
  return !!role && role.startsWith(REVIEW_ROLE_PREFIX);
}

export function reviewBoardFromRole(role?: string): ReviewBoardType | null {
  if (role === 'review:REVIEW') return 'REVIEW';
  if (role === 'review:EREVIEW') return 'EREVIEW';
  return null;
}
