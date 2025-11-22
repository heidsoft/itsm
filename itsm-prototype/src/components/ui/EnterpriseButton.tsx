'use client';

import React from 'react';
import { Button, ButtonProps } from 'antd';
import { cn } from '@/lib/utils';

export interface EnterpriseButtonProps extends ButtonProps {
  /**
   * 按钮变体
   * @default 'primary'
   */
  variant?: 'primary' | 'secondary' | 'success' | 'warning' | 'danger' | 'ghost' | 'link';
  
  /**
   * 是否显示渐变背景
   * @default true (仅primary生效)
   */
  gradient?: boolean;
  
  /**
   * 是否显示阴影效果
   * @default true
   */
  shadow?: boolean;
  
  /**
   * 是否全宽显示
   * @default false
   */
  fullWidth?: boolean;
}

/**
 * 企业级按钮组件
 * 
 * 特性：
 * - 统一的高度和圆角
 * - 渐变背景效果
 * - 悬浮阴影动画
 * - 多种预设变体
 * - GPU加速动画
 * 
 * @example
 * ```tsx
 * <EnterpriseButton variant="primary" icon={<PlusOutlined />}>
 *   创建工单
 * </EnterpriseButton>
 * ```
 */
export const EnterpriseButton: React.FC<EnterpriseButtonProps> = ({
  variant = 'primary',
  gradient = true,
  shadow = true,
  fullWidth = false,
  className,
  style,
  size = 'middle',
  children,
  ...props
}) => {
  // 高度映射
  const heightMap = {
    small: 32,
    middle: 40,
    large: 48,
  };

  // 基础样式
  const baseStyles: React.CSSProperties = {
    height: heightMap[size as keyof typeof heightMap] || 40,
    borderRadius: '8px',
    fontWeight: 500,
    fontSize: size === 'large' ? '16px' : '14px',
    padding: size === 'small' ? '0 16px' : '0 24px',
    transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
    ...style,
  };

  // 变体样式
  const variantStyles: Record<string, React.CSSProperties> = {
    primary: gradient ? {
      background: 'linear-gradient(135deg, #3b82f6 0%, #2563eb 100%)',
      border: 'none',
      color: '#ffffff',
    } : {
      background: '#3b82f6',
      border: 'none',
      color: '#ffffff',
    },
    secondary: {
      background: '#ffffff',
      border: '1px solid #e5e7eb',
      color: '#374151',
    },
    success: {
      background: 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)',
      border: 'none',
      color: '#ffffff',
    },
    warning: {
      background: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
      border: 'none',
      color: '#ffffff',
    },
    danger: {
      background: 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)',
      border: 'none',
      color: '#ffffff',
    },
    ghost: {
      background: 'transparent',
      border: '1px solid #3b82f6',
      color: '#3b82f6',
    },
    link: {
      background: 'transparent',
      border: 'none',
      color: '#3b82f6',
      padding: '0',
    },
  };

  const finalStyles = {
    ...baseStyles,
    ...variantStyles[variant],
    width: fullWidth ? '100%' : undefined,
  };

  // Ant Design 按钮类型映射
  const typeMap: Record<string, ButtonProps['type']> = {
    primary: 'primary',
    secondary: 'default',
    success: 'primary',
    warning: 'primary',
    danger: 'primary',
    ghost: 'default',
    link: 'link',
  };

  return (
    <Button
      type={typeMap[variant]}
      size={size}
      className={cn(
        'enterprise-button',
        'transition-all duration-300 ease-in-out',
        shadow && variant !== 'link' && 'hover:shadow-lg',
        variant === 'primary' && 'hover:opacity-90',
        variant === 'secondary' && 'hover:border-blue-500 hover:text-blue-600',
        variant === 'ghost' && 'hover:bg-blue-50',
        'will-change-transform',
        fullWidth && 'w-full',
        className
      )}
      style={finalStyles}
      {...props}
    >
      {children}
    </Button>
  );
};

/**
 * 企业级按钮组
 * 
 * 用于组合多个按钮，并保持一致的间距
 */
export interface EnterpriseButtonGroupProps {
  children: React.ReactNode;
  spacing?: 'small' | 'medium' | 'large';
  align?: 'left' | 'center' | 'right';
  vertical?: boolean;
  className?: string;
}

export const EnterpriseButtonGroup: React.FC<EnterpriseButtonGroupProps> = ({
  children,
  spacing = 'medium',
  align = 'left',
  vertical = false,
  className,
}) => {
  const spacingMap = {
    small: 8,
    medium: 12,
    large: 16,
  };

  const alignMap = {
    left: 'flex-start',
    center: 'center',
    right: 'flex-end',
  };

  return (
    <div
      className={cn(
        'enterprise-button-group',
        'flex',
        vertical ? 'flex-col' : 'flex-row',
        className
      )}
      style={{
        gap: spacingMap[spacing],
        justifyContent: alignMap[align],
      }}
    >
      {children}
    </div>
  );
};

/**
 * 企业级图标按钮
 * 
 * 正方形图标按钮，适用于工具栏等场景
 */
export interface EnterpriseIconButtonProps extends Omit<EnterpriseButtonProps, 'children'> {
  icon: React.ReactNode;
  tooltip?: string;
  'aria-label': string;
}

export const EnterpriseIconButton: React.FC<EnterpriseIconButtonProps> = ({
  icon,
  tooltip,
  size = 'middle',
  ...props
}) => {
  const sizeMap = {
    small: 32,
    middle: 40,
    large: 48,
  };

  const buttonSize = sizeMap[size as keyof typeof sizeMap] || 40;

  return (
    <EnterpriseButton
      {...props}
      size={size}
      icon={icon}
      title={tooltip}
      style={{
        minWidth: buttonSize,
        width: buttonSize,
        height: buttonSize,
        padding: 0,
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        ...props.style,
      }}
    />
  );
};

export default EnterpriseButton;

