'use client';

import React from 'react';
import { Select as AntSelect } from 'antd';
import type { SelectProps as AntSelectProps } from 'antd';
import type { DefaultOptionType } from 'antd/es/select';

/**
 * Select - 兼容 shadcn 风格的 Select 组件
 * 基于 Ant Design Select 实现
 *
 * 设计差异：
 * - shadcn: <Select value={x} onValueChange={fn}><SelectTrigger><SelectValue placeholder="..." /></SelectTrigger><SelectContent><SelectItem value="a">A</SelectItem></SelectContent></Select>
 * - antd: <Select value={x} onChange={fn} options={[{ value: 'a', label: 'A' }]} placeholder="..." />
 */

export interface SelectContextValue {
  registerOption: (value: string, label: React.ReactNode) => void;
}

const SelectContext = React.createContext<SelectContextValue | null>(null);

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

export const Select: React.FC<SelectProps> = ({
  value,
  onValueChange,
  placeholder,
  className,
  children,
  ...rest
}) => {
  const optionsRef = React.useRef<DefaultOptionType[]>([]);
  const [, force] = React.useReducer((x: number) => x + 1, 0);

  const registerOption = React.useCallback((val: string, label: React.ReactNode) => {
    const idx = optionsRef.current.findIndex((o) => o.value === val);
    const next = { value: val, label };
    if (idx >= 0) optionsRef.current[idx] = next;
    else optionsRef.current.push(next);
    force();
  }, []);

  const ctxValue = React.useMemo<SelectContextValue>(
    () => ({ registerOption }),
    [registerOption],
  );

  return (
    <SelectContext.Provider value={ctxValue}>
      <span style={{ display: 'inline-block', width: '100%' }} className={className}>
        <AntSelect
          {...rest}
          value={value === undefined || value === null ? undefined : value}
          onChange={(v) => onValueChange?.(String(v))}
          placeholder={placeholder}
          options={optionsRef.current}
          style={{ width: '100%' }}
        >
          {children}
        </AntSelect>
      </span>
    </SelectContext.Provider>
  );
};

export const SelectTrigger: React.FC<SelectTriggerProps> = ({ children }) => <>{children}</>;
export const SelectValue: React.FC<SelectValueProps> = () => null;
export const SelectContent: React.FC<SelectContentProps> = ({ children }) => <>{children}</>;

export const SelectItem: React.FC<SelectItemProps> = ({ value, children }) => {
  const ctx = React.useContext(SelectContext);
  React.useEffect(() => {
    ctx?.registerOption(value, children);
  }, [ctx, value, children]);
  return null;
};

export default Select;