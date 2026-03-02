'use client';

import React, { Component, ErrorInfo, ReactNode, useState } from 'react';
import { Spin, Alert, Button, Result } from 'antd';

interface LoadingWrapperProps {
  loading: boolean;
  children: ReactNode;
  tip?: string;
  size?: 'small' | 'default' | 'large';
}

export function LoadingWrapper({
  loading,
  children,
  tip = '加载中...',
  size = 'default',
}: LoadingWrapperProps) {
  return (
    <Spin spinning={loading} tip={tip} size={size}>
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
    return <>{loadingFallback ?? <Spin tip='加载中...' spinning />}</>;
  }

  if (error) {
    return (
      errorFallback ?? (
        <Alert
          type='error'
          message='加载失败'
          description={error.message}
          action={
            <Button size='small' danger onClick={() => window.location.reload()}>
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

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('[ErrorBoundary]', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <Result
            status='error'
            title='出错了'
            subTitle={this.state.error?.message || '未知错误'}
            extra={[
              <Button key='retry' onClick={() => this.setState({ hasError: false, error: null })}>
                重试
              </Button>,
              <Button key='reload' type='primary' onClick={() => window.location.reload()}>
                刷新页面
              </Button>,
            ]}
          />
        )
      );
    }
    return this.props.children;
  }
}

interface ShowWhenProps {
  condition: boolean;
  children: ReactNode;
  otherwise?: ReactNode;
}

export function ShowWhen({ condition, children, otherwise }: ShowWhenProps) {
  return condition ? <>{children}</> : <>{otherwise}</>;
}
