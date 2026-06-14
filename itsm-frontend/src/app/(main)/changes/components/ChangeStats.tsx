'use client';

import React from 'react';
import { GitBranch, AlertTriangle } from 'lucide-react';
import { BusinessStatsGrid } from '@/components/common/BusinessStatsGrid';
import type { ChangeStats as ChangeStatsData } from '@/lib/services/change-service';

interface ChangeStatsProps {
  stats: ChangeStatsData;
}

const statCards = [
  {
    key: 'total',
    label: '总变更数',
    icon: <GitBranch className="w-5 h-5" />,
    tone: 'blue',
  },
  {
    key: 'pending',
    label: '待审批',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'orange',
  },
  {
    key: 'approved',
    label: '已批准',
    icon: <GitBranch className="w-5 h-5" />,
    tone: 'green',
  },
  {
    key: 'completed',
    label: '已完成',
    icon: <GitBranch className="w-5 h-5" />,
    tone: 'purple',
  },
] as const;

export const ChangeStats: React.FC<ChangeStatsProps> = ({ stats }) => {
  return (
    <BusinessStatsGrid
      items={statCards.map(card => ({
        label: card.label,
        value: stats[card.key],
        icon: card.icon,
        tone: card.tone,
      }))}
    />
  );
};
