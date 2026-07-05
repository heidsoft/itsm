import React from 'react';
import { Input as AntInput } from 'antd';

interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  className?: string;
}

export const Input = ({ className, ...props }: InputProps) => (
  <AntInput
    className={className}
    value={props.value as string}
    onChange={props.onChange as any}
    placeholder={props.placeholder}
    disabled={props.disabled}
  />
);
