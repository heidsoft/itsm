
'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';

// --- 模拟数据 ---

// 1. 多云资源分布数据
const multiCloudResourceData = [
    { name: '虚拟机', '阿里云': 40, '腾讯云': 24, '私有云': 20 },
    { name: '数据库', '阿里云': 22, '腾讯云': 18, '私有云': 30 },
    { name: '存储桶', '阿里云': 55, '腾讯云': 32, '私有云': 15 },
    { name: '网络', '阿里云': 30, '腾讯云': 20, '私有云': 25 },
];

// 2. 资源健康状态数据
const resourceHealthData = [
    { name: '运行中', value: 400 },
    { name: '已停止', value: 78 },
    { name: '警告', value: 32 },
    { name: '错误', value: 15 },
];

const COLORS: Record<string, string> = { '运行中': '#22c55e', '已停止': '#f97316', '警告': '#facc15', '错误': '#ef4444' };

// --- 图表组件 ---

// 1. 资源分布条形图
export const ResourceDistributionChart = () => (
    <div className="h-96">
        <ResponsiveContainer width="100%" height="100%">
            <BarChart data={multiCloudResourceData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
                <XAxis dataKey="name" stroke="#6b7280" />
                <YAxis stroke="#6b7280" />
                <Tooltip wrapperClassName="rounded-lg shadow-lg bg-white" cursor={{ fill: 'rgba(243, 244, 246, 0.5)' }} />
                <Legend />
                <Bar dataKey="阿里云" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                <Bar dataKey="腾讯云" fill="#10b981" radius={[4, 4, 0, 0]} />
                <Bar dataKey="私有云" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
            </BarChart>
        </ResponsiveContainer>
    </div>
);

// 2. 资源健康状态饼图
export const ResourceHealthPieChart = () => (
    <div className="h-96">
        <ResponsiveContainer width="100%" height="100%">
            <PieChart>
                <Pie
                    data={resourceHealthData}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    outerRadius={120}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${percent ? (percent * 100).toFixed(0) : 0}%`}
                >
                    {resourceHealthData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[entry.name]} />
                    ))}
                </Pie>
                <Tooltip wrapperClassName="rounded-lg shadow-lg bg-white" />
                <Legend />
            </PieChart>
        </ResponsiveContainer>
    </div>
);
