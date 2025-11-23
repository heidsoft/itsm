'use client';

import React from 'react';
import { Tag } from 'antd';
import { Column } from '@ant-design/charts';
import { TrendingUp } from 'lucide-react';
import { DashboardChartCard } from './DashboardChartCard';

const PeakHoursChart: React.FC<{ data: PeakHourData[] }> = React.memo(({ data }) => {
  const config = {
    data: data.map(item => ({
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
        name: '工单数',
        value: `${datum.count}个`,
      }),
    },
    responsive: true,
  };

  const peakHour = data.reduce((max, item) => (item.count > max.count ? item : max), data[0]);
  const totalCount = data.reduce((sum, item) => sum + item.count, 0);

  return (
    <DashboardChartCard
      title='高峰时段分析'
      subtitle='24小时工单创建分布'
      icon={<TrendingUp style={{ width: 20, height: 20 }} />}
      iconColor='#06b6d4'
      extra={
        <Tag color='cyan'>
          高峰时段: {peakHour.hour}:00
        </Tag>
      }
    >
      <div style={{ height: '280px' }}>
        <Column {...config} />
      </div>
          </DashboardChartCard>  );
});

PeakHoursChart.displayName = 'PeakHoursChart';

export default PeakHoursChart;
