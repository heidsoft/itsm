'use client';

import React from 'react';
import { AlertTriangle, Clock, CheckCircle, AlertCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { BusinessStatsGrid } from '@/components/common/BusinessStatsGrid';

interface IncidentStatsProps {
  metrics?: {
    totalIncidents?: number;
    openIncidents?: number;
    criticalIncidents?: number;
    majorIncidents?: number;
    avgResolutionTime?: number;
  };
  className?: string;
}

/**
 * 格式化平均解决时间
 * 输入: 小时为单位的数值
 * 输出: 友好格式如 "2.5h" 或 "1天2小时"
 */
const formatAvgResolutionTime = (hours: number | undefined): string => {
  if (!hours || hours <= 0) return '-';

  if (hours < 1) {
    const minutes = Math.round(hours * 60);
    return `${minutes}分钟`;
  }

  if (hours < 24) {
    return `${hours.toFixed(1)}小时`;
  }

  const days = Math.floor(hours / 24);
  const remainingHours = (hours % 24).toFixed(1);
  return `${days}天${remainingHours}小时`;
};

export const IncidentStats: React.FC<IncidentStatsProps> = ({ metrics, className }) => {
  const { t } = useI18n();

  if (!metrics) {
    return <BusinessStatsGrid loading items={[]} className={className} />;
  }

  const cards = [
    {
      tone: 'blue' as const,
      icon: <AlertTriangle className="w-5 h-5" />,
      value: metrics.totalIncidents ?? '-',
      label: t('incidents.total'),
    },
    {
      tone: 'orange' as const,
      icon: <Clock className="w-5 h-5" />,
      value: metrics.criticalIncidents ?? '-',
      label: t('incidents.critical'),
    },
    {
      tone: 'green' as const,
      icon: <CheckCircle className="w-5 h-5" />,
      value: metrics.majorIncidents ?? '-',
      label: t('incidents.major'),
    },
    {
      tone: 'purple' as const,
      icon: <AlertCircle className="w-5 h-5" />,
      value: formatAvgResolutionTime(metrics.avgResolutionTime),
      label: t('incidents.avgResolutionTime'),
    },
  ];

  return <BusinessStatsGrid items={cards} className={className} />;
};
