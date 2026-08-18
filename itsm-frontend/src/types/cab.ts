/**
 * CAB（变更咨询委员会）成员名册类型定义。
 * 后端仅负责名册管理；CAB 审批流转由审批链引擎（cab:CAB / cab:ECAB 步骤）驱动。
 */

export type CabBoardType = 'CAB' | 'ECAB';

export type CabMemberRole = 'member' | 'chair' | 'secretary';

export interface CabMember {
  id: number;
  userId: number;
  userName: string;
  email: string;
  /** CAB 或 ECAB */
  type: CabBoardType;
  /** member, chair, secretary */
  role: CabMemberRole;
  isActive: boolean;
  tenantId: number;
  createdAt: string;
}

export interface AddCabMemberRequest {
  userId: number;
  type: CabBoardType;
  role?: CabMemberRole;
}

export interface UpdateCabMemberRequest {
  role: CabMemberRole;
  isActive: boolean;
}

/** 审批链步骤 role 中以 cab: 前缀标识 CAB 步骤，例如 cab:CAB / cab:ECAB。 */
export const CAB_ROLE_PREFIX = 'cab:';

export function isCabRole(role?: string): boolean {
  return !!role && role.startsWith(CAB_ROLE_PREFIX);
}

export function cabBoardFromRole(role?: string): CabBoardType | null {
  if (role === 'cab:CAB') return 'CAB';
  if (role === 'cab:ECAB') return 'ECAB';
  return null;
}
