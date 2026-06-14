'use client';

import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { BusinessStatsGrid } from '@/components/common/BusinessStatsGrid';

interface ProblemStatsProps {
  metrics: {
    total: number;
    open: number;
    inProgress: number;
    resolved: number;
  };
}

const statCards = [
  {
    key: 'total',
    label: '总问题数',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'blue',
  },
  {
    key: 'open',
    label: '待处理',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'orange',
  },
  {
    key: 'inProgress',
    label: '处理中',
    icon: <RefreshCw className="w-5 h-5" />,
    tone: 'cyan',
  },
  {
    key: 'resolved',
    label: '已解决',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'green',
  },
] as const;

export const ProblemStats: React.FC<ProblemStatsProps> = ({ metrics }) => {
  return (
    <BusinessStatsGrid
      items={statCards.map(card => ({
        label: card.label,
        value: metrics[card.key],
        icon: card.icon,
        tone: card.tone,
      }))}
    />
  );
};
