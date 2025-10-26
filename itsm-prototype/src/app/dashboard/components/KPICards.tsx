'use client';

import React from 'react';
import { Card, Row, Col, Statistic, Tooltip, Spin } from 'antd';
import {
  TrendingUp,
  TrendingDown,
  Minus,
  Activity,
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

// KPI卡片组件
const KPICard: React.FC<{ metric: KPIMetric }> = React.memo(({ metric }) => {
  // 获取趋势图标
  const getTrendIcon = () => {
    switch (metric.trend) {
      case 'up':
        return <TrendingUp size={16} className='text-green-500' />;
      case 'down':
        return <TrendingDown size={16} className='text-red-500' />;
      default:
        return <Minus size={16} className='text-gray-500' />;
    }
  };

  // 获取变化颜色
  const getChangeColor = () => {
    switch (metric.changeType) {
      case 'increase':
        return 'text-green-600';
      case 'decrease':
        return 'text-red-600';
      default:
        return 'text-gray-600';
    }
  };

  // 获取变化符号
  const getChangeSymbol = () => {
    switch (metric.changeType) {
      case 'increase':
        return '+';
      case 'decrease':
        return '-';
      default:
        return '';
    }
  };

  // 获取默认图标
  const getDefaultIcon = () => {
    switch (metric.id) {
      case 'total-tickets':
        return <Activity size={20} className='text-blue-500' />;
      case 'open-tickets':
        return <AlertTriangle size={20} className='text-orange-500' />;
      case 'resolved-tickets':
        return <CheckCircle size={20} className='text-green-500' />;
      case 'sla-compliance':
        return <Clock size={20} className='text-purple-500' />;
      case 'avg-resolution':
        return <Clock size={20} className='text-cyan-500' />;
      case 'user-satisfaction':
        return <User size={20} className='text-pink-500' />;
      default:
        return <Activity size={20} className='text-gray-500' />;
    }
  };

  return (
    <Col xs={24} sm={12} lg={8} xl={4}>
      <Card
        className='h-full hover:shadow-lg transition-all duration-300 border-0'
        style={{
          background: `linear-gradient(135deg, ${metric.color}15 0%, ${metric.color}05 100%)`,
          borderLeft: `4px solid ${metric.color}`,
        }}
        bodyStyle={{ padding: '20px' }}
      >
        <div className='flex items-start justify-between mb-3'>
          <div className='flex items-center gap-2'>
            {metric.icon || getDefaultIcon()}
            <span className='text-sm font-medium text-gray-600'>{metric.title}</span>
          </div>
          {metric.trend && <Tooltip title={`Trend: ${metric.trend}`}>{getTrendIcon()}</Tooltip>}
        </div>

        <div className='mb-2'>
          <Statistic
            value={metric.value}
            suffix={metric.unit}
            valueStyle={{
              fontSize: '24px',
              fontWeight: 'bold',
              color: metric.color,
            }}
          />
        </div>

        {metric.change !== undefined && (
          <div className='flex items-center gap-1'>
            <span className={`text-xs font-medium ${getChangeColor()}`}>
              {getChangeSymbol()}
              {Math.abs(metric.change)}%
            </span>
            <span className='text-xs text-gray-500'>vs last period</span>
          </div>
        )}

        {metric.description && (
          <Tooltip title={metric.description}>
            <div className='mt-2 text-xs text-gray-500 truncate'>{metric.description}</div>
          </Tooltip>
        )}
      </Card>
    </Col>
  );
});

KPICard.displayName = 'KPICard';

// KPI卡片组组件
export const KPICards: React.FC<KPICardsProps> = React.memo(({ metrics, loading = false }) => {
  if (loading) {
    return (
      <div className='mb-6'>
        <Row gutter={[16, 16]}>
          {Array.from({ length: 6 }).map((_, index) => (
            <Col key={index} xs={24} sm={12} lg={8} xl={4}>
              <Card className='h-32'>
                <div className='flex items-center justify-center h-full'>
                  <Spin size='large' />
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
        <Card className='text-center py-8'>
          <div className='text-gray-500'>
            <Activity size={48} className='mx-auto mb-4 text-gray-300' />
            <p>暂无KPI数据</p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className='mb-6'>
      <div className='mb-4'>
        <h2 className='text-lg font-semibold text-gray-900 mb-1'>关键绩效指标</h2>
        <p className='text-sm text-gray-600'>实时指标和系统性能</p>
      </div>

      <Row gutter={[16, 16]}>
        {metrics.map(metric => (
          <KPICard key={metric.id} metric={metric} />
        ))}
      </Row>
    </div>
  );
});

KPICards.displayName = 'KPICards';
