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
    label: 'Total Problems',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'blue',
  },
  {
    key: 'open',
    label: 'Pending',
    icon: <AlertTriangle className="w-5 h-5" />,
    tone: 'orange',
  },
  {
    key: 'inProgress',
    label: 'In Progress',
    icon: <RefreshCw className="w-5 h-5" />,
    tone: 'cyan',
  },
  {
    key: 'resolved',
    label: 'Resolved',
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
