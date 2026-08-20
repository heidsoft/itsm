'use client';

import React from 'react';
import { Card as AntCard } from 'antd';
import type { CardProps as AntCardProps } from 'antd';

/**
 * Card - 兼容 shadcn 风格的 Card 组件
 * 基于 Ant Design Card 实现，提供 CardHeader/CardTitle/CardDescription/CardContent/CardFooter 子组件
 *
 * 说明：shadcn 风格的 CardHeader/Title/Description/Content/Footer 是无样式的 div，
 * 这里采用包装 div 策略，保留原页面的 className 排版。
 */

const baseHeaderClass = 'flex flex-col space-y-1.5 p-6';
const baseTitleClass = 'text-xl font-semibold leading-none tracking-tight';
const baseDescriptionClass = 'text-sm text-gray-500';
const baseContentClass = 'p-6 pt-0';
const baseFooterClass = 'flex items-center p-6 pt-0';

export interface CardProps extends AntCardProps {
  className?: string;
}

export const Card: React.FC<CardProps> = ({ className, children, ...rest }) => (
  <AntCard {...rest} className={className}>
    {children}
  </AntCard>
);

export interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
}

export const CardHeader: React.FC<CardHeaderProps> = ({ className, children, ...rest }) => (
  <div {...rest} className={className ? `${baseHeaderClass} ${className}` : baseHeaderClass}>
    {children}
  </div>
);

export interface CardTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {
  className?: string;
}

export const CardTitle: React.FC<CardTitleProps> = ({ className, children, ...rest }) => (
  <h3 {...rest} className={className ? `${baseTitleClass} ${className}` : baseTitleClass}>
    {children}
  </h3>
);

export interface CardDescriptionProps extends React.HTMLAttributes<HTMLParagraphElement> {
  className?: string;
}

export const CardDescription: React.FC<CardDescriptionProps> = ({ className, children, ...rest }) => (
  <p {...rest} className={className ? `${baseDescriptionClass} ${className}` : baseDescriptionClass}>
    {children}
  </p>
);

export interface CardContentProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
}

export const CardContent: React.FC<CardContentProps> = ({ className, children, ...rest }) => (
  <div {...rest} className={className ? `${baseContentClass} ${className}` : baseContentClass}>
    {children}
  </div>
);

export interface CardFooterProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
}

export const CardFooter: React.FC<CardFooterProps> = ({ className, children, ...rest }) => (
  <div {...rest} className={className ? `${baseFooterClass} ${className}` : baseFooterClass}>
    {children}
  </div>
);

export default Card;