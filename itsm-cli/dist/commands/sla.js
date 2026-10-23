import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const SlaStatsCommand = () => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getSLAStats().then((d) => setData(d)).catch((e) => setErr(e.message));
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
        apiClient.getSLAViolations()
            .then((d) => {
            const raw = d.items ?? (Array.isArray(d) ? d : []);
            setItems(raw);
            setLoading(false);
        })
            .catch((e) => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading SLA violations..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "red" },
            "\u26A0 SLA violations (",
            items.length,
            ")"),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'ticketNumber', header: 'Number', width: 12 },
                { key: 'ticketTitle', header: 'Title', width: 40 },
                { key: 'violationType', header: 'Type', width: 14 },
                { key: 'breachedAt', header: 'Breached', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Raw payload: ",
            JSON.stringify(items).slice(0, 400))));
};
