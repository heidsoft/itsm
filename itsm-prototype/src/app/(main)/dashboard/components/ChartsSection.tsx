'use client';

import React from 'react';
import { Card, Row, Col, Spin, Empty, Badge, Progress, Tag } from 'antd';
import { Line, Pie, Gauge, Column } from '@ant-design/charts';
import {
  LineChartOutlined,
  PieChartOutlined,
  DashboardOutlined,
  RiseOutlined,
  FundOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons';
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
  showTitle?: boolean;
}

// 企业级图表卡片包装器
const EnterpriseChartCard: React.FC<{
  title: string;
  subtitle?: string;
  icon: React.ReactNode;
  iconColor: string;
  iconBgGradient: string;
  extra?: React.ReactNode;
  trend?: { value: number; isPositive: boolean };
  children: React.ReactNode;
  height?: string;
}> = ({
  title,
  subtitle,
  icon,
  iconColor,
  iconBgGradient,
  extra,
  trend,
  children,
  height = '420px',
}) => {
  return (
    <Card
      className='enterprise-chart-card h-full border-0 hover:shadow-2xl transition-all duration-300 overflow-hidden group'
      style={{
        borderRadius: '12px',
        background: '#ffffff',
        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.08)',
      }}
      styles={{
        body: { padding: 0, minHeight: height },
      }}
    >
      {/* 卡片头部 */}
      <div
        className='px-6 py-4 border-b border-gray-100 bg-gradient-to-r from-gray-50/50 to-transparent'
        style={{
          backdropFilter: 'blur(10px)',
        }}
      >
        <div className='flex items-center justify-between'>
          <div className='flex items-center gap-3'>
            <div
              className='w-10 h-10 rounded-xl flex items-center justify-center shadow-md transition-transform duration-300 group-hover:scale-110 group-hover:rotate-3'
              style={{
                background: iconBgGradient,
                boxShadow: `0 4px 12px ${iconColor}30`,
              }}
            >
              <div style={{ color: '#ffffff' }}>{icon}</div>
            </div>
            <div>
              <h3 className='text-base font-bold text-gray-900 mb-0.5'>{title}</h3>
              {subtitle && <p className='text-xs text-gray-500 m-0'>{subtitle}</p>}
            </div>
          </div>
          <div className='flex items-center gap-2'>
            {trend && (
              <div
                className={`flex items-center gap-1 px-2.5 py-1 rounded-lg ${
                  trend.isPositive
                    ? 'bg-green-50 border border-green-200'
                    : 'bg-red-50 border border-red-200'
                }`}
              >
                {trend.isPositive ? (
                  <ArrowUpOutlined style={{ fontSize: 14, color: '#16a34a' }} />
                ) : (
                  <ArrowDownOutlined style={{ fontSize: 14, color: '#dc2626' }} />
                )}
                <span
                  className={`text-xs font-bold ${
                    trend.isPositive ? 'text-green-600' : 'text-red-600'
                  }`}
                >
                  {Math.abs(trend.value).toFixed(1)}%
                </span>
              </div>
            )}
            {extra}
          </div>
        </div>
      </div>

      {/* 图表内容 */}
      <div className='p-6'>{children}</div>
    </Card>
  );
};

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
    },
    responsive: true,
  };

  const totalTickets = data.reduce(
    (sum, item) => sum + item.open + item.inProgress + item.resolved + item.closed,
    0
  );
  const trend = data.length > 1 ? (data[data.length - 1].resolved / data[0].resolved - 1) * 100 : 0;

  return (
    <EnterpriseChartCard
      title='工单趋势分析'
      subtitle='过去7天工单状态变化趋势'
      icon={<LineChartOutlined style={{ fontSize: 20 }} />}
      iconColor='#3b82f6'
      iconBgGradient='linear-gradient(135deg, #3b82f6 0%, #2563eb 100%)'
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
    </EnterpriseChartCard>
  );
});

