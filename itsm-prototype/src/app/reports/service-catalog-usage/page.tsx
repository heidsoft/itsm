'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { mockRequestsData } from '../../lib/mock-data';

const ServiceCatalogUsagePage = () => {

    // Aggregate data for charts
    const requestsByService = mockRequestsData.reduce((acc, request) => {
        acc[request.serviceName] = (acc[request.serviceName] || 0) + 1;
        return acc;
    }, {});

    const serviceData = Object.keys(requestsByService).map(service => ({
        name: service,
        value: requestsByService[service]
    }));

    const requestsByStatus = mockRequestsData.reduce((acc, request) => {
        acc[request.status] = (acc[request.status] || 0) + 1;
        return acc;
    }, {});

    const statusData = Object.keys(requestsByStatus).map(status => ({
        status,
        count: requestsByStatus[status]
    }));

    const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042'];

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">Service Catalog Usage Report</h2>
                <p className="text-gray-500 mt-1">This report shows the usage of the service catalog.</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                <div className="bg-white p-6 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Requests by Service</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                            <Pie
                                data={serviceData}
                                cx="50%"
                                cy="50%"
                                labelLine={false}
                                outerRadius={80}
                                fill="#8884d8"
                                dataKey="value"
                                nameKey="name"
                                label
                            >
                                {serviceData.map((entry, index) => (
                                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                ))}
                            </Pie>
                            <Tooltip />
                            <Legend />
                        </PieChart>
                    </ResponsiveContainer>
                </div>

                <div className="bg-white p-6 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Requests by Status</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={statusData}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="status" />
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

export default ServiceCatalogUsagePage;