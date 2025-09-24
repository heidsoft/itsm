'use client';

import React from 'react';
import { cn } from '@/lib/utils';
import { LoadingSpinner } from './LoadingSpinner';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  icon?: React.ReactNode;
  iconPosition?: 'left' | 'right';
  fullWidth?: boolean;
}

/**
 * 通用按钮组件
 * 支持多种样式变体、尺寸和状态
 */
export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({
    className,
    variant = 'primary',
    size = 'md',
    loading = false,
    icon,
    iconPosition = 'left',
    fullWidth = false,
    disabled,
    children,
    ...props
  }, ref) => {
    const baseClasses = [
      'inline-flex items-center justify-center',
      'font-medium rounded-lg transition-all duration-200',
      'focus:outline-none focus:ring-2 focus:ring-offset-2',
      'disabled:opacity-50 disabled:cursor-not-allowed',
      'active:scale-95',
    ];

    const variantClasses = {
      primary: [
        'bg-blue-600 text-white',
        'hover:bg-blue-700 focus:ring-blue-500',
        'shadow-sm hover:shadow-md',
      ],
      secondary: [
        'bg-gray-100 text-gray-900',
        'hover:bg-gray-200 focus:ring-gray-500',
        'border border-gray-300',
      ],
      outline: [
        'bg-transparent text-blue-600',
        'border border-blue-600',
        'hover:bg-blue-50 focus:ring-blue-500',
      ],
      ghost: [
        'bg-transparent text-gray-700',
        'hover:bg-gray-100 focus:ring-gray-500',
      ],
      danger: [
        'bg-red-600 text-white',
        'hover:bg-red-700 focus:ring-red-500',
        'shadow-sm hover:shadow-md',
      ],
      success: [
        'bg-green-600 text-white',
        'hover:bg-green-700 focus:ring-green-500',
        'shadow-sm hover:shadow-md',
      ],
    };

    const sizeClasses = {
      sm: 'px-3 py-1.5 text-sm gap-1.5',
      md: 'px-4 py-2 text-sm gap-2',
      lg: 'px-6 py-3 text-base gap-2.5',
    };

    const widthClasses = fullWidth ? 'w-full' : '';

    const isDisabled = disabled || loading;

    return (
      <button
        ref={ref}
        className={cn(
          baseClasses,
          variantClasses[variant],
          sizeClasses[size],
          widthClasses,
          className
        )}
        disabled={isDisabled}
        {...props}
      >
        {loading && (
          <LoadingSpinner 
            size={size === 'sm' ? 'sm' : size === 'lg' ? 'md' : 'sm'} 
            className="text-current" 
          />
        )}
        
        {!loading && icon && iconPosition === 'left' && (
          <span className="flex-shrink-0">{icon}</span>
        )}
        
        {children && (
          <span className={loading ? 'opacity-0' : ''}>{children}</span>
        )}
        
        {!loading && icon && iconPosition === 'right' && (
          <span className="flex-shrink-0">{icon}</span>
        )}
      </button>
    );
  }
);

Button.displayName = 'Button';

/**
 * 图标按钮组件
 * 专门用于只显示图标的按钮
 */
export interface IconButtonProps extends Omit<ButtonProps, 'children' | 'icon' | 'iconPosition'> {
  icon: React.ReactNode;
  'aria-label': string;
}

export const IconButton = React.forwardRef<HTMLButtonElement, IconButtonProps>(
  ({ icon, className, size = 'md', ...props }, ref) => {
    const sizeClasses = {
      sm: 'p-1.5',
      md: 'p-2',
      lg: 'p-3',
    };

    return (
      <Button
        ref={ref}
        className={cn('!px-0 !py-0', sizeClasses[size], className)}
        size={size}
        {...props}
      >
        {icon}
      </Button>
    );
  }
);

IconButton.displayName = 'IconButton';

/**
 * 按钮组组件
 * 用于将多个按钮组合在一起
 */
export interface ButtonGroupProps {
  children: React.ReactNode;
  className?: string;
  orientation?: 'horizontal' | 'vertical';
  spacing?: 'none' | 'sm' | 'md';
}

export const ButtonGroup: React.FC<ButtonGroupProps> = ({
  children,
  className,
  orientation = 'horizontal',
  spacing = 'sm',
}) => {
  const orientationClasses = {
    horizontal: 'flex-row',
    vertical: 'flex-col',
  };

  const spacingClasses = {
    none: 'gap-0',
    sm: 'gap-2',
    md: 'gap-4',
  };

  return (
    <div
      className={cn(
        'flex',
        orientationClasses[orientation],
        spacingClasses[spacing],
        className
      )}
    >
      {children}
    </div>
  );
};