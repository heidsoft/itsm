'use client';

import React from 'react';
import { Tag } from 'antd';
import { Pie } from '@ant-design/charts';
import { PieChart } from 'lucide-react';
import { IncidentDistributionData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const IncidentDistributionChart: React.FC<{ data: IncidentDistributionData[] }> = React.memo(
  ({ data }) => {
    // 确保数据有效性
    const validData = data.filter(
      item => item && typeof item.count === 'number' && !isNaN(item.count) && item.count >= 0
    );
    const totalCount = validData.reduce((sum, item) => sum + (item.count || 0), 0);

    const config = {
      data: validData.map(item => ({
        type: item.category || '未知',
        value: item.count || 0,
      })),
      angleField: 'value',
      colorField: 'type',
      radius: 0.85,
      innerRadius: 0.6,
      label: false, // 禁用label，使用legend和tooltip显示信息，避免兼容性问题
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
      color: validData.map(item => item.color || '#6b7280'),
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

    const topCategory =
      validData.length > 0
        ? validData.reduce((prev, current) =>
            (prev.count || 0) > (current.count || 0) ? prev : current
          )
        : { category: '无数据', count: 0 };

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
