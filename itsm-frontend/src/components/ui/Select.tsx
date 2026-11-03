'use client';

import React from 'react';
import { Select as AntSelect } from 'antd';
import type { SelectProps as AntSelectProps } from 'antd';
import type { DefaultOptionType } from 'antd/es/select';

/**
 * Select - 兼容 shadcn 风格的 Select 组件
 * 基于 Ant Design Select 实现
 *
 * shadcn: <Select value={x} onValueChange={fn}><SelectTrigger><SelectValue placeholder="..." /></SelectTrigger><SelectContent><SelectItem value="a">A</SelectItem></SelectContent></Select>
 * antd:   <Select value={x} onChange={fn} options={[{ value: 'a', label: 'A' }]} placeholder="..." />
 *
 * 选项与 placeholder 在渲染时用 React.Children 递归收集。此前用 effect 注册选项，
 * 首帧 options 为空，AntSelect 会把原始 value（例如 `all`）直接显示出来，
 * 表现为「all all」这类未翻译值。
 */

export interface SelectProps extends Omit<AntSelectProps, 'value' | 'onChange' | 'options' | 'placeholder'> {
  value?: string | number | null;
  onValueChange?: (value: string) => void;
  placeholder?: string;
  className?: string;
  children?: React.ReactNode;
}

export interface SelectTriggerProps {
  children?: React.ReactNode;
}

export interface SelectValueProps {
  placeholder?: string;
}

export interface SelectContentProps {
  className?: string;
  children?: React.ReactNode;
}

export interface SelectItemProps {
  value: string;
  className?: string;
  children?: React.ReactNode;
  disabled?: boolean;
}

interface MarkerProps {
  value?: unknown;
  placeholder?: unknown;
  children?: React.ReactNode;
  disabled?: boolean;
}

interface CollectedSelect {
  options: DefaultOptionType[];
  placeholder?: string;
}

const collectSelect = (children: React.ReactNode, acc: CollectedSelect): void => {
  React.Children.forEach(children, (child) => {
    if (!React.isValidElement(child)) return;
    const props = child.props as MarkerProps;

    if (child.type === SelectItem) {
      const value = typeof props.value === 'string' ? props.value : String(props.value ?? '');
      if (value) acc.options.push({ value, label: props.children, disabled: props.disabled });
      return;
    }

    if (child.type === SelectValue) {
      if (typeof props.placeholder === 'string' && props.placeholder) {
        acc.placeholder = props.placeholder;
      }
      return;
    }

    // SelectTrigger / SelectContent / Fragment 等包装层：继续向下收集。
    if (props.children) collectSelect(props.children, acc);
  });
};

export const Select: React.FC<SelectProps> = ({
  value,
  onValueChange,
  placeholder,
  className,
  children,
  ...rest
}) => {
  const collected: CollectedSelect = { options: [] };
  collectSelect(children, collected);

  return (
    <span style={{ display: 'inline-block', width: '100%' }} className={className}>
      <AntSelect
        {...rest}
        value={value === undefined || value === null ? undefined : value}
        onChange={(v) => onValueChange?.(String(v))}
        placeholder={placeholder ?? collected.placeholder}
        options={collected.options}
        style={{ width: '100%' }}
      />
    </span>
  );
};

// 以下三个组件只是标记，选项由 Select 在渲染时收集。
export const SelectTrigger: React.FC<SelectTriggerProps> = () => null;
export const SelectValue: React.FC<SelectValueProps> = () => null;
export const SelectContent: React.FC<SelectContentProps> = () => null;
export const SelectItem: React.FC<SelectItemProps> = () => null;

export default Select;
