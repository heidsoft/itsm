'use client';

import React from 'react';
import { Tag } from 'antd';
import { Column } from '@ant-design/charts';
import { TrendingUp } from 'lucide-react';
import { DashboardChartCard } from './DashboardChartCard';
import type { PeakHourData } from '../types/dashboard.types';

const PeakHoursChart: React.FC<{ data: PeakHourData[] }> = React.memo(({ data }) => {
  // 确保数据有效性
  const validData = data.filter(
    item => item && item.hour !== undefined && typeof item.count === 'number'
  );

  // 如果没有有效数据，显示空状态
  if (validData.length === 0) {
    return (
      <DashboardChartCard
        title="Peak hour analysis"
        subtitle="24-hour work order creation distribution"
        icon={<TrendingUp style={{ width: 20, height: 20 }} />}
        iconColor="#06b6d4"
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            height: '280px',
            color: '#999',
          }}
        >

          No data
        </div>
      </DashboardChartCard>
    );
  }

  const config = {
    data: validData.map(item => ({
      hour: `${item.hour}:00`,
      count: item.count,
    })),
    xField: 'hour',
    yField: 'count',
    color: '#06b6d4',
    columnStyle: {
      radius: [8, 8, 0, 0],
    },
    smooth: true,
    label: false,
    tooltip: {
      formatter: (datum: any) => ({
        name: "Number of work orders",
        value: `${datum.count} items`,
      }),
    },
    responsive: true,
  };

  const peakHour = validData.reduce(
    (max, item) => (item.count > max.count ? item : max),
    validData[0]
  );
  const totalCount = validData.reduce((sum, item) => sum + item.count, 0);

  return (
    <DashboardChartCard
      title="Peak hour analysis"
      subtitle="24-hour work order creation distribution"
      icon={<TrendingUp style={{ width: 20, height: 20 }} />}
      iconColor="#06b6d4"
      extra={<Tag color="cyan">Peak hours: {peakHour.hour}:00</Tag>}
    >
      <div style={{ height: '280px' }}>
        <Column {...config} />
      </div>
    </DashboardChartCard>
  );
});

PeakHoursChart.displayName = 'PeakHoursChart';

export default PeakHoursChart;
