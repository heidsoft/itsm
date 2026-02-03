'use client';

import React from 'react';
import { Card, Space, Typography } from 'antd';
import type { CardProps } from 'antd';

const { Text } = Typography;

/**
 * 仪表盘图表卡片 - 使用 Ant Design Card 组件
 * 这是一个简单的包装器，使用 Ant Design 原生组件
 */
export interface DashboardChartCardProps extends Omit<CardProps, 'title'> {
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  iconColor?: string;
  trend?: { value: number; isPositive: boolean };
  children: React.ReactNode;
}

export const DashboardChartCard: React.FC<DashboardChartCardProps> = ({
  title,
  subtitle,
  icon,
  iconColor = '#3b82f6',
  trend,
  extra,
  children,
  ...cardProps
}) => {
  // 构建标题内容
  const titleContent = (
    <Space size='small' align='start'>
      {icon && (
        <div
          className="w-8 h-8 rounded-md flex items-center justify-center"
          style={{
            backgroundColor: `${iconColor}15`,
            color: iconColor,
          }}
        >
          {icon}
        </div>
      )}
      <div>
        <div className="font-semibold text-base">{title}</div>
        {subtitle && (
          <Text type='secondary' className="text-xs">
            {subtitle}
          </Text>
        )}
      </div>
    </Space>
  );

  // 构建 extra 内容
  const extraContent = (
    <Space>
      {trend && (
        <Text
          type={trend.isPositive ? 'success' : 'danger'}
          className="text-sm font-semibold"
        >
          {trend.isPositive ? '+' : ''}
          {trend.value.toFixed(1)}%
        </Text>
      )}
      {extra}
    </Space>
  );

  return (
    <Card
      title={titleContent}
      extra={extraContent}
      className="h-full rounded-lg shadow-sm border border-gray-200"
      variant="borderless"
      {...cardProps}
    >
      {children}
    </Card>
  );
};
