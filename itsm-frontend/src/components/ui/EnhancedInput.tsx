'use client';

import React, { useState, useRef, forwardRef, useImperativeHandle } from 'react';
import { Input, InputProps, AutoComplete } from 'antd';
import { cn } from '@/lib/utils';
import { TouchFeedback } from './TouchFeedback';
import { useIsTouchDevice } from '@/hooks/useResponsive';

const { TextArea } = Input;

export interface EnhancedInputProps extends Omit<InputProps, 'size'> {
  /** 输入框变体 */
  variant?: 'default' | 'filled' | 'outlined' | 'underlined';
  /** 尺寸 */
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** 是否显示清除按钮 */
  allowClear?: boolean;
  /** 前缀图标 */
  prefixIcon?: React.ReactNode;
  /** 后缀图标 */
  suffixIcon?: React.ReactNode;
  /** 验证状态 */
  validation?: {
    status: 'success' | 'warning' | 'error';
    message?: string;
  };
  /** 输入提示 */
  hints?: string[];
  /** 自动完成选项 */
  options?: Array<{ value: string; label?: string; description?: string }>;
  /** 最大字符数 */
  maxLength?: number;
  /** 是否显示字符计数 */
  showCount?: boolean;
  /** 输入类型扩展 */
  inputType?: 'text' | 'email' | 'tel' | 'url' | 'password' | 'search';
  /** 格式化输入 */
  formatter?: (value: string) => string;
  /** 移动端优化 */
  mobileOptimized?: boolean;
  /** 自动聚焦 */
  autoFocus?: boolean;
  /** 自定义类名 */
  className?: string;
}

export interface EnhancedTextAreaProps extends Omit<Input.TextAreaProps, 'size'> {
  /** 尺寸 */
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** 最大高度 */
  maxHeight?: number;
  /** 自动调整高度 */
  autoSize?: boolean | { minRows?: number; maxRows?: number };
  /** 是否显示字符计数 */
  showCount?: boolean;
  /** 占位符提示 */
  placeholderHint?: string;
  /** 移动端优化 */
  mobileOptimized?: boolean;
}

/**
 * 增强型输入框组件
 * 提供更丰富的样式和交互体验
 */
