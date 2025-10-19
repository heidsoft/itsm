"use client";

import React, { ReactNode, useState, useEffect } from "react";
import { theme } from "antd";
import { cn } from "@/lib/utils";
import { animation } from "@/lib/design-system/spacing";
import {
  AlertCircle,
  CheckCircle,
  Info,
  AlertTriangle,
  X,
  Loader2,
} from "lucide-react";

const { token } = theme.useToken();

/**
 * 加载状态属性接口
 */
export interface LoadingStateProps {
  /** 是否显示加载状态 */
  loading: boolean;
  /** 加载文本 */
  text?: string;
  /** 加载图标 */
  icon?: ReactNode;
  /** 加载尺寸 */
  size?: "sm" | "md" | "lg";
  /** 是否全屏加载 */
  fullscreen?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 加载状态组件
 * 提供统一的加载状态显示
 */
export const LoadingState: React.FC<LoadingStateProps> = ({
  loading,
  text = "加载中...",
  icon,
  size = "md",
  fullscreen = false,
  className,
}) => {
  const getSizeClasses = () => {
    switch (size) {
      case "sm":
        return "w-4 h-4";
      case "lg":
        return "w-8 h-8";
      default:
        return "w-6 h-6";
    }
  };

  const getTextSizeClasses = () => {
    switch (size) {
      case "sm":
        return "text-sm";
      case "lg":
        return "text-lg";
      default:
        return "text-base";
    }
  };

  if (!loading) return null;

  const content = (
    <div
      className={cn(
        "flex flex-col items-center justify-center",
        fullscreen ? "fixed inset-0 z-50" : "py-8",
        className
      )}
      style={{
        backgroundColor: fullscreen
          ? "rgba(255, 255, 255, 0.8)"
          : "transparent",
        backdropFilter: fullscreen ? "blur(4px)" : "none",
      }}
    >
      <div className="flex flex-col items-center space-y-3">
        {icon || (
          <Loader2
            className={cn("animate-spin", getSizeClasses())}
            style={{ color: token.colorPrimary }}
          />
        )}
        {text && (
          <p
            className={cn("font-medium", getTextSizeClasses())}
            style={{ color: token.colorText }}
          >
            {text}
          </p>
        )}
      </div>
    </div>
  );

  return content;
};

/**
 * 错误状态属性接口
 */
export interface ErrorStateProps {
  /** 错误信息 */
  error: string | Error | null;
  /** 错误标题 */
  title?: string;
  /** 重试回调 */
  onRetry?: () => void;
  /** 是否显示重试按钮 */
  showRetry?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 错误状态组件
 * 提供统一的错误状态显示
 */
export const ErrorState: React.FC<ErrorStateProps> = ({
  error,
  title = "出现错误",
  onRetry,
  showRetry = true,
  className,
}) => {
  if (!error) return null;

  const errorMessage = error instanceof Error ? error.message : error;

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-8 px-4",
        className
      )}
    >
      <div className="text-center max-w-md">
        <div className="mb-4">
          <AlertCircle
            className="w-12 h-12 mx-auto"
            style={{ color: token.colorError }}
          />
        </div>
        <h3
          className="text-lg font-semibold mb-2"
          style={{ color: token.colorText }}
        >
          {title}
        </h3>
        <p className="text-sm mb-4" style={{ color: token.colorTextSecondary }}>
          {errorMessage}
        </p>
        {showRetry && onRetry && (
          <button
            onClick={onRetry}
            className="px-4 py-2 rounded-md font-medium transition-colors"
            style={{
              backgroundColor: token.colorPrimary,
              color: token.colorWhite,
            }}
          >
            重试
          </button>
        )}
      </div>
    </div>
  );
};

/**
 * 空状态属性接口
 */
export interface EmptyStateProps {
  /** 空状态标题 */
  title: string;
  /** 空状态描述 */
  description?: string;
  /** 空状态图标 */
  icon?: ReactNode;
  /** 操作按钮 */
  action?: ReactNode;
  /** 自定义类名 */
  className?: string;
}

/**
 * 空状态组件
 * 提供统一的空状态显示
 */
export const EmptyState: React.FC<EmptyStateProps> = ({
  title,
  description,
  icon,
  action,
  className,
}) => {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-12 px-4",
        className
      )}
    >
      <div className="text-center max-w-sm">
        {icon && (
          <div className="mb-4">
            <div
              className="w-16 h-16 mx-auto rounded-full flex items-center justify-center"
              style={{
                backgroundColor: token.colorFillSecondary,
                color: token.colorTextTertiary,
              }}
            >
              {icon}
            </div>
          </div>
        )}
        <h3
          className="text-lg font-semibold mb-2"
          style={{ color: token.colorText }}
        >
          {title}
        </h3>
        {description && (
          <p
            className="text-sm mb-4"
            style={{ color: token.colorTextSecondary }}
          >
            {description}
          </p>
        )}
        {action && <div>{action}</div>}
      </div>
    </div>
  );
};

/**
 * 通知消息类型
 */
export type NotificationType = "success" | "error" | "warning" | "info";

/**
 * 通知消息属性接口
 */
export interface NotificationProps {
  /** 通知类型 */
  type: NotificationType;
  /** 通知标题 */
  title: string;
  /** 通知内容 */
  message?: string;
  /** 是否显示关闭按钮 */
  closable?: boolean;
  /** 自动关闭时间（毫秒） */
  duration?: number;
  /** 关闭回调 */
  onClose?: () => void;
  /** 自定义类名 */
  className?: string;
}

/**
 * 通知消息组件
 * 提供统一的通知消息显示
 */
