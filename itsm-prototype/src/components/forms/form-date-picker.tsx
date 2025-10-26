'use client';

import React, { useState, useRef, useEffect, forwardRef } from 'react';
import { FormDatePickerProps } from './types';
import { FormField } from './form-field';

const DatePicker = forwardRef<HTMLInputElement, Omit<FormDatePickerProps, 'label' | 'error' | 'help'>>(
  ({ 
    value, 
    showTime = false,
    showToday = true,
    disabledDate,
    disabled = false,
    placeholder,
    onChange,
    onBlur,
    onFocus,
    className = '',
    ...props 
  }, ref) => {
    const [isOpen, setIsOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [selectedDate, setSelectedDate] = useState<Date | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);

    // 格式化日期
    const formatDate = React.useCallback((date: Date): string => {
      if (showTime) {
        return date.toLocaleString('zh-CN', {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit'
        });
      }
      return date.toLocaleDateString('zh-CN');
    }, [showTime]);

    // 解析输入值
    const parseValue = React.useCallback((val: string | Date | undefined): Date | null => {
      if (!val) return null;
      if (val instanceof Date) return val;
      const parsed = new Date(val);
      return isNaN(parsed.getTime()) ? null : parsed;
    }, []);

    // 初始化值
    useEffect(() => {
      const parsed = parseValue(value);
      setSelectedDate(parsed);
      setInputValue(parsed ? formatDate(parsed) : '');
    }, [value, formatDate, parseValue]);

    // 点击外部关闭
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
          setIsOpen(false);
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    // 处理日期选择
    const handleDateSelect = (date: Date) => {
      if (disabledDate && disabledDate(date)) return;
      
      setSelectedDate(date);
      setInputValue(formatDate(date));
      onChange?.(date);
      
      if (!showTime) {
        setIsOpen(false);
      }
    };

    // 处理今天按钮
    const handleToday = () => {
      const today = new Date();
      handleDateSelect(today);
    };

    // 生成日历
    const generateCalendar = () => {
      const now = new Date();
      const year = selectedDate?.getFullYear() || now.getFullYear();
      const month = selectedDate?.getMonth() || now.getMonth();
      
      const firstDay = new Date(year, month, 1);
      const startDate = new Date(firstDay);
      startDate.setDate(startDate.getDate() - firstDay.getDay());
      
      const days = [];
      const current = new Date(startDate);
      
      for (let i = 0; i < 42; i++) {
        days.push(new Date(current));
        current.setDate(current.getDate() + 1);
      }
      
      return { year, month, days };
    };

    const { year, month, days } = generateCalendar();

    return (
      <div ref={containerRef} className={`relative ${className}`}>
        <input
          ref={ref}
          type="text"
          value={inputValue}
          placeholder={placeholder || (showTime ? '请选择日期时间' : '请选择日期')}
          disabled={disabled}
          readOnly
          onClick={() => !disabled && setIsOpen(!isOpen)}
          onFocus={onFocus}
          onBlur={onBlur}
          className={`
            w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm
            focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500
            ${disabled ? 'bg-gray-50 cursor-not-allowed' : 'bg-white cursor-pointer'}
            ${isOpen ? 'ring-2 ring-blue-500 border-blue-500' : ''}
          `}
          {...props}
        />
        
        <div className="absolute right-3 top-1/2 transform -translate-y-1/2 pointer-events-none">
          <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
        </div>

        {isOpen && (
          <div className="absolute z-50 mt-1 bg-white border border-gray-200 rounded-md shadow-lg">
            <div className="p-4">
              {/* 月份导航 */}
              <div className="flex items-center justify-between mb-4">
                <button
                  type="button"
                  onClick={() => {
                    const newDate = new Date(year, month - 1, 1);
                    setSelectedDate(newDate);
                  }}
                  className="p-1 hover:bg-gray-100 rounded"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                  </svg>
                </button>
                
                <span className="font-medium">
                  {year}年{month + 1}月
                </span>
                
                <button
                  type="button"
                  onClick={() => {
                    const newDate = new Date(year, month + 1, 1);
                    setSelectedDate(newDate);
                  }}
                  className="p-1 hover:bg-gray-100 rounded"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </button>
              </div>

              {/* 星期标题 */}
              <div className="grid grid-cols-7 gap-1 mb-2">
                {['日', '一', '二', '三', '四', '五', '六'].map(day => (
                  <div key={day} className="text-center text-sm text-gray-500 py-1">
                    {day}
                  </div>
                ))}
              </div>

              {/* 日期网格 */}
              <div className="grid grid-cols-7 gap-1">
                {days.map((day, index) => {
                  const isCurrentMonth = day.getMonth() === month;
                  const isSelected = selectedDate && 
                    day.getFullYear() === selectedDate.getFullYear() &&
                    day.getMonth() === selectedDate.getMonth() &&
                    day.getDate() === selectedDate.getDate();
                  const isToday = day.toDateString() === new Date().toDateString();
                  const isDisabled = disabledDate && disabledDate(day);

                  return (
                    <button
                      key={index}
                      type="button"
                      onClick={() => handleDateSelect(day)}
                      disabled={isDisabled}
                      className={`
                        w-8 h-8 text-sm rounded hover:bg-blue-50
                        ${isCurrentMonth ? 'text-gray-900' : 'text-gray-400'}
                        ${isSelected ? 'bg-blue-600 text-white hover:bg-blue-700' : ''}
                        ${isToday && !isSelected ? 'bg-blue-100 text-blue-600' : ''}
                        ${isDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
                      `}
                    >
                      {day.getDate()}
                    </button>
                  );
                })}
              </div>

              {/* 底部按钮 */}
              <div className="flex justify-between items-center mt-4 pt-3 border-t border-gray-200">
                {showToday && (
                  <button
                    type="button"
                    onClick={handleToday}
                    className="text-sm text-blue-600 hover:text-blue-700"
                  >
                    今天
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setIsOpen(false)}
                  className="text-sm text-gray-600 hover:text-gray-700"
                >
                  确定
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  }
);

DatePicker.displayName = 'DatePicker';

export const FormDatePicker: React.FC<FormDatePickerProps> = (props) => {
  const { label, error, help, ...datePickerProps } = props;

  return (
    <FormField label={label} error={error} help={help} name={props.name}>
      <DatePicker {...datePickerProps} />
    </FormField>
  );
};

export { DatePicker };