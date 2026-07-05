import React from 'react';
import { Select as AntSelect } from 'antd';

interface SelectProps {
  value?: string;
  onValueChange?: (value: string) => void;
  children?: React.ReactNode;
}

interface SelectTriggerProps {
  className?: string;
  children?: React.ReactNode;
}

interface SelectContentProps {
  children?: React.ReactNode;
}

interface SelectItemProps {
  value: string;
  children?: React.ReactNode;
}

interface SelectValueProps {
  placeholder?: string;
}

// Collect SelectItem values from children for Ant Select options
function extractOptions(children: React.ReactNode): { value: string; label: React.ReactNode }[] {
  const options: { value: string; label: React.ReactNode }[] = [];
  React.Children.forEach(children, (child) => {
    if (React.isValidElement<SelectItemProps>(child) && child.type === SelectItem) {
      options.push({ value: child.props.value, label: child.props.children });
    }
  });
  return options;
}

export const Select = ({ value, onValueChange, children }: SelectProps) => {
  // Find SelectContent children and extract SelectItems
  let options: { value: string; label: React.ReactNode }[] = [];
  React.Children.forEach(children, (child) => {
    if (React.isValidElement<SelectContentProps>(child) && child.type === SelectContent) {
      options = extractOptions(child.props.children);
    }
  });

  return (
    <AntSelect
      value={value}
      onChange={onValueChange}
      style={{ width: '100%' }}
      options={options}
    />
  );
};

export const SelectTrigger = ({ children }: SelectTriggerProps) => <>{children}</>;
export const SelectContent = ({ children }: SelectContentProps) => <>{children}</>;
export const SelectItem = ({ value, children }: SelectItemProps) => <>{children}</>;
export const SelectValue = ({ placeholder }: SelectValueProps) => <>{placeholder}</>;
