'use client';

import React, { forwardRef, useState, useRef, useEffect } from 'react';
import { ChevronDown, X, Check, Search } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * 选项接口
 */
export interface Option {
  /** 选项值 */
  value: string | number;
  /** 选项标签 */
  label: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 选项图标 */
  icon?: React.ReactNode;
  /** 选项描述 */
  description?: string;
}

/**
 * 选择器组件属性
 */
export interface SelectProps {
  /** 选项列表 */
  options: Option[];
  /** 当前值 */
  value?: string | number | (string | number)[];
  /** 默认值 */
  defaultValue?: string | number | (string | number)[];
  /** 占位符 */
  placeholder?: string;
  /** 是否多选 */
  multiple?: boolean;
  /** 是否可搜索 */
  searchable?: boolean;
  /** 是否可清除 */
  clearable?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否加载中 */
  loading?: boolean;
  /** 尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 状态 */
  variant?: 'default' | 'error' | 'success' | 'warning';
  /** 标签 */
  label?: string;
  /** 帮助文本 */
  helperText?: string;
  /** 错误信息 */
  error?: string;
  /** 是否必填 */
  required?: boolean;
  /** 最大显示选项数 */
  maxTagCount?: number;
  /** 下拉框最大高度 */
  maxHeight?: number;
  /** 容器类名 */
  containerClassName?: string;
  /** 标签类名 */
  labelClassName?: string;
  /** 选择器类名 */
  className?: string;
  /** 值变化回调 */
  onChange?: (value: string | number | (string | number)[] | undefined) => void;
  /** 搜索回调 */
  onSearch?: (searchText: string) => void;
  /** 清除回调 */
  onClear?: () => void;
}

/**
 * 获取尺寸样式
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
 * 获取状态样式
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
 * 选择器组件
 */
