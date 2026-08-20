'use client';

import React from 'react';
import { Input as AntInput } from 'antd';
import type { InputProps as AntInputProps } from 'antd';
import type { TextAreaProps } from 'antd/es/input/TextArea';

/**
 * Input - 兼容 shadcn 风格的 Input 组件
 * 基于 Ant Design Input 实现
 */

export interface InputProps
  extends Omit<AntInputProps, 'value' | 'onChange' | 'prefix' | 'suffix'> {
  value?: string | number;
  onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
  className?: string;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
}

export const Input: React.FC<InputProps> = ({ className, ...rest }) => (
  <AntInput {...rest} className={className} />
);

export interface TextareaProps extends TextAreaProps {
  className?: string;
}

export const Textarea: React.FC<TextareaProps> = ({ className, ...rest }) => (
  <AntInput.TextArea {...rest} className={className} />
);

export default Input;