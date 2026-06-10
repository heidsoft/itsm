import React, { useState, useEffect } from 'react';
import { Box, Text, Newline } from 'ink';
import { apiClient } from '../lib/api-client.js';
import { Table } from '../components/table.js';
import { Spinner } from '../components/spinner.js';
export const IncidentListCommand = ({ status, priority, page = 1 }) => {
    const [items, setItems] = useState([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.listIncidents({ status, priority, page, pageSize: 20 })
            .then(r => { setItems(r.items || []); setTotal(r.total); setLoading(false); })
            .catch(e => { setErr(e.message); setLoading(false); });
    }, []);
    if (loading)
        return React.createElement(Spinner, { text: "Loading incidents..." });
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    return (React.createElement(Box, { flexDirection: "column" },
        React.createElement(Text, { bold: true, color: "cyan" },
            "Incidents (",
            total,
            " total)"),
        React.createElement(Text, { dimColor: true },
            "filters: ",
            status ? `status=${status}` : 'all-status',
            " ",
            priority ? `priority=${priority}` : ''),
        React.createElement(Newline, null),
        React.createElement(Table, { data: items, columns: [
                { key: 'number', header: 'Number', width: 12 },
                { key: 'title', header: 'Title', width: 35 },
                { key: 'status', header: 'Status', width: 12 },
                { key: 'priority', header: 'Prio', width: 8 },
                { key: 'assignee_name', header: 'Assignee', width: 16 },
                { key: 'sla_status', header: 'SLA', width: 10 },
            ] }),
        React.createElement(Newline, null),
        React.createElement(Text, { dimColor: true },
            "Page ",
            page,
            ". Use --page N for more, or `itsm incident <id>` for details.")));
};
export const IncidentViewCommand = ({ id }) => {
    const [data, setData] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.getIncident(id)
            .then(d => setData(d))
            .catch(e => setErr(e.message));
    }, [id]);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!data)
        return React.createElement(Spinner, { text: `Loading incident ${id}...` });
    const rows = [
        ['Number', data.number],
        ['Title', data.title],
        ['Status', data.status],
        ['Priority', data.priority],
        ['Category', data.category],
        ['Severity', data.severity],
        ['Assignee', data.assignee_name],
        ['Reporter', data.reporter_name],
        ['CMDB CI', data.cmdb_ci_name],
        ['SLA', data.sla_status],
        ['Due', data.due_date],
        ['Created', data.created_at],
        ['Updated', data.updated_at],
        ['Resolved', data.resolved_at],
    ];
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" },
            "\uD83D\uDCCB Incident ",
            data.number),
        React.createElement(Newline, null),
        rows.filter(([, v]) => v !== undefined && v !== '').map(([k, v]) => (React.createElement(Box, { key: k },
            React.createElement(Text, { color: "gray" },
                k.padEnd(12),
                " "),
            React.createElement(Text, null, String(v))))),
        data.description && (React.createElement(Box, { flexDirection: "column", marginTop: 1 },
            React.createElement(Text, { color: "gray" }, "Description"),
            React.createElement(Text, null, data.description)))));
};
export const IncidentTriageCommand = ({ title, description }) => {
    const [result, setResult] = useState(null);
    const [err, setErr] = useState('');
    useEffect(() => {
        apiClient.aiTriage({ title, description })
            .then(r => setResult(r))
            .catch(e => setErr(e.message));
    }, []);
    if (err)
        return React.createElement(Text, { color: "red" },
            "\u2717 ",
            err);
    if (!result)
        return React.createElement(Spinner, { text: "AI is triaging..." });
    return (React.createElement(Box, { flexDirection: "column", padding: 1 },
        React.createElement(Text, { bold: true, color: "cyan" }, "\uD83E\uDD16 AI Triage Result"),
        React.createElement(Newline, null),
        React.createElement(Text, null, JSON.stringify(result, null, 2))));
};
