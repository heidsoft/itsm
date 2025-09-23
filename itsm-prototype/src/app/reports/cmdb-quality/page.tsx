'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

const data = [
  { name: 'Servers', completeness: 95, accuracy: 90 },
  { name: 'Applications', completeness: 80, accuracy: 85 },
  { name: 'Databases', completeness: 90, accuracy: 92 },
  { name: 'Network Devices', completeness: 85, accuracy: 88 },
];

export default function CMDBQualityReport() {
  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <h2 className="text-4xl font-bold text-gray-800">CMDB Data Quality Report</h2>
        <p className="text-gray-500 mt-1">This report shows the data quality of the Configuration Management Database (CMDB).</p>
      </header>
      <div className="bg-white p-6 rounded-lg shadow-md">
        <h3 className="text-xl font-semibold text-gray-800 mb-4">CMDB Data Quality</h3>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="completeness" fill="#8884d8" />
            <Bar dataKey="accuracy" fill="#82ca9d" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}