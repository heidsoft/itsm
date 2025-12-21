'use client';

import React, { useState, useRef, useEffect } from 'react';
import { FormSelectProps, SelectOption } from './types';
import { FormField } from './form-field';

const Select: React.FC<Omit<FormSelectProps, 'label' | 'error' | 'help'>> = ({
  value,
  options = [],
  multiple = false,
  searchable = false,
  clearable = false,
  loading = false,
  loadingText = '加载中...',
  noOptionsText = '暂无选项',
  maxTagCount = 3,
  placeholder,
  disabled = false,
  onChange,
  onSearch,
  className = ''
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [filteredOptions, setFilteredOptions] = useState(options);
  const selectRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // 处理选中值
  const selectedValues = Array.isArray(value) ? value : value !== undefined ? [value] : [];
  const selectedOptions = options.filter(option => selectedValues.includes(option.value));

  // 过滤选项
  useEffect(() => {
    if (!searchQuery) {
      setFilteredOptions(options);
    } else {
      const filtered = options.filter(option =>
        option.label.toLowerCase().includes(searchQuery.toLowerCase())
      );
      setFilteredOptions(filtered);
    }
  }, [searchQuery, options]);

  // 点击外部关闭下拉框
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (selectRef.current && !selectRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearchQuery('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 处理选项点击
  const handleOptionClick = (option: SelectOption) => {
    if (option.disabled) return;

    let newValue;
    if (multiple) {
      const currentValues = Array.isArray(value) ? value as (string | number)[] : [];
      if (currentValues.includes(option.value)) {
        newValue = currentValues.filter(v => v !== option.value);
      } else {
        newValue = [...currentValues, option.value];
      }
    } else {
      newValue = option.value;
      setIsOpen(false);
    }

    onChange?.(newValue);
  };

  // 处理清除
  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange?.(multiple ? [] : undefined);
  };

  // 处理搜索
  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    const query = e.target.value;
    setSearchQuery(query);
    onSearch?.(query);
  };

  // 渲染选中的标签
  const renderSelectedTags = () => {
    if (!multiple || selectedOptions.length === 0) return null;

    const visibleTags = selectedOptions.slice(0, maxTagCount);
    const hiddenCount = selectedOptions.length - maxTagCount;

    return (
      <div className="flex flex-wrap gap-1">
        {visibleTags.map(option => (
          <span
            key={option.value}
            className="inline-flex items-center px-2 py-1 rounded text-xs bg-blue-100 text-blue-800"
          >
            {option.label}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                handleOptionClick(option);
              }}
              className="ml-1 hover:text-blue-600"
            >
              <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </span>
        ))}
        {hiddenCount > 0 && (
          <span className="inline-flex items-center px-2 py-1 rounded text-xs bg-gray-100 text-gray-600">
            +{hiddenCount}
          </span>
        )}
      </div>
    );
  };

  // 渲染显示文本
  const renderDisplayText = () => {
    if (multiple) {
      return selectedOptions.length > 0 ? null : placeholder;
    }
    return selectedOptions.length > 0 ? selectedOptions[0].label : placeholder;
  };

  const selectClasses = [
    'relative w-full cursor-pointer rounded-md border border-gray-300 bg-white py-2 pl-3 pr-10 text-left shadow-sm',
    'focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500',
    disabled ? 'bg-gray-50 cursor-not-allowed' : 'hover:border-gray-400',
    className
  ].filter(Boolean).join(' ');

  return (
    <div ref={selectRef} className="relative">
      <div
        className={selectClasses}
        onClick={() => !disabled && setIsOpen(!isOpen)}
      >
        <div className="flex items-center min-h-[20px]">
          {multiple ? (
            <div className="flex-1">
              {renderSelectedTags()}
              {selectedOptions.length === 0 && (
                <span className="text-gray-500">{placeholder}</span>
              )}
            </div>
          ) : (
            <span className={selectedOptions.length > 0 ? 'text-gray-900' : 'text-gray-500'}>
              {renderDisplayText()}
            </span>
          )}
        </div>

        <div className="absolute inset-y-0 right-0 flex items-center pr-2">
          {clearable && selectedOptions.length > 0 && (
            <button
              type="button"
              onClick={handleClear}
              className="mr-1 hover:text-gray-600"
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
          <svg
            className={`h-4 w-4 text-gray-400 transition-transform ${
              isOpen ? 'rotate-180' : ''
            }`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      {isOpen && (
        <div className="absolute z-10 mt-1 w-full rounded-md bg-white shadow-lg border border-gray-200">
          {searchable && (
            <div className="p-2 border-b border-gray-200">
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={handleSearch}
                placeholder="搜索选项..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                autoFocus
              />
            </div>
          )}

          <div className="max-h-60 overflow-auto py-1">
            {loading ? (
              <div className="px-3 py-2 text-gray-500 text-center">{loadingText}</div>
            ) : filteredOptions.length === 0 ? (
              <div className="px-3 py-2 text-gray-500 text-center">{noOptionsText}</div>
            ) : (
              filteredOptions.map(option => {
                const isSelected = selectedValues.includes(option.value);
                return (
                  <div
                    key={option.value}
                    onClick={() => handleOptionClick(option)}
                    className={[
                      'cursor-pointer select-none relative py-2 pl-3 pr-9',
                      option.disabled
                        ? 'text-gray-400 cursor-not-allowed'
                        : isSelected
                        ? 'bg-blue-100 text-blue-900'
                        : 'text-gray-900 hover:bg-gray-100'
                    ].filter(Boolean).join(' ')}
                  >
                    <span className="block truncate">{option.label}</span>
                    {isSelected && (
                      <span className="absolute inset-y-0 right-0 flex items-center pr-4">
                        <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                          <path
                            fillRule="evenodd"
                            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                            clipRule="evenodd"
                          />
                        </svg>
                      </span>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export const FormSelect: React.FC<FormSelectProps> = (props) => {
  const { label, error, help, ...selectProps } = props;

  if (label || error || help) {
    return (
      <FormField
        name={props.name}
        label={label}
        required={props.required}
        error={error}
        help={help}
      >
        <Select {...selectProps} />
      </FormField>
    );
  }

  return <Select {...selectProps} />;
};