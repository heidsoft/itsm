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
      <Tooltip title={`Required ${board === 'REVIEW' ? 'General' : 'Urgent'} Review Board Approval`}>
        <Tag color={color} icon={<Users size={12} />}>
          {board === 'REVIEW' ? 'General review' : 'Emergency review'}
        </Tag>
      </Tooltip>
    );
  }

  const quorumText =
    approvalType === 'or'
      ? 'Or sign (either passed)'
      : approvalType === 'parallel' || approvalType === 'all'
        ? `Countersign (${threshold ?? approverCount ?? 'Full'} People pass)`
        : null;

  return (
    <Space size={4}>
      {role && (
        <Tag color="blue" icon={<GitBranch size={12} />}>
          {role}
        </Tag>
      )}
      {quorumText && <Tag>{quorumText}</Tag>}
      {isRequired === false && <Tag color="default">Optional</Tag>}
    </Space>
  );
};

export default ReviewStepBadge;
