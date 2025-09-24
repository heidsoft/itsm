'use client';

import { cn } from '@/lib/utils';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
  color?: 'primary' | 'secondary' | 'white';
}

/**
 * 加载旋转器组件
 * 用于显示加载状态
 */
export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'md',
  className,
  color = 'primary',
}) => {
  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-6 h-6',
    lg: 'w-8 h-8',
    xl: 'w-12 h-12',
  };

  const colorClasses = {
    primary: 'text-blue-600',
    secondary: 'text-gray-600',
    white: 'text-white',
  };

  return (
    <div
      className={cn(
        'animate-spin rounded-full border-2 border-current border-t-transparent',
        sizeClasses[size],
        colorClasses[color],
        className
      )}
      role="status"
      aria-label="加载中"
    >
      <span className="sr-only">加载中...</span>
    </div>
  );
};

/**
 * 页面加载组件
 * 用于全屏加载状态
 */
export const PageLoading: React.FC<{ message?: string }> = ({
  message = '加载中...',
}) => {
  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-50">
      <LoadingSpinner size="xl" />
      <p className="mt-4 text-gray-600">{message}</p>
    </div>
  );
};

/**
 * 按钮加载组件
 * 用于按钮内的加载状态
 */
export const ButtonLoading: React.FC<{ className?: string }> = ({
  className,
}) => {
  return (
    <LoadingSpinner
      size="sm"
      color="white"
      className={cn('mr-2', className)}
    />
  );
};

/**
 * 卡片加载组件
 * 用于卡片内容的加载状态
 */
export const CardLoading: React.FC<{ message?: string }> = ({
  message = '加载中...',
}) => {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <LoadingSpinner size="lg" />
      <p className="mt-4 text-gray-600">{message}</p>
    </div>
  );
};