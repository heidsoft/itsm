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

/**
 * 格式化平均解决时间
 * 输入: 小时为单位的数值
 * 输出: 友好格式如 "2.5h" 或 "1天2小时"
 */
const formatAvgResolutionTime = (hours: number | undefined): string => {
  if (!hours || hours <= 0) return '-';

  if (hours < 1) {
    // 少于1小时，显示为分钟
    const minutes = Math.round(hours * 60);
    return `${minutes}分钟`;
  }

  if (hours < 24) {
    // 少于24小时，显示为小时
    return `${hours.toFixed(1)}小时`;
  }

  // 超过24小时，显示为天和小时
  const days = Math.floor(hours / 24);
  const remainingHours = (hours % 24).toFixed(1);
  return `${days}天${remainingHours}小时`;
};

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

  return (
    <div className={`mb-6 ${className || ''}`}>
      <Row gutter={[16, 16]} align="stretch">
        <Col xs={24} sm={12} md={6} lg={6} className="flex">
          <Card
            className="h-full w-full text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white overflow-hidden relative"
            styles={{ body: { padding: '16px' } }}
          >
            <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3">
              <AlertTriangle className="w-5 h-5 text-white" />
            </div>
            <div className="text-2xl font-bold mb-1">{metrics.totalIncidents ?? '-'}</div>
            <div className="text-white/90 font-semibold text-xs">{t('incidents.total')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6} className="flex">
          <Card
            className="h-full w-full text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-orange-500 to-orange-600 text-white overflow-hidden relative"
            styles={{ body: { padding: '16px' } }}
          >
            <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3">
              <Clock className="w-5 h-5 text-white" />
            </div>
            <div className="text-2xl font-bold mb-1">{metrics.criticalIncidents ?? '-'}</div>
            <div className="text-white/90 font-semibold text-xs">{t('incidents.critical')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6} className="flex">
          <Card
            className="h-full w-full text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-green-500 to-green-600 text-white overflow-hidden relative"
            styles={{ body: { padding: '16px' } }}
          >
            <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3">
              <CheckCircle className="w-5 h-5 text-white" />
            </div>
            <div className="text-2xl font-bold mb-1">{metrics.majorIncidents ?? '-'}</div>
            <div className="text-white/90 font-semibold text-xs">{t('incidents.major')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6} className="flex">
          <Card
            className="h-full w-full text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-purple-500 to-purple-600 text-white overflow-hidden relative"
            styles={{ body: { padding: '16px' } }}
          >
            <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3">
              <AlertCircle className="w-5 h-5 text-white" />
            </div>
            <div className="text-2xl font-bold mb-1">
              {formatAvgResolutionTime(metrics.avgResolutionTime)}
            </div>
            <div className="text-white/90 font-semibold text-xs">
              {t('incidents.avgResolutionTime')}
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
