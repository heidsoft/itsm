'use client';

import React, { forwardRef } from 'react';
import { Button } from '@/components/ui';
import { theme } from 'antd';
import { cn } from '@/lib/utils';

const { token } = theme.useToken();

/**
 * 认证按钮属性接口
 */
export interface AuthButtonProps {
  /** 按钮文本 */
  children: React.ReactNode;
  /** 按钮类型 */
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success';
  /** 按钮尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 是否全宽 */
  fullWidth?: boolean;
  /** 是否显示加载状态 */
  loading?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 图标 */
  icon?: React.ReactNode;
  /** 图标位置 */
  iconPosition?: 'left' | 'right';
  /** 按钮类型 */
  type?: 'button' | 'submit' | 'reset';
  /** 点击回调 */
  onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  /** 自定义类名 */
  className?: string;
  /** 自定义样式 */
  style?: React.CSSProperties;
  /** HTML属性 */
  htmlType?: 'button' | 'submit' | 'reset';
  /** 是否显示波纹效果 */
  ripple?: boolean;
  /** 按钮形状 */
  shape?: 'default' | 'round' | 'circle';
}

/**
 * 认证按钮组件
 * 提供统一的按钮样式和交互体验
 */
export const AuthButton = forwardRef<HTMLButtonElement, AuthButtonProps>(
  (
    {
      children,
      variant = 'primary',
      size = 'lg',
      fullWidth = false,
      loading = false,
      disabled = false,
      icon,
      iconPosition = 'left',
      type = 'button',
      onClick,
      className,
      style,
      htmlType,
      ripple = true,
      shape = 'default',
      ...props
    },
    ref
  ) => {
    // 获取按钮样式
    const getButtonStyles = () => {
      const baseStyles = {
        transition: 'all 0.2s ease-in-out',
        fontWeight: 500,
        borderRadius:
          shape === 'round' ? '9999px' : shape === 'circle' ? '50%' : token.borderRadius,
      };

      // 尺寸样式
      const sizeStyles = {
        sm: {
          height: token.controlHeight,
          padding: `0 ${token.padding}px`,
          fontSize: token.fontSizeSM,
        },
        md: {
          height: token.controlHeight,
          padding: `0 ${token.paddingLG}px`,
          fontSize: token.fontSize,
        },
        lg: {
          height: token.controlHeightLG,
          padding: `0 ${token.paddingXL}px`,
          fontSize: token.fontSizeLG,
        },
      };

      // 变体样式
      const variantStyles = {
        primary: {
          backgroundColor: token.colorPrimary,
          color: token.colorWhite,
          border: `1px solid ${token.colorPrimary}`,
          boxShadow: token.boxShadow,
        },
        secondary: {
          backgroundColor: token.colorBgContainer,
          color: token.colorText,
          border: `1px solid ${token.colorBorder}`,
          boxShadow: token.boxShadow,
        },
        outline: {
          backgroundColor: 'transparent',
          color: token.colorPrimary,
          border: `1px solid ${token.colorPrimary}`,
          boxShadow: 'none',
        },
        ghost: {
          backgroundColor: 'transparent',
          color: token.colorText,
          border: 'none',
          boxShadow: 'none',
        },
        danger: {
          backgroundColor: token.colorError,
          color: token.colorWhite,
          border: `1px solid ${token.colorError}`,
          boxShadow: token.boxShadow,
        },
        success: {
          backgroundColor: token.colorSuccess,
          color: token.colorWhite,
          border: `1px solid ${token.colorSuccess}`,
          boxShadow: token.boxShadow,
        },
      };

      return {
        ...baseStyles,
        ...sizeStyles[size],
        ...variantStyles[variant],
        width: fullWidth ? '100%' : 'auto',
        ...style,
      };
    };

    // 获取悬停样式
    const getHoverStyles = () => {
      const hoverStyles = {
        primary: {
          backgroundColor: token.colorPrimaryHover,
          borderColor: token.colorPrimaryHover,
          transform: 'translateY(-1px)',
          boxShadow: token.boxShadowSecondary,
        },
        secondary: {
          backgroundColor: token.colorFillSecondary,
          borderColor: token.colorBorderSecondary,
          transform: 'translateY(-1px)',
          boxShadow: token.boxShadowSecondary,
        },
        outline: {
          backgroundColor: token.colorPrimaryBg,
          transform: 'translateY(-1px)',
          boxShadow: token.boxShadow,
        },
        ghost: {
          backgroundColor: token.colorFillSecondary,
          transform: 'translateY(-1px)',
        },
        danger: {
          backgroundColor: token.colorErrorHover,
          borderColor: token.colorErrorHover,
          transform: 'translateY(-1px)',
          boxShadow: token.boxShadowSecondary,
        },
        success: {
          backgroundColor: token.colorSuccessHover,
          borderColor: token.colorSuccessHover,
          transform: 'translateY(-1px)',
          boxShadow: token.boxShadowSecondary,
        },
      };

      return hoverStyles[variant];
    };

    // 获取激活样式
    const getActiveStyles = () => {
      return {
        transform: 'translateY(0px)',
        boxShadow: token.boxShadow,
      };
    };

    return (
      <button
        ref={ref}
        type={htmlType || type}
        disabled={disabled || loading}
        onClick={onClick}
        className={cn(
          'inline-flex items-center justify-center gap-2',
          'focus:outline-none focus:ring-2 focus:ring-offset-2',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          'active:scale-95',
          ripple && 'hover:scale-105',
          className
        )}
        style={
          {
            ...getButtonStyles(),
            '--hover-bg': (getHoverStyles() as any).backgroundColor,
            '--hover-border': (getHoverStyles() as any).borderColor,
            '--hover-transform': (getHoverStyles() as any).transform,
            '--hover-shadow': (getHoverStyles() as any).boxShadow,
            '--active-transform': (getActiveStyles() as any).transform,
            '--active-shadow': (getActiveStyles() as any).boxShadow,
          } as unknown as React.CSSProperties
        }
        onMouseEnter={e => {
          if (!disabled && !loading) {
            const target = e.currentTarget;
            Object.assign(target.style, getHoverStyles());
          }
        }}
        onMouseLeave={e => {
          if (!disabled && !loading) {
            const target = e.currentTarget;
            Object.assign(target.style, getButtonStyles());
          }
        }}
        onMouseDown={e => {
          if (!disabled && !loading) {
            const target = e.currentTarget;
            Object.assign(target.style, getActiveStyles());
          }
        }}
        onMouseUp={e => {
          if (!disabled && !loading) {
            const target = e.currentTarget;
            Object.assign(target.style, getButtonStyles());
          }
        }}
        {...props}
      >
        {loading && <div className='animate-spin rounded-full h-4 w-4 border-b-2 border-current' />}

        {!loading && icon && iconPosition === 'left' && (
          <span className='flex-shrink-0'>{icon}</span>
        )}

        {children && <span className={loading ? 'opacity-0' : ''}>{children}</span>}

        {!loading && icon && iconPosition === 'right' && (
          <span className='flex-shrink-0'>{icon}</span>
        )}
      </button>
    );
  }
);

AuthButton.displayName = 'AuthButton';

/**
 * 认证按钮组组件
 * 用于将多个按钮组合在一起
 */
export interface AuthButtonGroupProps {
  children: React.ReactNode;
  /** 按钮组方向 */
  direction?: 'horizontal' | 'vertical';
  /** 按钮间距 */
  spacing?: 'none' | 'sm' | 'md' | 'lg';
  /** 是否全宽 */
  fullWidth?: boolean;
  /** 自定义类名 */
  className?: string;
}

export const AuthButtonGroup: React.FC<AuthButtonGroupProps> = ({
  children,
  direction = 'horizontal',
  spacing = 'sm',
  fullWidth = false,
  className,
}) => {
  const directionClasses = {
    horizontal: 'flex-row',
    vertical: 'flex-col',
  };

  const spacingClasses = {
    none: 'gap-0',
    sm: 'gap-2',
    md: 'gap-4',
    lg: 'gap-6',
  };

  return (
    <div
      className={cn(
        'flex',
        directionClasses[direction],
        spacingClasses[spacing],
        fullWidth && 'w-full',
        className
      )}
    >
      {children}
    </div>
  );
};

export default AuthButton;
