'use client';

import React from 'react';
import { Tag, Space, Tooltip } from 'antd';
import { Users, GitBranch } from 'lucide-react';
import { isReviewRole, reviewBoardFromRole, type ReviewBoardType } from '@/types/change-review';

export interface ReviewStepBadgeProps {
  /** 审批链步骤的 role，例如 review:REVIEW / review:EREVIEW / manager */
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
 * 通用的评审 / 会签步骤徽标。
 * - role 以 review: 开头 → 渲染 REVIEW / EREVIEW 标签（绿/紫）。
 * - 否则展示普通的角色 / 会签类型标签。
 * 被 change / service_request 的审批流详情复用。
 */
const ReviewStepBadge: React.FC<ReviewStepBadgeProps> = ({
  role,
  approvalType,
  threshold,
  approverCount,
  isRequired,
}) => {
  const board: ReviewBoardType | null = reviewBoardFromRole(role);

  if (board) {
    const color = board === 'REVIEW' ? 'green' : 'purple';
    return (
      <Tooltip title={`需要${board === 'REVIEW' ? '常规' : '紧急'}评审组审批`}>
        <Tag color={color} icon={<Users size={12} />}>
          {board === 'REVIEW' ? '常规评审' : '紧急评审'}
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

export default ReviewStepBadge;
