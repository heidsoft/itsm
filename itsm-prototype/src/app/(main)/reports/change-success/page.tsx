'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { mockChangesData } from '@/app/lib/mock-data';

// 按状态统计变更
const statusData = mockChangesData.reduce((acc, change) => {
  const status = change.status;
  acc[status] = (acc[status] || 0) + 1;
  return acc;
}, {} as Record<string, number>);

const pieData = Object.entries(statusData).map(([name, value]) => ({
  name,
  value,
}));

// 按类型统计变更
const typeData = mockChangesData.reduce((acc, change) => {
  const type = change.type;
  acc[type] = (acc[type] || 0) + 1;
  return acc;
}, {} as Record<string, number>);

const barData = Object.entries(typeData).map(([type, count]) => ({
  type,
  count,
}));

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042'];

export default function ChangeSuccessReport() {
  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <h2 className="text-4xl font-bold text-gray-800">Change Success Report</h2>
        <p className="text-gray-500 mt-1">This report shows the success rate of changes and the distribution of change types.</p>
      </header>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">Changes by Status</h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={pieData}
                cx="50%"
                cy="50%"
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
                nameKey="name"
              >
                {pieData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">Changes by Type</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={barData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="type" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="count" fill="#82ca9d" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}