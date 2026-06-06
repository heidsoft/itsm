'use client';

import React from 'react';
import { Card, Col, Row, Statistic } from 'antd';

export interface StatsOverviewItem {
  key: string;
  title: React.ReactNode;
  value: number | string;
  prefix?: React.ReactNode;
  suffix?: React.ReactNode;
  accentColor?: string;
  helper?: React.ReactNode;
}

interface StatsOverviewProps {
  items: StatsOverviewItem[];
  columns?: {
    xs?: number;
    sm?: number;
    lg?: number;
  };
  className?: string;
}

export function StatsOverview({
  items,
  columns = { xs: 24, sm: 12, lg: 6 },
  className,
}: StatsOverviewProps) {
  return (
    <Row gutter={[16, 16]} className={className}>
      {items.map(item => (
        <Col key={item.key} xs={columns.xs} sm={columns.sm} lg={columns.lg}>
          <Card className="h-full rounded-xl shadow-sm">
            <Statistic
              title={item.title}
              value={item.value}
              prefix={item.prefix}
              suffix={item.suffix}
              styles={item.accentColor ? { content: { color: item.accentColor } } : undefined}
            />
            {item.helper && <div className="mt-2 text-xs text-slate-500">{item.helper}</div>}
          </Card>
        </Col>
      ))}
    </Row>
  );
}