TicketTrendChart.displayName = 'TicketTrendChart';

// 事件分布图
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
          const percent = ((datum.value / totalCount) * 100).toFixed(1);
          return `${datum.type}\n${percent}%`;
        },
        style: {
          fontSize: 12,
          fontWeight: 600,
        },
      },
      statistic: {
        title: {
          offsetY: -8,
          style: {
            fontSize: '14px',
            color: '#666',
          },
          formatter: () => '总事件数',
        },
        content: {
          offsetY: 8,
          style: {
            fontSize: '32px',
            fontWeight: 'bold',
            color: '#10b981',
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
      <EnterpriseChartCard
        title='事件分类分布'
        subtitle='按事件类型统计分析'
        icon={<PieChartOutlined style={{ fontSize: 20 }} />}
        iconColor='#10b981'
        iconBgGradient='linear-gradient(135deg, #10b981 0%, #059669 100%)'
        extra={
          <Tag color='success' className='font-semibold'>
            {topCategory.category}占比最高
          </Tag>
        }
      >
        <div style={{ height: '320px' }}>
          <Pie {...config} />
        </div>
      </EnterpriseChartCard>
    );
  }
);

IncidentDistributionChart.displayName = 'IncidentDistributionChart';

// SLA达成率仪表盘
const SLAComplianceChart: React.FC<{ data: SLAData[] }> = React.memo(({ data }) => {
  const averageSLA =
    data.length > 0 ? data.reduce((sum, item) => sum + item.actual, 0) / data.length : 0;

  const config = {
    percent: averageSLA / 100,
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
          fontSize: '40px',
          fontWeight: 'bold',
          color: '#8b5cf6',
        },
        formatter: () => `${averageSLA.toFixed(1)}%`,
      },
      title: {
        offsetY: -15,
        style: {
          fontSize: '16px',
          color: '#666',
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
    <EnterpriseChartCard
      title='SLA 达成率监控'
      subtitle='服务水平协议整体达成情况'
      icon={<DashboardOutlined style={{ fontSize: 20 }} />}
      iconColor='#8b5cf6'
      iconBgGradient='linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%)'
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
    </EnterpriseChartCard>
  );
});

SLAComplianceChart.displayName = 'SLAComplianceChart';

// 用户满意度柱状图
const SatisfactionChart: React.FC<{ data: SatisfactionData[] }> = React.memo(({ data }) => {
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
        alias: '满意度评分',
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
      formatter: (datum: any) => `${datum.rating}`,
    },
    animation: {
      appear: {
        animation: 'scale-in-y',
        duration: 1200,
      },
    },
    tooltip: {
      formatter: (datum: any) => ({
        name: '满意度评分',
        value: `${datum.rating}/5.0 (${datum.responses}份反馈)`,
      }),
    },
    responsive: true,
  };

  const avgRating =
    data.length > 0 ? data.reduce((sum, item) => sum + item.rating, 0) / data.length : 0;
  const totalResponses = data.reduce((sum, item) => sum + item.responses, 0);
  const trend = data.length > 1 ? (data[data.length - 1].rating / data[0].rating - 1) * 100 : 0;

  return (
    <EnterpriseChartCard
      title='用户满意度趋势'
      subtitle='月度服务满意度评分变化'
      icon={<RiseOutlined style={{ fontSize: 20 }} />}
      iconColor='#ec4899'
      iconBgGradient='linear-gradient(135deg, #ec4899 0%, #db2777 100%)'
      trend={{ value: trend, isPositive: trend > 0 }}
      extra={
        <div className='text-right'>
          <div className='text-xs text-gray-500'>平均评分</div>
          <div className='text-base font-bold text-pink-600'>{avgRating.toFixed(1)}/5.0</div>
        </div>
      }
    >
      <div style={{ height: '280px' }}>
        <Column {...config} />
      </div>

      {/* 满意度统计摘要 */}
      <div className='mt-4 pt-4 border-t border-gray-100'>
        <div className='grid grid-cols-3 gap-4'>
          <div className='text-center'>
            <div className='text-xs text-gray-500 mb-1'>总反馈数</div>
            <div className='text-lg font-bold text-gray-900'>{totalResponses}</div>
          </div>
          <div className='text-center'>
            <div className='text-xs text-gray-500 mb-1'>最高评分</div>
            <div className='text-lg font-bold text-pink-600'>
              {Math.max(...data.map(d => d.rating)).toFixed(1)}
            </div>
          </div>
          <div className='text-center'>
            <div className='text-xs text-gray-500 mb-1'>最低评分</div>
            <div className='text-lg font-bold text-orange-600'>
              {Math.min(...data.map(d => d.rating)).toFixed(1)}
            </div>
          </div>
        </div>
      </div>
    </EnterpriseChartCard>
  );
});

