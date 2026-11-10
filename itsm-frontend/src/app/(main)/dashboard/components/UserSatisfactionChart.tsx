'use client';

import React from 'react';
import { Column } from '@ant-design/charts';
import { Smile } from 'lucide-react';
import { DashboardChartCard } from './DashboardChartCard';

interface SatisfactionData {
  month: string;
  rating: number;
  responses: number;
}

const UserSatisfactionChart: React.FC<{ data: SatisfactionData[] }> = React.memo(({ data }) => {
  const config = {
    data: data.map(item => ({
      month: item.month,
      rating: item.rating,
      responses: item.responses,
    })),
    xField: 'month',
    yField: 'rating',
    meta: {
      rating: {
        alias: 'Satisfaction Score',
        min: 0,
        max: 5,
      },
    },
    color: '#ec4899',
    columnStyle: {
      radius: [8, 8, 0, 0],
      fill: 'l(270) 0:#ec4899 1:#db2777',
    },
    columnWidthRatio: 0.5,
    label: {
      position: 'top' as const,
      style: {
        fill: '#ec4899',
        fontSize: 13,
        fontWeight: 'bold',
      },
      formatter: (datum: { rating: number }) => `${datum.rating}`,
    },
    animation: {
      appear: {
        duration: 1200,
      },
    },
    tooltip: {
      formatter: (datum: { rating: number; responses: number }) => ({
        name: "Satisfaction Score",
        value: `${datum.rating}/5.0 (${datum.responses} feedback)`,
      }),
    },
    responsive: true,
  };

  const avgRating =
    data.length > 0 ? data.reduce((sum, item) => sum + item.rating, 0) / data.length : 0;
  const totalResponses = data.reduce((sum, item) => sum + item.responses, 0);
  const trend = data.length > 1 ? (data[data.length - 1].rating / data[0].rating - 1) * 100 : 0;
  const maxRating = data.length > 0 ? Math.max(...data.map(d => d.rating)) : 0;
  const minRating = data.length > 0 ? Math.min(...data.map(d => d.rating)) : 0;

  return (
    <DashboardChartCard
      title="User satisfaction trends"
      subtitle="Changes in monthly service satisfaction scores"
      icon={<Smile style={{ width: 20, height: 20 }} />}
      iconColor="#ec4899"
      trend={{ value: trend, isPositive: trend > 0 }}
      extra={
        <div style={{ textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>average rating</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: '#eb2f96' }}>
            {avgRating.toFixed(1)}/5.0
          </div>
        </div>
      }
    >
      <div style={{ height: '280px' }}>
        <Column {...config} />
      </div>

      {/* 满意度统计摘要 */}
      <div className="mt-4 pt-4 border-t border-gray-100">
        <div className="grid grid-cols-3 gap-4">
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-1">Total number of feedbacks</div>
            <div className="text-lg font-bold text-gray-900">{totalResponses}</div>
          </div>
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-1">Highest Rated</div>
            <div className="text-lg font-bold text-pink-600">
              {data.length > 0 ? maxRating.toFixed(1) : '-'}
            </div>
          </div>
          <div className="text-center">
            <div className="text-xs text-gray-500 mb-1">lowest rating</div>
            <div className="text-lg font-bold text-orange-600">
              {data.length > 0 ? minRating.toFixed(1) : '-'}
            </div>
          </div>
        </div>
      </div>
    </DashboardChartCard>
  );
});

UserSatisfactionChart.displayName = 'UserSatisfactionChart';

export default UserSatisfactionChart;
