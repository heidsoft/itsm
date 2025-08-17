"use client";

import React from "react";
import { Button, Empty, Result, Spin, Typography } from "antd";
import {
  RotateCcw,
  Plus,
  Search,
  FileText,
  AlertTriangle,
  CheckCircle,
  Clock,
  User,
  Database,
  Settings,
} from "lucide-react";

const { Text, Title } = Typography;

export type LoadingEmptyErrorState = "loading" | "empty" | "error" | "success";

interface LoadingEmptyErrorProps {
  state: LoadingEmptyErrorState;
  loadingText?: string;
  empty?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    icon?: React.ReactNode;
    showAction?: boolean;
  };
  error?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    showRetry?: boolean;
    showAction?: boolean;
  };
  success?: {
    title?: string;
    description?: string;
    actionText?: string;
    onAction?: () => void;
    showAction?: boolean;
  };
  minHeight?: number;
  className?: string;
  children?: React.ReactNode;
}

// 预定义的图标映射
const getDefaultIcon = (state: LoadingEmptyErrorState, context?: string) => {
  switch (state) {
    case "empty":
      switch (context) {
        case "tickets":
          return <FileText size={48} />;
        case "incidents":
          return <AlertTriangle size={48} />;
        case "problems":
          return <AlertTriangle size={48} />;
        case "changes":
          return <Settings size={48} />;
        case "cmdb":
          return <Database size={48} />;
        case "users":
          return <User size={48} />;
        case "workflows":
          return <Settings size={48} />;
        default:
          return <FileText size={48} />;
      }
    case "error":
      return <AlertTriangle size={48} />;
    case "success":
      return <CheckCircle size={48} />;
    default:
      return <FileText size={48} />;
  }
};

// 预定义的空状态配置
const getDefaultEmptyConfig = (context?: string) => {
  switch (context) {
    case "tickets":
      return {
        title: "暂无工单",
        description: "当前没有工单数据，点击下方按钮创建第一个工单",
        actionText: "创建工单",
        icon: <FileText size={48} />,
      };
    case "incidents":
      return {
        title: "暂无事件",
        description: "当前没有事件数据，点击下方按钮创建第一个事件",
        actionText: "创建事件",
        icon: <AlertTriangle size={48} />,
      };
    case "problems":
      return {
        title: "暂无问题",
        description: "当前没有问题数据，点击下方按钮创建第一个问题",
        actionText: "创建问题",
        icon: <AlertTriangle size={48} />,
      };
    case "changes":
      return {
        title: "暂无变更",
        description: "当前没有变更数据，点击下方按钮创建第一个变更",
        actionText: "创建变更",
        icon: <Settings size={48} />,
      };
    case "cmdb":
      return {
        title: "暂无配置项",
        description: "当前没有配置项数据，点击下方按钮创建第一个配置项",
        actionText: "创建配置项",
        icon: <Database size={48} />,
      };
    case "users":
      return {
        title: "暂无用户",
        description: "当前没有用户数据，点击下方按钮创建第一个用户",
        actionText: "创建用户",
        icon: <User size={48} />,
      };
    case "workflows":
      return {
        title: "暂无工作流",
        description: "当前没有工作流数据，点击下方按钮创建第一个工作流",
        actionText: "创建工作流",
        icon: <Settings size={48} />,
      };
    default:
      return {
        title: "暂无数据",
        description: "当前没有数据，点击下方按钮创建第一条记录",
        actionText: "创建",
        icon: <FileText size={48} />,
      };
  }
};

export const LoadingEmptyError: React.FC<LoadingEmptyErrorProps> = ({
  state,
  loadingText = "加载中...",
  empty,
  error,
  success,
  minHeight = 200,
  className = "",
  children,
}) => {
  // 成功状态直接显示内容
  if (state === "success") {
    return (
      <div className={className}>
        {success?.title && (
          <div className="mb-4 text-center">
            <Title level={4} className="text-gray-700">
              {success.title}
            </Title>
            {success.description && (
              <Text type="secondary" className="text-sm">
                {success.description}
              </Text>
            )}
            {success.showAction && success.actionText && success.onAction && (
              <div className="mt-3">
                <Button type="primary" onClick={success.onAction}>
                  {success.actionText}
                </Button>
              </div>
            )}
          </div>
        )}
        {children}
      </div>
    );
  }

  // 加载状态
  if (state === "loading") {
    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <Spin size="large" />
        <Text className="mt-4 text-gray-500">{loadingText}</Text>
      </div>
    );
  }

  // 空状态
  if (state === "empty") {
    const defaultConfig = getDefaultEmptyConfig();
    const config = {
      ...defaultConfig,
      ...empty,
      icon: empty?.icon || defaultConfig.icon,
    };

    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <div className="text-gray-300 mb-4">{config.icon}</div>
        <Title level={4} className="text-gray-600 mb-2">
          {config.title}
        </Title>
        <Text type="secondary" className="text-center mb-6 max-w-md">
          {config.description}
        </Text>
        {config.showAction !== false &&
          config.actionText &&
          config.onAction && (
            <Button
              type="primary"
              icon={<Plus size={16} />}
              onClick={config.onAction}
            >
              {config.actionText}
            </Button>
          )}
      </div>
    );
  }

  // 错误状态
  if (state === "error") {
    const defaultErrorConfig = {
      title: "加载失败",
      description: "数据加载失败，请检查网络连接或稍后重试",
      actionText: "重试",
      showRetry: true,
      showAction: false,
    };
    const config = { ...defaultErrorConfig, ...error };

    return (
      <div
        className={`flex flex-col items-center justify-center ${className}`}
        style={{ minHeight }}
      >
        <Result
          status="error"
          icon={<AlertTriangle style={{ color: "#ff4d4f" }} />}
          title={config.title}
          subTitle={config.description}
          extra={
            <div className="flex gap-2">
              {config.showRetry && (
                <Button
                  type="primary"
                  icon={<RotateCcw size={16} />}
                  onClick={config.onAction}
                >
                  {config.actionText}
                </Button>
              )}
              {config.showAction && config.actionText && config.onAction && (
                <Button onClick={config.onAction}>{config.actionText}</Button>
              )}
            </div>
          }
        />
      </div>
    );
  }

  return null;
};

// 预定义的上下文组件
export const TicketsLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("tickets")} />;

export const IncidentsLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("incidents")} />
);

export const ProblemsLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("problems")} />;

export const ChangesLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("changes")} />;

export const CMDBLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("cmdb")} />;

export const UsersLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("users")} />;

export const WorkflowsLoadingEmptyError = (
  props: Omit<LoadingEmptyErrorProps, "empty">
) => (
  <LoadingEmptyError {...props} empty={getDefaultEmptyConfig("workflows")} />
);

// 简化的状态组件
export const Loading = ({ text = "加载中...", className = "" }) => (
  <LoadingEmptyError state="loading" loadingText={text} className={className} />
);

export const Empty = ({
  title = "暂无数据",
  description = "当前没有数据",
  actionText,
  onAction,
  className = "",
}) => (
  <LoadingEmptyError
    state="empty"
    empty={{ title, description, actionText, onAction }}
    className={className}
  />
);

export const Error = ({
  title = "加载失败",
  description = "数据加载失败",
  actionText = "重试",
  onAction,
  className = "",
}) => (
  <LoadingEmptyError
    state="error"
    error={{ title, description, actionText, onAction }}
    className={className}
  />
);

export default LoadingEmptyError;
