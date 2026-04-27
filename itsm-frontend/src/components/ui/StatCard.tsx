'use client';

import React from 'react';
import { Card, Statistic } from 'antd';
import { LucideIcon } from 'lucide-react';

export interface StatCardProps {
  title: string;
  value: number;
  icon?: React.ReactNode;
  prefix?: React.ReactNode;
  suffix?: React.ReactNode;
  color?: string;
  loading?: boolean;
}

export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  icon,
  prefix,
  suffix,
  color = '#1890ff',
  loading = false,
}) => {
  return (
    <Card
      className="rounded-lg shadow-sm"
      loading={loading}
      styles={{
        body: { padding: '20px' },
      }}
    >
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <div className="text-sm text-gray-500 mb-1">{title}</div>
          <Statistic
            value={value}
            prefix={prefix}
            suffix={suffix}
            valueStyle={{ color, fontSize: '24px', fontWeight: 'bold' }}
          />
        </div>
        {icon && (
          <div
            className="w-12 h-12 rounded-lg flex items-center justify-center"
            style={{ backgroundColor: `${color}15`, color }}
          >
            {icon}
          </div>
        )}
      </div>
    </Card>
  );
};

export default StatCard;
