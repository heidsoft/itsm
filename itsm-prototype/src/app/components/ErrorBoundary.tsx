"use client";

import React, { Component, ErrorInfo, ReactNode } from "react";
import { Button, Result, Typography } from "antd";
import { AlertTriangle, RefreshCw, Home } from "lucide-react";

const { Text } = Typography;

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Error caught by boundary:", error, errorInfo);
    this.setState({
      error,
      errorInfo,
    });

    // 可以在这里发送错误到监控服务
    // logErrorToService(error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined });
  };

  handleGoHome = () => {
    window.location.href = "/dashboard";
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div style={{ 
          minHeight: "100vh", 
          display: "flex", 
          alignItems: "center", 
          justifyContent: "center",
          background: "#f5f5f5"
        }}>
          <Result
            status="error"
            icon={<AlertTriangle style={{ color: "#ff4d4f" }} />}
            title="页面出现错误"
            subTitle={
              <div style={{ textAlign: "left", maxWidth: 500 }}>
                <Text type="secondary">
                  很抱歉，页面遇到了一个意外错误。这可能是由于：
                </Text>
                <ul style={{ marginTop: 16, paddingLeft: 20 }}>
                  <li>
                    <Text type="secondary">网络连接问题</Text>
                  </li>
                  <li>
                    <Text type="secondary">服务器暂时不可用</Text>
                  </li>
                  <li>
                    <Text type="secondary">浏览器兼容性问题</Text>
                  </li>
                </ul>
                {process.env.NODE_ENV === "development" && this.state.error && (
                  <details style={{ marginTop: 16 }}>
                    <summary style={{ cursor: "pointer", color: "#1890ff" }}>
                      查看错误详情
                    </summary>
                    <div style={{ 
                      marginTop: 8, 
                      padding: 12, 
                      background: "#f6f6f6", 
                      borderRadius: 4,
                      fontFamily: "monospace",
                      fontSize: "12px",
                      overflow: "auto"
                    }}>
                      <div><strong>错误信息:</strong> {this.state.error.message}</div>
                      <div><strong>错误堆栈:</strong></div>
                      <pre style={{ margin: "8px 0", whiteSpace: "pre-wrap" }}>
                        {this.state.error.stack}
                      </pre>
                      {this.state.errorInfo && (
                        <>
                          <div><strong>组件堆栈:</strong></div>
                          <pre style={{ margin: "8px 0", whiteSpace: "pre-wrap" }}>
                            {this.state.errorInfo.componentStack}
                          </pre>
                        </>
                      )}
                    </div>
                  </details>
                )}
              </div>
            }
            extra={[
              <Button
                key="retry"
                type="primary"
                icon={<RefreshCw size={16} />}
                onClick={this.handleRetry}
                size="large"
              >
                重试
              </Button>,
              <Button
                key="home"
                icon={<Home size={16} />}
                onClick={this.handleGoHome}
                size="large"
              >
                返回首页
              </Button>,
            ]}
          />
        </div>
      );
    }

    return this.props.children;
  }
}

// 函数式组件的错误边界 Hook
export const useErrorHandler = () => {
  const [error, setError] = React.useState<Error | null>(null);

  const handleError = React.useCallback((error: Error) => {
    console.error("Error caught by hook:", error);
    setError(error);
  }, []);

  const clearError = React.useCallback(() => {
    setError(null);
  }, []);

  return { error, handleError, clearError };
};

// 异步错误处理 Hook
export const useAsyncError = () => {
  const [, setError] = React.useState();
  return React.useCallback(
    (e: Error) => {
      setError(() => {
        throw e;
      });
    },
    []
  );
};