export const Select = forwardRef<HTMLDivElement, SelectProps>(
  ({
    options = [],
    value,
    defaultValue,
    placeholder = '请选择',
    multiple = false,
    searchable = false,
    clearable = false,
    disabled = false,
    loading = false,
    size = 'md',
    variant = 'default',
    label,
    helperText,
    error,
    required = false,
    maxTagCount = 3,
    maxHeight = 200,
    containerClassName,
    labelClassName,
    className,
    onChange,
    onSearch,
    onClear,
    ...props
  }, ref) => {
    const [isOpen, setIsOpen] = useState(false);
    const [searchText, setSearchText] = useState('');
    const [internalValue, setInternalValue] = useState(defaultValue);
    const selectRef = useRef<HTMLDivElement>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    const currentValue = value !== undefined ? value : internalValue;
    const selectVariant = error ? 'error' : variant;

    // 过滤选项
    const filteredOptions = searchable && searchText
      ? options.filter(option => 
          option.label.toLowerCase().includes(searchText.toLowerCase())
        )
      : options;

    // 获取选中的选项
    const getSelectedOptions = () => {
      if (!currentValue) return [];
      const values = Array.isArray(currentValue) ? currentValue : [currentValue];
      return options.filter(option => values.includes(option.value));
    };

    const selectedOptions = getSelectedOptions();

    // 处理选项点击
    const handleOptionClick = (option: Option) => {
      if (option.disabled) return;

      let newValue: string | number | (string | number)[] | undefined;

      if (multiple) {
        const currentValues = Array.isArray(currentValue) ? currentValue : [];
        if (currentValues.includes(option.value)) {
          newValue = currentValues.filter(v => v !== option.value);
        } else {
          newValue = [...currentValues, option.value];
        }
      } else {
        newValue = option.value;
        setIsOpen(false);
      }

      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(newValue);
    };

    // 处理清除
    const handleClear = (e: React.MouseEvent) => {
      e.stopPropagation();
      const newValue = multiple ? [] : undefined;
      
      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(newValue);
      onClear?.();
    };

    // 处理搜索
    const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
      const text = e.target.value;
      setSearchText(text);
      onSearch?.(text);
    };

    // 移除标签
    const removeTag = (optionValue: string | number, e: React.MouseEvent) => {
      e.stopPropagation();
      if (!multiple || !Array.isArray(currentValue)) return;
      
      const newValue = currentValue.filter(v => v !== optionValue);
      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(newValue);
    };

    // 点击外部关闭
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (selectRef.current && !selectRef.current.contains(event.target as Node)) {
          setIsOpen(false);
          setSearchText('');
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    // 自动聚焦搜索框
    useEffect(() => {
      if (isOpen && searchable && searchInputRef.current) {
        searchInputRef.current.focus();
      }
    }, [isOpen, searchable]);

    // 渲染选中值
    const renderSelectedValue = () => {
      if (!selectedOptions.length) {
        return <span className="text-gray-400">{placeholder}</span>;
      }

      if (!multiple) {
        const option = selectedOptions[0];
        return (
          <div className="flex items-center">
            {option.icon && <span className="mr-2">{option.icon}</span>}
            <span>{option.label}</span>
          </div>
        );
      }

      // 多选标签显示
      const visibleTags = selectedOptions.slice(0, maxTagCount);
      const remainingCount = selectedOptions.length - maxTagCount;

      return (
        <div className="flex flex-wrap gap-1">
          {visibleTags.map(option => (
            <span
              key={option.value}
              className="inline-flex items-center px-2 py-1 rounded text-xs bg-blue-100 text-blue-800"
            >
              {option.icon && <span className="mr-1">{option.icon}</span>}
              {option.label}
              <button
                type="button"
                onClick={(e) => removeTag(option.value, e)}
                className="ml-1 hover:text-blue-600"
              >
                <X size={12} />
              </button>
            </span>
          ))}
          {remainingCount > 0 && (
            <span className="inline-flex items-center px-2 py-1 rounded text-xs bg-gray-100 text-gray-600">
              +{remainingCount}
            </span>
          )}
        </div>
      );
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

        {/* 选择器 */}
        <div ref={selectRef} className="relative">
          <div
            ref={ref}
            className={cn(
              'w-full rounded-md border bg-white shadow-sm cursor-pointer transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-opacity-50',
              'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-gray-50',
              getSizeStyles(size),
              getVariantStyles(selectVariant),
              disabled && 'cursor-not-allowed',
              className
            )}
            onClick={() => !disabled && setIsOpen(!isOpen)}
            {...props}
          >
            <div className="flex items-center justify-between">
              <div className="flex-1 min-w-0">
                {renderSelectedValue()}
              </div>
              
              <div className="flex items-center space-x-1">
                {/* 清除按钮 */}
                {clearable && selectedOptions.length > 0 && !disabled && (
                  <button
                    type="button"
                    onClick={handleClear}
                    className="text-gray-400 hover:text-gray-600 transition-colors"
                  >
                    <X size={16} />
                  </button>
                )}
                
                {/* 下拉箭头 */}
                <ChevronDown
                  size={16}
                  className={cn(
                    'text-gray-400 transition-transform',
                    isOpen && 'transform rotate-180'
                  )}
                />
              </div>
            </div>
          </div>

          {/* 下拉选项 */}
          {isOpen && (
            <div
              className="absolute z-50 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg"
              style={{ maxHeight: maxHeight + 'px' }}
            >
              {/* 搜索框 */}
              {searchable && (
                <div className="p-2 border-b border-gray-200">
                  <div className="relative">
                    <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
                    <input
                      ref={searchInputRef}
                      type="text"
                      placeholder="搜索选项..."
                      value={searchText}
                      onChange={handleSearch}
                      className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>
              )}

              {/* 选项列表 */}
              <div className="max-h-48 overflow-y-auto">
                {loading ? (
                  <div className="p-3 text-center text-gray-500">
                    加载中...
                  </div>
                ) : filteredOptions.length === 0 ? (
                  <div className="p-3 text-center text-gray-500">
                    {searchText ? '未找到匹配选项' : '暂无选项'}
                  </div>
                ) : (
                  filteredOptions.map(option => {
                    const isSelected = Array.isArray(currentValue)
                      ? currentValue.includes(option.value)
                      : currentValue === option.value;

                    return (
                      <div
                        key={option.value}
                        className={cn(
                          'px-3 py-2 cursor-pointer transition-colors',
                          'hover:bg-gray-50 flex items-center justify-between',
                          option.disabled && 'opacity-50 cursor-not-allowed',
                          isSelected && 'bg-blue-50 text-blue-600'
                        )}
                        onClick={() => handleOptionClick(option)}
                      >
                        <div className="flex items-center flex-1 min-w-0">
                          {option.icon && (
                            <span className="mr-2 flex-shrink-0">{option.icon}</span>
                          )}
                          <div className="min-w-0">
                            <div className="truncate">{option.label}</div>
                            {option.description && (
                              <div className="text-xs text-gray-500 truncate">
                                {option.description}
                              </div>
                            )}
                          </div>
                        </div>
                        
                        {isSelected && (
                          <Check size={16} className="text-blue-600 flex-shrink-0" />
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          )}
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

Select.displayName = 'Select';

/**
 * 多选选择器组件
 */
export const MultiSelect = forwardRef<HTMLDivElement, Omit<SelectProps, 'multiple'>>(
  (props, ref) => {
    return <Select ref={ref} multiple {...props} />;
  }
);

MultiSelect.displayName = 'MultiSelect';

/**
 * 可搜索选择器组件
 */
export const SearchableSelect = forwardRef<HTMLDivElement, Omit<SelectProps, 'searchable'>>(
  (props, ref) => {
    return <Select ref={ref} searchable {...props} />;
  }
);

SearchableSelect.displayName = 'SearchableSelect';