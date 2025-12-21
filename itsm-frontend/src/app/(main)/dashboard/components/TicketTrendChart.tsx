'use client';

import React from 'react';
import { Badge } from 'antd';
import { Line } from '@ant-design/charts';
import { LineChart } from 'lucide-react'; // Import Lucide LineChart
import { TicketTrendData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard'; // Import from new file

// 工单趋势图
const TicketTrendChart: React.FC<{ data: TicketTrendData[] }> = React.memo(({ data }) => {
  const config = {
    data: data.flatMap(item => [
      { date: item.date, type: '待处理', value: item.open },
      { date: item.date, type: '处理中', value: item.inProgress },
      { date: item.date, type: '已解决', value: item.resolved },
      { date: item.date, type: '已关闭', value: item.closed },
    ]),
    xField: 'date',
    yField: 'value',
    seriesField: 'type',
    smooth: true,
    animation: {
      appear: {
        animation: 'path-in',
        duration: 1200,
      },
    },
    color: ['#3b82f6', '#f59e0b', '#10b981', '#8b5cf6'],
    lineStyle: {
      lineWidth: 3,
    },
    point: {
      size: 5,
      shape: 'circle',
      style: {
        fill: '#fff',
        stroke: '#000',
        lineWidth: 2,
      },
    },
    legend: {
      position: 'top' as const,
      offsetY: -5,
    },
    tooltip: {
      shared: true,
      showCrosshairs: true,
      customItems: (originalItems: any[]) => {
        return originalItems.map(item => {
          return {
            ...item,
            value: `${item.value} 个`,
          };
        });
      },
    },
    slider: {
      start: 0,
      end: 1,
    },
    scrollbar: {
      type: 'horizontal',
    },
    responsive: true,
  };

  const totalTickets = data.reduce(
    (sum, item) => sum + item.open + item.inProgress + item.resolved + item.closed,
    0
  );
  const trend = data.length > 1 ? (data[data.length - 1].resolved / data[0].resolved - 1) * 100 : 0;

  return (
    <DashboardChartCard
      title='工单趋势分析'
      subtitle='过去7天工单状态变化趋势'
      icon={<LineChart style={{ width: 20, height: 20 }} />}
      iconColor='#3b82f6'
      trend={{ value: trend, isPositive: trend > 0 }}
      extra={
        <Badge
          count={totalTickets}
          showZero
          style={{
            backgroundColor: '#3b82f6',
            boxShadow: '0 2px 8px rgba(59, 130, 246, 0.3)',
          }}
        />
      }
    >
      <div style={{ height: '320px' }}>
        <Line {...config} />
      </div>
    </DashboardChartCard>
  );
});

TicketTrendChart.displayName = 'TicketTrendChart';

export default TicketTrendChart;
