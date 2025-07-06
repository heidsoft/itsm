'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

const slaData = [
    { name: 'Incident Resolution', breached: 5, met: 95 },
    { name: 'Service Request Fulfillment', breached: 2, met: 98 },
    { name: 'Uptime Guarantee', breached: 1, met: 99 },
];

const SlaPerformancePage = () => {
    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">SLA Performance Report</h2>
                <p className="text-gray-500 mt-1">This report shows the performance of Service Level Agreements (SLAs).</p>
            </header>

            <div className="bg-white p-6 rounded-lg shadow-md">
                <h3 className="text-xl font-semibold text-gray-800 mb-4">SLA Status</h3>
                <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={slaData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="name" />
                        <YAxis />
                        <Tooltip />
                        <Legend />
                        <Bar dataKey="met" stackId="a" fill="#82ca9d" />
                        <Bar dataKey="breached" stackId="a" fill="#ffc658" />
                    </BarChart>
                </ResponsiveContainer>
            </div>
        </div>
    );
};

export default SlaPerformancePage;