'use client';

import React, { forwardRef } from 'react';
import { Button, type ButtonProps as AntButtonProps } from 'antd';
import { cn } from '@/lib/utils';

/**
 * 认证按钮属性接口
 * 继承 Ant Design Button 属性，但覆盖部分属性以保持 API 兼容
 */
export interface AuthButtonProps extends Omit<AntButtonProps, 'type' | 'size'> {
  /** 按钮类型 */
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success';
  /** 按钮尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 是否全宽 */
  fullWidth?: boolean;
  /** 是否显示波纹效果 (Antd 默认支持) */
  ripple?: boolean;
}

/**
 * 认证按钮组件
 * 基于 Ant Design Button 的封装，提供统一的认证页面按钮样式
 */
export const AuthButton = forwardRef<HTMLElement, AuthButtonProps>(
  (
    { children, variant = 'primary', size = 'lg', fullWidth = false, className, style, ...props },
    ref
  ) => {
    // 映射 variant 到 Ant Design 的 type
    let type: AntButtonProps['type'] = 'default';
    let danger = false;
    let customClass = '';

    switch (variant) {
      case 'primary':
        type = 'primary';
        break;
      case 'secondary':
        type = 'default';
        break;
      case 'outline':
        type = 'default';
        customClass = 'border-primary text-primary';
        break;
      case 'ghost':
        type = 'text';
        break;
      case 'danger':
        type = 'primary';
        danger = true;
        break;
      case 'success':
        type = 'primary';
        customClass = 'bg-green-600 hover:!bg-green-500 border-green-600 hover:!border-green-500';
        break;
    }

    // 映射 size
    const antdSize = size === 'lg' ? 'large' : size === 'sm' ? 'small' : 'middle';

    return (
      <Button
        ref={ref}
        type={type}
        danger={danger}
        size={antdSize}
        block={fullWidth}
        className={cn(customClass, className)}
        style={style}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

AuthButton.displayName = 'AuthButton';

/**
 * 认证按钮组组件
 */
export interface AuthButtonGroupProps {
  children: React.ReactNode;
  direction?: 'horizontal' | 'vertical';
  spacing?: 'none' | 'sm' | 'md' | 'lg';
  fullWidth?: boolean;
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
