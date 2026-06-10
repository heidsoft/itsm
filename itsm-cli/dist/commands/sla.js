import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const SlaStatsCommand = () => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getSLAStats().then(d => setData(d)).catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: "Loading SLA stats..." });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "\uD83D\uDCCA SLA Stats"),
        React.createElement(Newline, null),
        React.createElement(Text, null, JSON.stringify(data, null, 2))));
};
export const SlaOverdueCommand = () => {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getSLAOverdue()
            .then(d => { setItems(d || []); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading overdue tickets..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "red" },
            "\u26A0 Overdue tickets (",
            items.length,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'ticket_number', header: 'Number', width: 12 },
                { key: 'title', header: 'Title', width: 40 },
                { key: 'priority', header: 'Prio', width: 8 },
                { key: 'sla_status', header: 'SLA', width: 12 },
                { key: 'assignee_name', header: 'Assignee', width: 16 },
            ] })));
};
