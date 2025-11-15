'use client';

import React from 'react';
import { Spin, Empty, Result, Button } from 'antd';
import { LoadingOutlined, CloseCircleOutlined, ReloadOutlined } from '@ant-design/icons';

// 加载中组件
export const LoadingSpinner: React.FC<{
  size?: 'small' | 'default' | 'large';
  tip?: string;
  fullscreen?: boolean;
}> = ({ size = 'default', tip = '加载中...', fullscreen = false }) => {
  const antIcon = <LoadingOutlined style={{ fontSize: size === 'small' ? 20 : size === 'large' ? 36 : 28 }} spin />;

  if (fullscreen) {
    return (
      <div className='fixed inset-0 flex items-center justify-center bg-white bg-opacity-90 z-50'>
        <div className='text-center'>
          <Spin indicator={antIcon} size={size} />
          {tip && <div className='mt-4 text-sm text-gray-600'>{tip}</div>}
        </div>
      </div>
    );
  }

  return (
    <div className='flex items-center justify-center py-12'>
      <div className='text-center'>
        <Spin indicator={antIcon} size={size} />
        {tip && <div className='mt-2 text-sm text-gray-600'>{tip}</div>}
      </div>
    </div>
  );
};

// 空状态组件
export const EmptyState: React.FC<{
  title?: string;
  description?: string;
  image?: React.ReactNode;
  action?: {
    text: string;
    onClick: () => void;
    icon?: React.ReactNode;
  };
}> = ({ 
  title = '暂无数据', 
  description,
  image,
  action 
}) => {
  return (
    <div className='flex items-center justify-center py-12'>
      <Empty
        image={image || Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <div>
            <p className='text-base font-medium text-gray-900 mb-1'>{title}</p>
            {description && <p className='text-sm text-gray-500'>{description}</p>}
          </div>
        }
      >
        {action && (
          <Button 
            type='primary' 
            onClick={action.onClick}
            icon={action.icon}
          >
            {action.text}
          </Button>
        )}
      </Empty>
    </div>
  );
};

// 错误状态组件
export const ErrorState: React.FC<{
  title?: string;
  description?: string;
  error?: Error | string;
  onRetry?: () => void;
  showDetails?: boolean;
}> = ({ 
  title = '加载失败',
  description = '抱歉，数据加载失败，请稍后重试',
  error,
  onRetry,
  showDetails = false
}) => {
  const errorMessage = error instanceof Error ? error.message : error;

  return (
    <div className='flex items-center justify-center py-12'>
      <Result
        status='error'
        icon={<CloseCircleOutlined style={{ color: '#ef4444' }} />}
        title={<span className='text-lg font-semibold'>{title}</span>}
        subTitle={<span className='text-sm text-gray-600'>{description}</span>}
        extra={[
          onRetry && (
            <Button 
              key='retry' 
              type='primary' 
              onClick={onRetry}
              icon={<ReloadOutlined />}
            >
              重新加载
            </Button>
          ),
        ].filter(Boolean)}
      >
        {showDetails && errorMessage && (
          <div className='mt-4 p-4 bg-red-50 rounded-lg border border-red-200'>
            <p className='text-xs text-red-800 font-mono'>{errorMessage}</p>
          </div>
        )}
      </Result>
    </div>
  );
};

// 骨架屏加载组件
export const SkeletonLoader: React.FC<{
  type?: 'card' | 'list' | 'table' | 'form';
  count?: number;
}> = ({ type = 'card', count = 3 }) => {
  if (type === 'card') {
    return (
      <div className='space-y-4'>
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className='bg-white rounded-lg p-4 border border-gray-200 animate-pulse'>
            <div className='h-4 bg-gray-200 rounded w-3/4 mb-3'></div>
            <div className='space-y-2'>
              <div className='h-3 bg-gray-200 rounded'></div>
              <div className='h-3 bg-gray-200 rounded w-5/6'></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (type === 'list') {
    return (
      <div className='space-y-3'>
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className='flex items-center space-x-3 animate-pulse'>
            <div className='h-10 w-10 bg-gray-200 rounded-full'></div>
            <div className='flex-1 space-y-2'>
              <div className='h-3 bg-gray-200 rounded w-1/4'></div>
              <div className='h-2 bg-gray-200 rounded w-1/2'></div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (type === 'table') {
    return (
      <div className='space-y-2'>
        <div className='h-10 bg-gray-200 rounded animate-pulse'></div>
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className='h-12 bg-gray-100 rounded animate-pulse'></div>
        ))}
      </div>
    );
  }

  if (type === 'form') {
    return (
      <div className='space-y-4'>
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className='animate-pulse'>
            <div className='h-3 bg-gray-200 rounded w-1/4 mb-2'></div>
            <div className='h-10 bg-gray-100 rounded'></div>
          </div>
        ))}
      </div>
    );
  }

  return null;
};

// 页面加载包装器
export const PageLoader: React.FC<{
  loading: boolean;
  error?: Error | string;
  isEmpty?: boolean;
  onRetry?: () => void;
  emptyConfig?: {
    title?: string;
    description?: string;
    action?: {
      text: string;
      onClick: () => void;
    };
  };
  children: React.ReactNode;
}> = ({ 
  loading, 
  error, 
  isEmpty = false,
  onRetry,
  emptyConfig,
  children 
}) => {
  if (loading) {
    return <LoadingSpinner />;
  }

  if (error) {
    return <ErrorState error={error} onRetry={onRetry} />;
  }

  if (isEmpty) {
    return <EmptyState {...emptyConfig} />;
  }

  return <>{children}</>;
};

