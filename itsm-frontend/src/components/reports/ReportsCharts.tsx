'use client';

import React from 'react';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import { Table, Empty, Spin } from 'antd';
import type { AnalyticsDataPoint } from '@/lib/api/ticket-analytics-api';

interface ReportsChartsProps {
  data: AnalyticsDataPoint[];
  loading?: boolean;
  chartType?: 'line' | 'bar' | 'pie' | 'area' | 'table';
  height?: number;
}

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2', '#eb2f96', '#fa541c'];

const ReportsCharts: React.FC<ReportsChartsProps> = ({
  data,
  loading = false,
  chartType = 'bar',
  height = 400,
}) => {
  // 表格列定义
  const tableColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: '数值',
      dataIndex: 'value',
      key: 'value',
      render: (value: number) => <span className="text-blue-600 font-semibold">{value.toLocaleString()}</span>,
    },
    {
      title: '数量',
      dataIndex: 'count',
      key: 'count',
      render: (count?: number) => count ? count.toLocaleString() : '-',
    },
    {
      title: '平均时间',
      dataIndex: 'avg_time',
      key: 'avg_time',
      render: (avgTime?: number) => avgTime ? `${avgTime.toFixed(2)}小时` : '-',
    },
  ];

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 border border-gray-200 rounded shadow-lg">
          <p className="font-medium">{`${label}`}</p>
          {payload.map((entry: any, index: number) => (
            <p key={index} style={{ color: entry.color }}>
              {`${entry.name}: ${entry.value.toLocaleString()}`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  // 饼图自定义标签
  const renderCustomLabel = (entry: any) => {
    const RADIAN = Math.PI / 180;
    const radius = entry.innerRadius + (entry.outerRadius - entry.innerRadius) * 0.5;
    const x = entry.cx + radius * Math.cos(-entry.midAngle * RADIAN);
    const y = entry.cy + radius * Math.sin(-entry.midAngle * RADIAN);

    return (
      <text 
        x={x} 
        y={y} 
        fill="white" 
        textAnchor={x > entry.cx ? 'start' : 'end'} 
        dominantBaseline="central"
        className="text-xs font-medium"
      >
        {`${entry.percent ? (entry.percent * 100).toFixed(0) : 0}%`}
      </text>
    );
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center" style={{ height }}>
        <Spin size="large" tip="加载图表数据..." />
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="flex justify-center items-center" style={{ height }}>
        <Empty description="暂无数据" />
      </div>
    );
  }

  // 渲染图表
  const renderChart = () => {
    switch (chartType) {
      case 'line':
        return (
          <ResponsiveContainer width="100%" height={height}>
            <LineChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <Line 
                type="monotone" 
                dataKey="value" 
                stroke="#1890ff" 
                strokeWidth={2}
                dot={{ fill: '#1890ff', r: 4 }}
                activeDot={{ r: 6 }}
              />
              {data.some(item => item.avg_time) && (
                <Line 
                  type="monotone" 
                  dataKey="avg_time" 
                  stroke="#52c41a" 
                  strokeWidth={2}
                  name="平均时间(小时)"
                />
              )}
            </LineChart>
          </ResponsiveContainer>
        );

      case 'area':
        return (
          <ResponsiveContainer width="100%" height={height}>
            <AreaChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <Area 
                type="monotone" 
                dataKey="value" 
                stroke="#1890ff" 
                fill="#1890ff" 
                fillOpacity={0.6}
              />
            </AreaChart>
          </ResponsiveContainer>
        );

      case 'pie':
        return (
          <ResponsiveContainer width="100%" height={height}>
            <PieChart>
              <Pie
                data={data}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={renderCustomLabel}
                outerRadius={120}
                fill="#8884d8"
                dataKey="value"
              >
                {data.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip content={<CustomTooltip />} />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        );

      case 'table':
        return (
          <Table
            columns={tableColumns}
            dataSource={data}
            pagination={false}
            size="middle"
            rowKey={(record) => record.name}
          />
        );

      default: // bar
        return (
          <ResponsiveContainer width="100%" height={height}>
            <BarChart data={data} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <Bar dataKey="value" fill="#1890ff" radius={[4, 4, 0, 0]}>
                {data.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Bar>
              {data.some(item => item.avg_time) && (
                <Bar dataKey="avg_time" fill="#52c41a" name="平均时间(小时)" radius={[4, 4, 0, 0]} />
              )}
            </BarChart>
          </ResponsiveContainer>
        );
    }
  };

  return (
    <div className="reports-charts">
      {renderChart()}
    </div>
  );
};

export default ReportsCharts;