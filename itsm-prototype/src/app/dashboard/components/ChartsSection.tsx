'use client';

import React from 'react';
import { Card, Row, Col, Spin, Empty, Tooltip } from 'antd';
import { Line, Pie, Gauge, Column } from '@ant-design/charts';
import { BarChart3, PieChart, Gauge as GaugeIcon, TrendingUp } from 'lucide-react';
import {
  TicketTrendData,
  IncidentDistributionData,
  SLAData,
  SatisfactionData,
} from '../types/dashboard.types';

interface ChartsSectionProps {
  ticketTrend: TicketTrendData[];
  incidentDistribution: IncidentDistributionData[];
  slaData: SLAData[];
  satisfactionData: SatisfactionData[];
  loading?: boolean;
}

// 工单趋势图组件
const TicketTrendChart: React.FC<{ data: TicketTrendData[] }> = React.memo(({ data }) => {
  const config = {
    data: data.flatMap(item => [
      { date: item.date, type: 'Open', value: item.open },
      { date: item.date, type: 'In Progress', value: item.inProgress },
      { date: item.date, type: 'Resolved', value: item.resolved },
      { date: item.date, type: 'Closed', value: item.closed },
    ]),
    xField: 'date',
    yField: 'value',
    seriesField: 'type',
    smooth: true,
    animation: {
      appear: {
        animation: 'path-in',
        duration: 1000,
      },
    },
    color: ['#1890ff', '#fa8c16', '#52c41a', '#722ed1'],
    legend: {
      position: 'top' as const,
    },
    tooltip: {
      shared: true,
      showCrosshairs: true,
    },
    responsive: true,
  };

  return (
    <Card
      title={
        <div className='flex items-center gap-2'>
          <BarChart3 size={18} className='text-blue-500' />
          <span>工单趋势</span>
        </div>
      }
      className='h-full'
      extra={
        <Tooltip title='过去7天工单状态分布'>
          <span className='text-xs text-gray-500'>7天</span>
        </Tooltip>
      }
    >
      <div style={{ height: '300px' }}>
        <Line {...config} />
      </div>
    </Card>
  );
});

TicketTrendChart.displayName = 'TicketTrendChart';

// 事件分布图组件
const IncidentDistributionChart: React.FC<{ data: IncidentDistributionData[] }> = React.memo(
  ({ data }) => {
    const config = {
      data: data.map(item => ({
        type: item.category,
        value: item.count,
      })),
      angleField: 'value',
      colorField: 'type',
      radius: 0.8,
      label: {
        type: 'outer',
        content: '{name} {percentage}',
      },
      color: data.map(item => item.color),
      legend: {
        position: 'bottom' as const,
      },
      animation: {
        appear: {
          animation: 'scale-in',
          duration: 1000,
        },
      },
      responsive: true,
    };

    return (
      <Card
        title={
          <div className='flex items-center gap-2'>
            <PieChart size={18} className='text-green-500' />
            <span>事件分布</span>
          </div>
        }
        className='h-full'
        extra={
          <Tooltip title='事件分类统计'>
            <span className='text-xs text-gray-500'>按分类</span>
          </Tooltip>
        }
      >
        <div style={{ height: '300px' }}>
          <Pie {...config} />
        </div>
      </Card>
    );
  }
);

IncidentDistributionChart.displayName = 'IncidentDistributionChart';

// SLA达成率仪表盘组件
const SLAComplianceChart: React.FC<{ data: SLAData[] }> = React.memo(({ data }) => {
  const averageSLA = data.reduce((sum, item) => sum + item.actual, 0) / data.length;

  const config = {
    percent: averageSLA / 100,
    range: {
      color: ['#ff4d4f', '#faad14', '#52c41a'],
    },
    indicator: {
      pointer: {
        style: {
          stroke: '#D0D0D0',
        },
      },
      pin: {
        style: {
          stroke: '#D0D0D0',
        },
      },
    },
    statistic: {
      content: {
        style: {
          whiteSpace: 'pre-wrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        },
        formatter: () => `${averageSLA.toFixed(1)}%\nSLA Compliance`,
      },
    },
    animation: {
      appear: {
        animation: 'gauge-in',
        duration: 1000,
      },
    },
  };

  return (
    <Card
      title={
        <div className='flex items-center gap-2'>
          <GaugeIcon size={18} className='text-purple-500' />
          <span>SLA达成率</span>
        </div>
      }
      className='h-full'
      extra={
        <Tooltip title='所有服务的平均SLA达成率'>
          <span className='text-xs text-gray-500'>平均值</span>
        </Tooltip>
      }
    >
      <div style={{ height: '300px' }}>
        <Gauge {...config} />
      </div>
    </Card>
  );
});

SLAComplianceChart.displayName = 'SLAComplianceChart';

// 用户满意度柱状图组件
const SatisfactionChart: React.FC<{ data: SatisfactionData[] }> = React.memo(({ data }) => {
  const config = {
    data: data.map(item => ({
      month: item.month,
      rating: item.rating,
      responses: item.responses,
    })),
    xField: 'month',
    yField: 'rating',
    seriesField: 'responses',
    color: '#1890ff',
    columnStyle: {
      radius: [4, 4, 0, 0],
    },
    animation: {
      appear: {
        animation: 'scale-in-y',
        duration: 1000,
      },
    },
    tooltip: {
      formatter: (datum: any) => ({
        name: 'Rating',
        value: `${datum.rating}/5 (${datum.responses} responses)`,
      }),
    },
    responsive: true,
  };

  return (
    <Card
      title={
        <div className='flex items-center gap-2'>
          <TrendingUp size={18} className='text-pink-500' />
          <span>用户满意度</span>
        </div>
      }
      className='h-full'
      extra={
        <Tooltip title='月度用户满意度评分'>
          <span className='text-xs text-gray-500'>月度</span>
        </Tooltip>
      }
    >
      <div style={{ height: '300px' }}>
        <Column {...config} />
      </div>
    </Card>
  );
});

SatisfactionChart.displayName = 'SatisfactionChart';

// 图表区域主组件
export const ChartsSection: React.FC<ChartsSectionProps> = React.memo(
  ({ ticketTrend, incidentDistribution, slaData, satisfactionData, loading = false }) => {
    if (loading) {
      return (
        <div className='mb-6'>
          <Row gutter={[16, 16]}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Col key={index} xs={24} lg={12}>
                <Card className='h-80'>
                  <div className='flex items-center justify-center h-full'>
                    <Spin size='large' />
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      );
    }

    const hasData =
      ticketTrend.length > 0 ||
      incidentDistribution.length > 0 ||
      slaData.length > 0 ||
      satisfactionData.length > 0;

    if (!hasData) {
      return (
        <div className='mb-6'>
          <Card className='text-center py-12'>
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无图表数据' />
          </Card>
        </div>
      );
    }

    return (
      <div className='mb-6'>
        <div className='mb-4'>
          <h2 className='text-lg font-semibold text-gray-900 mb-1'>数据分析与趋势</h2>
          <p className='text-sm text-gray-600'>系统性能和趋势的可视化洞察</p>
        </div>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <TicketTrendChart data={ticketTrend} />
          </Col>
          <Col xs={24} lg={12}>
            <IncidentDistributionChart data={incidentDistribution} />
          </Col>
          <Col xs={24} lg={12}>
            <SLAComplianceChart data={slaData} />
          </Col>
          <Col xs={24} lg={12}>
            <SatisfactionChart data={satisfactionData} />
          </Col>
        </Row>
      </div>
    );
  }
);

ChartsSection.displayName = 'ChartsSection';
