'use client';

import React from 'react';
import { Select as AntSelect } from 'antd';
import type { SelectProps } from 'antd';

export type AppSelectProps = SelectProps;

const defaultStyle: React.CSSProperties = { width: '100%' };

const AppSelect: React.FC<AppSelectProps> = ({
  size = 'middle',
  showSearch = true,
  optionFilterProp = 'label',
  allowClear = true,
  style,
  ...rest
}) => {
  const mergedStyle: React.CSSProperties = { ...defaultStyle, ...style };
  return (
    <AntSelect
      size={size}
      showSearch={showSearch}
      optionFilterProp={optionFilterProp}
      allowClear={allowClear}
      style={mergedStyle}
      {...rest}
    />
  );
};

export default AppSelect;