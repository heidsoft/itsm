'use client';

import React from 'react';
import { Tag, Space, Tooltip } from 'antd';
import { Users, GitBranch } from 'lucide-react';
import { isCabRole, cabBoardFromRole, type CabBoardType } from '@/types/cab';

export interface CabStepBadgeProps {
  /** 审批链步骤的 role，例如 cab:CAB / cab:ECAB / manager */
  role?: string;
  /** 会签类型：serial | parallel | or | all */
  approvalType?: string;
  /** 阈值（parallel/or 时有效） */
  threshold?: number;
  /** 候选人数量（用于 or/parallel 文案） */
  approverCount?: number;
  /** 是否必需审批 */
  isRequired?: boolean;
}

/**
 * 通用的 CAB / 会签步骤徽标。
 * - role 以 cab: 开头 → 渲染 CAB / ECAB 标签（绿/紫）。
 * - 否则展示普通的角色 / 会签类型标签。
 * 被 change / service_request 的审批流详情复用。
 */
const CabStepBadge: React.FC<CabStepBadgeProps> = ({
  role,
  approvalType,
  threshold,
  approverCount,
  isRequired,
}) => {
  const board: CabBoardType | null = cabBoardFromRole(role);

  if (board) {
    const color = board === 'CAB' ? 'green' : 'purple';
    return (
      <Tooltip title={`需要 ${board}（变更咨询委员会）审批`}>
        <Tag color={color} icon={<Users size={12} />}>
          {board === 'CAB' ? 'CAB 审批' : 'ECAB 审批'}
        </Tag>
      </Tooltip>
    );
  }

  const quorumText =
    approvalType === 'or'
      ? '或签（任一通过）'
      : approvalType === 'parallel' || approvalType === 'all'
        ? `会签（${threshold ?? approverCount ?? '全'} 人通过）`
        : null;

  return (
    <Space size={4}>
      {role && (
        <Tag color="blue" icon={<GitBranch size={12} />}>
          {role}
        </Tag>
      )}
      {quorumText && <Tag>{quorumText}</Tag>}
      {isRequired === false && <Tag color="default">可选</Tag>}
    </Space>
  );
};

export default CabStepBadge;
