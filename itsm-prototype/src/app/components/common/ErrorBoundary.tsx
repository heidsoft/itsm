"use client";

import React, { Component, ErrorInfo, ReactNode } from 'react';
import { Result, Button, Card, Typography, Space } from 'antd';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';

const { Text, Paragraph } = Typography;

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  showDetails?: boolean;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  errorId: string;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: '',
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    // 生成错误ID用于追踪
    const errorId = `ERR_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    return {
      hasError: true,
      error,
      errorId,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 记录错误信息
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    
    this.setState({
      error,
      errorInfo,
    });

    // 调用外部错误处理函数
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // 在生产环境中，这里可以发送错误报告到监控服务
    if (process.env.NODE_ENV === 'production') {
      // 示例：发送错误到监控服务
      // errorReportingService.captureException(error, {
      //   extra: errorInfo,
      //   tags: { errorBoundary: true }
      // });
    }
  }

  handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      errorId: '',
    });
  };

  handleGoHome = () => {
    window.location.href = '/dashboard';
  };

  render() {
    if (this.state.hasError) {
      // 如果提供了自定义fallback，使用它
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // 默认错误UI
      return (
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <Card className="max-w-2xl w-full">
            <Result
              icon={<AlertTriangle className="text-red-500" size={64} />}
              title="页面出现错误"
              subTitle="抱歉，页面遇到了一些问题。请尝试刷新页面或联系技术支持。"
              extra={
                <Space direction="vertical" size="large" className="w-full">
                  <Space>
                    <Button 
                      type="primary" 
                      icon={<RefreshCw size={16} />}
                      onClick={this.handleRetry}
                    >
                      重试
                    </Button>
                    <Button 
                      icon={<Home size={16} />}
                      onClick={this.handleGoHome}
                    >
                      返回首页
                    </Button>
                  </Space>
                  
                  {this.props.showDetails && this.state.error && (
                    <Card size="small" className="text-left">
                      <Space direction="vertical" size="small" className="w-full">
                        <Text strong>错误详情:</Text>
                        <Text code className="text-red-600">
                          {this.state.error.message}
                        </Text>
                        
                        <Text strong>错误ID:</Text>
                        <Text code>{this.state.errorId}</Text>
                        
                        {this.state.errorInfo && (
                          <>
                            <Text strong>组件堆栈:</Text>
                            <Paragraph>
                              <pre className="text-xs bg-gray-100 p-2 rounded overflow-auto max-h-32">
                                {this.state.errorInfo.componentStack}
                              </pre>
                            </Paragraph>
                          </>
                        )}
                      </Space>
                    </Card>
                  )}
                  
                  <Text type="secondary" className="text-sm">
                    如果问题持续存在，请将错误ID提供给技术支持团队
                  </Text>
                </Space>
              }
            />
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}

// 函数式组件包装器，用于更简单的使用
export const withErrorBoundary = <P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<Props, 'children'>
) => {
  const WrappedComponent = (props: P) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <Component {...props} />
    </ErrorBoundary>
  );
  
  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`;
  
  return WrappedComponent;
};

// 轻量级错误边界，用于小组件
export const SimpleErrorBoundary: React.FC<{
  children: ReactNode;
  fallback?: ReactNode;
}> = ({ children, fallback }) => {
  return (
    <ErrorBoundary
      fallback={
        fallback || (
          <div className="p-4 text-center">
            <AlertTriangle className="text-red-500 mx-auto mb-2" size={24} />
            <Text type="secondary">组件加载失败</Text>
          </div>
        )
      }
    >
      {children}
    </ErrorBoundary>
  );
};

export default ErrorBoundary;