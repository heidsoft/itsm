"use client";

import React, { useState, useCallback } from "react";
import { Form, Space, Divider } from "antd";
import { Button, Input, PasswordInput } from "@/components/ui";
import { theme } from "antd";
import { cn } from "@/lib/utils";

const { token } = theme.useToken();

/**
 * 表单字段配置接口
 */
export interface AuthFieldConfig {
  /** 字段名称 */
  name: string;
  /** 字段标签 */
  label: string;
  /** 字段类型 */
  type: "text" | "email" | "password" | "tel" | "url";
  /** 占位符文本 */
  placeholder?: string;
  /** 是否必填 */
  required?: boolean;
  /** 验证规则 */
  rules?: any[];
  /** 前缀图标 */
  prefix?: React.ReactNode;
  /** 后缀图标 */
  suffix?: React.ReactNode;
  /** 帮助文本 */
  helpText?: string;
  /** 是否显示密码强度 */
  showPasswordStrength?: boolean;
  /** 是否可清除 */
  clearable?: boolean;
}

/**
 * 表单按钮配置接口
 */
export interface AuthButtonConfig {
  /** 按钮文本 */
  text: string;
  /** 按钮类型 */
  type?: "primary" | "secondary" | "outline" | "ghost";
  /** 按钮尺寸 */
  size?: "sm" | "md" | "lg";
  /** 是否全宽 */
  fullWidth?: boolean;
  /** 是否显示加载状态 */
  loading?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 图标 */
  icon?: React.ReactNode;
  /** 图标位置 */
  iconPosition?: "left" | "right";
  /** 点击回调 */
  onClick?: () => void;
}

/**
 * 认证表单属性接口
 */
export interface AuthFormProps {
  /** 表单标题 */
  title?: string;
  /** 表单副标题 */
  subtitle?: string;
  /** 表单字段配置 */
  fields: AuthFieldConfig[];
  /** 主要按钮配置 */
  primaryButton: AuthButtonConfig;
  /** 次要按钮配置 */
  secondaryButton?: AuthButtonConfig;
  /** 其他操作区域 */
  extraActions?: React.ReactNode;
  /** 是否显示分割线 */
  showDivider?: boolean;
  /** 分割线文本 */
  dividerText?: string;
  /** 表单提交回调 */
  onSubmit?: (values: Record<string, any>) => void;
  /** 表单验证失败回调 */
  onValidationFailed?: (errors: any) => void;
  /** 表单值变化回调 */
  onValuesChange?: (changedValues: any, allValues: any) => void;
  /** 初始值 */
  initialValues?: Record<string, any>;
  /** 自定义类名 */
  className?: string;
  /** 是否显示移动端Logo */
  showMobileLogo?: boolean;
  /** 错误信息 */
  error?: string;
  /** 成功信息 */
  success?: string;
}

/**
 * 认证表单组件
 * 提供统一的表单样式、验证和交互体验
 */
