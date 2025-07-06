'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { mockProblemsData } from '../../lib/mock-data';

const ProblemEfficiencyPage = () => {

    // Aggregate data for charts
    const problemsByStatus = mockProblemsData.reduce((acc, problem) => {
        acc[problem.status] = (acc[problem.status] || 0) + 1;
        return acc;
    }, {});

    const statusData = Object.keys(problemsByStatus).map(status => ({
        name: status,
        value: problemsByStatus[status]
    }));

    const problemsByPriority = mockProblemsData.reduce((acc, problem) => {
        acc[problem.priority] = (acc[problem.priority] || 0) + 1;
        return acc;
    }, {});

    const priorityData = Object.keys(problemsByPriority).map(priority => ({
        priority,
        count: problemsByPriority[priority]
    }));

    const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042'];

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">Problem Management Efficiency Report</h2>
                <p className="text-gray-500 mt-1">This report shows the efficiency of the problem management process.</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                <div className="bg-white p-6 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Problems by Status</h3>
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
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Problems by Priority</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={priorityData}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="priority" />
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

export default ProblemEfficiencyPage;