'use client';

import React from 'react';
import { Column } from '@ant-design/charts';
import { Empty, Typography } from 'antd';
import { Users } from 'lucide-react';
import type { TeamWorkload } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const TeamWorkloadChart: React.FC<{ data: TeamWorkload[] }> = React.memo(({ data }) => {
  const config = {
    data: data.map(item => ({
      assignee: item.assignee,
      ticketCount: item.ticketCount,
      completionRate: item.completionRate,
    })),
    xField: 'ticketCount',
    yField: 'assignee',
    seriesField: 'assignee',
    color: ['#3b82f6', '#10b981', '#f59e0b', '#ec4899', '#8b5cf6'],
    barStyle: {
      radius: [0, 8, 8, 0],
    },
    label: {
      position: 'right' as const,
      style: {
        fill: '#666',
        fontSize: 12,
      },
      formatter: (datum: { ticketCount: number }) => `${datum.ticketCount} items`,
    },
    tooltip: {
      formatter: (datum: { ticketCount: number; completionRate: number }) => ({
        name: "Number of work orders",
        value: `${datum.ticketCount}(Completion rate: ${datum.completionRate}%)`,
      }),
    },
    legend: false,
    responsive: true,
  };

  const totalTickets = data.reduce((sum, item) => sum + item.ticketCount, 0);
  // 空数据时 avgCompletion 为 NaN（0/0），2026-10-14 修复：显式守卫
  const avgCompletion =
    data.length > 0
      ? data.reduce((sum, item) => sum + item.completionRate, 0) / data.length
      : 0;

  return (
    <DashboardChartCard
      title="Team Workload"
      subtitle="Processing of employee orders for each employee"
      icon={<Users style={{ width: 20, height: 20 }} />}
      iconColor="#3b82f6"
      extra={
        <div style={{ textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>Average completion rate</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: '#1890ff' }}>
            {avgCompletion.toFixed(1)}%
          </div>
        </div>
      }
    >
      <div style={{ height: '280px' }}>
        {data.length > 0 ? (
          <Column {...config} />
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="No team workload data yet"
            style={{ paddingTop: 60 }}
          />
        )}
      </div>
    </DashboardChartCard>
  );
});

TeamWorkloadChart.displayName = 'TeamWorkloadChart';

export default TeamWorkloadChart;
