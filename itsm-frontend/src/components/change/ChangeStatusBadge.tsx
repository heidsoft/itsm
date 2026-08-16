'use client';

/**
 * 变更状态徽章组件
 * 用于显示变更请求的状态标签
 */

import React from 'react';
import { Tag } from 'antd';
import type { ChangeStatus} from '@/constants/taxonomy';
import { ChangeStatusConfig } from '@/constants/taxonomy';
import { useI18n } from '@/lib/i18n/useI18n';

interface ChangeStatusBadgeProps {
  /** 变更状态 */
  status: ChangeStatus | string;
  /** 是否显示文本，默认为 true */
  showText?: boolean;
}

const STATUS_LABEL_KEYS: Record<string, string> = {
  draft: 'change.status.draft',
  pending: 'change.status.pending',
  approved: 'change.status.approved',
  rejected: 'change.status.rejected',
  scheduled: 'change.status.scheduled',
  in_progress: 'change.status.inProgress',
  completed: 'change.status.completed',
  failed: 'change.status.failed',
  rolled_back: 'change.status.rolledBack',
  cancelled: 'change.status.cancelled',
  closed: 'change.status.closed',
};

/**
 * 变更状态徽章组件
 * 根据变更状态显示不同颜色的标签
 */
export const ChangeStatusBadge: React.FC<ChangeStatusBadgeProps> = ({
  status,
  showText = true,
}) => {
  const { t } = useI18n();

  const config = React.useMemo(() => {
    const base =
      ChangeStatusConfig[status as ChangeStatus] || {
        label: status,
        color: 'default',
        badgeStatus: 'default' as const,
      };
    const statusKey = String(status);
    const i18nKey = STATUS_LABEL_KEYS[statusKey];
    return {
      ...base,
      label: i18nKey ? t(i18nKey) : base.label,
    };
  }, [status, t]);

  if (!showText) {
    return null;
  }

  return (
    <Tag color={config.color} style={{ margin: 0 }}>
      {config.label}
    </Tag>
  );
};

export default ChangeStatusBadge;
