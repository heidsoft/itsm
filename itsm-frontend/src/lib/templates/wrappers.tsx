'use client';

import type { ReactNode } from 'react';
import React from 'react';
import { Spin, Alert, Button, Result } from 'antd';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';

interface LoadingWrapperProps {
  loading: boolean;
  children: ReactNode;
  description?: string;
  size?: 'small' | 'default' | 'large';
}

export function LoadingWrapper({
  loading,
  children,
  description: desc = '加载中...',
  size = 'default',
}: LoadingWrapperProps) {
  return (
    <Spin spinning={loading} description={desc} size={size}>
      {children}
    </Spin>
  );
}

interface AsyncDataWrapperProps<T> {
  data: T | undefined;
  isLoading: boolean;
  error: Error | null;
  children: (data: T) => ReactNode;
  loadingFallback?: ReactNode;
  errorFallback?: ReactNode;
  emptyFallback?: ReactNode;
}

export function AsyncDataWrapper<T>({
  data,
  isLoading,
  error,
  children,
  loadingFallback,
  errorFallback,
  emptyFallback,
}: AsyncDataWrapperProps<T>) {
  if (isLoading) {
    return <>{loadingFallback ?? <Spin description="加载中..." spinning />}</>;
  }

  if (error) {
    return (
      errorFallback ?? (
        <Alert
          type="error"
          message="加载失败"
          description={error.message}
          action={
            <Button size="small" danger onClick={() => window.location.reload()}>
              刷新
            </Button>
          }
        />
      )
    );
  }

  if (emptyFallback && (!data || (Array.isArray(data) && data.length === 0))) {
    return <>{emptyFallback}</>;
  }

  return <>{children(data as T)}</>;
}

// Re-export ErrorBoundary from the centralized component
export { ErrorBoundary };

interface ShowWhenProps {
  condition: boolean;
  children: ReactNode;
  otherwise?: ReactNode;
}

export function ShowWhen({ condition, children, otherwise }: ShowWhenProps) {
  return condition ? <>{children}</> : <>{otherwise}</>;
}
