'use client';

import React, { forwardRef, useState, useRef, useEffect } from 'react';
import { Calendar as CalendarIcon, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * 日期选择器组件属性
 */
export interface DatePickerProps {
  /** 当前值 */
  value?: Date | [Date, Date] | null;
  /** 默认值 */
  defaultValue?: Date | [Date, Date] | null;
  /** 占位符 */
  placeholder?: string;
  /** 是否范围选择 */
  range?: boolean;
  /** 是否包含时间 */
  showTime?: boolean;
  /** 时间格式 */
  timeFormat?: '12' | '24';
  /** 日期格式 */
  dateFormat?: string;
  /** 是否可清除 */
  clearable?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
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
  /** 最小日期 */
  minDate?: Date;
  /** 最大日期 */
  maxDate?: Date;
  /** 禁用日期函数 */
  disabledDate?: (date: Date) => boolean;
  /** 容器类名 */
  containerClassName?: string;
  /** 标签类名 */
  labelClassName?: string;
  /** 输入框类名 */
  className?: string;
  /** 值变化回调 */
  onChange?: (value: Date | [Date, Date] | null) => void;
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
 * 格式化日期
 */
const formatDate = (date: Date, format: string = 'YYYY-MM-DD', showTime: boolean = false, timeFormat: '12' | '24' = '24') => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  
  let dateStr = format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day);
  
  if (showTime) {
    const hours = date.getHours();
    const minutes = String(date.getMinutes()).padStart(2, '0');
    
    if (timeFormat === '12') {
      const period = hours >= 12 ? 'PM' : 'AM';
      const displayHours = hours % 12 || 12;
      dateStr += ` ${displayHours}:${minutes} ${period}`;
    } else {
      dateStr += ` ${String(hours).padStart(2, '0')}:${minutes}`;
    }
  }
  
  return dateStr;
};

/**
 * 获取月份的天数
 */
const getDaysInMonth = (year: number, month: number) => {
  return new Date(year, month + 1, 0).getDate();
};

/**
 * 获取月份第一天是星期几
 */
const getFirstDayOfMonth = (year: number, month: number) => {
  return new Date(year, month, 1).getDay();
};

/**
 * 检查日期是否相等
 */
const isSameDay = (date1: Date, date2: Date) => {
  return date1.getFullYear() === date2.getFullYear() &&
         date1.getMonth() === date2.getMonth() &&
         date1.getDate() === date2.getDate();
};

/**
 * 检查日期是否在范围内
 */
const isDateInRange = (date: Date, start: Date | null, end: Date | null) => {
  if (!start || !end) return false;
  return date >= start && date <= end;
};

/**
 * 日历组件
 */
