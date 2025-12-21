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
          style={{
            width: 32,
            height: 32,
            borderRadius: 6,
            backgroundColor: `${iconColor}15`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: iconColor,
          }}
        >
          {icon}
        </div>
      )}
      <div>
        <div style={{ fontWeight: 600, fontSize: 16 }}>{title}</div>
        {subtitle && (
          <Text type='secondary' style={{ fontSize: 12 }}>
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
          style={{ fontSize: 14, fontWeight: 600 }}
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
      {...cardProps}
      style={{
        height: '100%',
        ...cardProps.style,
      }}
    >
      {children}
    </Card>
  );
};
