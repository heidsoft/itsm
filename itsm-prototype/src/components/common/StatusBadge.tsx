'use client';

import React from 'react';
import { Badge, Tag } from 'antd';
import { getStatusConfig } from '@/lib/constants/ticket-constants';

interface StatusBadgeProps {
  status: string;
  showText?: boolean;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, showText = true }) => {
  const config = getStatusConfig(status);

  return (
    <Badge
      status={config.badgeStatus}
      text={
        showText ? (
          <Tag color={config.color} style={{ margin: 0 }}>
            {config.text}
          </Tag>
        ) : null
      }
    />
  );
};

