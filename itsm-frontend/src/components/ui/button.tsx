import React from 'react';
import { Button as AntButton } from 'antd';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'outline' | 'ghost' | 'secondary' | 'link' | 'destructive';
  size?: 'sm' | 'md' | 'lg';
  children?: React.ReactNode;
}

export const Button = ({ variant, size, className, children, ...props }: ButtonProps) => {
  const antType = variant === 'outline' || variant === 'ghost' ? 'default' : variant === 'link' ? 'link' : 'primary';
  const antSize = size === 'sm' ? 'small' : size === 'lg' ? 'large' : 'middle';

  return (
    <AntButton type={antType} size={antSize} className={className} {...(props as any)}>
      {children}
    </AntButton>
  );
};
