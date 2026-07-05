import React from 'react';
import { Tag } from 'antd';

interface BadgeProps {
  variant?: 'default' | 'secondary' | 'outline' | 'destructive';
  className?: string;
  children?: React.ReactNode;
}

const variantColorMap: Record<string, string> = {
  default: 'blue',
  secondary: 'default',
  outline: 'default',
  destructive: 'red',
};

export const Badge = ({ variant = 'default', className, children }: BadgeProps) => (
  <Tag color={variantColorMap[variant]} className={className}>
    {children}
  </Tag>
);
