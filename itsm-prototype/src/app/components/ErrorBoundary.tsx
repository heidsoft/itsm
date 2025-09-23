"use client";

import React, { Component, ErrorInfo, ReactNode } from "react";
import { Button, Result, Typography } from "antd";
import { RefreshCw, Home, Bug } from "lucide-react";

const { Paragraph, Text } = Typography;

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
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
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("ErrorBoundary caught error:", error, errorInfo);
    this.setState({
      error,
      errorInfo,
    });

    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  handleReload = () => {
    window.location.reload();
  };

  handleGoHome = () => {
    window.location.href = "/dashboard";
  };

  handleReportError = () => {
    const errorReport = {
      error: this.state.error?.message,
      stack: this.state.error?.stack,
      componentStack: this.state.errorInfo?.componentStack,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
      url: window.location.href,
    };

    console.log("Error report:", errorReport);
    // Here you would typically send the error report to your error tracking service
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

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
          <div
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
            <Result
              status="error"
              title="System encountered some issues"
              subTitle="Sorry, an error occurred while loading the page. Our technical team has been notified and is actively working on it."
              extra={[
                <Button
                  key="reload"
                  type="primary"
                  icon={<RefreshCw />}
                  onClick={this.handleReload}
                  size="large"
                >
                  Reload Page
                </Button>,
                <Button
                  key="home"
                  icon={<Home />}
                  onClick={this.handleGoHome}
                  size="large"
                >
                  Back to Home
                </Button>,
                <Button
                  key="report"
                  icon={<Bug />}
                  onClick={this.handleReportError}
                  size="large"
                >
                  Report Issue
                </Button>,
              ]}
            />

            <div style={{ marginTop: "24px", textAlign: "left" }}>
              <Paragraph style={{ marginBottom: "16px" }}>
                <strong>Error Details:</strong>
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
                      Stack Trace
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
                  If the problem persists, please contact technical support or try refreshing the page.
                </Text>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

// Global error handler hook
export const useErrorHandler = () => {
  React.useEffect(() => {
    const handleError = (event: ErrorEvent) => {
      console.error("Hook caught error:", event.error);
    };

    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      console.error("Hook caught unhandled Promise rejection:", event.reason);
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
};

// Simple error fallback component
export const SimpleErrorFallback: React.FC<{
  error?: Error;
}> = ({ error }) => {
  return (
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
        Something went wrong
      </h2>
      <p style={{ color: "#666", marginBottom: "24px" }}>
        {error?.message || "Unknown error"}
      </p>
      <Button type="primary" onClick={() => window.location.reload()}>
        Reload Page
      </Button>
    </div>
  );
};

export default ErrorBoundary;