SatisfactionChart.displayName = 'SatisfactionChart';

// 图表区域主组件
export const ChartsSection: React.FC<ChartsSectionProps> = React.memo(
  ({
    ticketTrend,
    incidentDistribution,
    slaData,
    satisfactionData,
    loading = false,
    showTitle = true,
  }) => {
    if (loading) {
      return (
        <div className='mb-6'>
          <Row gutter={[16, 16]}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Col key={index} xs={24} lg={12}>
                <Card
                  className='border-0'
                  style={{
                    borderRadius: '12px',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
                    minHeight: '420px',
                  }}
                >
                  <div className='flex flex-col items-center justify-center h-full min-h-[400px]'>
                    <Spin size='large' />
                    <p className='text-sm text-gray-400 mt-4'>加载图表数据...</p>
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
          <Card
            className='text-center py-16 border-0'
            style={{
              borderRadius: '12px',
              background: 'linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%)',
              boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
            }}
          >
            <div className='w-20 h-20 rounded-2xl bg-gradient-to-br from-gray-100 to-gray-200 flex items-center justify-center mx-auto mb-5'>
              <FundOutlined style={{ fontSize: 40, color: '#9ca3af' }} />
            </div>
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div>
                  <p className='text-base font-semibold text-gray-700 mb-1'>暂无图表数据</p>
                  <p className='text-sm text-gray-500'>系统正在收集和分析数据，请稍后查看</p>
                </div>
              }
            />
          </Card>
        </div>
      );
    }

    return (
      <div className={showTitle ? 'mb-6' : ''}>
        {showTitle && (
          <div className='mb-5 flex items-center justify-between'>
            <div>
              <h2 className='text-xl font-bold text-gray-900 mb-1.5 flex items-center gap-2'>
                <div className='w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-md'>
                  <FundOutlined style={{ fontSize: 18, color: '#ffffff' }} />
                </div>
                数据分析与趋势
              </h2>
              <p className='text-sm text-gray-600'>系统性能和业务趋势的深度洞察</p>
            </div>
            <Badge
              status='processing'
              text='实时更新'
              className='text-sm'
              style={{ fontWeight: 500 }}
            />
          </div>
        )}

        <Row gutter={[16, 16]}>
          {ticketTrend.length > 0 && (
            <Col xs={24} lg={12}>
              <TicketTrendChart data={ticketTrend} />
            </Col>
          )}
          {incidentDistribution.length > 0 && (
            <Col xs={24} lg={12}>
              <IncidentDistributionChart data={incidentDistribution} />
            </Col>
          )}
          {slaData.length > 0 && (
            <Col xs={24} lg={12}>
              <SLAComplianceChart data={slaData} />
            </Col>
          )}
          {satisfactionData.length > 0 && (
            <Col xs={24} lg={12}>
              <SatisfactionChart data={satisfactionData} />
            </Col>
          )}
        </Row>
      </div>
    );
  }
);

ChartsSection.displayName = 'ChartsSection';
