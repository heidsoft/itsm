'use client';

import React, { forwardRef, useState } from 'react';
import { Eye, EyeOff, Search, X } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * 输入框组件的属性接口
 */
export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size' | 'prefix'> {
  /** 输入框尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 输入框状态 */
  variant?: 'default' | 'error' | 'success' | 'warning';
  /** 是否显示清除按钮 */
  clearable?: boolean;
  /** 前缀图标 */
  prefix?: React.ReactNode;
  /** 后缀图标 */
  suffix?: React.ReactNode;
  /** 标签文本 */
  label?: string;
  /** 帮助文本 */
  helperText?: string;
  /** 错误信息 */
  error?: string;
  /** 是否必填 */
  required?: boolean;
  /** 容器类名 */
  containerClassName?: string;
  /** 标签类名 */
  labelClassName?: string;
  /** 清除回调函数 */
  onClear?: () => void;
}

/**
 * 密码输入框组件属性
 */
export interface PasswordInputProps extends Omit<InputProps, 'type'> {
  /** 是否显示密码强度指示器 */
  showStrength?: boolean;
}

/**
 * 搜索输入框组件属性
 */
export interface SearchInputProps extends Omit<InputProps, 'type'> {
  /** 搜索回调函数 */
  onSearch?: (value: string) => void;
  /** 搜索按钮文本 */
  searchText?: string;
  /** 是否显示搜索按钮 */
  showSearchButton?: boolean;
}

/**
 * 获取输入框尺寸样式
 */
const getSizeStyles = (size: 'sm' | 'md' | 'lg') => {
  const styles = {
    sm: 'h-8 px-3 text-sm',
    md: 'h-10 px-3 text-sm',
    lg: 'h-12 px-4 text-base'
  };
  return styles[size];
};

/**
 * 获取输入框状态样式
 */
const getVariantStyles = (variant: 'default' | 'error' | 'success' | 'warning') => {
  const styles = {
    default: 'border-gray-300 focus:border-blue-500 focus:ring-blue-500',
    error: 'border-red-300 focus:border-red-500 focus:ring-red-500',
    success: 'border-green-300 focus:border-green-500 focus:ring-green-500',
    warning: 'border-yellow-300 focus:border-yellow-500 focus:ring-yellow-500'
  };
  return styles[variant];
};

