"use client";

import React, { Component, ErrorInfo, ReactNode } from "react";
import { Button, Result, Typography } from "antd";
import { RefreshCw, Home, Bug } from "lucide-react";

const { Text, Paragraph } = Typography;

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    // 更新状态，下次渲染时显示错误UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 记录错误信息
    console.error("ErrorBoundary捕获到错误:", error, errorInfo);

    this.setState({
      error,
      errorInfo,
    });

    // 这里可以发送错误到错误报告服务
    // logErrorToService(error, errorInfo);
  }

  handleReload = () => {
    window.location.reload();
  };

  handleGoHome = () => {
    window.location.href = "/dashboard";
  };

  handleReportError = () => {
    // 这里可以实现错误报告功能
    const errorReport = {
      message: this.state.error?.message,
      stack: this.state.error?.stack,
      componentStack: this.state.errorInfo?.componentStack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href,
    };

    console.log("错误报告:", errorReport);

    // 可以发送到后端API或错误报告服务
    // sendErrorReport(errorReport);
  };

  render() {
    if (this.state.hasError) {
      // 自定义错误UI
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // 默认错误UI
      return (
        <div
          style={{
            minHeight: "100vh",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
            padding: "20px",
          }}
        >
          <Result
            status="error"
            title="系统遇到了一些问题"
            subTitle="抱歉，页面加载时出现了错误。我们的技术团队已经收到通知，正在积极处理。"
            extra={[
              <Button
                key="reload"
                type="primary"
                icon={<RefreshCw />}
                onClick={this.handleReload}
                size="large"
                style={{ marginRight: 8 }}
              >
                重新加载
              </Button>,
              <Button
                key="home"
                icon={<Home />}
                onClick={this.handleGoHome}
                size="large"
                style={{ marginRight: 8 }}
              >
                返回首页
              </Button>,
              <Button
                key="report"
                icon={<Bug />}
                onClick={this.handleReportError}
                size="large"
              >
                报告问题
              </Button>,
            ]}
            style={{
              background: "white",
              borderRadius: "16px",
              boxShadow:
                "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)",
              padding: "40px",
              maxWidth: "600px",
              width: "100%",
            }}
          >
            <div style={{ marginTop: "24px", textAlign: "left" }}>
              <Paragraph style={{ marginBottom: "16px" }}>
                <Text strong>错误详情：</Text>
              </Paragraph>
              <div
                style={{
                  background: "#f5f5f5",
                  padding: "16px",
                  borderRadius: "8px",
                  fontFamily: "monospace",
                  fontSize: "12px",
                  overflow: "auto",
                  maxHeight: "200px",
                }}
              >
                <Text type="secondary">{this.state.error?.message}</Text>
                {process.env.NODE_ENV === "development" && (
                  <details style={{ marginTop: "12px" }}>
                    <summary style={{ cursor: "pointer", color: "#1890ff" }}>
                      查看技术详情
                    </summary>
                    <div style={{ marginTop: "8px" }}>
                      <Text type="secondary" style={{ whiteSpace: "pre-wrap" }}>
                        {this.state.error?.stack}
                      </Text>
                    </div>
                  </details>
                )}
              </div>

              <div
                style={{
                  marginTop: "16px",
                  padding: "12px",
                  background: "#e6f7ff",
                  border: "1px solid #91d5ff",
                  borderRadius: "6px",
                }}
              >
                <Text type="secondary" style={{ fontSize: "12px" }}>
                  💡
                  提示：如果问题持续存在，请尝试清除浏览器缓存或联系技术支持。
                </Text>
              </div>
            </div>
          </Result>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;

// 函数式组件的错误边界Hook
export function useErrorBoundary() {
  const [hasError, setHasError] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    const handleError = (event: ErrorEvent) => {
      setHasError(true);
      setError(event.error);
      console.error("Hook捕获到错误:", event.error);
    };

    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      setHasError(true);
      setError(new Error(event.reason));
      console.error("Hook捕获到未处理的Promise拒绝:", event.reason);
    };

    window.addEventListener("error", handleError);
    window.addEventListener("unhandledrejection", handleUnhandledRejection);

    return () => {
      window.removeEventListener("error", handleError);
      window.removeEventListener(
        "unhandledrejection",
        handleUnhandledRejection
      );
    };
  }, []);

  return { hasError, error };
}

// 简化的错误边界组件
export function SimpleErrorBoundary({ children, fallback }: Props) {
  const { hasError, error } = useErrorBoundary();

  if (hasError) {
    return (
      fallback || (
        <div
          style={{
            padding: "40px",
            textAlign: "center",
            background: "#fafafa",
            minHeight: "400px",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <Bug
            style={{ fontSize: "48px", color: "#ff4d4f", marginBottom: "16px" }}
          />
          <h2 style={{ color: "#ff4d4f", marginBottom: "8px" }}>
            页面出现错误
          </h2>
          <p style={{ color: "#666", marginBottom: "24px" }}>
            {error?.message || "未知错误"}
          </p>
          <Button type="primary" onClick={() => window.location.reload()}>
            重新加载
          </Button>
        </div>
      )
    );
  }

  return <>{children}</>;
}
