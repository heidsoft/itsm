import React from 'react';
import { cn } from '@/lib/utils';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info';
  size?: 'sm' | 'md' | 'lg';
  dot?: boolean;
  removable?: boolean;
  onRemove?: () => void;
}

/**
 * 统一的徽章组件
 * 基于设计系统的徽章样式
 */
export const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(
  ({
    className,
    variant = 'default',
    size = 'md',
    dot = false,
    removable = false,
    onRemove,
    children,
    ...props
  }, ref) => {
    const baseClasses = [
      'inline-flex items-center font-medium rounded-full',
      'transition-all duration-200',
    ];

    // 变体样式
    const variantClasses = {
      default: [
        'bg-gray-100 text-gray-800',
        'border border-gray-200',
      ],
      primary: [
        'bg-blue-100 text-blue-800',
        'border border-blue-200',
      ],
      secondary: [
        'bg-indigo-100 text-indigo-800',
        'border border-indigo-200',
      ],
      success: [
        'bg-green-100 text-green-800',
        'border border-green-200',
      ],
      warning: [
        'bg-yellow-100 text-yellow-800',
        'border border-yellow-200',
      ],
      error: [
        'bg-red-100 text-red-800',
        'border border-red-200',
      ],
      info: [
        'bg-cyan-100 text-cyan-800',
        'border border-cyan-200',
      ],
    };

    // 尺寸样式
    const sizeClasses = {
      sm: dot ? 'w-2 h-2' : 'px-2 py-0.5 text-xs gap-1',
      md: dot ? 'w-2.5 h-2.5' : 'px-2.5 py-1 text-sm gap-1.5',
      lg: dot ? 'w-3 h-3' : 'px-3 py-1.5 text-base gap-2',
    };

    // 点状徽章样式
    const dotClasses = dot ? [
      'rounded-full border-2 border-white shadow-sm',
    ] : [];

    return (
      <span
        ref={ref}
        className={cn(
          baseClasses,
          variantClasses[variant],
          sizeClasses[size],
          dotClasses,
          className
        )}
        {...props}
      >
        {!dot && children}
        {removable && !dot && (
          <button
            type="button"
            onClick={onRemove}
            className={cn(
              'ml-1 inline-flex items-center justify-center',
              'rounded-full hover:bg-black/10 transition-colors',
              size === 'sm' ? 'w-3 h-3' : size === 'md' ? 'w-4 h-4' : 'w-5 h-5'
            )}
          >
            <svg
              className={cn(
                'fill-current',
                size === 'sm' ? 'w-2 h-2' : size === 'md' ? 'w-2.5 h-2.5' : 'w-3 h-3'
              )}
              viewBox="0 0 20 20"
            >
              <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
            </svg>
          </button>
        )}
      </span>
    );
  }
);

Badge.displayName = 'Badge';

// 状态徽章组件
export interface StatusBadgeProps extends Omit<BadgeProps, 'variant'> {
  status: 'active' | 'inactive' | 'pending' | 'completed' | 'failed' | 'cancelled';
}

export const StatusBadge = React.forwardRef<HTMLSpanElement, StatusBadgeProps>(
  ({ status, ...props }, ref) => {
    const statusVariants = {
      active: 'success' as const,
      inactive: 'default' as const,
      pending: 'warning' as const,
      completed: 'success' as const,
      failed: 'error' as const,
      cancelled: 'default' as const,
    };

    const statusLabels = {
      active: '活跃',
      inactive: '非活跃',
      pending: '待处理',
      completed: '已完成',
      failed: '失败',
      cancelled: '已取消',
    };

    return (
      <Badge
        ref={ref}
        variant={statusVariants[status]}
        {...props}
      >
        {statusLabels[status]}
      </Badge>
    );
  }
);

StatusBadge.displayName = 'StatusBadge';

// 优先级徽章组件
export interface PriorityBadgeProps extends Omit<BadgeProps, 'variant'> {
  priority: 'low' | 'medium' | 'high' | 'critical';
}

export const PriorityBadge = React.forwardRef<HTMLSpanElement, PriorityBadgeProps>(
  ({ priority, ...props }, ref) => {
    const priorityVariants = {
      low: 'info' as const,
      medium: 'warning' as const,
      high: 'error' as const,
      critical: 'error' as const,
    };

    const priorityLabels = {
      low: '低',
      medium: '中',
      high: '高',
      critical: '紧急',
    };

    return (
      <Badge
        ref={ref}
        variant={priorityVariants[priority]}
        {...props}
      >
        {priorityLabels[priority]}
      </Badge>
    );
  }
);

PriorityBadge.displayName = 'PriorityBadge';