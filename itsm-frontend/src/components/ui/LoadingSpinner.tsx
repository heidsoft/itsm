'use client';

import React from 'react';
import { Spin } from 'antd';
import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface LoadingSpinnerProps {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
  color?: 'primary' | 'secondary' | 'white';
  tip?: string;
  ariaLabel?: string;
}

/**
 * 加载旋转器组件
 * 基于 Ant Design Spin 组件
 */
export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'md',
  className,
  color = 'primary',
  tip,
  ariaLabel = '加载中',
}) => {
  const getSize = () => {
    switch (size) {
      case 'xs':
        return 12;
      case 'sm':
        return 16;
      case 'md':
        return 24;
      case 'lg':
        return 32;
      case 'xl':
        return 48;
      default:
        return 24;
    }
  };

  const getColorClass = () => {
    switch (color) {
      case 'primary':
        return 'text-blue-600';
      case 'secondary':
        return 'text-gray-600';
      case 'white':
        return 'text-white';
      default:
        return 'text-blue-600';
    }
  };

  return (
    <span role="status" aria-label={ariaLabel} className="inline-flex items-center">
      <Spin
        indicator={
          <Loader2
            size={getSize()}
            className={cn('animate-spin', getColorClass(), className)}
          />
        }
      />
      {tip ? <span className="ml-2">{tip}</span> : null}
    </span>
  );
};

/**
 * 页面加载组件
 * 用于全屏加载状态
 */
export const PageLoading: React.FC<{ message?: string }> = ({ message = '加载中...' }) => {
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
export const ButtonLoading: React.FC<{ className?: string }> = ({ className }) => {
  return <LoadingSpinner size="sm" color="white" className={cn('mr-2', className)} />;
};

/**
 * 卡片加载组件
 * 用于卡片内容的加载状态
 */
export const CardLoading: React.FC<{ message?: string }> = ({ message = '加载中...' }) => {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <LoadingSpinner size="lg" />
      <p className="mt-4 text-gray-600">{message}</p>
    </div>
  );
};
