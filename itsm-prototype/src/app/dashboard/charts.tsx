'use client';

import React from 'react';
import { Card } from 'antd';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';

// 多云资源分布图
export const ResourceDistributionChart: React.FC = () => {
  const data = [
    { name: '阿里云', value: 400 },
    { name: '腾讯云', value: 300 },
    { name: '私有云', value: 300 },
    { name: 'AWS', value: 200 },
  ];

  return (
    <ResponsiveContainer width='100%' height={300}>
      <BarChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        <CartesianGrid strokeDasharray='3 3' vertical={false} />
        <XAxis dataKey='name' />
        <YAxis />
        <Tooltip />
        <Legend />
        <Bar dataKey='value' fill='#8884d8' radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
};

// 资源健康状态饼图
export const ResourceHealthPieChart: React.FC = () => {
  const data = [
    { name: '运行中', value: 500, color: '#22c55e' },
    { name: '已停止', value: 150, color: '#64748b' },
    { name: '警告', value: 80, color: '#f59e0b' },
    { name: '错误', value: 30, color: '#ef4444' },
  ];

  return (
    <ResponsiveContainer width='100%' height={300}>
      <PieChart margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        <Pie
          data={data}
          cx='50%'
          cy='50%'
          labelLine={false}
          outerRadius={100}
          fill='#8884d8'
          dataKey='value'
          label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
};

// 默认导出
const Charts = {
  ResourceDistributionChart,
  ResourceHealthPieChart,
};

export default Charts;