const CalendarComponent: React.FC<{
  currentDate: Date;
  selectedDate: Date | [Date, Date] | null;
  range: boolean;
  minDate?: Date;
  maxDate?: Date;
  disabledDate?: (date: Date) => boolean;
  onDateSelect: (date: Date) => void;
  onMonthChange: (year: number, month: number) => void;
}> = ({
  currentDate,
  selectedDate,
  range,
  minDate,
  maxDate,
  disabledDate,
  onDateSelect,
  onMonthChange
}) => {
  const year = currentDate.getFullYear();
  const month = currentDate.getMonth();
  const daysInMonth = getDaysInMonth(year, month);
  const firstDay = getFirstDayOfMonth(year, month);
  
  const [tempRangeStart, setTempRangeStart] = useState<Date | null>(null);

  // 生成日历天数
  const days = [];
  
  // 上个月的天数（填充）
  const prevMonth = month === 0 ? 11 : month - 1;
  const prevYear = month === 0 ? year - 1 : year;
  const daysInPrevMonth = getDaysInMonth(prevYear, prevMonth);
  
  for (let i = firstDay - 1; i >= 0; i--) {
    const day = daysInPrevMonth - i;
    const date = new Date(prevYear, prevMonth, day);
    days.push({ date, isCurrentMonth: false });
  }
  
  // 当前月的天数
  for (let day = 1; day <= daysInMonth; day++) {
    const date = new Date(year, month, day);
    days.push({ date, isCurrentMonth: true });
  }
  
  // 下个月的天数（填充到42天）
  const nextMonth = month === 11 ? 0 : month + 1;
  const nextYear = month === 11 ? year + 1 : year;
  const remainingDays = 42 - days.length;
  
  for (let day = 1; day <= remainingDays; day++) {
    const date = new Date(nextYear, nextMonth, day);
    days.push({ date, isCurrentMonth: false });
  }

  // 处理日期点击
  const handleDateClick = (date: Date) => {
    if (isDateDisabled(date)) return;
    
    if (range) {
      if (!tempRangeStart) {
        setTempRangeStart(date);
      } else {
        const start = tempRangeStart <= date ? tempRangeStart : date;
        const end = tempRangeStart <= date ? date : tempRangeStart;
        onDateSelect([start, end] as [Date, Date]);
        setTempRangeStart(null);
      }
    } else {
      onDateSelect(date);
    }
  };

  // 检查日期是否禁用
  const isDateDisabled = (date: Date) => {
    if (minDate && date < minDate) return true;
    if (maxDate && date > maxDate) return true;
    if (disabledDate && disabledDate(date)) return true;
    return false;
  };

  // 检查日期是否选中
  const isDateSelected = (date: Date) => {
    if (!selectedDate) return false;
    
    if (Array.isArray(selectedDate)) {
      return selectedDate.some(d => isSameDay(d, date));
    }
    
    return isSameDay(selectedDate, date);
  };

  // 检查日期是否在范围内
  const isDateInSelectedRange = (date: Date) => {
    if (!range || !Array.isArray(selectedDate) || selectedDate.length !== 2) return false;
    return isDateInRange(date, selectedDate[0], selectedDate[1]);
  };

  // 检查日期是否在临时范围内
  const isDateInTempRange = (date: Date) => {
    if (!range || !tempRangeStart) return false;
    const start = tempRangeStart <= date ? tempRangeStart : date;
    const end = tempRangeStart <= date ? date : tempRangeStart;
    return isDateInRange(date, start, end);
  };

  return (
    <div className="p-4">
      {/* 月份导航 */}
      <div className="flex items-center justify-between mb-4">
        <button
          type="button"
          onClick={() => onMonthChange(month === 0 ? year - 1 : year, month === 0 ? 11 : month - 1)}
          className="p-1 hover:bg-gray-100 rounded"
        >
          <ChevronLeft size={16} />
        </button>
        
        <h3 className="text-sm font-medium">
          {year}年{month + 1}月
        </h3>
        
        <button
          type="button"
          onClick={() => onMonthChange(month === 11 ? year + 1 : year, month === 11 ? 0 : month + 1)}
          className="p-1 hover:bg-gray-100 rounded"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      {/* 星期标题 */}
      <div className="grid grid-cols-7 gap-1 mb-2">
        {['日', '一', '二', '三', '四', '五', '六'].map(day => (
          <div key={day} className="text-center text-xs text-gray-500 py-2">
            {day}
          </div>
        ))}
      </div>

      {/* 日期网格 */}
      <div className="grid grid-cols-7 gap-1">
        {days.map(({ date, isCurrentMonth }, index) => {
          const disabled = isDateDisabled(date);
          const selected = isDateSelected(date);
          const inRange = isDateInSelectedRange(date) || isDateInTempRange(date);
          
          return (
            <button
              key={index}
              type="button"
              disabled={disabled}
              onClick={() => handleDateClick(date)}
              className={cn(
                'w-8 h-8 text-xs rounded transition-colors',
                'hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500',
                !isCurrentMonth && 'text-gray-400',
                disabled && 'opacity-50 cursor-not-allowed',
                selected && 'bg-blue-600 text-white hover:bg-blue-700',
                inRange && !selected && 'bg-blue-100 text-blue-600'
              )}
            >
              {date.getDate()}
            </button>
          );
        })}
      </div>
    </div>
  );
};

