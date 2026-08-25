'use client';

import React from 'react';
import { Card, Row, Col, Statistic, Tooltip, Spin, Progress } from 'antd';
import {
  ArrowUp,
  ArrowDown,
  Minus,
  LayoutDashboard,
  Clock,
  CheckCircle,
  AlertTriangle,
  User,
} from 'lucide-react';
import type { KPIMetric } from '../types/dashboard.types';

interface KPICardsProps {
  metrics: KPIMetric[];
  loading?: boolean;
}

// 指标极性：positive = 越大越好，negative = 越小越好
type MetricPolarity = 'positive' | 'negative';

const getMetricPolarity = (id: string): MetricPolarity => {
  // 越小越好的指标
  if (
    id === 'pending-tickets' ||
    id === 'overdue-tickets' ||
    id === 'avg-first-response' ||
    id === 'avg-resolution' ||
    id.includes('response-time') ||
    id.includes('resolution-time')
  ) {
    return 'negative';
  }
  return 'positive';
};

// 企业级KPI卡片组件
const EnterpriseKPICard: React.FC<{ metric: KPIMetric }> = React.memo(({ metric }) => {
  const polarity = getMetricPolarity(metric.id);

  // 根据业务极性判断趋势颜色：
  // - positive 指标（总工单、已完成、SLA达成率）：上升=绿色（好），下降=红色（坏）
  // - negative 指标（待处理、超时、响应时间）：上升=红色（坏），下降=绿色（好）
  const isGoodTrend =
    metric.trend === 'stable'
      ? true
      : polarity === 'positive'
        ? metric.trend === 'up'
        : metric.trend === 'down';

  const trendColor = isGoodTrend ? 'text-green-500' : 'text-red-500';

  const getTrendIcon = () => {
    switch (metric.trend) {
      case 'up':
        return <ArrowUp className={`w-4 h-4 ${trendColor}`} />;
      case 'down':
        return <ArrowDown className={`w-4 h-4 ${trendColor}`} />;
      default:
        return <Minus className="w-4 h-4 text-gray-400" />;
    }
  };

  // 获取默认图标
  const getDefaultIcon = () => {
    const iconClass = 'w-7 h-7';
    switch (metric.id) {
      case 'total-tickets':
        return <LayoutDashboard className={iconClass} />;
      case 'open-tickets':
        return <AlertTriangle className={iconClass} />;
      case 'resolved-tickets':
        return <CheckCircle className={iconClass} />;
      case 'sla-compliance':
        return <Clock className={iconClass} />;
      case 'avg-resolution':
        return <Clock className={iconClass} />;
      case 'user-satisfaction':
        return <User className={iconClass} />;
      default:
        return <LayoutDashboard className={iconClass} />;
    }
  };

  // 计算进度百分比（用于装饰性进度条）
  const getProgressPercent = () => {
    if (metric.id === 'sla-compliance' && typeof metric.value === 'number') {
      return metric.value;
    }
    if (metric.change !== undefined) {
      return Math.min(Math.abs(metric.change) * 10, 100);
    }
    return 75; // 默认值
  };

  return (
    <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={6}>
      <Card
        className="h-full transition-all duration-200 hover:border-blue-500 hover:shadow-md group rounded-lg bg-white shadow-sm border border-gray-200"
       
        styles={{
          body: {
            padding: '20px',
            minHeight: '180px',
            display: 'flex',
            flexDirection: 'column',
            position: 'relative',
          },
        }}
      >
        {/* 简化设计：去除背景装饰 */}
        <div className="flex flex-col h-full">
          {/* 顶部区域：图标和趋势 */}
          <div className="flex items-start justify-between mb-4">
            {/* 简化图标容器：去除渐变和缩放动画 */}
            <div
              className="w-12 h-12 rounded-lg flex items-center justify-center transition-colors duration-200"
              style={{
                backgroundColor: `${metric.color}15`,
              }}
            >
              <div style={{ color: metric.color }}>{metric.icon || getDefaultIcon()}</div>
            </div>

            {/* 简化趋势指示器 */}
            {metric.change !== undefined && (
              <div
                className={`text-sm font-semibold flex items-center gap-1 ${trendColor}`}
              >
                {getTrendIcon()}
                {metric.change > 0 ? '+' : ''}
                {metric.change.toFixed(1)}%
              </div>
            )}
          </div>

          {/* 标题 */}
          <div className="mb-3">
            <Tooltip title={metric.description || metric.title}>
              <h3 className="text-base font-semibold text-gray-800 leading-tight line-clamp-2">
                {' '}
                {/* Changed text size and color */}
                {metric.title}
              </h3>
            </Tooltip>
          </div>

          {/* 数值显示 */}
          <div className="flex-1 flex flex-col justify-center mb-3">
            <div className="flex items-baseline gap-2">
              <span
                className="text-4xl font-bold leading-none" // Removed tracking-tight
                style={{ color: metric.color }}
              >
                {typeof metric.value === 'number' ? metric.value.toLocaleString() : metric.value}
              </span>
              {metric.unit && (
                <span className="text-base font-medium text-gray-500">{metric.unit}</span>
              )}
            </div>
          </div>

          {/* 简化底部：去除装饰性进度条 */}
          {metric.change !== undefined && (
            <div className="mt-auto pt-3 border-t border-gray-100">
              <span className="text-xs text-gray-500">
                相比上期
                <span
                  className="ml-2 font-semibold"
                  style={{
                    color: isGoodTrend ? '#10b981' : '#ef4444',
                  }}
                >
                  {metric.trend === 'up'
                    ? '↑'
                    : metric.trend === 'down'
                      ? '↓'
                      : '—'}{' '}
                  {Math.abs(metric.change).toFixed(1)}%
                </span>
              </span>
            </div>
          )}
        </div>
      </Card>
    </Col>
  );
});

EnterpriseKPICard.displayName = 'EnterpriseKPICard';

// KPI卡片组主组件
export const KPICards: React.FC<KPICardsProps> = React.memo(({ metrics, loading = false }) => {
  if (loading) {
    return (
      <div className="mb-6">
        <Row gutter={[16, 16]}>
          {Array.from({ length: 6 }).map((_, index) => (
            <Col key={index} xs={24} sm={12} md={12} lg={8} xl={6} xxl={4}>
              <Card
                className="h-44 rounded-lg shadow-sm border border-gray-200"
               
              >
                <div className="flex items-center justify-center h-full">
                  <div className="text-center">
                    <Spin size="large" />
                    <p className="text-xs text-gray-400 mt-3">加载中...</p>
                  </div>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </div>
    );
  }

  if (!metrics || metrics.length === 0) {
    return (
      <div className="mb-6">
        <Card
          className="text-center py-12 rounded-lg bg-gray-50 border border-dashed border-gray-300"
         
        >
          <div className="text-gray-500">
            <div className="w-16 h-16 rounded-lg bg-gray-100 flex items-center justify-center mx-auto mb-4">
              <LayoutDashboard className="text-3xl text-gray-400" />
            </div>
            <p className="text-base font-medium text-gray-700 mb-1">暂无KPI数据</p>
            <p className="text-sm text-gray-500">系统正在收集数据，请稍后查看</p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="mb-6">
      <Row gutter={[16, 16]}>
        {metrics.map(metric => (
          <EnterpriseKPICard key={metric.id} metric={metric} />
        ))}
        {/* Placeholder for two more cards if needed, maintain layout */}
        {metrics.length === 4 && (
          <>
            <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={6} />
            <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={6} />
          </>
        )}
        {metrics.length === 5 && <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={6} />}
      </Row>
    </div>
  );
});

KPICards.displayName = 'KPICards';
