'use client';

import React from 'react';
import { Card, Statistic } from 'antd';
import { LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: number;
  icon?: LucideIcon;
  iconColor?: string;
  valueColor?: string;
  suffix?: string;
  loading?: boolean;
  className?: string;
}

/**
 * 通用统计卡片组件
 * 用于各个页面的统计展示，统一风格
 */
export const StatsCard: React.FC<StatsCardProps> = ({
  title,
  value,
  icon: Icon,
  iconColor = '#1890ff',
  valueColor,
  suffix,
  loading = false,
  className = '',
}) => {
  return (
    <Card className={`rounded-lg shadow-sm ${className}`} loading={loading}>
      <Statistic
        title={title}
        value={value}
        suffix={suffix}
        prefix={Icon ? <Icon className="mr-2" style={{ color: iconColor }} /> : undefined}
        styles={{
          content: {
            color: valueColor || iconColor,
          },
        }}
      />
    </Card>
  );
};

/**
 * 统计卡片网格布局组件
 * 用于快速创建统计卡片行
 */
interface StatsGridProps {
  stats: Array<{
    title: string;
    value: number;
    icon?: LucideIcon;
    iconColor?: string;
    valueColor?: string;
    suffix?: string;
  }>;
  loading?: boolean;
}

export const StatsGrid: React.FC<StatsGridProps> = ({
  stats,
  loading = false,
}) => {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      {stats.map((stat, index) => (
        <StatsCard
          key={index}
          title={stat.title}
          value={stat.value}
          icon={stat.icon}
          iconColor={stat.iconColor}
          valueColor={stat.valueColor}
          suffix={stat.suffix}
          loading={loading}
        />
      ))}
    </div>
  );
};

export default StatsCard;
