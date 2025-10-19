"use client";

import React, { forwardRef } from "react";
import { Input, PasswordInput } from "@/components/ui";
import { theme } from "antd";
import { cn } from "@/lib/utils";

const { token } = theme.useToken();

/**
 * 认证字段属性接口
 */
export interface AuthFieldProps {
  /** 字段标签 */
  label?: string;
  /** 字段类型 */
  type?: "text" | "email" | "password" | "tel" | "url" | "number";
  /** 字段名称 */
  name?: string;
  /** 占位符文本 */
  placeholder?: string;
  /** 字段值 */
  value?: string;
  /** 默认值 */
  defaultValue?: string;
  /** 是否必填 */
  required?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readOnly?: boolean;
  /** 前缀图标 */
  prefix?: React.ReactNode;
  /** 后缀图标 */
  suffix?: React.ReactNode;
  /** 帮助文本 */
  helpText?: string;
  /** 错误信息 */
  error?: string;
  /** 成功信息 */
  success?: string;
  /** 字段尺寸 */
  size?: "sm" | "md" | "lg";
  /** 是否显示密码强度 */
  showPasswordStrength?: boolean;
  /** 是否可清除 */
  clearable?: boolean;
  /** 验证规则 */
  rules?: any[];
  /** 值变化回调 */
  onChange?: (value: string) => void;
  /** 焦点事件回调 */
  onFocus?: (e: React.FocusEvent<HTMLInputElement>) => void;
  /** 失焦事件回调 */
  onBlur?: (e: React.FocusEvent<HTMLInputElement>) => void;
  /** 键盘事件回调 */
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  /** 自定义类名 */
  className?: string;
  /** 自定义样式 */
  style?: React.CSSProperties;
  /** 容器类名 */
  containerClassName?: string;
  /** 标签类名 */
  labelClassName?: string;
}

/**
 * 认证字段组件
 * 提供统一的输入框样式和交互体验
 */
export const AuthField = forwardRef<HTMLInputElement, AuthFieldProps>(
  (
    {
      label,
      type = "text",
      name,
      placeholder,
      value,
      defaultValue,
      required = false,
      disabled = false,
      readOnly = false,
      prefix,
      suffix,
      helpText,
      error,
      success,
      size = "lg",
      showPasswordStrength = false,
      clearable = false,
      rules,
      onChange,
      onFocus,
      onBlur,
      onKeyDown,
      className,
      style,
      containerClassName,
      labelClassName,
      ...props
    },
    ref
  ) => {
    // 处理值变化
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(e.target.value);
    };

    // 获取字段状态
    const getFieldVariant = () => {
      if (error) return "error";
      if (success) return "success";
      return "default";
    };

    // 渲染标签
    const renderLabel = () => {
      if (!label) return null;

      return (
        <label
          className={cn(
            "block text-sm font-medium mb-2",
            labelClassName
          )}
          style={{ color: token.colorText }}
        >
          {label}
          {required && (
            <span style={{ color: token.colorError, marginLeft: 4 }}>*</span>
          )}
        </label>
      );
    };

    // 渲染帮助文本或状态信息
    const renderHelpText = () => {
      if (!helpText && !error && !success) return null;

      const text = error || success || helpText;
      const color = error ? token.colorError : success ? token.colorSuccess : token.colorTextSecondary;

      return (
        <p
          className="mt-1 text-sm"
          style={{ color }}
        >
          {text}
        </p>
      );
    };

    // 渲染输入框
    const renderInput = () => {
      const commonProps = {
        ref,
        name,
        placeholder,
        value,
        defaultValue,
        disabled,
        required,
        onChange: handleChange,
        onFocus,
        onBlur,
        onKeyDown,
        className,
        style,
        size,
        variant: getFieldVariant(),
        prefix,
        suffix,
        clearable,
        ...props,
      };

      if (type === "password") {
        return (
          <PasswordInput
            {...commonProps}
            showStrength={showPasswordStrength}
            helpText={helpText}
          />
        );
      }

      return (
        <Input
          {...commonProps}
          type={type}
          helpText={helpText}
        />
      );
    };

    return (
      <div className={cn("w-full", containerClassName)}>
        {renderLabel()}
        {renderInput()}
        {renderHelpText()}
      </div>
    );
  }
);

AuthField.displayName = "AuthField";

/**
 * 认证字段组组件
 * 用于将多个字段组合在一起
 */
export interface AuthFieldGroupProps {
  children: React.ReactNode;
  /** 字段组标题 */
  title?: string;
  /** 字段组描述 */
  description?: string;
  /** 字段间距 */
  spacing?: "none" | "sm" | "md" | "lg";
  /** 是否显示边框 */
  bordered?: boolean;
  /** 自定义类名 */
  className?: string;
}

export const AuthFieldGroup: React.FC<AuthFieldGroupProps> = ({
  children,
  title,
  description,
  spacing = "md",
  bordered = false,
  className,
}) => {
  const { token } = theme.useToken();

  const spacingClasses = {
    none: "space-y-0",
    sm: "space-y-2",
    md: "space-y-4",
    lg: "space-y-6",
  };

  return (
    <div
      className={cn(
        "w-full",
        spacingClasses[spacing],
        bordered && "p-4 rounded-lg border",
        className
      )}
      style={bordered ? { borderColor: token.colorBorder } : undefined}
    >
      {(title || description) && (
        <div className="mb-4">
          {title && (
            <h3
              className="text-lg font-semibold mb-1"
              style={{ color: token.colorText }}
            >
              {title}
            </h3>
          )}
          {description && (
            <p style={{ color: token.colorTextSecondary }}>{description}</p>
          )}
        </div>
      )}
      {children}
    </div>
  );
};

export default AuthField;
