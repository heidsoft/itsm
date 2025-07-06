'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { mockChangesData } from '../../lib/mock-data';

const ChangeSuccessPage = () => {

    // Aggregate data for charts
    const changesByStatus = mockChangesData.reduce((acc, change) => {
        acc[change.status] = (acc[change.status] || 0) + 1;
        return acc;
    }, {});

    const statusData = Object.keys(changesByStatus).map(status => ({
        name: status,
        value: changesByStatus[status]
    }));

    const changesByType = mockChangesData.reduce((acc, change) => {
        acc[change.type] = (acc[change.type] || 0) + 1;
        return acc;
    }, {});

    const typeData = Object.keys(changesByType).map(type => ({
        type,
        count: changesByType[type]
    }));

    const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042'];

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
                                data={statusData}
                                cx="50%"
                                cy="50%"
                                labelLine={false}
                                outerRadius={80}
                                fill="#8884d8"
                                dataKey="value"
                                nameKey="name"
                                label
                            >
                                {statusData.map((entry, index) => (
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
                        <BarChart data={typeData}>
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
};

export default ChangeSuccessPage;