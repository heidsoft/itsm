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
import { KPIMetric } from '../types/dashboard.types';

interface KPICardsProps {
  metrics: KPIMetric[];
  loading?: boolean;
}

// 企业级KPI卡片组件
const EnterpriseKPICard: React.FC<{ metric: KPIMetric }> = React.memo(({ metric }) => {
  // 获取趋势图标
  const getTrendIcon = () => {
    switch (metric.trend) {
      case 'up':
        return <ArrowUp style={{ width: 16, height: 16, color: '#10b981' }} />; // Smaller icon
      case 'down':
        return <ArrowDown style={{ width: 16, height: 16, color: '#ef4444' }} />; // Smaller icon
      default:
        return <Minus style={{ width: 16, height: 16, color: '#9ca3af' }} />; // Smaller icon
    }
  };

  // 获取默认图标
  const getDefaultIcon = () => {
    const iconStyle = { width: 28, height: 28 }; // Slightly larger icon
    switch (metric.id) {
      case 'total-tickets':
        return <LayoutDashboard style={iconStyle} />;
      case 'open-tickets':
        return <AlertTriangle style={iconStyle} />;
      case 'resolved-tickets':
        return <CheckCircle style={iconStyle} />;
      case 'sla-compliance':
        return <Clock style={iconStyle} />;
      case 'avg-resolution':
        return <Clock style={iconStyle} />;
      case 'user-satisfaction':
        return <User style={iconStyle} />;
      default:
        return <LayoutDashboard style={iconStyle} />;
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
        className='enterprise-kpi-card h-full transition-all duration-200 hover:border-blue-500 hover:shadow-md group'
        style={{
          borderRadius: 8, // antdTheme.token.borderRadiusLG
          background: '#ffffff', // antdTheme.token.colorBgContainer
          boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)', // designSystem.boxShadow.base
          borderColor: '#e2e8f0', // antdTheme.token.colorBorder
          borderWidth: 1,
          borderStyle: 'solid',
        }}
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
        <div className='flex flex-col h-full'>
          {/* 顶部区域：图标和趋势 */}
          <div className='flex items-start justify-between mb-4'>
            {/* 简化图标容器：去除渐变和缩放动画 */}
            <div
              className='w-12 h-12 rounded-lg flex items-center justify-center transition-colors duration-200'
              style={{
                backgroundColor: `${metric.color}15`,
              }}
            >
              <div style={{ color: metric.color }}>{metric.icon || getDefaultIcon()}</div>
            </div>

            {/* 简化趋势指示器 */}
            {metric.change !== undefined && (
              <div
                className={`text-sm font-semibold flex items-center gap-1 ${ // Added flex items-center gap-1
                  metric.trend === 'up'
                    ? 'text-green-500' // antdTheme.token.colorSuccess
                    : metric.trend === 'down'
                    ? 'text-red-500' // antdTheme.token.colorError
                    : 'text-gray-500' // antdTheme.token.colorTextSecondary
                }`}
              >
                {getTrendIcon()}
                {metric.change > 0 ? '+' : ''}
                {metric.change}%
              </div>
            )}
          </div>

          {/* 标题 */}
          <div className='mb-3'>
            <Tooltip title={metric.description || metric.title}>
              <h3 className='text-base font-semibold text-gray-800 leading-tight line-clamp-2'> {/* Changed text size and color */}
                {metric.title}
              </h3>
            </Tooltip>
          </div>

          {/* 数值显示 */}
          <div className='flex-1 flex flex-col justify-center mb-3'>
            <div className='flex items-baseline gap-2'>
              <span
                className='text-4xl font-bold leading-none' // Removed tracking-tight
                style={{ color: metric.color }}
              >
                {typeof metric.value === 'number' ? metric.value.toLocaleString() : metric.value}
              </span>
              {metric.unit && (
                <span className='text-base font-medium text-gray-500'>{metric.unit}</span>
              )}
            </div>
          </div>

          {/* 简化底部：去除装饰性进度条 */}
          {metric.change !== undefined && (
            <div className='mt-auto pt-3 border-t border-gray-100'>
              <span className='text-xs text-gray-500'>
                相比上期
                <span
                  className='ml-2 font-semibold'
                  style={{
                    color:
                      metric.changeType === 'increase'
                        ? '#10b981' // antdTheme.token.colorSuccess
                        : metric.changeType === 'decrease'
                        ? '#ef4444' // antdTheme.token.colorError
                        : '#6b7280', // antdTheme.token.colorTextSecondary
                  }}
                >
                  {metric.changeType === 'increase' ? '↑' : metric.changeType === 'decrease' ? '↓' : '—'}{' '}
                  {Math.abs(metric.change)}%
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
      <div className='mb-6'>
        <Row gutter={[16, 16]}>
          {Array.from({ length: 6 }).map((_, index) => (
            <Col key={index} xs={24} sm={12} md={12} lg={8} xl={6} xxl={4}>
              <Card
                className='h-44 border-0'
                style={{
                  borderRadius: 8, // antdTheme.token.borderRadiusLG
                  boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)', // designSystem.boxShadow.base
                }}
              >
                <div className='flex items-center justify-center h-full'>
                  <div className='text-center'>
                    <Spin size='large' />
                    <p className='text-xs text-gray-400 mt-3'>加载中...</p>
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
      <div className='mb-6'>
        <Card
          className='text-center py-12 border-0'
          style={{
            borderRadius: 8, // antdTheme.token.borderRadiusLG
            background: '#f8fafc', // antdTheme.token.colorBgLayout
            boxShadow: 'none', // No shadow for empty state background
            border: '1px dashed #cbd5e1', // designSystem.colors.neutral[300]
          }}
        >
          <div className='text-gray-500'>
            <div className='w-16 h-16 rounded-lg bg-gray-100 flex items-center justify-center mx-auto mb-4'>
              <DashboardOutlined style={{ fontSize: 32, color: '#9ca3af' }} />
            </div>
            <p className='text-base font-medium text-gray-700 mb-1'>暂无KPI数据</p>
            <p className='text-sm text-gray-500'>系统正在收集数据，请稍后查看</p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className='mb-6'>
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
        {metrics.length === 5 && (
          <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={6} />
        )}
      </Row>
    </div>
  );
});

KPICards.displayName = 'KPICards';