/**
 * 时间选择器组件
 */
const TimePicker: React.FC<{
  value: Date;
  timeFormat: '12' | '24';
  onChange: (date: Date) => void;
}> = ({ value, timeFormat, onChange }) => {
  const hours = value.getHours();
  const minutes = value.getMinutes();

  const handleHourChange = (hour: number) => {
    const newDate = new Date(value);
    newDate.setHours(hour);
    onChange(newDate);
  };

  const handleMinuteChange = (minute: number) => {
    const newDate = new Date(value);
    newDate.setMinutes(minute);
    onChange(newDate);
  };

  const displayHours = timeFormat === '12' ? (hours % 12 || 12) : hours;
  const period = hours >= 12 ? 'PM' : 'AM';

  return (
    <div className="p-4 border-t border-gray-200">
      <div className="flex items-center justify-center space-x-2">
        {/* 小时 */}
        <select
          value={displayHours}
          onChange={(e) => {
            const hour = parseInt(e.target.value);
            const actualHour = timeFormat === '12' 
              ? (period === 'PM' && hour !== 12 ? hour + 12 : (period === 'AM' && hour === 12 ? 0 : hour))
              : hour;
            handleHourChange(actualHour);
          }}
          className="px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {Array.from({ length: timeFormat === '12' ? 12 : 24 }, (_, i) => {
            const hour = timeFormat === '12' ? i + 1 : i;
            return (
              <option key={hour} value={hour}>
                {String(hour).padStart(2, '0')}
              </option>
            );
          })}
        </select>

        <span className="text-gray-500">:</span>

        {/* 分钟 */}
        <select
          value={minutes}
          onChange={(e) => handleMinuteChange(parseInt(e.target.value))}
          className="px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {Array.from({ length: 60 }, (_, i) => (
            <option key={i} value={i}>
              {String(i).padStart(2, '0')}
            </option>
          ))}
        </select>

        {/* AM/PM */}
        {timeFormat === '12' && (
          <select
            value={period}
            onChange={(e) => {
              const newPeriod = e.target.value;
              const hour = displayHours === 12 ? 0 : displayHours;
              const actualHour = newPeriod === 'PM' ? hour + 12 : hour;
              handleHourChange(actualHour);
            }}
            className="px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="AM">AM</option>
            <option value="PM">PM</option>
          </select>
        )}
      </div>
    </div>
  );
};

/**
 * 日期选择器组件
 */
