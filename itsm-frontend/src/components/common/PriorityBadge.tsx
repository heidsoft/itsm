'use client';

import React from 'react';
import { Tag } from 'antd';
import { getPriorityConfig } from '@/lib/constants/ticket-constants';

interface PriorityBadgeProps {
  priority: string;
  showDescription?: boolean;
}

export const PriorityBadge: React.FC<PriorityBadgeProps> = ({ priority, showDescription = false }) => {
  const config = getPriorityConfig(priority);

  return (
    <Tag color={config.color} style={{ margin: 0 }}>
      <span className='mr-1'>{config.icon}</span>
      {config.text}
      {showDescription && (
        <span className='ml-2 text-xs opacity-75'>({config.description})</span>
      )}
    </Tag>
  );
};

