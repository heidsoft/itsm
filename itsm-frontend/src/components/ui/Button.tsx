'use client';

import React from 'react';
import { cn } from '@/lib/utils';
import { LoadingSpinner } from './LoadingSpinner';
import { TouchFeedback } from './TouchFeedback';
import { useIsTouchDevice } from '@/hooks/useResponsive';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success' | 'link' | 'dashed' | 'text';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  loading?: boolean;
  icon?: React.ReactNode;
  iconPosition?: 'left' | 'right';
  fullWidth?: boolean;
  /** 移动端最小触摸区域(px) */
  minTouchArea?: number;
  /** 是否为移动端优化 */
  mobileOptimized?: boolean;
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
    minTouchArea = 44,
    mobileOptimized = true,
    ...props
  }, ref) => {
    const isTouchDevice = useIsTouchDevice();
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
      link: [
        'text-blue-600 bg-transparent',
        'hover:text-blue-800 hover:underline',
        'focus:ring-blue-500',
        'shadow-none',
      ],
      dashed: [
        'text-gray-700 bg-transparent',
        'border border-dashed border-gray-300',
        'hover:border-gray-400 hover:bg-gray-50',
        'focus:ring-gray-500',
      ],
      text: [
        'text-gray-700 bg-transparent',
        'hover:bg-gray-100',
        'focus:ring-gray-500',
        'shadow-none',
      ],
    };

    const sizeClasses = {
      xs: 'px-2 py-1 text-xs gap-1 h-6 min-w-[2.5rem]',
      sm: 'px-3 py-1.5 text-sm gap-1.5 h-8 min-w-[3rem]',
      md: 'px-4 py-2 text-sm gap-2 h-10 min-w-[3.5rem]',
      lg: 'px-6 py-3 text-base gap-2.5 h-12 min-w-[4rem]',
      xl: 'px-8 py-4 text-lg gap-3 h-14 min-w-[5rem]',
    };

    const widthClasses = fullWidth ? 'w-full' : '';

    const isDisabled = disabled || loading;

    const buttonContent = (
      <>
        {loading && (
          <LoadingSpinner 
            size={size === 'xs' ? 'sm' : size === 'sm' ? 'sm' : size === 'lg' ? 'md' : size === 'xl' ? 'lg' : 'sm'} 
            className="text-current" 
          />
        )}
        
        {!loading && icon && iconPosition === 'left' && (
          <span className="flex-shrink-0">{icon}</span>
        )}
        
        {children && (
          <span className={cn('truncate', loading && 'opacity-0')}>{children}</span>
        )}
        
        {!loading && icon && iconPosition === 'right' && (
          <span className="flex-shrink-0">{icon}</span>
        )}
      </>
    );

    const buttonElement = (
      <button
        ref={ref}
        className={cn(
          baseClasses,
          variantClasses[variant],
          sizeClasses[size],
          widthClasses,
          isTouchDevice && mobileOptimized && 'touch-manipulation',
          className
        )}
        disabled={isDisabled}
        {...props}
      >
        {buttonContent}
      </button>
    );

    // 移动端优化：使用TouchFeedback包装
    if (isTouchDevice && mobileOptimized) {
      return (
        <TouchFeedback
          enableFeedback={true}
          touchArea={{ 
            width: sizeClasses[size].match(/min-w-\[(.*?)rem\]/)?.[1] ? 
              parseFloat(sizeClasses[size].match(/min-w-\[(.*?)rem\]/)![1]) * 16 : 
              minTouchArea,
            height: sizeClasses[size].match(/h-(\d+)/)?.[1] ? 
              parseInt(sizeClasses[size].match(/h-(\d+)/)![1]) * 4 : 
              minTouchArea
          }}
          preventDefault={false}
          className="w-full h-full"
        >
          {buttonElement}
        </TouchFeedback>
      );
    }

    return buttonElement;
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
    xs: 'p-1',
    sm: 'p-1.5',
    md: 'p-2',
    lg: 'p-3',
    xl: 'p-4',
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