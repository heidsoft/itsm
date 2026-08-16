'use client';

import React from 'react';
import { DatePicker } from 'antd';
import type { DatePickerProps } from 'antd';
import type { RangePickerProps } from 'antd/es/date-picker';
import dayjs, { type Dayjs } from 'dayjs';

const fullStyle: React.CSSProperties = { width: '100%' };

export type AppDatePickerProps = DatePickerProps;

const AppDatePicker: React.FC<AppDatePickerProps> = ({
  size = 'middle',
  allowClear = true,
  format,
  showTime,
  style,
  ...rest
}) => {
  const finalFormat = format ?? (showTime ? 'YYYY-MM-DD HH:mm' : 'YYYY-MM-DD');
  const mergedStyle: React.CSSProperties = { ...fullStyle, ...style };
  return (
    <DatePicker
      size={size}
      allowClear={allowClear}
      format={finalFormat}
      showTime={showTime}
      style={mergedStyle}
      {...rest}
    />
  );
};

export type AppDateRangePickerProps = RangePickerProps;

const defaultPresets: NonNullable<RangePickerProps['presets']> = [
  { label: '今天', value: [dayjs(), dayjs()] as [Dayjs, Dayjs] },
  { label: '昨天', value: [dayjs().subtract(1, 'day'), dayjs().subtract(1, 'day')] as [Dayjs, Dayjs] },
  { label: '近 7 天', value: [dayjs().subtract(6, 'day'), dayjs()] as [Dayjs, Dayjs] },
  { label: '近 30 天', value: [dayjs().subtract(29, 'day'), dayjs()] as [Dayjs, Dayjs] },
  { label: '本月', value: [dayjs().startOf('month'), dayjs().endOf('month')] as [Dayjs, Dayjs] },
  { label: '上月', value: [dayjs().subtract(1, 'month').startOf('month'), dayjs().subtract(1, 'month').endOf('month')] as [Dayjs, Dayjs] },
];

const AppDateRangePicker: React.FC<AppDateRangePickerProps> = ({
  size = 'middle',
  allowClear = true,
  format = 'YYYY-MM-DD',
  showTime,
  presets,
  style,
  ...rest
}) => {
  const finalFormat = showTime && format === 'YYYY-MM-DD' ? 'YYYY-MM-DD HH:mm' : format;
  const finalPresets = presets ?? defaultPresets;
  const mergedStyle: React.CSSProperties = { ...fullStyle, ...style };
  return (
    <DatePicker.RangePicker
      size={size}
      allowClear={allowClear}
      format={finalFormat}
      showTime={showTime}
      presets={finalPresets}
      style={mergedStyle}
      {...rest}
    />
  );
};

export { AppDatePicker, AppDateRangePicker };
export default AppDatePicker;