/**
 * 基础输入框组件
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({
    className,
    containerClassName,
    labelClassName,
    size = 'md',
    variant = 'default',
    clearable = false,
    prefix,
    suffix,
    label,
    helperText,
    error,
    required = false,
    disabled,
    value,
    onChange,
    onClear,
    ...props
  }, ref) => {
    const [internalValue, setInternalValue] = useState(value || '');
    const currentValue = value !== undefined ? value : internalValue;
    
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(e);
    };

    const handleClear = () => {
      const event = {
        target: { value: '' },
        currentTarget: { value: '' }
      } as React.ChangeEvent<HTMLInputElement>;
      
      if (value === undefined) {
        setInternalValue('');
      }
      onChange?.(event);
      onClear?.();
    };

    const inputVariant = error ? 'error' : variant;
    const showClearButton = clearable && currentValue && !disabled;

    return (
      <div className={cn('w-full', containerClassName)}>
        {/* 标签 */}
        {label && (
          <label className={cn(
            'block text-sm font-medium text-gray-700 mb-1',
            labelClassName
          )}>
            {label}
            {required && <span className="text-red-500 ml-1">*</span>}
          </label>
        )}

        {/* 输入框容器 */}
        <div className="relative">
          {/* 前缀图标 */}
          {prefix && (
            <div className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 pointer-events-none">
              {prefix}
            </div>
          )}

          {/* 输入框 */}
          <input
            ref={ref}
            className={cn(
              'w-full rounded-md border bg-white shadow-sm transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-opacity-50',
              'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-gray-50',
              getSizeStyles(size),
              getVariantStyles(inputVariant),
              prefix && 'pl-10',
              (suffix || showClearButton) && 'pr-10',
              className
            )}
            disabled={disabled}
            value={currentValue}
            onChange={handleChange}
            {...props}
          />

          {/* 后缀内容 */}
          <div className="absolute right-3 top-1/2 transform -translate-y-1/2 flex items-center space-x-1">
            {/* 清除按钮 */}
            {showClearButton && (
              <button
                type="button"
                onClick={handleClear}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <X size={16} />
              </button>
            )}
            
            {/* 后缀图标 */}
            {suffix && (
              <div className="text-gray-400">
                {suffix}
              </div>
            )}
          </div>
        </div>

        {/* 帮助文本或错误信息 */}
        {(helperText || error) && (
          <p className={cn(
            'mt-1 text-sm',
            error ? 'text-red-600' : 'text-gray-500'
          )}>
            {error || helperText}
          </p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

/**
 * 密码输入框组件
 */
export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(
  ({ showStrength = false, ...props }, ref) => {
    const [showPassword, setShowPassword] = useState(false);
    const [strength, setStrength] = useState(0);

    // 计算密码强度
    const calculateStrength = (password: string): number => {
      let score = 0;
      if (password.length >= 8) score += 1;
      if (/[a-z]/.test(password)) score += 1;
      if (/[A-Z]/.test(password)) score += 1;
      if (/[0-9]/.test(password)) score += 1;
      if (/[^A-Za-z0-9]/.test(password)) score += 1;
      return score;
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (showStrength) {
        setStrength(calculateStrength(e.target.value));
      }
      props.onChange?.(e);
    };

    const getStrengthColor = (score: number) => {
      if (score <= 1) return 'bg-red-500';
      if (score <= 2) return 'bg-yellow-500';
      if (score <= 3) return 'bg-blue-500';
      return 'bg-green-500';
    };

    const getStrengthText = (score: number) => {
      if (score <= 1) return '弱';
      if (score <= 2) return '一般';
      if (score <= 3) return '强';
      return '很强';
    };

    return (
      <div>
        <Input
          ref={ref}
          type={showPassword ? 'text' : 'password'}
          suffix={
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="text-gray-400 hover:text-gray-600 transition-colors"
            >
              {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          }
          onChange={handleChange}
          {...props}
        />

        {/* 密码强度指示器 */}
        {showStrength && props.value && (
          <div className="mt-2">
            <div className="flex items-center space-x-2">
              <div className="flex-1 bg-gray-200 rounded-full h-2">
                <div
                  className={cn(
                    'h-2 rounded-full transition-all duration-300',
                    getStrengthColor(strength)
                  )}
                  style={{ width: `${(strength / 5) * 100}%` }}
                />
              </div>
              <span className="text-xs text-gray-500">
                密码强度: {getStrengthText(strength)}
              </span>
            </div>
          </div>
        )}
      </div>
    );
  }
);

PasswordInput.displayName = 'PasswordInput';

/**
 * 搜索输入框组件
 */
export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  ({ 
    onSearch, 
    searchText = '搜索', 
    showSearchButton = false,
    onKeyDown,
    ...props 
  }, ref) => {
    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter' && onSearch) {
        onSearch(e.currentTarget.value);
      }
      onKeyDown?.(e);
    };

    const handleSearchClick = () => {
      if (onSearch && ref && 'current' in ref && ref.current) {
        onSearch(ref.current.value);
      }
    };

    return (
      <div className="flex">
        <Input
          ref={ref}
          prefix={<Search size={16} />}
          onKeyDown={handleKeyDown}
          className={showSearchButton ? 'rounded-r-none' : ''}
          {...props}
        />
        
        {showSearchButton && (
          <button
            type="button"
            onClick={handleSearchClick}
            className={cn(
              'px-4 py-2 bg-blue-600 text-white rounded-r-md',
              'hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500',
              'border border-l-0 border-blue-600'
            )}
          >
            {searchText}
          </button>
        )}
      </div>
    );
  }
);

SearchInput.displayName = 'SearchInput';

/**
 * 文本域组件
 */
export interface TextareaProps extends Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'size'> {
  size?: 'sm' | 'md' | 'lg';
  variant?: 'default' | 'error' | 'success' | 'warning';
  label?: string;
  helperText?: string;
  error?: string;
  required?: boolean;
  containerClassName?: string;
  labelClassName?: string;
  resize?: 'none' | 'vertical' | 'horizontal' | 'both';
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({
    className,
    containerClassName,
    labelClassName,
    size = 'md',
    variant = 'default',
    label,
    helperText,
    error,
    required = false,
    resize = 'vertical',
    ...props
  }, ref) => {
    const inputVariant = error ? 'error' : variant;

    const getResizeStyle = (resize: string) => {
      const styles = {
        none: 'resize-none',
        vertical: 'resize-y',
        horizontal: 'resize-x',
        both: 'resize'
      };
      return styles[resize as keyof typeof styles];
    };

    return (
      <div className={cn('w-full', containerClassName)}>
        {/* 标签 */}
        {label && (
          <label className={cn(
            'block text-sm font-medium text-gray-700 mb-1',
            labelClassName
          )}>
            {label}
            {required && <span className="text-red-500 ml-1">*</span>}
          </label>
        )}

        {/* 文本域 */}
        <textarea
          ref={ref}
          className={cn(
            'w-full rounded-md border bg-white shadow-sm transition-colors',
            'focus:outline-none focus:ring-2 focus:ring-opacity-50',
            'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-gray-50',
            getSizeStyles(size),
            getVariantStyles(inputVariant),
            getResizeStyle(resize),
            className
          )}
          {...props}
        />

        {/* 帮助文本或错误信息 */}
        {(helperText || error) && (
          <p className={cn(
            'mt-1 text-sm',
            error ? 'text-red-600' : 'text-gray-500'
          )}>
            {error || helperText}
          </p>
        )}
      </div>
    );
  }
);

Textarea.displayName = 'Textarea';