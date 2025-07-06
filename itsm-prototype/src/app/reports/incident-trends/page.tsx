'use client';

import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, LineChart, Line } from 'recharts';
import { mockIncidentsData } from '../../lib/mock-data';

const IncidentTrendsPage = () => {

    // Aggregate data for charts
    const incidentsByDay = mockIncidentsData.reduce((acc, incident) => {
        const date = new Date(incident.createdAt).toLocaleDateString();
        acc[date] = (acc[date] || 0) + 1;
        return acc;
    }, {});

    const chartData = Object.keys(incidentsByDay).map(date => ({
        date,
        count: incidentsByDay[date]
    }));

    const incidentsByPriority = mockIncidentsData.reduce((acc, incident) => {
        acc[incident.priority] = (acc[incident.priority] || 0) + 1;
        return acc;
    }, {});

    const priorityData = Object.keys(incidentsByPriority).map(priority => ({
        priority,
        count: incidentsByPriority[priority]
    }));

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">Incident Trends Report</h2>
                <p className="text-gray-500 mt-1">This report shows the trend of incidents over time and by priority.</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                <div className="bg-white p-6 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Incidents per Day</h3>
                    <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={chartData}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="date" />
                            <YAxis />
                            <Tooltip />
                            <Legend />
                            <Line type="monotone" dataKey="count" stroke="#8884d8" activeDot={{ r: 8 }} />
                        </LineChart>
                    </ResponsiveContainer>
                </div>

                <div className="bg-white p-6 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-800 mb-4">Incidents by Priority</h3>
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

export default IncidentTrendsPage;