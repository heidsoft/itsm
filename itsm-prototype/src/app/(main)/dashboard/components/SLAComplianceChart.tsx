'use client';

import React from 'react';
import { Progress, Tag } from 'antd';
import { Gauge } from '@ant-design/charts';
import { GaugeIcon } from 'lucide-react';
import { SLAData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const SLAComplianceChart: React.FC<{ data: SLAData[] }> = React.memo(({ data }) => {
  const averageSLA =
    data.length > 0 ? data.reduce((sum, item) => sum + (item.actual ?? 0), 0) / data.length : 0;

  const safePercent = isNaN(averageSLA / 100) ? 0 : averageSLA / 100;

  const config = {
    percent: safePercent,
    range: {
      ticks: [0, 0.25, 0.5, 0.75, 1],
      color: ['#ef4444', '#f59e0b', '#fbbf24', '#a3e635', '#22c55e'],
    },
    indicator: {
      pointer: {
        style: {
          stroke: '#8b5cf6',
          lineWidth: 3,
        },
      },
      pin: {
        style: {
          stroke: '#8b5cf6',
          fill: '#8b5cf6',
          r: 8,
        },
      },
    },
    statistic: {
      content: {
        offsetY: 10,
        style: {
          fontSize: '40px', // Keeping 40px for visual prominence
          fontWeight: 'bold',
          fill: '#8b5cf6', // Consistent with iconColor (designSystem.colors.secondary[500])
        },
        formatter: () => {
          const value = averageSLA;
          if (typeof value === 'number' && !isNaN(value)) {
            return `${value.toFixed(1)}%`;
          }
          return 'N/A';
        },
      },
      title: {
        offsetY: -15,
        style: {
          fontSize: '14px', // antdTheme.token.fontSize
          fill: '#64748b', // antdTheme.token.colorTextSecondary
          fontWeight: 600,
        },
        formatter: () => 'SLA 达成率',
      },
    },
    animation: {
      appear: {
        animation: 'fade-in',
        duration: 1200,
      },
    },
  };

  const getSLAStatus = (value: number) => {
    if (value >= 95) return { text: '优秀', color: 'success' };
    if (value >= 90) return { text: '良好', color: 'processing' };
    if (value >= 85) return { text: '一般', color: 'warning' };
    return { text: '需改进', color: 'error' };
  };

  const status = getSLAStatus(averageSLA);

  return (
    <DashboardChartCard
      title='SLA 达成率监控'
      subtitle='服务水平协议整体达成情况'
      icon={<GaugeIcon style={{ width: 20, height: 20 }} />}
      iconColor='#8b5cf6'
      extra={<Tag color={status.color as any}>{status.text}</Tag>}
    >
      <div style={{ height: '280px' }}>
        <Gauge {...config} />
      </div>

      {/* SLA服务详情 */}
      {data.length > 0 && (
        <div className='mt-4 space-y-3'>
          {data.map((item, index) => (
            <div key={index} className='flex items-center justify-between'>
              <span className='text-sm text-gray-600 font-medium'>{item.service}</span>
              <div className='flex items-center gap-3'>
                <Progress
                  percent={item.actual}
                  size='small'
                  strokeColor={item.actual >= item.target ? '#22c55e' : '#f59e0b'}
                  style={{ width: 120 }}
                  format={percent => `${percent}%`}
                />
                <span className='text-xs text-gray-500 w-12 text-right'>目标{item.target}%</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </DashboardChartCard>
  );
});

SLAComplianceChart.displayName = 'SLAComplianceChart';

export default SLAComplianceChart;
