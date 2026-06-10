import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Spinner } from '../components/spinner.js';
export const DashboardOverviewCommand = () => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getDashboardOverview().then(d => setData(d)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: "Loading dashboard..." });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "\uD83C\uDFE0 Dashboard Overview"),
        React.createElement(Newline, null),
        React.createElement(Text, null, JSON.stringify(data, null, 2))));
};
