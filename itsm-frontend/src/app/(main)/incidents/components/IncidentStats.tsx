'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import { AlertTriangle, Clock, CheckCircle, AlertCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

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

const CARD_STYLES = [
  { bg: 'linear-gradient(135deg, #3b82f6 0%, #2563eb 100%)' }, // blue
  { bg: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)' }, // orange
  { bg: 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)' }, // green
  { bg: 'linear-gradient(135deg, #a855f7 0%, #9333ea 100%)' }, // purple
];

const ICON_BG = 'rgba(255,255,255,0.2)';
const TEXT_COLOR = '#ffffff';
const LABEL_COLOR = 'rgba(255,255,255,0.9)';

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

const StatCard: React.FC<{
  bg: string;
  icon: React.ReactNode;
  value: string | number;
  label: string;
}> = ({ bg, icon, value, label }) => (
  <Card
    style={{
      backgroundImage: bg,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      border: 'none',
      borderRadius: '0.5rem',
    }}
    styles={{ body: { padding: '16px', color: TEXT_COLOR } }}
    className="shadow-sm overflow-hidden relative"
  >
    <div
      style={{
        width: '2.5rem',
        height: '2.5rem',
        background: ICON_BG,
        borderRadius: '0.5rem',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        margin: '0 auto 0.75rem',
      }}
    >
      {React.cloneElement(
        icon as React.ReactElement<{ className?: string; style?: React.CSSProperties }>,
        {
          style: { color: TEXT_COLOR },
        }
      )}
    </div>
    <div className="text-2xl font-bold mb-1" style={{ color: TEXT_COLOR }}>
      {value}
    </div>
    <div className="font-semibold text-xs" style={{ color: LABEL_COLOR }}>
      {label}
    </div>
  </Card>
);

export const IncidentStats: React.FC<IncidentStatsProps> = ({ metrics, className }) => {
  const { t } = useI18n();

  if (!metrics) {
    return (
      <div className={`mb-6 ${className || ''}`}>
        <Row gutter={[16, 16]}>
          {[1, 2, 3, 4].map(i => (
            <Col key={i} xs={24} sm={12} md={6} lg={6}>
              <Card loading className="rounded-lg" />
            </Col>
          ))}
        </Row>
      </div>
    );
  }

  const cards = [
    {
      bg: CARD_STYLES[0].bg,
      icon: <AlertTriangle className="w-5 h-5" />,
      value: metrics.totalIncidents ?? '-',
      label: t('incidents.total'),
    },
    {
      bg: CARD_STYLES[1].bg,
      icon: <Clock className="w-5 h-5" />,
      value: metrics.criticalIncidents ?? '-',
      label: t('incidents.critical'),
    },
    {
      bg: CARD_STYLES[2].bg,
      icon: <CheckCircle className="w-5 h-5" />,
      value: metrics.majorIncidents ?? '-',
      label: t('incidents.major'),
    },
    {
      bg: CARD_STYLES[3].bg,
      icon: <AlertCircle className="w-5 h-5" />,
      value: formatAvgResolutionTime(metrics.avgResolutionTime),
      label: t('incidents.avgResolutionTime'),
    },
  ];

  return (
    <div className={`mb-6 ${className || ''}`}>
      <Row gutter={[16, 16]} align="stretch">
        {cards.map((card, i) => (
          <Col key={i} xs={24} sm={12} md={6} lg={6} className="flex">
            <StatCard {...card} />
          </Col>
        ))}
      </Row>
    </div>
  );
};