export const AuthForm: React.FC<AuthFormProps> = ({
  title,
  subtitle,
  fields,
  primaryButton,
  secondaryButton,
  extraActions,
  showDivider = false,
  dividerText = "或",
  onSubmit,
  onValidationFailed,
  onValuesChange,
  initialValues,
  className,
  showMobileLogo = true,
  error,
  success,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [formValues, setFormValues] = useState(initialValues || {});

  // 处理表单值变化
  const handleValuesChange = useCallback(
    (changedValues: any, allValues: any) => {
      setFormValues(allValues);
      onValuesChange?.(changedValues, allValues);
    },
    [onValuesChange]
  );

  // 处理表单提交
  const handleSubmit = useCallback(
    async (values: Record<string, any>) => {
      try {
        setLoading(true);
        await onSubmit?.(values);
      } catch (err) {
        console.error("Form submission error:", err);
      } finally {
        setLoading(false);
      }
    },
    [onSubmit]
  );

  // 处理表单验证失败
  const handleValidationFailed = useCallback(
    (errorInfo: any) => {
      onValidationFailed?.(errorInfo);
    },
    [onValidationFailed]
  );

  // 渲染表单字段
  const renderField = (field: AuthFieldConfig) => {
    const commonProps = {
      label: field.label,
      placeholder: field.placeholder,
      required: field.required,
      disabled: loading,
      size: "lg" as const,
      prefix: field.prefix,
      suffix: field.suffix,
      clearable: field.clearable,
    };

    switch (field.type) {
      case "password":
        return (
          <PasswordInput
            {...commonProps}
            showStrength={field.showPasswordStrength}
            helperText={field.helpText}
          />
        );
      default:
        return (
          <Input
            {...commonProps}
            type={field.type}
            helperText={field.helpText}
          />
        );
    }
  };

  return (
    <div className={cn("w-full max-w-md", className)}>
      {/* 移动端Logo */}
      {showMobileLogo && (
        <div className="lg:hidden text-center mb-8">
          <div
            className="inline-flex items-center justify-center w-16 h-16 rounded-2xl mb-4"
            style={{ backgroundColor: token.colorPrimary }}
          >
            <svg
              className="w-8 h-8 text-white"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
              />
            </svg>
          </div>
          <h1 className="text-2xl font-bold" style={{ color: token.colorText }}>
            ITSM Pro
          </h1>
          <p style={{ color: token.colorTextSecondary }}>智能IT服务管理平台</p>
        </div>
      )}

      {/* 表单卡片 */}
      <div
        className="bg-white rounded-lg shadow-lg border"
        style={{
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowSecondary,
          border: `1px solid ${token.colorBorder}`,
          padding: token.paddingLG,
        }}
      >
        {/* 表单标题 */}
        {(title || subtitle) && (
          <div className="text-center mb-6">
            {title && (
              <h2
                className="text-xl font-semibold mb-2"
                style={{ color: token.colorText }}
              >
                {title}
              </h2>
            )}
            {subtitle && (
              <p style={{ color: token.colorTextSecondary }}>{subtitle}</p>
            )}
          </div>
        )}

        {/* 错误提示 */}
        {error && (
          <div
            className="mb-4 p-3 rounded-md flex items-center space-x-2"
            style={{
              backgroundColor: token.colorErrorBg,
              border: `1px solid ${token.colorErrorBorder}`,
            }}
          >
            <div
              className="w-4 h-4 rounded-full flex-shrink-0"
              style={{ backgroundColor: token.colorError }}
            />
            <span style={{ color: token.colorError, fontSize: token.fontSizeSM }}>
              {error}
            </span>
          </div>
        )}

        {/* 成功提示 */}
        {success && (
          <div
            className="mb-4 p-3 rounded-md flex items-center space-x-2"
            style={{
              backgroundColor: token.colorSuccessBg,
              border: `1px solid ${token.colorSuccessBorder}`,
            }}
          >
            <div
              className="w-4 h-4 rounded-full flex-shrink-0"
              style={{ backgroundColor: token.colorSuccess }}
            />
            <span style={{ color: token.colorSuccess, fontSize: token.fontSizeSM }}>
              {success}
            </span>
          </div>
        )}

        {/* 表单 */}
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          onFinishFailed={handleValidationFailed}
          onValuesChange={handleValuesChange}
          initialValues={initialValues}
          className="space-y-4"
        >
          {/* 表单字段 */}
          {fields.map((field) => (
            <Form.Item
              key={field.name}
              name={field.name}
              label={field.label}
              rules={field.rules}
              required={field.required}
            >
              {renderField(field)}
            </Form.Item>
          ))}

          {/* 主要按钮 */}
          <Form.Item className="mb-0">
            <Button
              type="submit"
              variant={primaryButton.type || "primary"}
              size={primaryButton.size || "lg"}
              fullWidth={primaryButton.fullWidth !== false}
              loading={loading || primaryButton.loading}
              disabled={primaryButton.disabled}
              icon={primaryButton.icon}
              iconPosition={primaryButton.iconPosition}
              style={{
                marginTop: token.marginLG,
                height: token.controlHeightLG,
              }}
            >
              {primaryButton.text}
            </Button>
          </Form.Item>
        </Form>

        {/* 分割线 */}
        {showDivider && (
          <Divider style={{ margin: `${token.marginLG}px 0` }}>
            <span style={{ color: token.colorTextTertiary }}>{dividerText}</span>
          </Divider>
        )}

        {/* 次要按钮 */}
        {secondaryButton && (
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <Button
              variant={secondaryButton.type || "outline"}
              size={secondaryButton.size || "lg"}
              fullWidth={secondaryButton.fullWidth !== false}
              disabled={loading || secondaryButton.disabled}
              icon={secondaryButton.icon}
              iconPosition={secondaryButton.iconPosition}
              onClick={secondaryButton.onClick}
              style={{ height: token.controlHeightLG }}
            >
              {secondaryButton.text}
            </Button>
          </Space>
        )}

        {/* 其他操作 */}
        {extraActions && (
          <div className="mt-4">{extraActions}</div>
        )}
      </div>
    </div>
  );
};

export default AuthForm;
