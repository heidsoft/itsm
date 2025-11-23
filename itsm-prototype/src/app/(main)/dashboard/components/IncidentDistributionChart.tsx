'use client';

import React from 'react';
import { Tag } from 'antd';
import { Pie } from '@ant-design/charts';
import { PieChart } from 'lucide-react';
import { IncidentDistributionData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const IncidentDistributionChart: React.FC<{ data: IncidentDistributionData[] }> = React.memo(
  ({ data }) => {
    const totalCount = data.reduce((sum, item) => sum + item.count, 0);

    const config = {
      data: data.map(item => ({
        type: item.category,
        value: item.count,
      })),
      angleField: 'value',
      colorField: 'type',
      radius: 0.85,
      innerRadius: 0.6,
      label: {
        type: 'spider' as const,
        formatter: (datum: any) => {
          if (!datum || typeof datum.value === 'undefined' || totalCount === 0) {
            return '';
          }
          const percent = ((datum.value / totalCount) * 100).toFixed(1);
          return `${datum.type}\n${percent}%`;
        },
        style: {
          fontSize: 12, // antdTheme.token.fontSizeSM
          fontWeight: 600,
          fill: '#1e293b', // antdTheme.token.colorText
        },
      },
      statistic: {
        title: {
          offsetY: -8,
          style: {
            fontSize: '12px', // antdTheme.token.fontSizeSM
            fill: '#64748b', // antdTheme.token.colorTextSecondary
          },
          formatter: () => '总事件数',
        },
        content: {
          offsetY: 8,
          style: {
            fontSize: '30px', // Adjusted to align with designSystem.typography.fontSize['3xl']
            fontWeight: 'bold',
            fill: '#10b981', // antdTheme.token.colorSuccess (consistent with iconColor)
          },
          formatter: () => totalCount.toString(),
        },
      },
      color: data.map(item => item.color),
      legend: {
        position: 'bottom' as const,
        offsetY: -10,
      },
      animation: {
        appear: {
          animation: 'fade-in',
          duration: 1200,
        },
      },
      interactions: [{ type: 'element-selected' }, { type: 'element-active' }],
      responsive: true,
    };

    const topCategory = data.reduce((prev, current) =>
      prev.count > current.count ? prev : current
    );

    return (
      <DashboardChartCard
        title='事件分类分布'
        subtitle='按事件类型统计分析'
        icon={<PieChart style={{ width: 20, height: 20 }} />}
        iconColor='#10b981'
        extra={
          <Tag color='success' className='font-semibold'>
            {topCategory.category}占比最高
          </Tag>
        }
      >
        <div style={{ height: '320px' }}>
          <Pie {...config} />
        </div>
      </DashboardChartCard>
    );
  }
);

IncidentDistributionChart.displayName = 'IncidentDistributionChart';

export default IncidentDistributionChart;
