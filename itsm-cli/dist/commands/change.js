import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const ChangeListCommand = ({ status, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listChanges({ status, page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading changes..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Changes (",
            total,
            " total)"),
        status && React.createElement(Text, { dimColor: true },
            "status filter: ",
            status),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'number', header: 'Number', width: 12 },
                { key: 'title', header: 'Title', width: 35 },
                { key: 'type', header: 'Type', width: 12 },
                { key: 'risk_level', header: 'Risk', width: 8 },
                { key: 'status', header: 'Status', width: 12 },
                { key: 'scheduled_at', header: 'Scheduled', width: 20 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true }, "Use `itsm change <id>` for details.")));
};
export const ChangeViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getChange(id).then(d => setData(d)).catch(e => setErr(e.message));
    }, [id]);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: `Loading change ${id}...` });
    const rows = [
        ['Number', data.number], ['Title', data.title], ['Type', data.type],
        ['Risk', data.risk_level], ['Status', data.status], ['Priority', data.priority],
        ['Assignee', data.assignee_name], ['Implementer', data.implementer_name],
        ['Scheduled', data.scheduled_at], ['Created', data.created_at], ['Updated', data.updated_at],
    ];
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDD27 Change ",
            data.number),
        React.createElement(Newline, null),
        rows.filter(([, v]) => v).map(([k, v]) => (React.createElement(Box, { key: k },
            React.createElement(Text, { color: "gray" },
                k.padEnd(12),
                " "),
            React.createElement(Text, null, v)))),
        data.description && React.createElement(Box, { flexDirection: "column", marginTop: 1 },
            React.createElement(Text, { color: "gray" }, "Description"),
            React.createElement(Text, null, data.description)),
        data.rollback_plan && React.createElement(Box, { flexDirection: "column", marginTop: 1 },
            React.createElement(Text, { color: "gray" }, "Rollback"),
            React.createElement(Text, null, data.rollback_plan))));
};
