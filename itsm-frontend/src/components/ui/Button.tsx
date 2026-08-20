'use client';

import React from 'react';
import { Button as AntButton } from 'antd';
import type { ButtonProps as AntButtonProps } from 'antd';

/**
 * Button - 兼容 shadcn 风格的 Button 组件
 * 基于 Ant Design Button 实现，支持 variant/size 简写
 */

export type ButtonVariant = 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link';
export type ButtonSize = 'sm' | 'default' | 'lg';

// 注：Antd ButtonProps 的 variant/color 是 antd@5 才有的可选字段；为兼容旧版本，
// 这里主动 omit 这些字段以避免与我们的 variant/color 语义冲突。
export interface ButtonProps extends Omit<AntButtonProps, 'type' | 'size' | 'danger' | 'variant'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
}

const variantToAntConfig: Record<ButtonVariant, { type: AntButtonProps['type']; danger?: boolean }> = {
  default: { type: 'primary' },
  secondary: { type: 'default' },
  destructive: { type: 'primary', danger: true },
  outline: { type: 'dashed' },
  ghost: { type: 'text' },
  link: { type: 'link' },
};

const sizeToAntSize: Record<ButtonSize, AntButtonProps['size']> = {
  sm: 'small',
  default: 'middle',
  lg: 'large',
};

export const Button: React.FC<ButtonProps> = (props) => {
  const { variant = 'default', size = 'default', children } = props;
  const { type: mappedType, danger: mappedDanger } = variantToAntConfig[variant] || variantToAntConfig.default;
  const antSize = sizeToAntSize[size] || 'middle';
  // 将 antd Button 的 type/danger/size 从 props 中取出后转发，确保我们的 variant/size 优先
  const {
    type: _ignoredType,
    danger: _ignoredDanger,
    size: _ignoredSize,
    variant: _ignoredVariant,
    ...rest
  } = props as ButtonProps & {
    type?: AntButtonProps['type'];
    danger?: boolean;
    size?: AntButtonProps['size'];
  };
  return (
    <AntButton {...rest} type={mappedType} size={antSize} danger={mappedDanger}>
      {children}
    </AntButton>
  );
};

export default Button;