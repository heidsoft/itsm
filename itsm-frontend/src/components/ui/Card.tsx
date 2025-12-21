'use client';

import React from 'react';
import { cn } from '@/lib/utils';

/**
 * 卡片属性
 */
export interface CardProps {
  /** 卡片内容 */
  children: React.ReactNode;
  /** 卡片标题 */
  title?: string;
  /** 卡片描述 */
  description?: string;
  /** 额外的操作区域 */
  extra?: React.ReactNode;
  /** 卡片封面 */
  cover?: React.ReactNode;
  /** 是否显示边框 */
  bordered?: boolean;
  /** 是否可悬停 */
  hoverable?: boolean;
  /** 是否加载中 */
  loading?: boolean;
  /** 卡片尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 卡片类型 */
  type?: 'default' | 'inner';
  /** 自定义类名 */
  className?: string;
  /** 点击事件 */
  onClick?: () => void;
}

/**
 * 卡片头部属性
 */
export interface CardHeaderProps {
  /** 标题 */
  title?: string;
  /** 描述 */
  description?: string;
  /** 额外内容 */
  extra?: React.ReactNode;
  /** 自定义类名 */
  className?: string;
}

/**
 * 卡片内容属性
 */
export interface CardContentProps {
  /** 内容 */
  children: React.ReactNode;
  /** 自定义类名 */
  className?: string;
}

/**
 * 卡片操作区属性
 */
export interface CardActionsProps {
  /** 操作内容 */
  children: React.ReactNode;
  /** 自定义类名 */
  className?: string;
}

/**
 * 卡片头部组件
 */
export const CardHeader: React.FC<CardHeaderProps> = ({
  title,
  description,
  extra,
  className,
}) => {
  return (
    <div className={cn('flex items-center justify-between p-6 border-b border-gray-100 bg-gradient-to-r from-gray-50/50 to-white', className)}>
      <div className="flex-1 min-w-0">
        {title && (
          <h3 className="text-lg font-bold text-gray-900 tracking-tight">{title}</h3>
        )}
        {description && (
          <p className="mt-1.5 text-sm text-gray-600 leading-relaxed">{description}</p>
        )}
      </div>
      {extra && (
        <div className="ml-6 flex-shrink-0">{extra}</div>
      )}
    </div>
  );
};

/**
 * 卡片内容组件
 */
export const CardContent: React.FC<CardContentProps> = ({
  children,
  className,
}) => {
  return (
    <div className={cn('p-4', className)}>
      {children}
    </div>
  );
};

/**
 * 卡片操作区组件
 */
export const CardActions: React.FC<CardActionsProps> = ({
  children,
  className,
}) => {
  return (
    <div className={cn('flex items-center justify-end gap-3 p-5 border-t border-gray-100 bg-gradient-to-b from-white to-gray-50/80', className)}>
      {children}
    </div>
  );
};

/**
 * 加载骨架屏
 */
const CardSkeleton: React.FC<{ size: 'sm' | 'md' | 'lg' }> = ({ size }) => {
  const heights = {
    sm: 'h-32',
    md: 'h-48',
    lg: 'h-64',
  };

  return (
    <div className="animate-pulse">
      <div className="p-4 border-b border-gray-200">
        <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
        <div className="h-3 bg-gray-200 rounded w-1/2"></div>
      </div>
      <div className={cn('bg-gray-200 rounded', heights[size])}></div>
    </div>
  );
};

/**
 * 通用卡片组件
 */
export const Card: React.FC<CardProps> = ({
  children,
  title,
  description,
  extra,
  cover,
  bordered = true,
  hoverable = false,
  loading = false,
  size = 'md',
  type = 'default',
  className,
  onClick,
}) => {
  // 尺寸样式
  const sizeClasses = {
    sm: 'text-sm',
    md: 'text-base',
    lg: 'text-lg',
  };

  // 类型样式
  const typeClasses = {
    default: 'bg-white',
    inner: 'bg-gray-50',
  };

  if (loading) {
    return (
      <div className={cn(
        'rounded-lg overflow-hidden',
        bordered && 'border border-gray-200',
        typeClasses[type],
        sizeClasses[size],
        className
      )}>
        <CardSkeleton size={size} />
      </div>
    );
  }

  return (
    <div
      className={cn(
        'rounded-xl overflow-hidden transition-all duration-300 ease-in-out',
        'bg-white border border-gray-200',
        'shadow-sm hover:shadow-lg hover:-translate-y-1 hover:border-blue-200',
        'backdrop-blur-sm supports-[backdrop-filter]:bg-white/95',
        bordered && 'border border-gray-200',
        hoverable && 'hover:shadow-lg hover:-translate-y-1 hover:border-blue-300 cursor-pointer active:scale-[0.98]',
        typeClasses[type],
        sizeClasses[size],
        onClick && 'cursor-pointer',
        className
      )}
      onClick={onClick}
    >
      {/* 封面 */}
      {cover && (
        <div className="overflow-hidden">
          {cover}
        </div>
      )}

      {/* 头部 */}
      {(title || description || extra) && (
        <CardHeader
          title={title}
          description={description}
          extra={extra}
        />
      )}

      {/* 内容 */}
      {children}
    </div>
  );
};