export const DatePicker = forwardRef<HTMLDivElement, DatePickerProps>(
  ({
    value,
    defaultValue,
    placeholder,
    range = false,
    showTime = false,
    timeFormat = '24',
    dateFormat = 'YYYY-MM-DD',
    clearable = false,
    disabled = false,
    size = 'md',
    variant = 'default',
    label,
    helperText,
    error,
    required = false,
    minDate,
    maxDate,
    disabledDate,
    containerClassName,
    labelClassName,
    className,
    onChange,
    onClear,
    ...props
  }, ref) => {
    const [isOpen, setIsOpen] = useState(false);
    const [internalValue, setInternalValue] = useState(defaultValue);
    const [currentDate, setCurrentDate] = useState(new Date());
    const pickerRef = useRef<HTMLDivElement>(null);

    const currentValue = value !== undefined ? value : internalValue;
    const pickerVariant = error ? 'error' : variant;

    // 格式化显示值
    const getDisplayValue = () => {
      if (!currentValue) return '';
      
      if (Array.isArray(currentValue)) {
        const [start, end] = currentValue;
        const startStr = formatDate(start, dateFormat, showTime, timeFormat);
        const endStr = formatDate(end, dateFormat, showTime, timeFormat);
        return `${startStr} ~ ${endStr}`;
      }
      
      return formatDate(currentValue, dateFormat, showTime, timeFormat);
    };

    // 处理日期选择
    const handleDateSelect = (date: Date | [Date, Date]) => {
      let newValue = date;
      
      // 如果显示时间且选择的是单个日期，保持原有时间
      if (showTime && !Array.isArray(date) && currentValue && !Array.isArray(currentValue)) {
        const newDate = new Date(date);
        newDate.setHours(currentValue.getHours());
        newDate.setMinutes(currentValue.getMinutes());
        newValue = newDate;
      }

      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(newValue);
      
      if (!showTime && !range) {
        setIsOpen(false);
      }
    };

    // 处理时间变化
    const handleTimeChange = (date: Date) => {
      if (value === undefined) {
        setInternalValue(date);
      }
      onChange?.(date);
    };

    // 处理清除
    const handleClear = (e: React.MouseEvent) => {
      e.stopPropagation();
      const newValue = null;
      
      if (value === undefined) {
        setInternalValue(newValue);
      }
      onChange?.(newValue);
      onClear?.();
    };

    // 点击外部关闭
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (pickerRef.current && !pickerRef.current.contains(event.target as Node)) {
          setIsOpen(false);
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

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

        {/* 输入框 */}
        <div ref={pickerRef} className="relative">
          <div
            ref={ref}
            className={cn(
              'w-full rounded-md border bg-white shadow-sm cursor-pointer transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-opacity-50',
              'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-gray-50',
              getSizeStyles(size),
              getVariantStyles(pickerVariant),
              disabled && 'cursor-not-allowed',
              className
            )}
            onClick={() => !disabled && setIsOpen(!isOpen)}
            {...props}
          >
            <div className="flex items-center justify-between">
              <div className="flex-1 min-w-0">
                {getDisplayValue() || (
                  <span className="text-gray-400">
                    {placeholder || (range ? '选择日期范围' : '选择日期')}
                  </span>
                )}
              </div>
              
              <div className="flex items-center space-x-1">
                {/* 清除按钮 */}
                {clearable && currentValue && !disabled && (
                  <button
                    type="button"
                    onClick={handleClear}
                    className="text-gray-400 hover:text-gray-600 transition-colors"
                  >
                    <X size={16} />
                  </button>
                )}
                
                {/* 日历图标 */}
                <CalendarIcon size={16} className="text-gray-400" />
              </div>
            </div>
          </div>

          {/* 日期选择面板 */}
          {isOpen && (
            <div className="absolute z-50 mt-1 bg-white border border-gray-300 rounded-md shadow-lg">
              <CalendarComponent
                currentDate={currentDate}
                selectedDate={currentValue}
                range={range}
                minDate={minDate}
                maxDate={maxDate}
                disabledDate={disabledDate}
                onDateSelect={handleDateSelect}
                onMonthChange={(year, month) => setCurrentDate(new Date(year, month, 1))}
              />
              
              {/* 时间选择器 */}
              {showTime && currentValue && !Array.isArray(currentValue) && (
                <TimePicker
                  value={currentValue}
                  timeFormat={timeFormat}
                  onChange={handleTimeChange}
                />
              )}
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

DatePicker.displayName = 'DatePicker';

/**
 * 日期范围选择器组件
 */
export const DateRangePicker = forwardRef<HTMLDivElement, Omit<DatePickerProps, 'range'>>(
  (props, ref) => {
    return <DatePicker ref={ref} range {...props} />;
  }
);

DateRangePicker.displayName = 'DateRangePicker';

/**
 * 日期时间选择器组件
 */
export const DateTimePicker = forwardRef<HTMLDivElement, Omit<DatePickerProps, 'showTime'>>(
  (props, ref) => {
    return <DatePicker ref={ref} showTime {...props} />;
  }
);

DateTimePicker.displayName = 'DateTimePicker';