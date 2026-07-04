'use client';

/**
 * 变更状态徽章组件
 * 用于显示变更请求的状态标签
 */

import React from 'react';
import { Tag } from 'antd';
import type { ChangeStatus} from '@/constants/taxonomy';
import { ChangeStatusConfig } from '@/constants/taxonomy';

interface ChangeStatusBadgeProps {
  /** 变更状态 */
  status: ChangeStatus | string;
  /** 是否显示文本，默认为 true */
  showText?: boolean;
}

/**
 * 获取变更状态配置
 */
const getChangeStatusConfig = (status: string) => {
  return (
    ChangeStatusConfig[status as ChangeStatus] || {
      label: status,
      color: 'default',
      badgeStatus: 'default' as const,
    }
  );
};

/**
 * 变更状态徽章组件
 * 根据变更状态显示不同颜色的标签
 */
export const ChangeStatusBadge: React.FC<ChangeStatusBadgeProps> = ({
  status,
  showText = true,
}) => {
  const config = getChangeStatusConfig(status);

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
