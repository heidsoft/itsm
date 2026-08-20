'use client';

import React from 'react';
import { Badge as AntBadge } from 'antd';
import type { BadgeProps as AntBadgeProps } from 'antd';

/**
 * Badge - 兼容 shadcn 风格的 Badge 组件
 * 基于 Ant Design Badge 实现
 */

export type BadgeVariant = 'default' | 'secondary' | 'outline' | 'destructive';

export interface BadgeProps extends Omit<AntBadgeProps, 'status' | 'color'> {
  variant?: BadgeVariant;
  className?: string;
  children?: React.ReactNode;
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'ant-badge-default',
  secondary: 'bg-gray-100 text-gray-800 border-gray-200',
  outline: 'bg-transparent text-gray-700 border border-gray-300',
  destructive: 'bg-red-100 text-red-800 border-red-200',
};

export const Badge: React.FC<BadgeProps> = ({
  variant = 'default',
  className,
  children,
  ...rest
}) => {
  const variantClass = variantStyles[variant] || variantStyles.default;
  const mergedClassName = className
    ? `${variantClass} ${className}`
    : variantClass;
  return (
    <AntBadge {...rest}>
      <span className={mergedClassName}>{children}</span>
    </AntBadge>
  );
};

export default Badge;