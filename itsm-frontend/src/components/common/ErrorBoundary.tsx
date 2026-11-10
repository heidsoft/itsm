'use client';

import type { ErrorInfo, ReactNode } from 'react';
import React, { Component } from 'react';
import { Result, Button, Card } from 'antd';
import { RefreshCw, Home, Bug } from 'lucide-react';
import { handleErrorBoundary } from '@/lib/hooks/useErrorHandler';

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  errorId: string;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  showDetails?: boolean;
  resetOnPropsChange?: boolean;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  private resetTimeoutId: number | null = null;

  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: '',
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return {
      hasError: true,
      error,
      errorId: `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({
      error,
      errorInfo,
    });

    // 调用全局错误处理
    handleErrorBoundary(error, errorInfo);

    // 调用自定义错误处理
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  componentDidUpdate(prevProps: ErrorBoundaryProps) {
    // 如果props发生变化且配置了重置，则重置错误状态
    if (this.props.resetOnPropsChange && this.state.hasError) {
      const propsChanged = Object.keys(this.props).some(
        key =>
          key !== 'children' &&
          this.props[key as keyof ErrorBoundaryProps] !== prevProps[key as keyof ErrorBoundaryProps]
      );

      if (propsChanged) {
        this.resetError();
      }
    }
  }

  componentWillUnmount() {
    if (this.resetTimeoutId) {
      clearTimeout(this.resetTimeoutId);
    }
  }

  private resetError = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: '',
    });
  };

  private handleRetry = () => {
    this.resetError();
  };

  private handleGoHome = () => {
    window.location.href = '/dashboard';
  };

  private handleReportBug = () => {
    const { error, errorInfo, errorId } = this.state;

    // 创建错误报告
    const errorReport = {
      errorId,
      message: error?.message,
      stack: error?.stack,
      componentStack: errorInfo?.componentStack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href,
    };

    // 复制到剪贴板
    navigator.clipboard
      .writeText(JSON.stringify(errorReport, null, 2))
      .then(() => {
        // 这里可以集成错误上报服务
      })
      .catch(err => {
        console.error('Failed to copy error report:', err);
      });
  };

  render() {
    if (this.state.hasError) {
      // 如果有自定义fallback，使用自定义fallback
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const { error, errorInfo } = this.state;

      return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
          <Card className="max-w-2xl w-full">
            <Result
              status="error"
              title="An error occurred on the page"
              subTitle="Sorry, the page encountered an unexpected error. We have logged this issue, please try refreshing the page or contact technical support."
              extra={[
                <Button
                  key="retry"
                  type="primary"
                  icon={<RefreshCw size={16} />}
                  onClick={this.handleRetry}
                  className="mr-2"
                >

                  Retry
                </Button>,
                <Button
                  key="home"
                  icon={<Home size={16} />}
                  onClick={this.handleGoHome}
                  className="mr-2"
                >

                  Return to homepage
                </Button>,
                <Button key="report" icon={<Bug size={16} />} onClick={this.handleReportBug}>

                  Report a problem
                </Button>,
              ]}
            />

            {this.props.showDetails && error && (
              <div className="mt-6 p-4 bg-red-50 rounded-lg">
                <h4 className="text-sm font-medium text-red-800 mb-2">Error details</h4>
                <div className="text-xs text-red-700 space-y-2">
                  <div>
                    <strong>Error message:</strong> {error.message}
                  </div>
                  <div>
                    <strong>Error ID:</strong> {this.state.errorId}
                  </div>
                  {error.stack && (
                    <div>
                      <strong>Stack information:</strong>
                      <pre className="mt-1 p-2 bg-red-100 rounded text-xs overflow-auto max-h-32">
                        {error.stack}
                      </pre>
                    </div>
                  )}
                  {errorInfo?.componentStack && (
                    <div>
                      <strong>Component stack:</strong>
                      <pre className="mt-1 p-2 bg-red-100 rounded text-xs overflow-auto max-h-32">
                        {errorInfo.componentStack}
                      </pre>
                    </div>
                  )}
                </div>
              </div>
            )}
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}


(ErrorBoundary as unknown as React.ComponentType<ErrorBoundaryProps>).displayName = 'ErrorBoundary';

// 高阶组件版本
export const withErrorBoundary = <P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<ErrorBoundaryProps, 'children'>
) => {
  const WrappedComponent = (props: P) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <Component {...props} />
    </ErrorBoundary>
  );

  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`;

  return WrappedComponent;
};

// Hook版本（用于函数组件）
export const useErrorBoundary = () => {
  const [error, setError] = React.useState<Error | null>(null);

  const resetError = React.useCallback(() => {
    setError(null);
  }, []);

  const captureError = React.useCallback((error: Error) => {
    setError(error);
  }, []);

  React.useEffect(() => {
    if (error) {
      throw error;
    }
  }, [error]);

  return { captureError, resetError };
};

export default ErrorBoundary;