export const EnhancedInput = forwardRef<any, EnhancedInputProps>(
  ({
    variant = 'default',
    size = 'md',
    allowClear = true,
    prefixIcon,
    suffixIcon,
    validation,
    hints,
    options = [],
    maxLength,
    showCount = false,
    inputType = 'text',
    formatter,
    mobileOptimized = true,
    autoFocus = false,
    className,
    value,
    onChange,
    ...props
  }, ref) => {
    const [internalValue, setInternalValue] = useState(value || '');
    const [focused, setFocused] = useState(false);
    const isTouchDevice = useIsTouchDevice();
    const inputRef = useRef<any>(null);

    // 暴露ref方法
    useImperativeHandle(ref, () => ({
      focus: () => inputRef.current?.focus(),
      blur: () => inputRef.current?.blur(),
      input: inputRef.current,
      value: internalValue,
    }));

    // 尺寸样式
    const sizeClasses = {
      xs: 'h-6 text-xs px-2',
      sm: 'h-8 text-sm px-3',
      md: 'h-10 text-base px-4',
      lg: 'h-12 text-lg px-5',
      xl: 'h-14 text-xl px-6',
    };

    // 变体样式
    const variantClasses = {
      default: 'border border-gray-300 bg-white hover:border-blue-400 focus:border-blue-500 focus:shadow-sm',
      filled: 'border-0 bg-gray-100 hover:bg-gray-50 focus:bg-white focus:shadow-md',
      outlined: 'border-2 border-gray-300 bg-white hover:border-blue-400 focus:border-blue-500',
      underlined: 'border-0 border-b-2 border-gray-300 bg-transparent hover:border-blue-400 focus:border-blue-500 rounded-none px-0',
    };

    // 处理值变化
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      const formattedValue = formatter ? formatter(newValue) : newValue;
      
      setInternalValue(formattedValue);
      
      if (onChange) {
        onChange({
          ...e,
          target: {
            ...e.target,
            value: formattedValue,
          },
        });
      }
    };

    // 构建输入框属性
    const inputProps: InputProps = {
      ...props,
      type: inputType,
      value: internalValue,
      onChange: handleChange,
      autoFocus,
      maxLength,
      allowClear,
      prefix: prefixIcon,
      suffix: suffixIcon,
      size: 'middle', // Ant Design的size，我们自己控制样式
      className: cn(
        'transition-all duration-200',
        sizeClasses[size],
        variantClasses[variant],
        validation?.status && {
          success: 'border-green-500',
          warning: 'border-yellow-500',
          error: 'border-red-500',
        }[validation.status],
        focused && 'ring-2 ring-blue-500 ring-opacity-20',
        className
      ),
    };

    // 自动完成
    if (options.length > 0) {
      return (
        <div className="relative">
          <AutoComplete
            {...inputProps}
            options={options.map(opt => ({
              value: opt.value,
              label: (
                <div className="py-1">
                  <div className="font-medium">{opt.label || opt.value}</div>
                  {opt.description && (
                    <div className="text-xs text-gray-500">{opt.description}</div>
                  )}
                </div>
              ),
            }))}
            filterOption={(inputValue, option) =>
              (option?.value ?? '').toLowerCase().includes(inputValue.toLowerCase())
            }
          />
          
          {/* 验证消息 */}
          {validation?.message && (
            <div className={cn(
              'mt-1 text-xs transition-all duration-200',
              {
                'text-green-600': validation.status === 'success',
                'text-yellow-600': validation.status === 'warning',
                'text-red-600': validation.status === 'error',
              }[validation.status]
            )}>
              {validation.message}
            </div>
          )}
          
          {/* 输入提示 */}
          {hints && hints.length > 0 && focused && (
            <div className="absolute z-10 w-full mt-1 p-2 bg-gray-50 border border-gray-200 rounded-md text-xs text-gray-600">
              <div className="font-medium mb-1">提示:</div>
              <ul className="space-y-1">
                {hints.map((hint, index) => (
                  <li key={index} className="flex items-start">
                    <span className="mr-1">•</span>
                    <span>{hint}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          
          {/* 字符计数 */}
          {showCount && maxLength && (
            <div className="absolute right-2 -bottom-5 text-xs text-gray-500">
              {internalValue.length}/{maxLength}
            </div>
          )}
        </div>
      );
    }

    // 移动端优化
    if (isTouchDevice && mobileOptimized) {
      return (
        <TouchFeedback
          touchArea={{ 
            width: sizeClasses[size].match(/px-(\d+)/)?.[1] ? 
              parseInt(sizeClasses[size].match(/px-(\d+)/)![1]) + 20 : 
              60,
            height: sizeClasses[size].match(/h-(\d+)/)?.[1] ? 
              parseInt(sizeClasses[size].match(/h-(\d+)/)![1]) * 4 : 
              44
          }}
          className="w-full"
        >
          <div className="relative">
            <Input {...inputProps} ref={inputRef} />
            
            {/* 验证消息 */}
            {validation?.message && (
              <div className={cn(
                'mt-1 text-xs transition-all duration-200',
                {
                  'text-green-600': validation.status === 'success',
                  'text-yellow-600': validation.status === 'warning',
                  'text-red-600': validation.status === 'error',
                }[validation.status]
              )}>
                {validation.message}
              </div>
            )}
            
            {/* 字符计数 */}
            {showCount && maxLength && (
              <div className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-gray-500 bg-white px-1">
                {internalValue.length}/{maxLength}
              </div>
            )}
          </div>
        </TouchFeedback>
      );
    }

    // 默认情况
    return (
      <div className="relative">
        <Input {...inputProps} ref={inputRef} />
        
        {/* 验证消息 */}
        {validation?.message && (
          <div className={cn(
            'mt-1 text-xs transition-all duration-200',
            {
              'text-green-600': validation.status === 'success',
              'text-yellow-600': validation.status === 'warning',
              'text-red-600': validation.status === 'error',
            }[validation.status]
          )}>
            {validation.message}
          </div>
        )}
        
        {/* 字符计数 */}
        {showCount && maxLength && (
          <div className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-gray-500 bg-white px-1">
            {internalValue.length}/{maxLength}
          </div>
        )}
      </div>
    );
  }
);

/**
 * 增强型文本域组件
 */
export const EnhancedTextArea: React.FC<EnhancedTextAreaProps> = ({
  size = 'md',
  maxHeight,
  autoSize = true,
  showCount = false,
  placeholderHint,
  mobileOptimized = true,
  className,
  value,
  onChange,
  ...props
}) => {
  const [internalValue, setInternalValue] = useState(value || '');
  const [focused, setFocused] = useState(false);
  const isTouchDevice = useIsTouchDevice();
  const textAreaRef = useRef<any>(null);

  // 尺寸样式
  const sizeClasses = {
    xs: 'text-xs p-2',
    sm: 'text-sm p-3',
    md: 'text-base p-4',
    lg: 'text-lg p-5',
    xl: 'text-xl p-6',
  };

  // 处理值变化
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newValue = e.target.value;
    setInternalValue(newValue);
    
    if (onChange) {
      onChange(e);
    }
  };

  // 自动调整高度的配置
  const autoSizeConfig = typeof autoSize === 'boolean' 
    ? { minRows: 2, maxRows: 6 }
    : autoSize;

  const textAreaProps = {
    ...props,
    value: internalValue,
    onChange: handleChange,
    autoSize: autoSize ? autoSizeConfig : false,
    className: cn(
      'transition-all duration-200 border-gray-300 hover:border-blue-400 focus:border-blue-500 focus:shadow-sm',
      sizeClasses[size],
      maxHeight && `max-h-[${maxHeight}px]`,
      focused && 'ring-2 ring-blue-500 ring-opacity-20',
      className
    ),
  };

  if (isTouchDevice && mobileOptimized) {
    return (
      <TouchFeedback
        touchArea={{ 
          width: '100%',
          height: '120px'
        }}
        className="w-full"
      >
        <div className="relative">
          <TextArea {...textAreaProps} ref={textAreaRef} />
          
          {/* 占位符提示 */}
          {placeholderHint && !internalValue && focused && (
            <div className="absolute top-4 left-4 text-xs text-gray-400 pointer-events-none">
              {placeholderHint}
            </div>
          )}
          
          {/* 字符计数 */}
          {showCount && props.maxLength && (
            <div className="absolute right-2 bottom-2 text-xs text-gray-500 bg-white px-1 rounded">
              {internalValue.length}/{props.maxLength}
            </div>
          )}
        </div>
      </TouchFeedback>
    );
  }

  return (
    <div className="relative">
      <TextArea {...textAreaProps} ref={textAreaRef} />
      
      {/* 占位符提示 */}
      {placeholderHint && !internalValue && focused && (
        <div className="absolute top-4 left-4 text-xs text-gray-400 pointer-events-none">
          {placeholderHint}
        </div>
      )}
      
      {/* 字符计数 */}
      {showCount && props.maxLength && (
        <div className="absolute right-2 bottom-2 text-xs text-gray-500 bg-white px-1 rounded">
          {internalValue.length}/{props.maxLength}
        </div>
      )}
    </div>
  );
};

// 设置displayName
EnhancedInput.displayName = 'EnhancedInput';

export default EnhancedInput;