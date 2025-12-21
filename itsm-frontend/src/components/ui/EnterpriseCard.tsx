// @ts-nocheck
'use client';

import React from 'react';
import { Card, CardProps } from 'antd';
import { cn } from '@/lib/utils';

export interface EnterpriseCardProps extends CardProps {
  /**
   * 是否启用悬浮效果
   * @default true
   */
  hover?: boolean;

  /**
   * 是否显示渐变边框/光晕效果
   * @default false
   */
  gradient?: boolean;

  /**
   * 渐变色方向
   * @default 'blue-purple'
   */
  gradientVariant?: 'blue-purple' | 'green-blue' | 'orange-red' | 'pink-purple';

  /**
   * 卡片大小
   * @default 'default'
   */
  size?: 'small' | 'default' | 'large';

  /**
   * 是否显示为玻璃态效果
   * @default false
   */
  glass?: boolean;

  /**
   * 自定义阴影级别
   * @default 'sm'
   */
  shadowLevel?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
}

const gradientMap = {
  'blue-purple': 'from-blue-500/10 to-purple-500/10',
  'green-blue': 'from-green-500/10 to-blue-500/10',
  'orange-red': 'from-orange-500/10 to-red-500/10',
  'pink-purple': 'from-pink-500/10 to-purple-500/10',
};

const shadowMap = {
  none: '',
  xs: 'shadow-xs',
  sm: 'shadow-sm',
  md: 'shadow-md',
  lg: 'shadow-lg',
  xl: 'shadow-xl',
  '2xl': 'shadow-2xl',
};

const sizeMap = {
  small: { padding: 16 },
  default: { padding: 24 },
  large: { padding: 32 },
};

/**
 * 企业级卡片组件
 *
 * 特性：
 * - 统一的圆角和阴影
 * - 悬浮动画效果
 * - 可选渐变光晕
 * - 玻璃态效果
 * - GPU加速动画
 *
 * @example
 * ```tsx
 * <EnterpriseCard hover gradient>
 *   <h3>标题</h3>
 *   <p>内容</p>
 * </EnterpriseCard>
 * ```
 */
export const EnterpriseCard: React.FC<EnterpriseCardProps> = ({
  children,
  hover = true,
  gradient = false,
  gradientVariant = 'blue-purple',
  size = 'default',
  glass = false,
  shadowLevel = 'sm',
  className,
  style,
  ...props
}) => {
  const baseStyles: React.CSSProperties = {
    borderRadius: '12px',
    border: '1px solid rgba(229, 231, 235, 0.8)',
    ...sizeMap[size],
    ...style,
  };

  return (
    <div className={cn('enterprise-card-wrapper', 'relative group', className)}>
      <Card
        className={cn(
          'enterprise-card',
          'transition-all duration-300 ease-in-out',
          shadowMap[shadowLevel],
          hover && 'hover:shadow-2xl hover:-translate-y-1',
          glass && 'backdrop-blur-sm bg-white/80',
          'will-change-transform'
        )}
        style={baseStyles}
        bordered={false}
        {...props}
      >
        {/* 渐变光晕背景 - 仅在悬浮时显示 */}
        {gradient && (
          <div
            className={cn(
              'absolute inset-0 rounded-xl',
              'opacity-0 group-hover:opacity-100',
              'transition-opacity duration-300',
              'pointer-events-none',
              '-z-10'
            )}
            style={{
              background: `linear-gradient(135deg, ${
                gradientVariant === 'blue-purple'
                  ? '#3b82f6, #8b5cf6'
                  : gradientVariant === 'green-blue'
                  ? '#22c55e, #3b82f6'
                  : gradientVariant === 'orange-red'
                  ? '#f97316, #ef4444'
                  : '#ec4899, #8b5cf6'
              })`,
              opacity: 0.1,
              filter: 'blur(20px)',
            }}
          />
        )}

        {children}
      </Card>
    </div>
  );
};

/**
 * 企业级统计卡片
 *
 * 专门用于KPI指标展示的卡片变体
 */
export interface EnterpriseStatCardProps extends EnterpriseCardProps {
  title: string;
  value: string | number;
  suffix?: string;
  prefix?: string;
  trend?: {
    value: number;
    isPositive?: boolean;
  };
  icon?: React.ReactNode;
  loading?: boolean;
}

export const EnterpriseStatCard: React.FC<EnterpriseStatCardProps> = ({
  title,
  value,
  suffix,
  prefix,
  trend,
  icon,
  loading,
  ...cardProps
}) => {
  return (
    <EnterpriseCard hover gradient {...cardProps}>
      <div className='flex items-start justify-between'>
        <div className='flex-1'>
          <div className='text-sm font-medium text-gray-600 mb-2'>{title}</div>
          <div className='text-2xl font-bold text-gray-900 mb-1'>
            {loading ? (
              <div className='h-8 w-24 bg-gray-200 animate-pulse rounded' />
            ) : (
              <>
                {prefix}
                {value}
                {suffix && <span className='text-lg text-gray-600 ml-1'>{suffix}</span>}
              </>
            )}
          </div>
          {trend && !loading && (
            <div
              className={cn(
                'text-sm font-medium',
                trend.isPositive ? 'text-green-600' : 'text-red-600'
              )}
            >
              {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value)}%
            </div>
          )}
        </div>
        {icon && <div className='text-3xl text-blue-500 opacity-80'>{icon}</div>}
      </div>
    </EnterpriseCard>
  );
};

/**
 * 图表卡片组件
 *
 * 专门用于图表展示的卡片容器
 */
export interface ChartCardProps extends EnterpriseCardProps {
  chartTitle?: string;
  extra?: React.ReactNode;
  loading?: boolean;
}

export const ChartCard: React.FC<ChartCardProps> = ({
  chartTitle,
  extra,
  loading,
  children,
  ...cardProps
}) => {
  return (
    <EnterpriseCard hover {...cardProps}>
      {chartTitle && (
        <div className='flex items-center justify-between mb-4 pb-3 border-b border-gray-100'>
          <h3 className='text-lg font-semibold text-gray-900'>{chartTitle}</h3>
          {extra && <div>{extra}</div>}
        </div>
      )}
      {loading ? (
        <div className='h-64 flex items-center justify-center'>
          <div className='text-gray-400'>加载中...</div>
        </div>
      ) : (
        children
      )}
    </EnterpriseCard>
  );
};

// 保持向后兼容的导出
export const EnterpriseChartCard = ChartCard;
export type EnterpriseChartCardProps = ChartCardProps;

export default EnterpriseCard;