export const Notification: React.FC<NotificationProps> = ({
  type,
  title,
  message,
  closable = true,
  duration = 5000,
  onClose,
  className,
}) => {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        setVisible(false);
        onClose?.();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [duration, onClose]);

  const handleClose = () => {
    setVisible(false);
    onClose?.();
  };

  if (!visible) return null;

  const getTypeStyles = () => {
    switch (type) {
      case "success":
        return {
          backgroundColor: token.colorSuccessBg,
          borderColor: token.colorSuccessBorder,
          iconColor: token.colorSuccess,
          icon: <CheckCircle className="w-5 h-5" />,
        };
      case "error":
        return {
          backgroundColor: token.colorErrorBg,
          borderColor: token.colorErrorBorder,
          iconColor: token.colorError,
          icon: <AlertCircle className="w-5 h-5" />,
        };
      case "warning":
        return {
          backgroundColor: token.colorWarningBg,
          borderColor: token.colorWarningBorder,
          iconColor: token.colorWarning,
          icon: <AlertTriangle className="w-5 h-5" />,
        };
      case "info":
        return {
          backgroundColor: token.colorInfoBg,
          borderColor: token.colorInfoBorder,
          iconColor: token.colorInfo,
          icon: <Info className="w-5 h-5" />,
        };
    }
  };

  const typeStyles = getTypeStyles();

  return (
    <div
      className={cn(
        "p-4 rounded-lg border shadow-sm transition-all duration-300",
        className
      )}
      style={{
        backgroundColor: typeStyles.backgroundColor,
        borderColor: typeStyles.borderColor,
        animation: `${animation.presets.slideUp.duration} ${animation.presets.slideUp.easing}`,
      }}
    >
      <div className="flex items-start space-x-3">
        <div
          className="flex-shrink-0 mt-0.5"
          style={{ color: typeStyles.iconColor }}
        >
          {typeStyles.icon}
        </div>
        <div className="flex-1 min-w-0">
          <h4
            className="text-sm font-medium"
            style={{ color: token.colorText }}
          >
            {title}
          </h4>
          {message && (
            <p
              className="mt-1 text-sm"
              style={{ color: token.colorTextSecondary }}
            >
              {message}
            </p>
          )}
        </div>
        {closable && (
          <button
            onClick={handleClose}
            className="flex-shrink-0 p-1 rounded-md hover:bg-black/5 transition-colors"
          >
            <X className="w-4 h-4" style={{ color: token.colorTextTertiary }} />
          </button>
        )}
      </div>
    </div>
  );
};

/**
 * 表单验证状态属性接口
 */
export interface FormValidationProps {
  /** 验证状态 */
  status?: "success" | "error" | "warning" | "validating";
  /** 验证消息 */
  message?: string;
  /** 是否显示图标 */
  showIcon?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 表单验证状态组件
 * 提供统一的表单验证状态显示
 */
export const FormValidation: React.FC<FormValidationProps> = ({
  status,
  message,
  showIcon = true,
  className,
}) => {
  if (!status || !message) return null;

  const getStatusStyles = () => {
    switch (status) {
      case "success":
        return {
          color: token.colorSuccess,
          icon: <CheckCircle className="w-4 h-4" />,
        };
      case "error":
        return {
          color: token.colorError,
          icon: <AlertCircle className="w-4 h-4" />,
        };
      case "warning":
        return {
          color: token.colorWarning,
          icon: <AlertTriangle className="w-4 h-4" />,
        };
      case "validating":
        return {
          color: token.colorPrimary,
          icon: <Loader2 className="w-4 h-4 animate-spin" />,
        };
    }
  };

  const statusStyles = getStatusStyles();

  return (
    <div
      className={cn("flex items-center space-x-2 mt-1", className)}
      style={{ color: statusStyles.color }}
    >
      {showIcon && statusStyles.icon}
      <span className="text-sm">{message}</span>
    </div>
  );
};

/**
 * 骨架屏属性接口
 */
export interface SkeletonProps {
  /** 骨架屏类型 */
  variant?: "text" | "rectangular" | "circular";
  /** 骨架屏宽度 */
  width?: string | number;
  /** 骨架屏高度 */
  height?: string | number;
  /** 骨架屏数量 */
  count?: number;
  /** 是否显示动画 */
  animated?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 骨架屏组件
 * 提供统一的骨架屏显示
 */
export const Skeleton: React.FC<SkeletonProps> = ({
  variant = "rectangular",
  width,
  height,
  count = 1,
  animated = true,
  className,
}) => {
  const getVariantClasses = () => {
    switch (variant) {
      case "text":
        return "h-4 rounded";
      case "circular":
        return "rounded-full";
      default:
        return "rounded";
    }
  };

  const skeletonStyle = {
    width: width || (variant === "circular" ? "40px" : "100%"),
    height:
      height ||
      (variant === "text" ? "16px" : variant === "circular" ? "40px" : "200px"),
    backgroundColor: token.colorFillSecondary,
  };

  const renderSkeleton = () => (
    <div
      className={cn(
        getVariantClasses(),
        animated && "animate-pulse",
        className
      )}
      style={skeletonStyle}
    />
  );

  if (count === 1) {
    return renderSkeleton();
  }

  return (
    <div className="space-y-2">
      {Array.from({ length: count }).map((_, index) => (
        <div key={index}>{renderSkeleton()}</div>
      ))}
    </div>
  );
};

const InteractionPatterns = {
  LoadingState,
  ErrorState,
  EmptyState,
  Notification,
  FormValidation,
  Skeleton,
};

export default InteractionPatterns;