/**
 * 统计卡片属性
 */
export interface StatCardProps {
  /** 标题 */
  title: string;
  /** 数值 */
  value: string | number;
  /** 描述 */
  description?: string;
  /** 图标 */
  icon?: React.ReactNode;
  /** 趋势 */
  trend?: {
    value: number;
    type: 'up' | 'down';
  };
  /** 颜色主题 */
  color?: 'blue' | 'green' | 'red' | 'yellow' | 'purple' | 'gray';
  /** 自定义类名 */
  className?: string;
}

/**
 * 统计卡片组件
 */
export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  description,
  icon,
  trend,
  color = 'blue',
  className,
}) => {
  const colorClasses = {
    blue: 'text-blue-600 bg-blue-50',
    green: 'text-green-600 bg-green-50',
    red: 'text-red-600 bg-red-50',
    yellow: 'text-yellow-600 bg-yellow-50',
    purple: 'text-purple-600 bg-purple-50',
    gray: 'text-gray-600 bg-gray-50',
  };

  const trendColors = {
    up: 'text-green-600',
    down: 'text-red-600',
  };

  return (
    <Card className={cn('p-6 border-0 shadow-md hover:shadow-xl transition-all duration-300', className)} hoverable>
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-600 mb-1">{title}</p>
          <p className="text-2xl font-bold text-gray-900 mb-1">{value}</p>
          {description && (
            <p className="text-sm text-gray-500">{description}</p>
          )}
          {trend && (
            <div className={cn('flex items-center mt-2 text-sm', trendColors[trend.type])}>
              <span className="font-medium">
                {trend.type === 'up' ? '↗' : '↘'} {Math.abs(trend.value)}%
              </span>
              <span className="ml-1 text-gray-500">vs 上期</span>
            </div>
          )}
        </div>
        {icon && (
          <div className={cn('p-3 rounded-lg', colorClasses[color])}>
            {icon}
          </div>
        )}
      </div>
    </Card>
  );
};

/**
 * 信息卡片属性
 */
export interface InfoCardProps {
  /** 标题 */
  title: string;
  /** 内容 */
  content: React.ReactNode;
  /** 图标 */
  icon?: React.ReactNode;
  /** 状态 */
  status?: 'success' | 'warning' | 'error' | 'info';
  /** 自定义类名 */
  className?: string;
}

/**
 * 信息卡片组件
 */
export const InfoCard: React.FC<InfoCardProps> = ({
  title,
  content,
  icon,
  status,
  className,
}) => {
  const statusClasses = {
    success: 'border-green-200 bg-green-50',
    warning: 'border-yellow-200 bg-yellow-50',
    error: 'border-red-200 bg-red-50',
    info: 'border-blue-200 bg-blue-50',
  };

  const iconColors = {
    success: 'text-green-600',
    warning: 'text-yellow-600',
    error: 'text-red-600',
    info: 'text-blue-600',
  };

  return (
    <div className={cn(
      'p-5 rounded-xl border transition-all duration-200 hover:shadow-md',
      status ? statusClasses[status] : 'border-gray-200 bg-white hover:border-gray-300',
      className
    )}>
      <div className="flex items-start gap-4">
        {icon && (
          <div className={cn(
            'flex-shrink-0 mt-0.5 p-2 rounded-lg bg-white shadow-sm',
            status ? `${iconColors[status]} ${statusClasses[status].split(' ')[0]}` : 'text-gray-600 bg-gray-50'
          )}>
            {icon}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-semibold text-gray-900 mb-2">{title}</h4>
          <div className="text-sm text-gray-600 leading-relaxed">{content}</div>
        </div>
      </div>
    </div>
  );
};

// 导出所有组件
export default Card;