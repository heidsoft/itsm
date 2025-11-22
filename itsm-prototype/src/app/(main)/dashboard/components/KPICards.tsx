'use client';

import React from 'react';
import { Card, Row, Col, Statistic, Tooltip, Spin, Progress } from 'antd';
import {
  RiseOutlined,
  FallOutlined,
  MinusOutlined,
  DashboardOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  UserOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
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
        return <ArrowUpOutlined style={{ fontSize: 18, color: '#10b981' }} />;
      case 'down':
        return <ArrowDownOutlined style={{ fontSize: 18, color: '#ef4444' }} />;
      default:
        return <MinusOutlined style={{ fontSize: 18, color: '#9ca3af' }} />;
    }
  };

  // 获取默认图标
  const getDefaultIcon = () => {
    const iconStyle = { fontSize: 24 };
    switch (metric.id) {
      case 'total-tickets':
        return <DashboardOutlined style={iconStyle} />;
      case 'open-tickets':
        return <WarningOutlined style={iconStyle} />;
      case 'resolved-tickets':
        return <CheckCircleOutlined style={iconStyle} />;
      case 'sla-compliance':
        return <ClockCircleOutlined style={iconStyle} />;
      case 'avg-resolution':
        return <ClockCircleOutlined style={iconStyle} />;
      case 'user-satisfaction':
        return <UserOutlined style={iconStyle} />;
      default:
        return <DashboardOutlined style={iconStyle} />;
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
    <Col xs={24} sm={12} md={12} lg={8} xl={6} xxl={4}>
      <Card
        className='enterprise-kpi-card h-full border-0 transition-all duration-300 hover:shadow-2xl hover:-translate-y-1 group overflow-hidden'
        style={{
          borderRadius: '12px',
          background: '#ffffff',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.08)',
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
        {/* 背景装饰渐变 */}
        <div
          className='absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500'
          style={{
            background: `radial-gradient(circle at top right, ${metric.color}15 0%, transparent 70%)`,
          }}
        />

        <div className='relative z-10 flex flex-col h-full'>
          {/* 顶部区域：图标和趋势 */}
          <div className='flex items-start justify-between mb-4'>
            <div
              className='w-12 h-12 rounded-xl flex items-center justify-center transition-all duration-300 group-hover:scale-110'
              style={{
                background: `linear-gradient(135deg, ${metric.color}20 0%, ${metric.color}10 100%)`,
                border: `2px solid ${metric.color}30`,
              }}
            >
              <div style={{ color: metric.color }}>{metric.icon || getDefaultIcon()}</div>
            </div>

            {metric.trend && (
              <div
                className='px-2.5 py-1.5 rounded-lg flex items-center gap-1.5 backdrop-blur-sm'
                style={{
                  backgroundColor:
                    metric.trend === 'up'
                      ? '#f0fdf4'
                      : metric.trend === 'down'
                      ? '#fef2f2'
                      : '#f9fafb',
                  border: `1px solid ${
                    metric.trend === 'up'
                      ? '#bbf7d0'
                      : metric.trend === 'down'
                      ? '#fecaca'
                      : '#e5e7eb'
                  }`,
                }}
              >
                {getTrendIcon()}
                {metric.change !== undefined && (
                  <span
                    className='text-xs font-bold'
                    style={{
                      color:
                        metric.trend === 'up'
                          ? '#16a34a'
                          : metric.trend === 'down'
                          ? '#dc2626'
                          : '#6b7280',
                    }}
                  >
                    {Math.abs(metric.change)}%
                  </span>
                )}
              </div>
            )}
          </div>

          {/* 标题 */}
          <div className='mb-3'>
            <Tooltip title={metric.description || metric.title}>
              <h3 className='text-sm font-medium text-gray-600 leading-tight line-clamp-2'>
                {metric.title}
              </h3>
            </Tooltip>
          </div>

          {/* 数值显示 */}
          <div className='flex-1 flex flex-col justify-center mb-3'>
            <div className='flex items-baseline gap-2'>
              <span
                className='text-4xl font-bold leading-none tracking-tight'
                style={{ color: metric.color }}
              >
                {typeof metric.value === 'number' ? metric.value.toLocaleString() : metric.value}
              </span>
              {metric.unit && (
                <span className='text-base font-medium text-gray-500'>{metric.unit}</span>
              )}
            </div>
          </div>

          {/* 底部进度条 */}
          <div className='mt-auto'>
            <Progress
              percent={getProgressPercent()}
              strokeColor={metric.color}
              showInfo={false}
              strokeWidth={4}
              trailColor='#f0f0f0'
              className='enterprise-kpi-progress'
            />
            {metric.change !== undefined && (
              <div className='flex items-center justify-between mt-2'>
                <span className='text-xs text-gray-500'>vs 上期</span>
                <span
                  className='text-xs font-semibold'
                  style={{
                    color:
                      metric.changeType === 'increase'
                        ? '#16a34a'
                        : metric.changeType === 'decrease'
                        ? '#dc2626'
                        : '#6b7280',
                  }}
                >
                  {metric.changeType === 'increase'
                    ? '+'
                    : metric.changeType === 'decrease'
                    ? '-'
                    : ''}
                  {Math.abs(metric.change)}%
                </span>
              </div>
            )}
          </div>
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
                  borderRadius: '12px',
                  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
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
            borderRadius: '12px',
            background: 'linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%)',
          }}
        >
          <div className='text-gray-500'>
            <div className='w-16 h-16 rounded-2xl bg-gray-100 flex items-center justify-center mx-auto mb-4'>
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
      </Row>
    </div>
  );
});

KPICards.displayName = 'KPICards';
