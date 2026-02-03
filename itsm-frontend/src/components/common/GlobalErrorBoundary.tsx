'use client';

import React, { Component, ErrorInfo, ReactNode } from 'react';
import { Result, Button, Card, Collapse, Typography } from 'antd';
import { BugOutlined, ReloadOutlined, HomeOutlined, WarningOutlined } from '@ant-design/icons';

const { Paragraph, Text } = Typography;

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  errorCount: number;
}

/**
 * 全局错误边界组件
 * 捕获子组件树中的JavaScript错误，记录错误并显示降级UI
 */
export class GlobalErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      errorCount: 0,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    // 更新 state 使下一次渲染能够显示降级后的 UI
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 记录错误信息
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    // 更新错误计数
    this.setState(prevState => ({
      errorInfo,
      errorCount: prevState.errorCount + 1,
    }));

    // 调用外部错误处理函数
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // 在生产环境中，这里可以发送错误到监控服务
    // 例如：Sentry, LogRocket, etc.
    if (process.env.NODE_ENV === 'production') {
      // sendErrorToMonitoringService(error, errorInfo);
    }
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  handleReload = () => {
    window.location.reload();
  };

  handleGoHome = () => {
    window.location.href = '/dashboard';
  };

  render() {
    const { hasError, error, errorInfo, errorCount } = this.state;
    const { children, fallback } = this.props;

    if (hasError) {
      // 如果提供了自定义fallback，使用它
      if (fallback) {
        return fallback;
      }

      // 默认错误UI
      return (
        <div
          style={{
            minHeight: '100vh',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            padding: '20px',
          }}
        >
          <Card
            style={{
              maxWidth: 800,
              width: '100%',
              boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
            }}
          >
            <Result
              status='error'
              icon={<BugOutlined style={{ fontSize: 72, color: '#ff4d4f' }} />}
              title={
                <span>
                  <WarningOutlined style={{ color: '#faad14', marginRight: 8 }} />
                  页面遇到了一些问题
                </span>
              }
              subTitle='我们已经记录了这个错误，开发团队会尽快修复。您可以尝试刷新页面或返回首页。'
              extra={[
                <Button
                  key='reload'
                  type='primary'
                  icon={<ReloadOutlined />}
                  onClick={this.handleReload}
                  size='large'
                >
                  刷新页面
                </Button>,
                <Button key='home' icon={<HomeOutlined />} onClick={this.handleGoHome} size='large'>
                  返回首页
                </Button>,
                <Button key='reset' onClick={this.handleReset} size='large'>
                  尝试恢复
                </Button>,
              ]}
            >
              {/* 错误详情（仅开发环境显示） */}
              {process.env.NODE_ENV === 'development' && error && (
                <div style={{ marginTop: 24, textAlign: 'left' }}>
                  <Collapse
                    bordered={false}
                    style={{ background: '#f5f5f5' }}
                    items={[
                      {
                        key: '1',
                        label: (
                          <Text strong>
                            <BugOutlined style={{ marginRight: 8 }} />
                            开发者信息 (仅开发环境显示)
                          </Text>
                        ),
                        children: (
                          <>
                            <div style={{ marginBottom: 16 }}>
                              <Text strong>错误次数: </Text>
                              <Text type='danger'>{errorCount}</Text>
                            </div>

                            <div style={{ marginBottom: 16 }}>
                              <Text strong>错误消息:</Text>
                              <Paragraph
                                code
                                copyable
                                style={{
                                  background: '#fff',
                                  padding: 12,
                                  borderRadius: 4,
                                  marginTop: 8,
                                }}
                              >
                                {error.message}
                              </Paragraph>
                            </div>

                            <div style={{ marginBottom: 16 }}>
                              <Text strong>错误堆栈:</Text>
                              <Paragraph
                                code
                                copyable
                                style={{
                                  background: '#fff',
                                  padding: 12,
                                  borderRadius: 4,
                                  marginTop: 8,
                                  maxHeight: 200,
                                  overflow: 'auto',
                                }}
                              >
                                {error.stack}
                              </Paragraph>
                            </div>

                            {errorInfo && (
                              <div>
                                <Text strong>组件堆栈:</Text>
                                <Paragraph
                                  code
                                  copyable
                                  style={{
                                    background: '#fff',
                                    padding: 12,
                                    borderRadius: 4,
                                    marginTop: 8,
                                    maxHeight: 200,
                                    overflow: 'auto',
                                  }}
                                >
                                  {errorInfo.componentStack}
                                </Paragraph>
                              </div>
                            )}
                          </>
                        ),
                      },
                    ]}
                  />
                </div>
              )}

              {/* 生产环境提示 */}
              {process.env.NODE_ENV === 'production' && (
                <div style={{ marginTop: 24 }}>
                  <Text type='secondary'>错误ID: {Date.now().toString(36).toUpperCase()}</Text>
                </div>
              )}
            </Result>
          </Card>
        </div>
      );
    }

    return children;
  }
}

/**
 * 功能组件错误边界Hook（实验性）
 * 注意：目前React不支持函数组件的错误边界，这个是包装器
 */
export const withErrorBoundary = <P extends object>(
  Component: React.ComponentType<P>,
  fallback?: ReactNode
) => {
  const WrappedComponent = (props: P) => (
    <GlobalErrorBoundary fallback={fallback}>
      <Component {...props} />
    </GlobalErrorBoundary>
  );

  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`;

  return WrappedComponent;
};

export default GlobalErrorBoundary;
