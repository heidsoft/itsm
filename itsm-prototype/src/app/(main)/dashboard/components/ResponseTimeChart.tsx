'use client';

import React from 'react';
import { Column } from '@ant-design/charts';
import { LineChart } from 'lucide-react';
import { ResponseTimeDistribution } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const ResponseTimeChart: React.FC<{ data: ResponseTimeDistribution[] }> = React.memo(({ data }) => {
  const config = {
    data: data.map(item => ({
      range: item.timeRange,
      count: item.count,
      percentage: item.percentage,
    })),
    xField: 'range',
    yField: 'count',
    color: '#8b5cf6',
    columnStyle: {
      radius: [8, 8, 0, 0],
    },
    label: {
      position: 'top' as const,
      style: {
        fill: '#8b5cf6',
        fontSize: 12,
        fontWeight: 'bold',
      },
      formatter: (datum: any) => `${datum.percentage}%`,
    },
    tooltip: {
      formatter: (datum: any) => ({
        name: '工单数',
        value: `${datum.count}个 (${datum.percentage}%)`,
      }),
    },
    responsive: true,
  };

  const totalTickets = data.reduce((sum, item) => sum + item.count, 0);
  const avgTime = data.reduce((sum, item) => sum + (item.avgTime || 0) * item.count, 0) / totalTickets;

  return (
    <DashboardChartCard
      title='响应时间分布'
      subtitle='工单首次响应时间统计'
      icon={<LineChart style={{ width: 20, height: 20 }} />}
      iconColor='#8b5cf6'
      extra={
        <div style={{ textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>平均响应</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: '#722ed1' }}>
            {avgTime.toFixed(1)}小时
          </div>
        </div>
      }
    >
      <div style={{ height: '320px' }}>
        <Column {...config} />
      </div>
    </DashboardChartCard>
  );
});

ResponseTimeChart.displayName = 'ResponseTimeChart';

export default ResponseTimeChart;
