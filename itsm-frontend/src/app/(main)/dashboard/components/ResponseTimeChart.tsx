'use client';

import React from 'react';
import { Column } from '@ant-design/charts';
import { LineChart } from 'lucide-react';
import type { ResponseTimeDistribution } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const ResponseTimeChart: React.FC<{ data: ResponseTimeDistribution[] }> = React.memo(({ data }) => {
  const totalTickets = data.reduce((sum, item) => sum + (item.count || 0), 0);

  // 百分比必须在本地兜底算一次：后端字段缺失/为 null 时，标签 formatter 会直接
  // 把 undefined 拼进模板，坐标轴上出现「undefined%」而柱子高度为 0，
  // 结果是一张既没数据也读不懂的空图（2026-10-25 浏览器实测）。
  const pct = (item: ResponseTimeDistribution): number => {
    if (typeof item.percentage === 'number' && Number.isFinite(item.percentage)) {
      return item.percentage;
    }
    return totalTickets > 0 ? Math.round(((item.count || 0) / totalTickets) * 1000) / 10 : 0;
  };

  const config = {
    data: data.map(item => ({
      range: item.timeRange,
      count: item.count || 0,
      percentage: pct(item),
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
      // 零值桶没有柱子，标签会落在坐标轴基线上与刻度重叠；直接不画标签。
      formatter: (datum: { count?: number; percentage?: number }) =>
        datum?.count ? `${datum.percentage ?? 0}%` : '',
    },
    tooltip: {
      formatter: (datum: { count?: number; percentage?: number }) => ({
        name: '工单数',
        value: `${datum?.count ?? 0}个 (${datum?.percentage ?? 0}%)`,
      }),
    },
    responsive: true,
  };

  // 计算平均响应时间，避免除以零
  const avgTime = totalTickets > 0
    ? data.reduce((sum, item) => sum + (item.avgTime || 0) * item.count, 0) / totalTickets
    : 0;

  return (
    <DashboardChartCard
      title="响应时间分布"
      subtitle="工单首次响应时间统计"
      icon={<LineChart style={{ width: 20, height: 20 }} />}
      iconColor="#8b5cf6"
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
        {/* 全部桶为 0 时渲染一张只有坐标轴的空图，比「暂无数据」更容易被误读成图表坏了 */}
        {totalTickets === 0 ? (
          <div
            style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#8c8c8c',
              fontSize: 13,
            }}
          >
            暂无响应时间数据
          </div>
        ) : (
          <Column {...config} />
        )}
      </div>
    </DashboardChartCard>
  );
});

ResponseTimeChart.displayName = 'ResponseTimeChart';

export default ResponseTimeChart;
